package qualitygate

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestFileCache_GetMissWhenEmpty(t *testing.T) {
	cache := NewFileCache(fs.NewFakeFileSystem(), "/project/.agents/generated/quality-gate-cache.json")

	_, hit, err := cache.Get("10.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit {
		t.Fatalf("expected no cache hit for empty cache")
	}
}

func TestFileCache_PutThenGetRoundTrips(t *testing.T) {
	cache := NewFileCache(fs.NewFakeFileSystem(), "/project/.agents/generated/quality-gate-cache.json")

	entry := CacheEntry{Fingerprint: "abc123", Decision: "ALLOW", PolicyID: "quality-gate:feature:low", GateID: GateID}
	if err := cache.Put("10.0", entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, hit, err := cache.Get("10.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit {
		t.Fatalf("expected cache hit")
	}
	if got != entry {
		t.Fatalf("got %+v, want %+v", got, entry)
	}
}

func TestFileCache_PutPreservesOtherTasks(t *testing.T) {
	cache := NewFileCache(fs.NewFakeFileSystem(), "/project/.agents/generated/quality-gate-cache.json")

	if err := cache.Put("10.0", CacheEntry{Fingerprint: "a"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cache.Put("11.0", CacheEntry{Fingerprint: "b"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, hit, err := cache.Get("10.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hit || got.Fingerprint != "a" {
		t.Fatalf("expected task 10.0 entry preserved, got %+v hit=%v", got, hit)
	}
}
