package detect

import (
	"path/filepath"
	"strings"
)

// commonPrefixDepth counts the number of shared directory segments between two paths.
func commonPrefixDepth(a, b string) int {
	aParts := strings.Split(filepath.ToSlash(filepath.Dir(a)), "/")
	bParts := strings.Split(filepath.ToSlash(filepath.Dir(b)), "/")

	if len(aParts) == 1 && aParts[0] == "." {
		aParts = []string{}
	}
	if len(bParts) == 1 && bParts[0] == "." {
		bParts = []string{}
	}

	count := 0
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] != "" && aParts[i] == bParts[i] {
			count++
		} else {
			break
		}
	}
	return count
}

// scoreManifest returns the maximum commonPrefixDepth between manifestPath and any focus path.
func scoreManifest(manifestPath string, focusPaths []string) int {
	max := 0
	for _, fp := range focusPaths {
		if s := commonPrefixDepth(manifestPath, fp); s > max {
			max = s
		}
	}
	return max
}
