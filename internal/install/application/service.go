package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/install/catalog"
	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	installinventory "github.com/jailtonjunior/orchestrator/internal/install/inventory"
	installproviders "github.com/jailtonjunior/orchestrator/internal/install/providers"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// InstallService coordinates install module use cases.
type InstallService struct {
	projectRoot          string
	projectInventoryPath string
	globalInventoryPath  string
	catalog              catalog.Catalog
	planner              *Planner
	inventoryStore       installinventory.Store
	adapters             *installproviders.Registry
	fileSystem           platform.FileSystem
	clock                platform.Clock
	logger               *slog.Logger
}

// NewService builds the install module application service.
func NewService(
	projectRoot string,
	projectInventoryPath string,
	globalInventoryPath string,
	sourceCatalog catalog.Catalog,
	planner *Planner,
	store installinventory.Store,
	adapters *installproviders.Registry,
	fileSystem platform.FileSystem,
	clock platform.Clock,
	logger *slog.Logger,
) Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &InstallService{
		projectRoot:          cleanPathOrEmpty(projectRoot),
		projectInventoryPath: cleanPathOrEmpty(projectInventoryPath),
		globalInventoryPath:  cleanPathOrEmpty(globalInventoryPath),
		catalog:              sourceCatalog,
		planner:              planner,
		inventoryStore:       store,
		adapters:             adapters,
		fileSystem:           fileSystem,
		clock:                clock,
		logger:               logger.With("module", "install"),
	}
}

// Preview builds the deterministic plan for a mutating operation without applying changes.
func (s *InstallService) Preview(ctx context.Context, req PreviewRequest) (*OperationPreview, error) {
	switch req.Operation {
	case install.OperationInstall, install.OperationUpdate, install.OperationRemove:
	default:
		return nil, fmt.Errorf("preview unsupported for operation %q", req.Operation)
	}

	previewReq := req.OperationRequest
	if previewReq.ConflictPolicy == install.ConflictPolicyAbort {
		previewReq.ConflictPolicy = install.ConflictPolicySkip
	}

	prepared, err := s.preparePlan(ctx, req.Operation, previewReq)
	if err != nil {
		return nil, err
	}

	return &OperationPreview{
		Operation:     req.Operation,
		Scope:         req.Scope,
		InventoryPath: prepared.inventoryPath,
		Plan:          prepared.plan,
	}, nil
}

// Install executes a new installation plan.
func (s *InstallService) Install(ctx context.Context, req InstallRequest) (*OperationResult, error) {
	return s.runMutation(ctx, install.OperationInstall, req.OperationRequest)
}

// Update reconciles managed assets with the current catalog.
func (s *InstallService) Update(ctx context.Context, req UpdateRequest) (*OperationResult, error) {
	return s.runMutation(ctx, install.OperationUpdate, req.OperationRequest)
}

// Remove removes managed assets from the selected scope.
func (s *InstallService) Remove(ctx context.Context, req RemoveRequest) (*OperationResult, error) {
	return s.runMutation(ctx, install.OperationRemove, req.OperationRequest)
}

// List returns the current inventory view for a scope.
func (s *InstallService) List(ctx context.Context, req ListRequest) (*InventoryView, error) {
	prepared, err := s.preparePlan(ctx, install.OperationList, OperationRequest{
		Scope:      req.Scope,
		Providers:  req.Providers,
		AssetNames: req.AssetNames,
		AssetKinds: req.AssetKinds,
	})
	if err != nil {
		return nil, err
	}

	items := buildInventoryItems(prepared.plan, prepared.assetsByID, prepared.inventory)
	return &InventoryView{
		Scope:         req.Scope,
		InventoryPath: prepared.inventoryPath,
		Plan:          prepared.plan,
		Items:         items,
	}, nil
}

// Verify audits managed assets for the selected scope.
func (s *InstallService) Verify(ctx context.Context, req VerifyRequest) (*VerificationReport, error) {
	prepared, err := s.preparePlan(ctx, install.OperationVerify, OperationRequest{
		Scope:      req.Scope,
		Providers:  req.Providers,
		AssetNames: req.AssetNames,
		AssetKinds: req.AssetKinds,
	})
	if err != nil {
		return nil, err
	}

	verifyChanges := buildVerifyChanges(prepared.plan, prepared.assetsByID, prepared.inventory)
	reports := make([]VerificationProviderReport, 0, len(verifyChanges))
	currentInventory := prepared.inventory

	for _, provider := range orderedProviders(verifyChanges) {
		changes := verifyChanges[provider]
		if len(changes) == 0 {
			continue
		}

		adapter, err := s.adapters.Adapter(provider)
		if err != nil {
			return &VerificationReport{
				Scope:         req.Scope,
				InventoryPath: prepared.inventoryPath,
				Providers:     reports,
			}, err
		}

		startedAt := s.clock.Now()
		providerReport, err := adapter.Verify(ctx, changes)
		duration := s.clock.Now().Sub(startedAt)
		if err != nil {
			return &VerificationReport{
				Scope:         req.Scope,
				InventoryPath: prepared.inventoryPath,
				Providers:     reports,
			}, fmt.Errorf("verify provider %q: %w", provider, err)
		}

		report := VerificationProviderReport{
			Provider: provider,
			Status:   providerReport.Status,
			Details:  append([]string(nil), providerReport.Details...),
		}

		if providerReport.Status != install.VerificationStatusFailed {
			updatedInventory, reconcileErr := reconcileVerifiedInventory(currentInventory, changes, providerReport.Status, s.clock.Now().UTC())
			if reconcileErr != nil {
				return &VerificationReport{
					Scope:         req.Scope,
					InventoryPath: prepared.inventoryPath,
					Providers:     append(reports, report),
				}, reconcileErr
			}
			if saveErr := s.inventoryStore.Save(ctx, prepared.scope, prepared.inventoryPath, updatedInventory); saveErr != nil {
				return &VerificationReport{
					Scope:         req.Scope,
					InventoryPath: prepared.inventoryPath,
					Providers:     append(reports, report),
				}, fmt.Errorf("save inventory after verify for provider %q: %w", provider, saveErr)
			}
			currentInventory = updatedInventory
			report.InventorySaved = true
		}

		s.logger.InfoContext(
			ctx,
			"provider verification completed",
			"provider", provider,
			"operation", install.OperationVerify,
			"duration_ms", duration.Milliseconds(),
			"status", providerReport.Status,
		)
		reports = append(reports, report)

		if providerReport.Status == install.VerificationStatusFailed {
			return &VerificationReport{
				Scope:         req.Scope,
				InventoryPath: prepared.inventoryPath,
				Providers:     reports,
			}, fmt.Errorf("provider %q verification failed", provider)
		}
	}

	return &VerificationReport{
		Scope:         req.Scope,
		InventoryPath: prepared.inventoryPath,
		Providers:     reports,
	}, nil
}

type preparedOperation struct {
	scope         install.Scope
	inventoryPath string
	inventory     *install.Inventory
	plan          *Plan
	assetsByID    map[string]install.Asset
}

func (s *InstallService) runMutation(
	ctx context.Context,
	operation install.Operation,
	req OperationRequest,
) (*OperationResult, error) {
	prepared, err := s.preparePlan(ctx, operation, req)
	if err != nil {
		return nil, err
	}

	result := &OperationResult{
		Operation:     operation,
		Scope:         req.Scope,
		InventoryPath: prepared.inventoryPath,
		Plan:          prepared.plan,
	}

	currentInventory := prepared.inventory
	for _, provider := range orderedProviders(groupChangesByProvider(prepared.plan.Changes)) {
		changes := groupChangesByProvider(prepared.plan.Changes)[provider]
		adapter, err := s.adapters.Adapter(provider)
		if err != nil {
			return result, err
		}

		startedAt := s.clock.Now()
		applied, err := adapter.Apply(ctx, changes)
		if err != nil {
			return result, fmt.Errorf("apply provider %q: %w", provider, err)
		}

		executedChanges := appliedChanges(applied)
		providerResult := ProviderResult{
			Provider:           provider,
			PlannedChangeCount: len(changes),
			AppliedChangeCount: len(executedChanges),
			Verification:       install.VerificationStatusUnknown,
		}

		if len(executedChanges) == 0 {
			providerResult.Details = []string{"no changes applied for this provider"}
			result.Providers = append(result.Providers, providerResult)
			continue
		}

		verifyReport, err := adapter.Verify(ctx, executedChanges)
		if err != nil {
			result.Providers = append(result.Providers, providerResult)
			return result, fmt.Errorf("verify provider %q: %w", provider, err)
		}

		providerResult.Verification = verifyReport.Status
		providerResult.Details = append([]string(nil), verifyReport.Details...)
		result.Providers = append(result.Providers, providerResult)

		s.logger.InfoContext(
			ctx,
			"provider operation completed",
			"provider", provider,
			"operation", operation,
			"duration_ms", s.clock.Now().Sub(startedAt).Milliseconds(),
			"status", verifyReport.Status,
		)

		if verifyReport.Status == install.VerificationStatusFailed {
			return result, fmt.Errorf("provider %q verification failed after %s", provider, operation)
		}

		updatedInventory, err := s.reconcileInventory(currentInventory, executedChanges, prepared.assetsByID, verifyReport.Status)
		if err != nil {
			return result, err
		}
		if err := s.inventoryStore.Save(ctx, prepared.scope, prepared.inventoryPath, updatedInventory); err != nil {
			return result, fmt.Errorf("save inventory after provider %q: %w", provider, err)
		}

		currentInventory = updatedInventory
		result.Providers[len(result.Providers)-1].InventorySaved = true
	}

	return result, nil
}

func (s *InstallService) preparePlan(
	ctx context.Context,
	operation install.Operation,
	req OperationRequest,
) (*preparedOperation, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if err := install.ValidateScope(req.Scope); err != nil {
		return nil, err
	}

	inventoryPath, err := s.inventoryPath(req.Scope)
	if err != nil {
		return nil, err
	}

	inventorySnapshot, err := s.inventoryStore.Load(ctx, req.Scope, inventoryPath)
	if err != nil {
		return nil, fmt.Errorf("load inventory %q: %w", inventoryPath, err)
	}

	assets, err := s.catalog.Discover(ctx, s.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("discover install catalog from %q: %w", s.projectRoot, err)
	}
	if operation == install.OperationRemove || operation == install.OperationList || operation == install.OperationVerify {
		assets, err = mergeAssetsWithInventory(assets, inventorySnapshot)
		if err != nil {
			return nil, err
		}
	}

	plan, err := s.planner.Plan(ctx, PlanInput{
		Operation:      operation,
		Scope:          req.Scope,
		Providers:      req.Providers,
		Assets:         assets,
		Inventory:      inventorySnapshot,
		AssetNames:     req.AssetNames,
		AssetKinds:     req.AssetKinds,
		ConflictPolicy: req.ConflictPolicy,
		Interactive:    req.Interactive,
	})
	if err != nil {
		return nil, err
	}

	return &preparedOperation{
		scope:         req.Scope,
		inventoryPath: inventoryPath,
		inventory:     inventorySnapshot,
		plan:          plan,
		assetsByID:    indexAssets(assets),
	}, nil
}

func (s *InstallService) validateDependencies() error {
	switch {
	case s.projectRoot == "":
		return errors.New("project root must not be empty")
	case s.projectInventoryPath == "":
		return errors.New("project inventory path must not be empty")
	case s.globalInventoryPath == "":
		return errors.New("global inventory path must not be empty")
	case s.catalog == nil:
		return errors.New("catalog must not be nil")
	case s.planner == nil:
		return errors.New("planner must not be nil")
	case s.inventoryStore == nil:
		return errors.New("inventory store must not be nil")
	case s.adapters == nil:
		return errors.New("adapter registry must not be nil")
	case s.fileSystem == nil:
		return errors.New("filesystem must not be nil")
	case s.clock == nil:
		return errors.New("clock must not be nil")
	default:
		return nil
	}
}

func (s *InstallService) inventoryPath(scope install.Scope) (string, error) {
	switch scope {
	case install.ScopeProject:
		return s.projectInventoryPath, nil
	case install.ScopeGlobal:
		return s.globalInventoryPath, nil
	default:
		return "", fmt.Errorf("unsupported scope %q", scope)
	}
}

func (s *InstallService) reconcileInventory(
	current *install.Inventory,
	changes []install.PlannedChange,
	assetsByID map[string]install.Asset,
	verification install.VerificationStatus,
) (*install.Inventory, error) {
	entryMap := make(map[string]install.InventoryEntry, len(current.Entries()))
	for _, entry := range current.Entries() {
		entryMap[inventoryEntryKey(entry.Provider(), entry.AssetID())] = entry
	}

	now := s.clock.Now().UTC()
	for _, change := range changes {
		key := inventoryEntryKey(change.Provider(), change.AssetID())
		switch change.Action() {
		case install.ActionInstall, install.ActionUpdate:
			asset, ok := assetsByID[change.AssetID()]
			if !ok {
				return nil, fmt.Errorf("asset %q not found during inventory reconciliation", change.AssetID())
			}
			fingerprint, err := computeFingerprint(s.fileSystem, asset.SourcePath())
			if err != nil {
				return nil, fmt.Errorf("fingerprint asset %q: %w", asset.ID(), err)
			}
			entry, err := install.NewInventoryEntry(
				change.Provider(),
				change.AssetID(),
				asset.Kind(),
				change.ManagedPath(),
				fingerprint,
				verification,
				&now,
			)
			if err != nil {
				return nil, err
			}
			entryMap[key] = entry
		case install.ActionRemove:
			delete(entryMap, key)
		}
	}

	return newInventoryFromEntries(current, entryMap, now)
}

func reconcileVerifiedInventory(
	current *install.Inventory,
	changes []install.PlannedChange,
	verification install.VerificationStatus,
	now time.Time,
) (*install.Inventory, error) {
	entryMap := make(map[string]install.InventoryEntry, len(current.Entries()))
	for _, entry := range current.Entries() {
		entryMap[inventoryEntryKey(entry.Provider(), entry.AssetID())] = entry
	}

	for _, change := range changes {
		key := inventoryEntryKey(change.Provider(), change.AssetID())
		entry, ok := entryMap[key]
		if !ok {
			continue
		}
		updatedEntry, err := install.NewInventoryEntry(
			entry.Provider(),
			entry.AssetID(),
			entry.AssetKind(),
			entry.ManagedPath(),
			entry.Fingerprint(),
			verification,
			&now,
		)
		if err != nil {
			return nil, err
		}
		entryMap[key] = updatedEntry
	}

	return newInventoryFromEntries(current, entryMap, now)
}

func newInventoryFromEntries(current *install.Inventory, entries map[string]install.InventoryEntry, now time.Time) (*install.Inventory, error) {
	items := make([]install.InventoryEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entry)
	}
	sort.Slice(items, func(i int, j int) bool {
		if items[i].Provider() == items[j].Provider() {
			return items[i].AssetID() < items[j].AssetID()
		}
		return items[i].Provider() < items[j].Provider()
	})

	return install.NewInventory(current.SchemaVersion(), current.Scope(), current.Location(), items, now)
}

func buildInventoryItems(plan *Plan, assetsByID map[string]install.Asset, inventorySnapshot *install.Inventory) []InventoryItem {
	inventoryIndex := make(map[string]install.InventoryEntry, len(inventorySnapshot.Entries()))
	for _, entry := range inventorySnapshot.Entries() {
		inventoryIndex[inventoryEntryKey(entry.Provider(), entry.AssetID())] = entry
	}

	items := make([]InventoryItem, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		asset := assetsByID[change.AssetID()]
		entry, managed := inventoryIndex[inventoryEntryKey(change.Provider(), change.AssetID())]
		verification := change.Verification()
		managedPath := change.ManagedPath()
		if managed {
			verification = entry.Verification()
			managedPath = entry.ManagedPath()
		}

		items = append(items, InventoryItem{
			Provider:     change.Provider(),
			AssetID:      change.AssetID(),
			Name:         asset.Name(),
			Kind:         asset.Kind(),
			TargetPath:   change.TargetPath(),
			ManagedPath:  managedPath,
			Managed:      managed,
			Verification: verification,
		})
	}

	return items
}

func buildVerifyChanges(
	plan *Plan,
	assetsByID map[string]install.Asset,
	inventorySnapshot *install.Inventory,
) map[install.Provider][]install.PlannedChange {
	inventoryIndex := make(map[string]install.InventoryEntry, len(inventorySnapshot.Entries()))
	for _, entry := range inventorySnapshot.Entries() {
		inventoryIndex[inventoryEntryKey(entry.Provider(), entry.AssetID())] = entry
	}

	grouped := make(map[install.Provider][]install.PlannedChange)
	for _, change := range plan.Changes {
		entry, ok := inventoryIndex[inventoryEntryKey(change.Provider(), change.AssetID())]
		if !ok {
			continue
		}

		asset, ok := assetsByID[change.AssetID()]
		sourcePath := entry.ManagedPath()
		if ok {
			sourcePath = asset.SourcePath()
		}

		verifyChange, err := install.NewPlannedChange(
			change.Provider(),
			change.Scope(),
			change.AssetID(),
			install.ActionInstall,
			sourcePath,
			entry.ManagedPath(),
			entry.ManagedPath(),
			nil,
			entry.Verification(),
		)
		if err != nil {
			continue
		}
		grouped[change.Provider()] = append(grouped[change.Provider()], verifyChange)
	}

	return grouped
}

func mergeAssetsWithInventory(assets []install.Asset, inventorySnapshot *install.Inventory) ([]install.Asset, error) {
	index := indexAssets(assets)
	merged := append([]install.Asset(nil), assets...)

	for _, entry := range inventorySnapshot.Entries() {
		if _, ok := index[entry.AssetID()]; ok {
			continue
		}

		asset, err := install.NewAsset(
			entry.AssetID(),
			assetNameFromID(entry.AssetID()),
			entry.AssetKind(),
			entry.ManagedPath(),
			[]install.Provider{entry.Provider()},
			install.AssetMetadata{
				EntryPath:     entry.ManagedPath(),
				SourceRoot:    filepath.Dir(entry.ManagedPath()),
				RelativePath:  filepath.Base(entry.ManagedPath()),
				ProviderHints: []install.Provider{entry.Provider()},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("rebuild inventory asset %q: %w", entry.AssetID(), err)
		}

		merged = append(merged, asset)
		index[asset.ID()] = asset
	}

	return merged, nil
}

func indexAssets(assets []install.Asset) map[string]install.Asset {
	index := make(map[string]install.Asset, len(assets))
	for _, asset := range assets {
		index[asset.ID()] = asset
	}
	return index
}

func groupChangesByProvider(changes []install.PlannedChange) map[install.Provider][]install.PlannedChange {
	grouped := make(map[install.Provider][]install.PlannedChange)
	for _, change := range changes {
		grouped[change.Provider()] = append(grouped[change.Provider()], change)
	}
	return grouped
}

func orderedProviders(grouped map[install.Provider][]install.PlannedChange) []install.Provider {
	providers := make([]install.Provider, 0, len(grouped))
	for provider := range grouped {
		providers = append(providers, provider)
	}
	sort.Slice(providers, func(i int, j int) bool {
		return providers[i] < providers[j]
	})
	return providers
}

func appliedChanges(results []installproviders.ApplyResult) []install.PlannedChange {
	changes := make([]install.PlannedChange, 0, len(results))
	for _, result := range results {
		changes = append(changes, result.Change)
	}
	return changes
}

func computeFingerprint(fileSystem platform.FileSystem, path string) (string, error) {
	info, err := fileSystem.Stat(path)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	if info.IsDir() {
		if err := hashDirectory(fileSystem, filepath.Clean(path), "", hash); err != nil {
			return "", err
		}
	} else {
		data, err := fileSystem.ReadFile(path)
		if err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte(filepath.Base(path))); err != nil {
			return "", err
		}
		if _, err := hash.Write(data); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func hashDirectory(fileSystem platform.FileSystem, root string, relative string, hash hashWriter) error {
	current := root
	if relative != "" {
		current = filepath.Join(root, relative)
	}

	entries, err := fileSystem.ReadDir(current)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryRelative := filepath.Join(relative, entry.Name())
		if _, err := hash.Write([]byte(filepath.ToSlash(entryRelative))); err != nil {
			return err
		}

		if entry.IsDir() {
			if err := hashDirectory(fileSystem, root, entryRelative, hash); err != nil {
				return err
			}
			continue
		}

		data, err := fileSystem.ReadFile(filepath.Join(root, entryRelative))
		if err != nil {
			return err
		}
		if _, err := hash.Write(data); err != nil {
			return err
		}
	}

	return nil
}

type hashWriter interface {
	Write(p []byte) (n int, err error)
}

func inventoryEntryKey(provider install.Provider, assetID string) string {
	return string(provider) + "::" + assetID
}

func assetNameFromID(assetID string) string {
	parts := strings.Split(assetID, ":")
	if len(parts) == 0 {
		return assetID
	}
	return parts[len(parts)-1]
}

func cleanPathOrEmpty(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

type registryTargetResolver struct {
	registry *installproviders.Registry
}

// NewRegistryTargetResolver adapts a provider registry to planner target resolution.
func NewRegistryTargetResolver(registry *installproviders.Registry) TargetResolver {
	return registryTargetResolver{registry: registry}
}

func (r registryTargetResolver) ResolveTarget(
	ctx context.Context,
	provider install.Provider,
	scope install.Scope,
	asset install.Asset,
) (Target, error) {
	adapter, err := r.registry.Adapter(provider)
	if err != nil {
		return Target{}, err
	}

	resolved, err := adapter.ResolveTarget(ctx, scope, asset)
	if err != nil {
		return Target{}, err
	}

	return Target{
		TargetPath:   resolved.TargetPath,
		ManagedPath:  resolved.ManagedPath,
		Verification: resolved.Verification,
	}, nil
}
