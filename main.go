package main

import (
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/cmd/ai_spec_harness"
)

func main() {
	if err := aispecharness.Execute(); err != nil {
		os.Exit(1)
	}
}
