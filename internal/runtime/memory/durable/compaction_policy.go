package durable

import (
	"fmt"
	"strings"
)

type CompactionConfig struct {
	LineLimit int
	ByteLimit int
}

type CompactionResult struct {
	ToArchive []Identity
	Remaining []Fact
	Achieved  bool
}

type CompactionPolicy struct{}

var DefaultCompactionPolicy = CompactionPolicy{}

func (p CompactionPolicy) Compact(facts []Fact, human HumanBlock, cfg CompactionConfig, activeTask string) (CompactionResult, error) {
	page := NewMarkdownPage()
	remaining := DefaultRelevancePolicy.Rank(facts, activeTask)
	archived := make([]Identity, 0)

	for {
		rendered, err := page.Serialize(remaining, human)
		if err != nil {
			return CompactionResult{}, fmt.Errorf("durable: compaction failed to render page: %w", err)
		}

		lines := strings.Count(string(rendered), "\n") + 1
		size := len(rendered)

		if lines <= cfg.LineLimit && size <= cfg.ByteLimit {
			return CompactionResult{ToArchive: archived, Remaining: remaining, Achieved: true}, nil
		}

		if len(remaining) == 0 {
			return CompactionResult{ToArchive: archived, Remaining: remaining, Achieved: false}, ErrLimitUnreachable
		}

		last := remaining[len(remaining)-1]
		archived = append(archived, last.Identity)
		remaining = remaining[:len(remaining)-1]
	}
}
