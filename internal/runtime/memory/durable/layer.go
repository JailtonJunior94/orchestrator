package durable

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const (
	projectMemoryDirName = ".aispec"
	memorySubDirName     = "memory"
	projectPageFileName  = "PROJECT.md"
	prdPageFileName      = "MEMORY.md"
	lockFileSuffix       = ".lock"
)

type Scope struct {
	Layer        TargetLayer
	ProjectDir   string
	TasksDir     string
	TaskFileName string
}

func (s Scope) directory() (string, error) {
	switch s.Layer {
	case TargetLayerProject:
		if s.ProjectDir == "" {
			return "", ErrProjectDirMissing
		}
		return filepath.Join(s.ProjectDir, projectMemoryDirName, memorySubDirName), nil
	case TargetLayerPRD, TargetLayerTask:
		if s.TasksDir == "" {
			return "", ErrTasksDirMissing
		}
		return filepath.Join(s.TasksDir, memorySubDirName), nil
	default:
		return "", ErrLayerUndefined
	}
}

func (s Scope) fileName() (string, error) {
	switch s.Layer {
	case TargetLayerProject:
		return projectPageFileName, nil
	case TargetLayerPRD:
		return prdPageFileName, nil
	case TargetLayerTask:
		if s.TaskFileName == "" {
			return "", ErrTaskFileNameMissing
		}
		return filepath.Base(s.TaskFileName), nil
	default:
		return "", ErrLayerUndefined
	}
}

func (s Scope) activePath() (string, error) {
	dir, err := s.directory()
	if err != nil {
		return "", err
	}
	name, err := s.fileName()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func (s Scope) sidecarPath(suffix string) (string, error) {
	active, err := s.activePath()
	if err != nil {
		return "", err
	}
	base := strings.TrimSuffix(active, ".md")
	return base + suffix, nil
}

func (s Scope) ActivePath() (string, error) {
	return s.activePath()
}

func (s Scope) SidecarPath(suffix string) (string, error) {
	return s.sidecarPath(suffix)
}

func (s Scope) lockPath() (string, error) {
	return s.sidecarPath(lockFileSuffix)
}

type ConsolidationResult struct {
	Added        []Identity
	Unchanged    []Identity
	Restored     []Identity
	Contradicted []Identity
}

type Layer interface {
	Read(ctx context.Context, scope Scope) ([]Fact, HumanBlock, error)
	Consolidate(ctx context.Context, scope Scope, newFacts []Fact) (ConsolidationResult, error)
	Archive(ctx context.Context, scope Scope, ids []Identity) error
	Promote(ctx context.Context, from, to Scope, id Identity) error
}

type layer struct {
	filesystem fs.FileSystem
	page       Page
	catalog    *Catalog
	locker     LayerLocker
}

var _ Layer = (*layer)(nil)

func NewLayer(filesystem fs.FileSystem) Layer {
	return NewLayerWithLocker(filesystem, DefaultLayerLocker)
}

func NewLayerWithLocker(filesystem fs.FileSystem, locker LayerLocker) Layer {
	return &layer{
		filesystem: filesystem,
		page:       NewMarkdownPage(),
		catalog:    NewCatalog(),
		locker:     locker,
	}
}

func (l *layer) Read(_ context.Context, scope Scope) ([]Fact, HumanBlock, error) {
	activePath, err := scope.activePath()
	if err != nil {
		return nil, HumanBlock{}, err
	}

	all, human, err := l.readPage(activePath)
	if err != nil {
		return nil, HumanBlock{}, err
	}

	active := make([]Fact, 0, len(all))
	for _, f := range all {
		if f.State == FactStateArchived || f.State == FactStatePromoted {
			continue
		}
		active = append(active, f)
	}
	return active, human, nil
}

func (l *layer) Consolidate(_ context.Context, scope Scope, newFacts []Fact) (ConsolidationResult, error) {
	for _, f := range newFacts {
		if err := l.catalog.ValidateFact(f); err != nil {
			return ConsolidationResult{}, err
		}
	}

	lockPath, err := scope.lockPath()
	if err != nil {
		return ConsolidationResult{}, err
	}
	release, err := l.locker.Lock(lockPath)
	if err != nil {
		return ConsolidationResult{}, err
	}
	defer func() { _ = release() }()

	activePath, err := scope.activePath()
	if err != nil {
		return ConsolidationResult{}, err
	}

	existing, human, err := l.readPage(activePath)
	if err != nil {
		return ConsolidationResult{}, err
	}

	merged, result := l.mergeFacts(existing, newFacts)

	if err := l.writePage(scope, activePath, merged, human); err != nil {
		return ConsolidationResult{}, err
	}

	return result, nil
}

func (l *layer) Archive(_ context.Context, scope Scope, ids []Identity) error {
	lockPath, err := scope.lockPath()
	if err != nil {
		return err
	}
	release, err := l.locker.Lock(lockPath)
	if err != nil {
		return err
	}
	defer func() { _ = release() }()

	activePath, err := scope.activePath()
	if err != nil {
		return err
	}

	existing, human, err := l.readPage(activePath)
	if err != nil {
		return err
	}

	toArchive := make(map[Identity]bool, len(ids))
	for _, id := range ids {
		toArchive[id] = true
	}

	changed := false
	for i := range existing {
		if toArchive[existing[i].Identity] && existing[i].State != FactStateArchived {
			existing[i].State = FactStateArchived
			changed = true
		}
	}

	if !changed {
		return nil
	}

	return l.writePage(scope, activePath, existing, human)
}

func (l *layer) Promote(_ context.Context, from, to Scope, id Identity) error {
	fromLockPath, err := from.lockPath()
	if err != nil {
		return err
	}
	toLockPath, err := to.lockPath()
	if err != nil {
		return err
	}

	releaseFns, err := l.lockInOrder(fromLockPath, toLockPath)
	if err != nil {
		return err
	}
	defer func() {
		for _, release := range releaseFns {
			_ = release()
		}
	}()

	fromActivePath, err := from.activePath()
	if err != nil {
		return err
	}
	fromFacts, fromHuman, err := l.readPage(fromActivePath)
	if err != nil {
		return err
	}

	idx := -1
	for i, f := range fromFacts {
		if f.Identity == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("durable: promote %s: %w", id.Key, ErrFactNotFound)
	}
	if fromFacts[idx].Durability != DurabilityDurable {
		return ErrPromotionWithoutMark
	}

	promoted := fromFacts[idx]
	fromFacts[idx].State = FactStatePromoted

	if err := l.writePage(from, fromActivePath, fromFacts, fromHuman); err != nil {
		return err
	}

	toActivePath, err := to.activePath()
	if err != nil {
		return err
	}
	toFacts, toHuman, err := l.readPage(toActivePath)
	if err != nil {
		return err
	}

	promoted.State = FactStateActive
	merged, _ := l.mergeFacts(toFacts, []Fact{promoted})

	return l.writePage(to, toActivePath, merged, toHuman)
}

func (l *layer) lockInOrder(pathA, pathB string) ([]func() error, error) {
	first, second := pathA, pathB
	if second < first {
		first, second = second, first
	}

	releaseFirst, err := l.locker.Lock(first)
	if err != nil {
		return nil, err
	}
	if first == second {
		return []func() error{releaseFirst}, nil
	}

	releaseSecond, err := l.locker.Lock(second)
	if err != nil {
		_ = releaseFirst()
		return nil, err
	}
	return []func() error{releaseFirst, releaseSecond}, nil
}

func (l *layer) mergeFacts(existing []Fact, candidates []Fact) ([]Fact, ConsolidationResult) {
	merged := make([]Fact, len(existing))
	copy(merged, existing)

	result := ConsolidationResult{}

	for _, candidate := range candidates {
		exactIdx := -1
		activeKeyIdx := -1
		for i, e := range merged {
			if e.Identity == candidate.Identity {
				exactIdx = i
				break
			}
			if e.State == FactStateActive && e.Identity.Key == candidate.Identity.Key {
				activeKeyIdx = i
			}
		}

		if exactIdx != -1 {
			if merged[exactIdx].State == FactStateArchived {
				merged[exactIdx].State = FactStateActive
				result.Restored = append(result.Restored, candidate.Identity)
			} else {
				result.Unchanged = append(result.Unchanged, candidate.Identity)
			}
			continue
		}

		if activeKeyIdx != -1 {
			contradictedIdentity := merged[activeKeyIdx].Identity
			merged[activeKeyIdx].State = FactStateContradicted

			next := candidate
			next.State = FactStateActive
			next.Links = append(append([]Link{}, next.Links...), Link{Type: LinkTypeContradicts, Target: contradictedIdentity})
			merged = append(merged, next)

			result.Contradicted = append(result.Contradicted, contradictedIdentity)
			result.Added = append(result.Added, candidate.Identity)
			continue
		}

		next := candidate
		next.State = FactStateActive
		merged = append(merged, next)
		result.Added = append(result.Added, candidate.Identity)
	}

	return merged, result
}

func (l *layer) readPage(path string) ([]Fact, HumanBlock, error) {
	if !l.filesystem.Exists(path) {
		return nil, HumanBlock{}, nil
	}
	content, err := l.filesystem.ReadFile(path)
	if err != nil {
		return nil, HumanBlock{}, fmt.Errorf("durable: read layer page %s: %w", path, err)
	}
	facts, human, err := l.page.Parse(content)
	if err != nil {
		return nil, HumanBlock{}, fmt.Errorf("durable: parse layer page %s: %w", path, err)
	}
	return facts, human, nil
}

func (l *layer) writePage(scope Scope, path string, facts []Fact, human HumanBlock) error {
	rendered, err := l.page.Serialize(facts, human)
	if err != nil {
		return err
	}

	projectDir := scope.ProjectDir
	if projectDir == "" {
		projectDir = scope.TasksDir
	}
	if projectDir != "" {
		if err := fs.RefuseExternalSymlink(l.filesystem, projectDir, path, false); err != nil {
			return fmt.Errorf("durable: write layer page: %w", err)
		}
	}

	if err := l.filesystem.WriteFileAtomic(path, rendered); err != nil {
		return fmt.Errorf("durable: write layer page %s: %w", path, err)
	}
	return nil
}
