package agents

import "github.com/JailtonJunior94/ai-spec-harness/internal/fs"

// ExportDiscoverAgents expoe discoverAgents para testes externos (package agents_test).
func ExportDiscoverAgents(fsys fs.FileSystem, scope Scope, root string) ([]ResolvedAgent, error) {
	return NewCatalog().discoverAgents(fsys, scope, root)
}

// ExportMergeWithShadowing expoe mergeWithShadowing para testes externos (package agents_test).
func ExportMergeWithShadowing(global, workspace []ResolvedAgent) (merged, shadowed []ResolvedAgent) {
	return NewCatalog().mergeWithShadowing(global, workspace)
}
