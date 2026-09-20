package qualitygate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type CacheEntry struct {
	Fingerprint string `json:"fingerprint"`
	Decision    string `json:"decision"`
	Reason      string `json:"reason"`
	PolicyID    string `json:"policy_id"`
	GateID      string `json:"gate_id"`
}

type Cache interface {
	Get(taskID string) (CacheEntry, bool, error)
	Put(taskID string, entry CacheEntry) error
}

type fileCache struct {
	fs   fs.FileSystem
	path string
}

var _ Cache = (*fileCache)(nil)

func NewFileCache(filesystem fs.FileSystem, path string) Cache {
	return &fileCache{fs: filesystem, path: path}
}

func (c *fileCache) load() (map[string]CacheEntry, error) {
	data, err := c.fs.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]CacheEntry{}, nil
		}
		return nil, fmt.Errorf("qualitygate: read cache %s: %w", c.path, err)
	}

	entries := map[string]CacheEntry{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("qualitygate: decode cache %s: %w", c.path, err)
	}
	return entries, nil
}

func (c *fileCache) Get(taskID string) (CacheEntry, bool, error) {
	entries, err := c.load()
	if err != nil {
		return CacheEntry{}, false, err
	}
	entry, ok := entries[taskID]
	return entry, ok, nil
}

func (c *fileCache) Put(taskID string, entry CacheEntry) error {
	entries, err := c.load()
	if err != nil {
		return err
	}
	entries[taskID] = entry

	encoded, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("qualitygate: encode cache: %w", err)
	}
	if err := c.fs.WriteFileAtomic(c.path, append(encoded, '\n')); err != nil {
		return fmt.Errorf("qualitygate: write cache %s: %w", c.path, err)
	}
	return nil
}
