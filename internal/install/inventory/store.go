package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

const schemaVersion = 1

// Store persists install inventory snapshots.
type Store interface {
	Load(ctx context.Context, scope install.Scope, location string) (*install.Inventory, error)
	Save(ctx context.Context, scope install.Scope, location string, inv *install.Inventory) error
}

// FileStore persists inventory snapshots as JSON files.
type FileStore struct {
	fs    platform.FileSystem
	clock platform.Clock
}

// NewStore creates a filesystem-backed inventory store.
func NewStore(fileSystem platform.FileSystem, clock platform.Clock) *FileStore {
	return &FileStore{
		fs:    fileSystem,
		clock: clock,
	}
}

type inventoryDTO struct {
	SchemaVersion int                 `json:"schema_version"`
	Scope         string              `json:"scope"`
	UpdatedAt     string              `json:"updated_at"`
	Entries       []inventoryEntryDTO `json:"entries"`
}

type inventoryEntryDTO struct {
	Provider       string  `json:"provider"`
	AssetID        string  `json:"asset_id"`
	AssetKind      string  `json:"asset_kind"`
	ManagedPath    string  `json:"managed_path"`
	Fingerprint    string  `json:"fingerprint,omitempty"`
	Verification   string  `json:"verification"`
	LastVerifiedAt *string `json:"last_verified_at,omitempty"`
}

// Load reads inventory data from the requested location or returns an empty aggregate when absent.
func (s *FileStore) Load(ctx context.Context, scope install.Scope, location string) (*install.Inventory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := install.ValidateScope(scope); err != nil {
		return nil, err
	}
	if location == "" {
		return nil, install.ErrEmptyInventoryLocation
	}

	data, err := s.fs.ReadFile(location)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return install.NewInventory(schemaVersion, scope, location, nil, s.clock.Now().UTC())
		}
		return nil, fmt.Errorf("read inventory %q: %w", location, err)
	}

	var dto inventoryDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("decode inventory %q: %w", location, err)
	}

	inventory, err := dto.toDomain(location)
	if err != nil {
		return nil, fmt.Errorf("map inventory %q: %w", location, err)
	}
	if inventory.Scope() != scope {
		return nil, fmt.Errorf("inventory scope mismatch: file=%q requested=%q", inventory.Scope(), scope)
	}

	return inventory, nil
}

// Save persists the provided inventory snapshot atomically.
func (s *FileStore) Save(ctx context.Context, scope install.Scope, location string, inv *install.Inventory) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := install.ValidateScope(scope); err != nil {
		return err
	}
	if location == "" {
		return install.ErrEmptyInventoryLocation
	}
	if inv == nil {
		return errors.New("inventory must not be nil")
	}
	if inv.Scope() != scope {
		return fmt.Errorf("inventory scope mismatch: inventory=%q requested=%q", inv.Scope(), scope)
	}

	dto, err := newInventoryDTO(inv)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		return fmt.Errorf("encode inventory %q: %w", location, err)
	}

	parentDir := filepath.Dir(location)
	if err := s.fs.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create inventory directory %q: %w", parentDir, err)
	}

	tempPath := location + ".tmp"
	if err := s.fs.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("write inventory temp file %q: %w", tempPath, err)
	}
	if err := s.fs.Rename(tempPath, location); err != nil {
		return fmt.Errorf("replace inventory file %q: %w", location, err)
	}

	return nil
}

func newInventoryDTO(inv *install.Inventory) (inventoryDTO, error) {
	entries := make([]inventoryEntryDTO, 0, len(inv.Entries()))
	for _, entry := range inv.Entries() {
		dtoEntry := inventoryEntryDTO{
			Provider:     string(entry.Provider()),
			AssetID:      entry.AssetID(),
			AssetKind:    string(entry.AssetKind()),
			ManagedPath:  entry.ManagedPath(),
			Fingerprint:  entry.Fingerprint(),
			Verification: string(entry.Verification()),
		}

		if timestamp := entry.LastVerifiedAt(); timestamp != nil {
			value := timestamp.UTC().Format(timeLayout)
			dtoEntry.LastVerifiedAt = &value
		}

		entries = append(entries, dtoEntry)
	}

	return inventoryDTO{
		SchemaVersion: inv.SchemaVersion(),
		Scope:         string(inv.Scope()),
		UpdatedAt:     inv.UpdatedAt().UTC().Format(timeLayout),
		Entries:       entries,
	}, nil
}

func (dto inventoryDTO) toDomain(location string) (*install.Inventory, error) {
	updatedAt, err := timeFromString(dto.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	entries := make([]install.InventoryEntry, 0, len(dto.Entries))
	for _, entryDTO := range dto.Entries {
		var lastVerifiedAt *time.Time
		if entryDTO.LastVerifiedAt != nil {
			parsed, err := timeFromString(*entryDTO.LastVerifiedAt)
			if err != nil {
				return nil, fmt.Errorf("parse last_verified_at for %q: %w", entryDTO.AssetID, err)
			}
			lastVerifiedAt = &parsed
		}

		entry, err := install.NewInventoryEntry(
			install.Provider(entryDTO.Provider),
			entryDTO.AssetID,
			install.AssetKind(entryDTO.AssetKind),
			entryDTO.ManagedPath,
			entryDTO.Fingerprint,
			install.VerificationStatus(entryDTO.Verification),
			lastVerifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("build inventory entry %q: %w", entryDTO.AssetID, err)
		}
		entries = append(entries, entry)
	}

	return install.NewInventory(dto.SchemaVersion, install.Scope(dto.Scope), location, entries, updatedAt)
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func timeFromString(value string) (time.Time, error) {
	parsed, err := time.Parse(timeLayout, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
