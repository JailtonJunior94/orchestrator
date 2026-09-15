package runtime

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
)

type ReviewOutputFn = autoReviewOutputFn

func (c *Catalog) WithReviewOutputFn(fn autoReviewOutputFn) Option {
	return func(r *ACPRunner) {
		r.reviewOutputFn = fn
	}
}

func (c *Catalog) BuildReviewPromptForTest(skillBody, gitDiff string) string {
	return NewCatalog().buildReviewPrompt(skillBody, gitDiff)
}

func (c *Catalog) ExtractHardIssuesForTest(reviewOutput string) []string {
	return NewCatalog().extractHardIssues(reviewOutput)
}

func (c *Catalog) InjectMemoryContextForTest(prompt string, wf, tk memory.Document, wfErr, tkErr error) string {
	return NewCatalog().injectMemoryContext(prompt, wf, tk, wfErr, tkErr)
}
