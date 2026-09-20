package qualitygate

import (
	"errors"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
)

var ErrVerdictSealed = errors.New("qualitygate: deterministic verdict is sealed and cannot be overridden")

type Verdict struct {
	result hookcontract.Result
}

func NewVerdict(result hookcontract.Result) *Verdict {
	return &Verdict{result: result}
}

func (v *Verdict) Result() hookcontract.Result {
	return v.result
}

func (v *Verdict) Override(hookcontract.Result) error {
	return ErrVerdictSealed
}
