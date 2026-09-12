package durable

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

const (
	DefaultCompactionLineLimit = 150
	DefaultCompactionByteLimit = 12288
)

type MemoryScope struct {
	TaskFileName string
	WindowClass  specs.WindowClass
}

type MemoryContext struct {
	Block          string
	Omitted        int
	Contradicted   int
	Unreadable     int
	FactsByLayer   map[string]int
	BudgetByLayer  map[string]int
	BuildLatencyMs int64
}

type SessionFacts struct {
	TaskFileName    string
	ExitStatus      string
	EventsCount     int
	ToolCalls       int
	DeclaredSection string
	SessionID       string
	CLI             string
}

type MemoryReport struct {
	Writes             int
	Redactions         int
	Archived           int
	BatonClaimed       bool
	BatonRefusalReason string
	Contradictions     int
	Compactions        int
	Promoted           int
	WritesByLayer      map[string]int
	ArchivedByLayer    map[string]int
	RecordLatencyMs    int64
}

type FacadeConfig struct {
	ProjectDir   string
	TasksDir     string
	Budget       BudgetConfig
	Sanitization SanitizationConfig
	Compaction   CompactionConfig
	LeaseTTL     time.Duration
}

type Facade struct {
	filesystem   fs.FileSystem
	layer        Layer
	relevance    RelevancePolicy
	budget       BudgetPolicy
	sanitization SanitizationPolicy
	compaction   CompactionPolicy
	durability   DurabilityPolicy
	handoff      *ContinuityHandoff
	cfg          FacadeConfig
}

func NewFacade(filesystem fs.FileSystem, cfg FacadeConfig) *Facade {
	return NewFacadeWithLayer(filesystem, NewLayer(filesystem), cfg)
}

func NewFacadeWithLayer(filesystem fs.FileSystem, layer Layer, cfg FacadeConfig) *Facade {
	return NewFacadeWithLayerAndLocker(filesystem, layer, DefaultLayerLocker, cfg)
}

func NewFacadeWithLayerAndLocker(filesystem fs.FileSystem, layer Layer, locker LayerLocker, cfg FacadeConfig) *Facade {
	handoffScope := Scope{Layer: TargetLayerPRD, TasksDir: cfg.TasksDir}
	handoffProjectDir := cfg.ProjectDir
	if handoffProjectDir == "" {
		handoffProjectDir = cfg.TasksDir
	}
	return &Facade{
		filesystem:   filesystem,
		layer:        layer,
		relevance:    DefaultRelevancePolicy,
		budget:       DefaultBudgetPolicy,
		sanitization: DefaultSanitizationPolicy,
		compaction:   DefaultCompactionPolicy,
		durability:   DefaultDurabilityPolicy,
		handoff:      NewContinuityHandoff(DefaultLeasePolicy, filesystem, locker, handoffScope, handoffProjectDir),
		cfg:          cfg,
	}
}

func (f *Facade) scopeForLayer(layer TargetLayer, taskFileName string) Scope {
	taskName := ""
	if layer == TargetLayerTask {
		taskName = taskFileName
	}
	return Scope{
		Layer:        layer,
		ProjectDir:   f.cfg.ProjectDir,
		TasksDir:     f.cfg.TasksDir,
		TaskFileName: taskName,
	}
}

func (f *Facade) BuildContext(ctx context.Context, scope MemoryScope) (MemoryContext, error) {
	start := time.Now()

	var allFacts []Fact
	unreadable := 0
	factsByLayer := make(map[string]int, 3)

	for _, layer := range []TargetLayer{TargetLayerTask, TargetLayerPRD, TargetLayerProject} {
		facts, _, err := f.layer.Read(ctx, f.scopeForLayer(layer, scope.TaskFileName))
		if err != nil {
			switch {
			case errors.Is(err, ErrProjectDirMissing), errors.Is(err, ErrTasksDirMissing), errors.Is(err, ErrTaskFileNameMissing):
				continue
			case errors.Is(err, ErrPageUnreadable):
				unreadable++
				log.Printf("durable: isolated unreadable page on layer %s: %v", layer, err)
				continue
			default:
				return MemoryContext{}, fmt.Errorf("durable: build context: %w", err)
			}
		}
		factsByLayer[layer.String()] = len(facts)
		allFacts = append(allFacts, facts...)
	}

	if len(allFacts) == 0 {
		return MemoryContext{Unreadable: unreadable, FactsByLayer: factsByLayer, BuildLatencyMs: time.Since(start).Milliseconds()}, nil
	}

	budget := f.budget.Resolve(scope.WindowClass, f.cfg.Budget)
	ranked := f.relevance.Rank(allFacts, scope.TaskFileName)
	allocation := f.budget.Allocate(budget, ranked)

	contradicted := 0
	for _, fct := range allocation.Selected {
		if fct.State == FactStateContradicted {
			contradicted++
		}
	}

	return MemoryContext{
		Block:          f.renderBlock(allocation.Selected),
		Omitted:        len(allocation.Omitted),
		Contradicted:   contradicted,
		Unreadable:     unreadable,
		FactsByLayer:   factsByLayer,
		BudgetByLayer:  allocation.UsedByLayer,
		BuildLatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

func (f *Facade) renderBlock(facts []Fact) string {
	if len(facts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Durable Memory\n")
	for _, fct := range facts {
		b.WriteString("\n- ")
		b.WriteString(fct.Content)
		if fct.State == FactStateContradicted {
			b.WriteString(" [CONTRADICTED]")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func (f *Facade) RecordSession(ctx context.Context, in SessionFacts) (MemoryReport, error) {
	start := time.Now()
	facts, err := f.deriveFacts(in)
	if err != nil {
		return MemoryReport{}, fmt.Errorf("durable: record session: %w", err)
	}
	if len(facts) == 0 {
		return MemoryReport{}, nil
	}

	sanitized := make([]Fact, 0, len(facts))
	redactionCount := 0
	for _, fct := range facts {
		result, sanErr := f.sanitization.Sanitize(fct.Content, f.cfg.Sanitization)
		if sanErr != nil {
			return MemoryReport{}, fmt.Errorf("durable: record session: %w", sanErr)
		}
		fct.Content = result.Content
		redactionCount += len(result.Redactions)
		sanitized = append(sanitized, fct)
	}

	batonClaimed := false
	batonRefusalReason := ""
	record, claimErr := f.claimBaton(time.Now())
	if claimErr != nil {
		batonRefusalReason = claimErr.Error()
		log.Printf("durable: baton claim refused, proceeding with fact consolidation (RF precedence: no-loss over single-baton-owner): %v", claimErr)
	} else {
		batonClaimed = record.Outcome == LeaseOutcomeGranted ||
			record.Outcome == LeaseOutcomeRenewed ||
			record.Outcome == LeaseOutcomeTransferred
	}

	catalog := NewCatalog()
	byLayer := make(map[TargetLayer][]Fact)
	for _, fct := range sanitized {
		layer, layerErr := catalog.ResolveLayer(fct.Durability)
		if layerErr != nil {
			return MemoryReport{}, fmt.Errorf("durable: record session: %w", layerErr)
		}
		byLayer[layer] = append(byLayer[layer], fct)
	}

	writes := 0
	archived := 0
	contradictions := 0
	compactions := 0
	writesByLayer := make(map[string]int, len(byLayer))
	archivedByLayer := make(map[string]int, len(byLayer))
	for layer, layerFacts := range byLayer {
		scope := f.scopeForLayer(layer, in.TaskFileName)

		if dir, dirErr := scope.directory(); dirErr == nil {
			if mkErr := f.filesystem.MkdirAll(dir); mkErr != nil {
				return MemoryReport{}, fmt.Errorf("durable: record session: ensure layer directory: %w", mkErr)
			}
		}

		consResult, consErr := f.layer.Consolidate(ctx, scope, layerFacts)
		if consErr != nil {
			if errors.Is(consErr, ErrProjectDirMissing) || errors.Is(consErr, ErrTasksDirMissing) || errors.Is(consErr, ErrTaskFileNameMissing) {
				log.Printf("durable: skip layer %s write (no scope resolved): %v", layer, consErr)
				continue
			}
			if errors.Is(consErr, ErrHumanContentNotNormalized) || errors.Is(consErr, ErrHumanBlockInterleaved) {
				log.Printf("durable: skip layer %s write (human content round-trip violation reported, not rewritten): %v", layer, consErr)
				continue
			}
			return MemoryReport{}, fmt.Errorf("durable: record session: %w", consErr)
		}
		writes += len(layerFacts)
		writesByLayer[layer.String()] = len(layerFacts)
		contradictions += len(consResult.Contradicted)

		active, human, readErr := f.layer.Read(ctx, scope)
		if readErr != nil {
			log.Printf("durable: skip compaction for layer %s: %v", layer, readErr)
			continue
		}
		result, compErr := f.compaction.Compact(active, human, f.resolveCompactionConfig(), in.TaskFileName)
		if compErr != nil {
			log.Printf("durable: compaction limit unreachable on layer %s: %v", layer, compErr)
			continue
		}
		if len(result.ToArchive) == 0 {
			continue
		}
		if archErr := f.layer.Archive(ctx, scope, result.ToArchive); archErr != nil {
			log.Printf("durable: archive failed on layer %s: %v", layer, archErr)
			continue
		}
		archived += len(result.ToArchive)
		archivedByLayer[layer.String()] = len(result.ToArchive)
		compactions++
	}

	return MemoryReport{
		Writes:             writes,
		Redactions:         redactionCount,
		Archived:           archived,
		BatonClaimed:       batonClaimed,
		BatonRefusalReason: batonRefusalReason,
		Contradictions:     contradictions,
		Compactions:        compactions,
		WritesByLayer:      writesByLayer,
		ArchivedByLayer:    archivedByLayer,
		RecordLatencyMs:    time.Since(start).Milliseconds(),
	}, nil
}

func (f *Facade) resolveCompactionConfig() CompactionConfig {
	cfg := f.cfg.Compaction
	if cfg.LineLimit == 0 {
		cfg.LineLimit = DefaultCompactionLineLimit
	}
	if cfg.ByteLimit == 0 {
		cfg.ByteLimit = DefaultCompactionByteLimit
	}
	return cfg
}

func (f *Facade) claimBaton(now time.Time) (LeaseRecord, error) {
	owner := LeaseOwner(fmt.Sprintf("pid:%d", os.Getpid()))
	ttl := f.cfg.LeaseTTL

	if current, ok := f.handoff.Current(); ok && current.Owner == owner {
		return f.handoff.Renew(owner, ttl, now)
	}
	return f.handoff.Claim(owner, CurrentProcessRef(), ttl, now)
}

func (f *Facade) deriveFacts(in SessionFacts) ([]Fact, error) {
	catalog := NewCatalog()
	subject := strings.TrimSpace(in.TaskFileName)
	if subject == "" {
		subject = "adhoc"
	}
	now := time.Now().UTC().Format(time.RFC3339)

	var facts []Fact

	if strings.TrimSpace(in.TaskFileName) != "" {
		summaryContent := fmt.Sprintf("Exit Status: %s | Events: %d | Tool Calls: %d", in.ExitStatus, in.EventsCount, in.ToolCalls)
		summaryKey, err := catalog.DeriveSemanticKey(StructuredSignal{Kind: "session-summary", Subject: subject})
		if err != nil {
			return nil, err
		}
		facts = append(facts, Fact{
			Identity:   Identity{Key: summaryKey, Hash: catalog.HashContent(summaryContent)},
			Content:    summaryContent,
			Durability: f.durability.Classify("session-summary"),
			Origin:     FactOrigin{Session: in.SessionID, CLI: in.CLI, Task: in.TaskFileName, Date: now},
			State:      FactStateActive,
		})
	}

	if declared := strings.TrimSpace(in.DeclaredSection); declared != "" {
		sectionKey, err := catalog.DeriveSemanticKey(StructuredSignal{Kind: "declared-section", Subject: subject})
		if err != nil {
			return nil, err
		}
		facts = append(facts, Fact{
			Identity:   Identity{Key: sectionKey, Hash: catalog.HashContent(declared)},
			Content:    declared,
			Durability: f.durability.Classify("declared-section"),
			Origin:     FactOrigin{Session: in.SessionID, CLI: in.CLI, Task: in.TaskFileName, Date: now},
			State:      FactStateActive,
		})
	}

	return facts, nil
}
