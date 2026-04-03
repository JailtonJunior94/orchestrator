package providers

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

const codexConfigQualifier = "codex-config"

// CodexAdapter manages install targets for Codex CLI.
type CodexAdapter struct {
	managed    *managedAdapter
	globalRoot string
}

// NewCodexAdapter creates a Codex target adapter.
func NewCodexAdapter(
	fileSystem platform.FileSystem,
	projectRoot string,
	globalRoot string,
) *CodexAdapter {
	return &CodexAdapter{
		managed: &managedAdapter{
			provider:        install.ProviderCodex,
			toolDir:         ".codex",
			projectRoot:     projectRoot,
			globalRoot:      globalRoot,
			fs:              fileSystem,
			backupQualifier: "codex",
		},
		globalRoot: globalRoot,
	}
}

// ResolveTarget returns the Codex destination for a skill asset.
func (a *CodexAdapter) ResolveTarget(ctx context.Context, scope install.Scope, asset install.Asset) (ResolvedTarget, error) {
	if asset.Kind() != install.AssetKindSkill {
		return ResolvedTarget{}, fmt.Errorf("provider %q does not support asset kind %q", install.ProviderCodex, asset.Kind())
	}
	return a.managed.ResolveTarget(ctx, scope, asset)
}

// Provider returns the target provider handled by this adapter.
func (a *CodexAdapter) Provider() install.Provider {
	return a.managed.Provider()
}

// Apply materializes Codex skills and reconciles config.toml atomically per scope.
func (a *CodexAdapter) Apply(ctx context.Context, changes []install.PlannedChange) ([]ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.managed == nil || a.managed.fs == nil {
		return nil, errors.New("filesystem must not be nil")
	}

	providerChanges, err := a.managed.validateChanges(changes)
	if err != nil {
		return nil, err
	}

	scopeGroups := make(map[install.Scope][]install.PlannedChange)
	scopeOrder := make([]install.Scope, 0, 2)
	for _, change := range providerChanges {
		if err := validateCodexChange(change); err != nil {
			return nil, err
		}
		if _, ok := scopeGroups[change.Scope()]; !ok {
			scopeOrder = append(scopeOrder, change.Scope())
		}
		scopeGroups[change.Scope()] = append(scopeGroups[change.Scope()], change)
	}

	results := make([]ApplyResult, 0, len(providerChanges))
	for _, scope := range scopeOrder {
		applied, err := a.applyScopeChanges(ctx, scope, scopeGroups[scope])
		if err != nil {
			return nil, err
		}
		results = append(results, applied...)
	}

	return results, nil
}

// Verify checks Codex skill materialization and config reconciliation.
func (a *CodexAdapter) Verify(ctx context.Context, changes []install.PlannedChange) (VerificationReport, error) {
	if err := ctx.Err(); err != nil {
		return VerificationReport{}, err
	}

	providerChanges := a.managed.validateProviderOnly(changes)
	details := make([]string, 0, len(providerChanges)+2)
	statuses := make([]install.VerificationStatus, 0, 4)

	scopeGroups := make(map[install.Scope][]install.PlannedChange)
	scopeOrder := make([]install.Scope, 0, 2)
	for _, change := range providerChanges {
		if err := validateCodexChange(change); err != nil {
			return VerificationReport{}, err
		}
		if _, ok := scopeGroups[change.Scope()]; !ok {
			scopeOrder = append(scopeOrder, change.Scope())
		}
		scopeGroups[change.Scope()] = append(scopeGroups[change.Scope()], change)
	}

	for _, scope := range scopeOrder {
		scopeStatus, scopeDetails, err := a.verifyScope(ctx, scope, scopeGroups[scope])
		if err != nil {
			return VerificationReport{}, err
		}
		statuses = append(statuses, scopeStatus)
		details = append(details, scopeDetails...)
	}

	if len(statuses) == 0 {
		statuses = append(statuses, install.VerificationStatusPartial)
		details = append(details, "no Codex changes to verify")
	}

	return VerificationReport{
		Provider: install.ProviderCodex,
		Status:   aggregateVerificationStatus(statuses...),
		Details:  details,
	}, nil
}

func (a *CodexAdapter) applyScopeChanges(
	ctx context.Context,
	scope install.Scope,
	changes []install.PlannedChange,
) ([]ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	configPath, err := a.configPath(scope)
	if err != nil {
		return nil, err
	}

	currentConfig, configExists, err := a.readConfig(configPath)
	if err != nil {
		return nil, err
	}

	results := make([]ApplyResult, 0, len(changes))
	for _, change := range changes {
		if change.Action() == install.ActionSkip {
			continue
		}

		result, err := a.managed.applyChange(change, len(results))
		if err != nil {
			rollbackErr := a.managed.rollback(results)
			if rollbackErr != nil {
				return nil, fmt.Errorf("apply change for provider %q failed: %w; rollback failed: %v", install.ProviderCodex, err, rollbackErr)
			}
			return nil, fmt.Errorf("apply change for provider %q asset %q target %q: %w", install.ProviderCodex, change.AssetID(), change.TargetPath(), err)
		}
		results = append(results, result)
	}

	reconciledConfig, err := reconcileCodexConfig(currentConfig, changes)
	if err != nil {
		rollbackErr := a.managed.rollback(results)
		if rollbackErr != nil {
			return nil, fmt.Errorf("reconcile Codex config %q failed: %w; rollback failed: %v", configPath, err, rollbackErr)
		}
		return nil, fmt.Errorf("reconcile Codex config %q: %w", configPath, err)
	}

	configBackupPath, err := a.writeConfig(configPath, configExists, reconciledConfig)
	if err != nil {
		rollbackErr := a.managed.rollback(results)
		if rollbackErr != nil {
			return nil, fmt.Errorf("persist Codex config %q failed: %w; rollback failed: %v", configPath, err, rollbackErr)
		}
		return nil, fmt.Errorf("persist Codex config %q: %w", configPath, err)
	}

	if err := a.managed.cleanupBackups(results); err != nil {
		_ = a.restoreConfig(configPath, configBackupPath)
		return nil, err
	}
	if err := cleanupConfigBackup(a.managed.fs, configBackupPath); err != nil {
		return nil, err
	}

	return results, nil
}

func (a *CodexAdapter) verifyScope(
	ctx context.Context,
	scope install.Scope,
	changes []install.PlannedChange,
) (install.VerificationStatus, []string, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	configPath, err := a.configPath(scope)
	if err != nil {
		return "", nil, err
	}

	currentConfig, _, err := a.readConfig(configPath)
	if err != nil {
		return "", nil, err
	}

	registeredPaths, err := codexConfiguredSkillPaths(currentConfig)
	if err != nil {
		return "", nil, fmt.Errorf("read Codex config %q: %w", configPath, err)
	}

	details := make([]string, 0, len(changes)+2)
	status := install.VerificationStatusComplete
	for _, change := range changes {
		expectedPresent := change.Action() != install.ActionRemove
		exists, err := pathExists(a.managed.fs, change.TargetPath())
		if err != nil {
			return "", nil, fmt.Errorf("stat target %q during verification: %w", change.TargetPath(), err)
		}

		configured := slices.Contains(registeredPaths, filepath.Clean(change.TargetPath()))
		switch {
		case expectedPresent && !exists:
			status = install.VerificationStatusFailed
			details = append(details, fmt.Sprintf("missing target: %s", change.TargetPath()))
		case expectedPresent && !configured:
			status = install.VerificationStatusFailed
			details = append(details, fmt.Sprintf("skill path missing from Codex config: %s", change.TargetPath()))
		case !expectedPresent && exists:
			status = install.VerificationStatusFailed
			details = append(details, fmt.Sprintf("target still present after removal: %s", change.TargetPath()))
		case !expectedPresent && configured:
			status = install.VerificationStatusFailed
			details = append(details, fmt.Sprintf("skill path still registered in Codex config: %s", change.TargetPath()))
		case expectedPresent:
			details = append(details, fmt.Sprintf("target present and registered: %s", change.TargetPath()))
		default:
			details = append(details, fmt.Sprintf("target removed and unregistered: %s", change.TargetPath()))
		}
	}

	if status == install.VerificationStatusFailed {
		return status, details, nil
	}

	switch scope {
	case install.ScopeProject:
		trustLevel, trustKnown, trustErr := a.projectTrustLevel()
		if trustErr != nil {
			return "", nil, trustErr
		}
		if trustKnown && trustLevel == "untrusted" {
			details = append(details, "project-scoped .codex/config.toml is ignored because Codex marks this project as untrusted")
			return install.VerificationStatusFailed, details, nil
		}
		if trustKnown && trustLevel == "trusted" {
			details = append(details, "project trust is configured as trusted; verification remains structural because no stronger Codex skill probe is used")
		} else {
			details = append(details, "Codex loads project-scoped .codex/config.toml only for trusted projects; trust level was not found in the user config")
		}
	default:
		details = append(details, "Codex skill registration is structurally verified through config.toml; no stronger functional probe is implemented")
	}

	return install.VerificationStatusPartial, details, nil
}

func (a *CodexAdapter) configPath(scope install.Scope) (string, error) {
	baseRoot, err := a.managed.baseRoot(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(baseRoot, ".codex", "config.toml"), nil
}

func (a *CodexAdapter) readConfig(path string) (map[string]any, bool, error) {
	exists, err := pathExists(a.managed.fs, path)
	if err != nil {
		return nil, false, fmt.Errorf("stat Codex config %q: %w", path, err)
	}
	if !exists {
		return map[string]any{}, false, nil
	}

	data, err := a.managed.fs.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("read Codex config %q: %w", path, err)
	}

	parsed, err := parseCodexConfig(data)
	if err != nil {
		return nil, false, fmt.Errorf("parse Codex config %q: %w", path, err)
	}

	return parsed, true, nil
}

func (a *CodexAdapter) writeConfig(path string, existing bool, config map[string]any) (string, error) {
	encoded, err := toml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}
	if err := a.managed.fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create Codex config parent %q: %w", filepath.Dir(path), err)
	}

	tempPath := buildTempPath(path, codexConfigQualifier)
	if err := a.managed.fs.WriteFile(tempPath, encoded, 0o644); err != nil {
		return "", fmt.Errorf("write temporary Codex config %q: %w", tempPath, err)
	}

	backupPath := ""
	if existing {
		backupPath = buildBackupPath(path, codexConfigQualifier, 0)
		if err := a.managed.fs.Rename(path, backupPath); err != nil {
			_ = a.managed.fs.RemoveAll(tempPath)
			return "", fmt.Errorf("backup Codex config %q to %q: %w", path, backupPath, err)
		}
	}

	if err := a.managed.fs.Rename(tempPath, path); err != nil {
		_ = a.managed.fs.RemoveAll(tempPath)
		if backupPath != "" {
			_ = a.managed.fs.Rename(backupPath, path)
		}
		return "", fmt.Errorf("activate Codex config %q: %w", path, err)
	}

	return backupPath, nil
}

func (a *CodexAdapter) restoreConfig(path string, backupPath string) error {
	if backupPath == "" {
		return a.managed.fs.RemoveAll(path)
	}
	if err := a.managed.fs.RemoveAll(path); err != nil {
		return err
	}
	return a.managed.fs.Rename(backupPath, path)
}

func (a *CodexAdapter) projectTrustLevel() (string, bool, error) {
	userConfigPath := filepath.Join(filepath.Clean(a.globalRoot), ".codex", "config.toml")
	config, exists, err := a.readConfig(userConfigPath)
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil
	}

	return codexProjectTrustLevel(config, a.managed.projectRoot)
}

func validateCodexChange(change install.PlannedChange) error {
	if change.Action() == install.ActionSkip {
		return nil
	}
	if !strings.Contains(filepath.Clean(change.TargetPath()), filepath.Clean(filepath.Join(".codex", "skills"))+string(filepath.Separator)) {
		return fmt.Errorf("provider %q only supports skill targets under .codex/skills: %q", install.ProviderCodex, change.TargetPath())
	}
	return nil
}

func parseCodexConfig(data []byte) (map[string]any, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, nil
	}

	config := make(map[string]any)
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func reconcileCodexConfig(config map[string]any, changes []install.PlannedChange) (map[string]any, error) {
	config = cloneMap(config)

	skillEntries, err := codexSkillEntries(config, true)
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		if change.Action() == install.ActionSkip {
			continue
		}

		targetPath := filepath.Clean(change.TargetPath())
		index := -1
		for entryIndex, entry := range skillEntries {
			currentPath, ok := entry["path"].(string)
			if !ok {
				return nil, fmt.Errorf("skills.config[%d].path must be a string", entryIndex)
			}
			if filepath.Clean(currentPath) == targetPath {
				index = entryIndex
				break
			}
		}

		switch change.Action() {
		case install.ActionInstall, install.ActionUpdate:
			if index == -1 {
				skillEntries = append(skillEntries, map[string]any{
					"path":    targetPath,
					"enabled": true,
				})
				continue
			}
			entry := cloneMap(skillEntries[index])
			entry["path"] = targetPath
			entry["enabled"] = true
			skillEntries[index] = entry
		case install.ActionRemove:
			if index == -1 {
				continue
			}
			skillEntries = append(skillEntries[:index], skillEntries[index+1:]...)
		default:
			return nil, fmt.Errorf("unsupported action %q", change.Action())
		}
	}

	skillsTable, err := codexTable(config, "skills", true)
	if err != nil {
		return nil, err
	}
	skillsTable["config"] = skillEntries
	return config, nil
}

func codexConfiguredSkillPaths(config map[string]any) ([]string, error) {
	entries, err := codexSkillEntries(config, false)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for index, entry := range entries {
		pathValue, ok := entry["path"].(string)
		if !ok {
			return nil, fmt.Errorf("skills.config[%d].path must be a string", index)
		}

		enabled, ok := entry["enabled"].(bool)
		if ok && !enabled {
			continue
		}
		paths = append(paths, filepath.Clean(pathValue))
	}

	slices.Sort(paths)
	return paths, nil
}

func codexProjectTrustLevel(config map[string]any, projectRoot string) (string, bool, error) {
	projectsTable, err := codexTable(config, "projects", false)
	if err != nil || projectsTable == nil {
		return "", false, err
	}

	cleanRoot := filepath.Clean(projectRoot)
	var projectConfig map[string]any
	for key, rawValue := range projectsTable {
		if filepath.Clean(key) != cleanRoot {
			continue
		}

		table, ok := rawValue.(map[string]any)
		if !ok {
			return "", false, fmt.Errorf("projects.%q must be a table", key)
		}
		projectConfig = table
		break
	}
	if projectConfig == nil {
		return "", false, nil
	}

	rawTrustLevel, ok := projectConfig["trust_level"]
	if !ok {
		return "", false, nil
	}

	trustLevel, ok := rawTrustLevel.(string)
	if !ok {
		return "", false, fmt.Errorf("projects.%q.trust_level must be a string", cleanRoot)
	}

	return trustLevel, true, nil
}

func codexSkillEntries(config map[string]any, create bool) ([]map[string]any, error) {
	skillsTable, err := codexTable(config, "skills", create)
	if err != nil || skillsTable == nil {
		return nil, err
	}

	rawConfig, ok := skillsTable["config"]
	if !ok {
		if create {
			return []map[string]any{}, nil
		}
		return nil, nil
	}

	rawEntries, ok := rawConfig.([]any)
	if !ok {
		return nil, errors.New("skills.config must be an array")
	}

	entries := make([]map[string]any, 0, len(rawEntries))
	for index, rawEntry := range rawEntries {
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("skills.config[%d] must be a table", index)
		}
		entries = append(entries, cloneMap(entry))
	}
	return entries, nil
}

func codexTable(config map[string]any, key string, create bool) (map[string]any, error) {
	rawValue, ok := config[key]
	if !ok {
		if !create {
			return nil, nil
		}
		table := make(map[string]any)
		config[key] = table
		return table, nil
	}

	table, ok := rawValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a table", key)
	}
	return table, nil
}

func cleanupConfigBackup(fileSystem platform.FileSystem, backupPath string) error {
	if backupPath == "" {
		return nil
	}

	exists, err := pathExists(fileSystem, backupPath)
	if err != nil {
		return fmt.Errorf("stat backup %q: %w", backupPath, err)
	}
	if !exists {
		return nil
	}

	if err := fileSystem.RemoveAll(backupPath); err != nil {
		return fmt.Errorf("remove backup %q: %w", backupPath, err)
	}
	return nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}

	output := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			output[key] = cloneMap(typed)
		case []any:
			output[key] = cloneSlice(typed)
		default:
			output[key] = typed
		}
	}
	return output
}

func cloneSlice(input []any) []any {
	if input == nil {
		return nil
	}

	output := make([]any, len(input))
	for index, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			output[index] = cloneMap(typed)
		case []any:
			output[index] = cloneSlice(typed)
		default:
			output[index] = typed
		}
	}
	return output
}
