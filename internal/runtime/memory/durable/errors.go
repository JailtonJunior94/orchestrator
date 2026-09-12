package durable

import "errors"

var (
	ErrSemanticKeyMissing        = errors.New("durable: semantic key missing")
	ErrDurabilityMissing         = errors.New("durable: durability missing")
	ErrSecretNotRedactable       = errors.New("durable: sensitive content not isolable")
	ErrPromotionWithoutMark      = errors.New("durable: promotion requires explicit mark")
	ErrBatonAlreadyClaimed       = errors.New("durable: baton held by live process")
	ErrRoundTripNotPreserved     = errors.New("durable: serialization does not preserve human content")
	ErrPageUnreadable            = errors.New("durable: page unreadable")
	ErrBudgetExceeded            = errors.New("durable: context budget exceeded")
	ErrLimitUnreachable          = errors.New("durable: limit unreachable due to human content")
	ErrMigrationAlreadyApplied   = errors.New("durable: migration already applied")
	ErrLayerUndefined            = errors.New("durable: layer undefined")
	ErrProjectDirMissing         = errors.New("durable: project directory missing")
	ErrTasksDirMissing           = errors.New("durable: tasks directory missing")
	ErrTaskFileNameMissing       = errors.New("durable: task file name missing")
	ErrFactNotFound              = errors.New("durable: fact not found")
	ErrLayerLocked               = errors.New("durable: layer locked by another process")
	ErrHumanBlockInterleaved     = errors.New("durable: human content interleaved between facts cannot be repositioned")
	ErrHumanContentNotNormalized = errors.New("durable: human content not normalized for fact append")
)
