package domain

import (
	"path/filepath"
	"slices"
	"time"
)

// InventoryEntry stores a managed asset installation.
type InventoryEntry struct {
	provider       Provider
	assetID        string
	assetKind      AssetKind
	managedPath    string
	fingerprint    string
	verification   VerificationStatus
	lastVerifiedAt *time.Time
}

// NewInventoryEntry creates a validated inventory entry.
func NewInventoryEntry(
	provider Provider,
	assetID string,
	assetKind AssetKind,
	managedPath string,
	fingerprint string,
	verification VerificationStatus,
	lastVerifiedAt *time.Time,
) (InventoryEntry, error) {
	if err := ValidateProvider(provider); err != nil {
		return InventoryEntry{}, err
	}
	if assetID == "" {
		return InventoryEntry{}, ErrEmptyInventoryEntryAssetID
	}
	if err := ValidateAssetKind(assetKind); err != nil {
		return InventoryEntry{}, err
	}
	if managedPath == "" {
		return InventoryEntry{}, ErrEmptyManagedPath
	}
	if err := ValidateVerificationStatus(verification); err != nil {
		return InventoryEntry{}, err
	}

	entry := InventoryEntry{
		provider:     provider,
		assetID:      assetID,
		assetKind:    assetKind,
		managedPath:  filepath.Clean(managedPath),
		fingerprint:  fingerprint,
		verification: verification,
	}
	if lastVerifiedAt != nil {
		copyValue := *lastVerifiedAt
		entry.lastVerifiedAt = &copyValue
	}

	return entry, nil
}

// Provider returns the entry provider.
func (e InventoryEntry) Provider() Provider { return e.provider }

// AssetID returns the entry asset identifier.
func (e InventoryEntry) AssetID() string { return e.assetID }

// AssetKind returns the entry asset kind.
func (e InventoryEntry) AssetKind() AssetKind { return e.assetKind }

// ManagedPath returns the installed path.
func (e InventoryEntry) ManagedPath() string { return e.managedPath }

// Fingerprint returns the current asset fingerprint.
func (e InventoryEntry) Fingerprint() string { return e.fingerprint }

// Verification returns the last verification status.
func (e InventoryEntry) Verification() VerificationStatus { return e.verification }

// LastVerifiedAt returns the last verification timestamp.
func (e InventoryEntry) LastVerifiedAt() *time.Time {
	if e.lastVerifiedAt == nil {
		return nil
	}
	copyValue := *e.lastVerifiedAt
	return &copyValue
}

// Inventory stores the managed install state for a scope.
type Inventory struct {
	schemaVersion int
	scope         Scope
	location      string
	entries       []InventoryEntry
	updatedAt     time.Time
}

// NewInventory creates a validated inventory aggregate.
func NewInventory(schemaVersion int, scope Scope, location string, entries []InventoryEntry, updatedAt time.Time) (*Inventory, error) {
	if err := ValidateScope(scope); err != nil {
		return nil, err
	}
	if location == "" {
		return nil, ErrEmptyInventoryLocation
	}

	return &Inventory{
		schemaVersion: schemaVersion,
		scope:         scope,
		location:      filepath.Clean(location),
		entries:       slices.Clone(entries),
		updatedAt:     updatedAt,
	}, nil
}

// SchemaVersion returns the inventory schema version.
func (i *Inventory) SchemaVersion() int { return i.schemaVersion }

// Scope returns the inventory scope.
func (i *Inventory) Scope() Scope { return i.scope }

// Location returns the inventory storage location.
func (i *Inventory) Location() string { return i.location }

// Entries returns a defensive copy of the inventory entries.
func (i *Inventory) Entries() []InventoryEntry { return slices.Clone(i.entries) }

// UpdatedAt returns the last update time.
func (i *Inventory) UpdatedAt() time.Time { return i.updatedAt }
