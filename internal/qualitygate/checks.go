package qualitygate

import (
	"errors"
	"fmt"
)

type CheckKind int

const (
	CheckFmt CheckKind = iota + 1
	CheckTest
	CheckLint
)

var allCheckKinds = []CheckKind{CheckFmt, CheckTest, CheckLint}

var ErrUnknownCheckKind = errors.New("qualitygate: unknown check kind")

func (k CheckKind) Valid() bool {
	return k >= CheckFmt && k <= CheckLint
}

func (k CheckKind) String() string {
	switch k {
	case CheckFmt:
		return "fmt"
	case CheckTest:
		return "test"
	case CheckLint:
		return "lint"
	default:
		return "unknown"
	}
}

func ParseCheckKind(s string) (CheckKind, error) {
	for _, kind := range allCheckKinds {
		if kind.String() == s {
			return kind, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownCheckKind, s)
}

func CheckKinds() []CheckKind {
	return append([]CheckKind(nil), allCheckKinds...)
}
