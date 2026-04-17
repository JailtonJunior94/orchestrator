package acp

import "errors"

// Sentinel errors for ACP domain failures.
var (
	// ErrProtocolVersionMismatch is returned when the agent reports an incompatible protocol version.
	ErrProtocolVersionMismatch = errors.New("acp protocol version incompatible")

	// ErrHandshakeTimeout is returned when the ACP handshake does not complete within the configured timeout.
	ErrHandshakeTimeout = errors.New("acp handshake timeout exceeded")

	// ErrSessionNotFound is returned when the requested session ID does not exist on the agent.
	ErrSessionNotFound = errors.New("acp session not found")

	// ErrSessionExpired is returned when the referenced session has expired or been evicted.
	ErrSessionExpired = errors.New("acp session expired")

	// ErrAgentNotAvailable is returned when neither the native binary nor the fallback command is found.
	ErrAgentNotAvailable = errors.New("acp agent binary not available")

	// ErrPathOutsideWorkDir is returned when a file operation targets a path outside the session CWD.
	ErrPathOutsideWorkDir = errors.New("file path outside session work directory")
)
