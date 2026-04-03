package domain

import (
	"path/filepath"
	"slices"
)

// AssetMetadata stores normalized source information for an asset.
type AssetMetadata struct {
	EntryPath     string
	SourceRoot    string
	RelativePath  string
	ProviderHints []Provider
}

// Asset is the normalized representation of a source asset.
type Asset struct {
	id         string
	name       string
	kind       AssetKind
	sourcePath string
	providers  []Provider
	metadata   AssetMetadata
}

// NewAsset creates a validated asset.
func NewAsset(id string, name string, kind AssetKind, sourcePath string, providers []Provider, metadata AssetMetadata) (Asset, error) {
	if id == "" {
		return Asset{}, ErrEmptyAssetID
	}
	if name == "" {
		return Asset{}, ErrEmptyAssetName
	}
	if err := ValidateAssetKind(kind); err != nil {
		return Asset{}, err
	}
	if sourcePath == "" {
		return Asset{}, ErrEmptyAssetSourcePath
	}

	normalizedProviders, err := NormalizeProviders(providers)
	if err != nil {
		return Asset{}, err
	}

	hints, err := NormalizeProvidersOrNil(metadata.ProviderHints)
	if err != nil {
		return Asset{}, err
	}

	metadata.EntryPath = cleanOptionalPath(metadata.EntryPath)
	metadata.SourceRoot = cleanOptionalPath(metadata.SourceRoot)
	metadata.RelativePath = cleanOptionalPath(metadata.RelativePath)
	metadata.ProviderHints = hints

	return Asset{
		id:         id,
		name:       name,
		kind:       kind,
		sourcePath: filepath.Clean(sourcePath),
		providers:  normalizedProviders,
		metadata:   metadata,
	}, nil
}

// ID returns the asset identifier.
func (a Asset) ID() string { return a.id }

// Name returns the asset display name.
func (a Asset) Name() string { return a.name }

// Kind returns the normalized asset kind.
func (a Asset) Kind() AssetKind { return a.kind }

// SourcePath returns the source materialized path.
func (a Asset) SourcePath() string { return a.sourcePath }

// Providers returns a defensive copy of the supported providers.
func (a Asset) Providers() []Provider {
	return slices.Clone(a.providers)
}

// Metadata returns the normalized metadata.
func (a Asset) Metadata() AssetMetadata {
	metadata := a.metadata
	metadata.ProviderHints = slices.Clone(metadata.ProviderHints)
	return metadata
}

// Supports reports whether an asset is eligible for the given provider.
func (a Asset) Supports(provider Provider) bool {
	return slices.Contains(a.providers, provider)
}

// NormalizeProvidersOrNil validates, deduplicates and orders providers when present.
func NormalizeProvidersOrNil(providers []Provider) ([]Provider, error) {
	if len(providers) == 0 {
		return nil, nil
	}
	return NormalizeProviders(providers)
}

func cleanOptionalPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
