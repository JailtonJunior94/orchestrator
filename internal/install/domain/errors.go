package domain

import "errors"

var (
	ErrEmptyAssetID               = errors.New("asset id must not be empty")
	ErrEmptyAssetName             = errors.New("asset name must not be empty")
	ErrEmptyAssetSourcePath       = errors.New("asset source path must not be empty")
	ErrEmptyTargetPath            = errors.New("target path must not be empty")
	ErrEmptyManagedPath           = errors.New("managed path must not be empty")
	ErrEmptyInventoryLocation     = errors.New("inventory location must not be empty")
	ErrEmptyConflictTargetPath    = errors.New("conflict target path must not be empty")
	ErrInvalidProvider            = errors.New("invalid provider")
	ErrInvalidScope               = errors.New("invalid scope")
	ErrInvalidAssetKind           = errors.New("invalid asset kind")
	ErrInvalidAction              = errors.New("invalid action")
	ErrInvalidOperation           = errors.New("invalid operation")
	ErrInvalidConflictPolicy      = errors.New("invalid conflict policy")
	ErrInvalidVerificationStatus  = errors.New("invalid verification status")
	ErrEmptyProviderSet           = errors.New("provider set must not be empty")
	ErrDuplicateProvider          = errors.New("duplicate provider")
	ErrEmptyInventoryEntryAssetID = errors.New("inventory entry asset id must not be empty")
	ErrConflictAborted            = errors.New("planning aborted because of conflict policy")
)
