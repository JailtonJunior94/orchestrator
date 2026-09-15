package approval

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

const fingerprintSeparator = "\x1e"

type Fingerprint struct {
	hash     string
	computed bool
}

type FingerprintCalculator struct{}

func NewFingerprintCalculator() FingerprintCalculator {
	return FingerprintCalculator{}
}

func (c FingerprintCalculator) Compute(findings []Finding) Fingerprint {
	keys := make([]string, 0, len(findings))
	for _, f := range findings {
		keys = append(keys, f.identityKey())
	}
	slices.Sort(keys)
	keys = slices.Compact(keys)

	sum := sha256.Sum256([]byte(strings.Join(keys, fingerprintSeparator)))
	return Fingerprint{hash: hex.EncodeToString(sum[:]), computed: true}
}

func (f Fingerprint) Equal(other Fingerprint) bool {
	if !f.computed || !other.computed {
		return false
	}
	return f.hash == other.hash
}

func (f Fingerprint) Computed() bool {
	return f.computed
}

func (f Fingerprint) String() string {
	return f.hash
}
