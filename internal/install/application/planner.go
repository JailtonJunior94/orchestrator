package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// Target describes the resolved destination for an asset/provider pair.
type Target struct {
	TargetPath   string
	ManagedPath  string
	Verification install.VerificationStatus
}

// TargetResolver resolves provider-specific destination paths.
type TargetResolver interface {
	ResolveTarget(ctx context.Context, provider install.Provider, scope install.Scope, asset install.Asset) (Target, error)
}

// ConflictResolver resolves detected conflicts before execution.
type ConflictResolver interface {
	Resolve(ctx context.Context, conflicts []install.Conflict) ([]install.ConflictDecision, error)
}

// PlanInput is the immutable request used to build an operation plan.
type PlanInput struct {
	Operation      install.Operation
	Scope          install.Scope
	Providers      []install.Provider
	Assets         []install.Asset
	Inventory      *install.Inventory
	AssetNames     []string
	AssetKinds     []install.AssetKind
	ConflictPolicy install.ConflictPolicy
	Interactive    bool
}

// PlanSummary aggregates counts required by rendering and auditing.
type PlanSummary struct {
	InstallCount  int
	UpdateCount   int
	RemoveCount   int
	SkippedCount  int
	ConflictCount int
}

// Plan is the deterministic output produced by the planner.
type Plan struct {
	Operation install.Operation
	Scope     install.Scope
	Changes   []install.PlannedChange
	Summary   PlanSummary
}

// Planner builds executable plans from the asset catalog and persisted inventory.
type Planner struct {
	fs              platform.FileSystem
	targetResolver  TargetResolver
	conflictResolve ConflictResolver
}

// NewPlanner creates a planner with provider path resolution and conflict handling.
func NewPlanner(fileSystem platform.FileSystem, targetResolver TargetResolver, conflictResolver ConflictResolver) *Planner {
	return &Planner{
		fs:              fileSystem,
		targetResolver:  targetResolver,
		conflictResolve: conflictResolver,
	}
}

// Plan builds a deterministic plan for the requested operation.
func (p *Planner) Plan(ctx context.Context, input PlanInput) (*Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.fs == nil {
		return nil, errors.New("filesystem must not be nil")
	}
	if p.targetResolver == nil {
		return nil, errors.New("target resolver must not be nil")
	}
	if err := install.ValidateOperation(input.Operation); err != nil {
		return nil, err
	}
	if err := install.ValidateScope(input.Scope); err != nil {
		return nil, err
	}

	providers, err := p.normalizeProviders(input)
	if err != nil {
		return nil, err
	}
	if err := validateKinds(input.AssetKinds); err != nil {
		return nil, err
	}

	assets := filterAssets(input.Assets, providers, input.AssetNames, input.AssetKinds)
	inventoryEntries := inventoryEntriesForScope(input.Inventory, input.Scope)

	changes, conflicts, err := p.planChanges(ctx, input, providers, assets, inventoryEntries)
	if err != nil {
		return nil, err
	}

	if len(conflicts) > 0 {
		changes, err = p.applyConflictPolicy(ctx, input, changes, conflicts)
		if err != nil {
			return nil, err
		}
	}

	sortChanges(changes)
	return &Plan{
		Operation: input.Operation,
		Scope:     input.Scope,
		Changes:   changes,
		Summary:   buildSummary(changes),
	}, nil
}

func (p *Planner) normalizeProviders(input PlanInput) ([]install.Provider, error) {
	if len(input.Providers) > 0 {
		return install.NormalizeProviders(input.Providers)
	}

	providerSet := make(map[install.Provider]struct{})
	for _, asset := range input.Assets {
		for _, provider := range asset.Providers() {
			providerSet[provider] = struct{}{}
		}
	}
	if input.Inventory != nil {
		for _, entry := range input.Inventory.Entries() {
			providerSet[entry.Provider()] = struct{}{}
		}
	}

	providers := make([]install.Provider, 0, len(providerSet))
	for provider := range providerSet {
		providers = append(providers, provider)
	}
	if len(providers) == 0 {
		return nil, install.ErrEmptyProviderSet
	}

	return install.NormalizeProviders(providers)
}

func (p *Planner) planChanges(
	ctx context.Context,
	input PlanInput,
	providers []install.Provider,
	assets []install.Asset,
	inventoryEntries []install.InventoryEntry,
) ([]install.PlannedChange, []install.Conflict, error) {
	inventoryIndex := newInventoryIndex(inventoryEntries)
	changes := make([]install.PlannedChange, 0)
	conflicts := make([]install.Conflict, 0)

	switch input.Operation {
	case install.OperationInstall, install.OperationUpdate, install.OperationList, install.OperationVerify:
		for _, provider := range providers {
			providerAssets := assetsForProvider(assets, provider)
			for _, asset := range providerAssets {
				if err := ctx.Err(); err != nil {
					return nil, nil, err
				}

				target, err := p.targetResolver.ResolveTarget(ctx, provider, input.Scope, asset)
				if err != nil {
					return nil, nil, fmt.Errorf("resolve target for provider %q asset %q: %w", provider, asset.ID(), err)
				}

				entry, managed := inventoryIndex.lookup(provider, asset.ID())
				change, conflict, err := p.buildAssetChange(input.Operation, input.Scope, provider, asset, target, entry, managed)
				if err != nil {
					return nil, nil, err
				}

				if conflict != nil {
					conflicts = append(conflicts, *conflict)
				}
				changes = append(changes, change)
			}
		}
	case install.OperationRemove:
		for _, provider := range providers {
			providerAssets := assetsForProvider(assets, provider)
			for _, asset := range providerAssets {
				if err := ctx.Err(); err != nil {
					return nil, nil, err
				}

				target, err := p.targetResolver.ResolveTarget(ctx, provider, input.Scope, asset)
				if err != nil {
					return nil, nil, fmt.Errorf("resolve target for provider %q asset %q: %w", provider, asset.ID(), err)
				}

				entry, managed := inventoryIndex.lookup(provider, asset.ID())
				change, conflict, err := p.buildRemoveChange(input.Scope, asset, provider, target, entry, managed)
				if err != nil {
					return nil, nil, err
				}

				if conflict != nil {
					conflicts = append(conflicts, *conflict)
				}
				changes = append(changes, change)
			}
		}
	default:
		return nil, nil, fmt.Errorf("unsupported operation %q", input.Operation)
	}

	return changes, conflicts, nil
}

func (p *Planner) buildAssetChange(
	operation install.Operation,
	scope install.Scope,
	provider install.Provider,
	asset install.Asset,
	target Target,
	entry install.InventoryEntry,
	managed bool,
) (install.PlannedChange, *install.Conflict, error) {
	action := install.ActionSkip
	switch operation {
	case install.OperationInstall:
		action = install.ActionInstall
		if managed {
			action = install.ActionUpdate
		}
	case install.OperationUpdate:
		if managed {
			action = install.ActionUpdate
		}
	case install.OperationList, install.OperationVerify:
		action = install.ActionSkip
	}

	conflict, err := p.detectConflict(operation, provider, asset, target, entry, managed)
	if err != nil {
		return install.PlannedChange{}, nil, err
	}
	if operation == install.OperationUpdate && !managed {
		action = install.ActionSkip
	}

	verification := target.Verification
	if verification == "" {
		verification = install.VerificationStatusUnknown
	}
	if managed && entry.Verification() != "" && operation == install.OperationList {
		verification = entry.Verification()
	}

	change, err := install.NewPlannedChange(
		provider,
		scope,
		asset.ID(),
		action,
		asset.SourcePath(),
		target.TargetPath,
		target.ManagedPath,
		conflict,
		verification,
	)
	if err != nil {
		return install.PlannedChange{}, nil, err
	}

	return change, conflict, nil
}

func (p *Planner) buildRemoveChange(
	scope install.Scope,
	asset install.Asset,
	provider install.Provider,
	target Target,
	entry install.InventoryEntry,
	managed bool,
) (install.PlannedChange, *install.Conflict, error) {
	conflict, err := p.detectConflict(install.OperationRemove, provider, asset, target, entry, managed)
	if err != nil {
		return install.PlannedChange{}, nil, err
	}

	action := install.ActionSkip
	managedPath := target.ManagedPath
	if managed {
		action = install.ActionRemove
		managedPath = entry.ManagedPath()
	}

	change, err := install.NewPlannedChange(
		provider,
		scope,
		asset.ID(),
		action,
		asset.SourcePath(),
		target.TargetPath,
		managedPath,
		conflict,
		install.VerificationStatusUnknown,
	)
	if err != nil {
		return install.PlannedChange{}, nil, err
	}

	return change, conflict, nil
}

func (p *Planner) detectConflict(
	operation install.Operation,
	provider install.Provider,
	asset install.Asset,
	target Target,
	entry install.InventoryEntry,
	managed bool,
) (*install.Conflict, error) {
	exists, err := pathExists(p.fs, target.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("stat target %q: %w", target.TargetPath, err)
	}

	switch operation {
	case install.OperationInstall:
		if exists && !managed {
			return &install.Conflict{
				Provider:   provider,
				AssetID:    asset.ID(),
				TargetPath: target.TargetPath,
				Managed:    false,
				Reason:     "target already exists and is not managed by ORQ",
			}, nil
		}
	case install.OperationUpdate:
		if !managed {
			return &install.Conflict{
				Provider:   provider,
				AssetID:    asset.ID(),
				TargetPath: target.TargetPath,
				Managed:    false,
				Reason:     "asset is not managed by ORQ",
			}, nil
		}
		if exists && entry.ManagedPath() != filepath.Clean(target.ManagedPath) {
			return &install.Conflict{
				Provider:   entry.Provider(),
				AssetID:    asset.ID(),
				TargetPath: target.TargetPath,
				Managed:    true,
				Reason:     "inventory managed path does not match resolved target",
			}, nil
		}
	case install.OperationRemove:
		if managed {
			return nil, nil
		}
		if exists {
			return &install.Conflict{
				Provider:   provider,
				AssetID:    asset.ID(),
				TargetPath: target.TargetPath,
				Managed:    false,
				Reason:     "target exists but is not managed by ORQ",
			}, nil
		}
	}

	return nil, nil
}

func (p *Planner) applyConflictPolicy(
	ctx context.Context,
	input PlanInput,
	changes []install.PlannedChange,
	conflicts []install.Conflict,
) ([]install.PlannedChange, error) {
	resolver := p.conflictResolve
	if resolver == nil {
		resolver = NewConflictPolicyResolver(input.ConflictPolicy, input.Interactive, nil)
	}

	decisions, err := resolver.Resolve(ctx, conflicts)
	if err != nil {
		return nil, err
	}

	decisionIndex := make(map[string]install.ConflictDecision, len(decisions))
	for _, decision := range decisions {
		decisionIndex[conflictKey(decision.Provider(), decision.AssetID(), decision.TargetPath())] = decision
	}

	resolved := make([]install.PlannedChange, 0, len(changes))
	for _, change := range changes {
		conflict := change.Conflict()
		if conflict == nil {
			resolved = append(resolved, change)
			continue
		}

		key := conflictKey(change.Provider(), change.AssetID(), conflict.TargetPath)
		decision, ok := decisionIndex[key]
		if !ok {
			return nil, fmt.Errorf("missing conflict decision for %s", key)
		}

		switch decision.Policy() {
		case install.ConflictPolicyAbort:
			return nil, fmt.Errorf("%w: provider=%q asset=%q target=%q", install.ErrConflictAborted, change.Provider(), change.AssetID(), conflict.TargetPath)
		case install.ConflictPolicySkip:
			change, err = install.NewPlannedChange(
				change.Provider(),
				change.Scope(),
				change.AssetID(),
				install.ActionSkip,
				change.SourcePath(),
				change.TargetPath(),
				change.ManagedPath(),
				conflict,
				change.Verification(),
			)
			if err != nil {
				return nil, err
			}
		case install.ConflictPolicyOverwrite:
		default:
			return nil, fmt.Errorf("unsupported conflict policy %q", decision.Policy())
		}

		resolved = append(resolved, change)
	}

	return resolved, nil
}

// InteractiveConflictPrompter resolves conflicts interactively.
type InteractiveConflictPrompter interface {
	ResolveConflicts(ctx context.Context, conflicts []install.Conflict) ([]install.ConflictDecision, error)
}

// ConflictPolicyResolver applies the default non-interactive policy and delegates when interactive.
type ConflictPolicyResolver struct {
	policy      install.ConflictPolicy
	interactive bool
	prompter    InteractiveConflictPrompter
}

// NewConflictPolicyResolver creates a resolver compatible with interactive and non-interactive modes.
func NewConflictPolicyResolver(
	policy install.ConflictPolicy,
	interactive bool,
	prompter InteractiveConflictPrompter,
) *ConflictPolicyResolver {
	if policy == "" {
		policy = install.ConflictPolicyAbort
	}

	return &ConflictPolicyResolver{
		policy:      policy,
		interactive: interactive,
		prompter:    prompter,
	}
}

// Resolve returns one decision per conflict.
func (r *ConflictPolicyResolver) Resolve(ctx context.Context, conflicts []install.Conflict) ([]install.ConflictDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := install.ValidateConflictPolicy(r.policy); err != nil {
		return nil, err
	}
	if len(conflicts) == 0 {
		return nil, nil
	}
	if r.interactive && r.prompter != nil && r.policy == install.ConflictPolicyAbort {
		return r.prompter.ResolveConflicts(ctx, conflicts)
	}

	decisions := make([]install.ConflictDecision, 0, len(conflicts))
	for _, conflict := range conflicts {
		decision, err := install.NewConflictDecision(conflict.Provider, conflict.AssetID, conflict.TargetPath, r.policy)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}

	return decisions, nil
}

func validateKinds(kinds []install.AssetKind) error {
	for _, kind := range kinds {
		if err := install.ValidateAssetKind(kind); err != nil {
			return err
		}
	}
	return nil
}

func filterAssets(
	assets []install.Asset,
	providers []install.Provider,
	names []string,
	kinds []install.AssetKind,
) []install.Asset {
	nameFilter := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		nameFilter[strings.ToLower(name)] = struct{}{}
	}

	kindFilter := make(map[install.AssetKind]struct{}, len(kinds))
	for _, kind := range kinds {
		kindFilter[kind] = struct{}{}
	}

	providerFilter := make(map[install.Provider]struct{}, len(providers))
	for _, provider := range providers {
		providerFilter[provider] = struct{}{}
	}

	filtered := make([]install.Asset, 0, len(assets))
	for _, asset := range assets {
		if len(nameFilter) > 0 {
			if _, ok := nameFilter[strings.ToLower(asset.Name())]; !ok {
				continue
			}
		}
		if len(kindFilter) > 0 {
			if _, ok := kindFilter[asset.Kind()]; !ok {
				continue
			}
		}
		if !supportsAny(asset, providerFilter) {
			continue
		}

		filtered = append(filtered, asset)
	}

	sort.Slice(filtered, func(i int, j int) bool {
		if filtered[i].Kind() == filtered[j].Kind() {
			return filtered[i].ID() < filtered[j].ID()
		}
		return filtered[i].Kind() < filtered[j].Kind()
	})

	return filtered
}

func supportsAny(asset install.Asset, providers map[install.Provider]struct{}) bool {
	for _, provider := range asset.Providers() {
		if _, ok := providers[provider]; ok {
			return true
		}
	}
	return false
}

func assetsForProvider(assets []install.Asset, provider install.Provider) []install.Asset {
	filtered := make([]install.Asset, 0)
	for _, asset := range assets {
		if asset.Supports(provider) {
			filtered = append(filtered, asset)
		}
	}
	return filtered
}

func sortChanges(changes []install.PlannedChange) {
	sort.Slice(changes, func(i int, j int) bool {
		if changes[i].Provider() == changes[j].Provider() {
			if changes[i].Action() == changes[j].Action() {
				return changes[i].AssetID() < changes[j].AssetID()
			}
			return changes[i].Action() < changes[j].Action()
		}
		return changes[i].Provider() < changes[j].Provider()
	})
}

func buildSummary(changes []install.PlannedChange) PlanSummary {
	summary := PlanSummary{}
	for _, change := range changes {
		if change.Conflict() != nil {
			summary.ConflictCount++
		}
		switch change.Action() {
		case install.ActionInstall:
			summary.InstallCount++
		case install.ActionUpdate:
			summary.UpdateCount++
		case install.ActionRemove:
			summary.RemoveCount++
		case install.ActionSkip:
			summary.SkippedCount++
		}
	}
	return summary
}

func inventoryEntriesForScope(inv *install.Inventory, scope install.Scope) []install.InventoryEntry {
	if inv == nil || inv.Scope() != scope {
		return nil
	}
	return inv.Entries()
}

type inventoryIndex struct {
	entries map[string]install.InventoryEntry
}

func newInventoryIndex(entries []install.InventoryEntry) inventoryIndex {
	index := inventoryIndex{entries: make(map[string]install.InventoryEntry, len(entries))}
	for _, entry := range entries {
		index.entries[providerAssetKey(entry.Provider(), entry.AssetID())] = entry
	}
	return index
}

func (i inventoryIndex) lookup(provider install.Provider, assetID string) (install.InventoryEntry, bool) {
	entry, ok := i.entries[providerAssetKey(provider, assetID)]
	return entry, ok
}

func providerAssetKey(provider install.Provider, assetID string) string {
	return string(provider) + "::" + assetID
}

func conflictKey(provider install.Provider, assetID string, targetPath string) string {
	return string(provider) + "::" + assetID + "::" + filepath.Clean(targetPath)
}

func pathExists(fileSystem platform.FileSystem, path string) (bool, error) {
	_, err := fileSystem.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
