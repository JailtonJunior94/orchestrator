package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
)

// FSHandler handles ReadTextFile and WriteTextFile requests from the agent.
// All file operations are scoped to the session's working directory (CWD).
type FSHandler struct {
	cwd    string
	logger *slog.Logger
	audit  FSAuditMetadata
}

// FSAuditMetadata identifies where ACP filesystem events should be persisted.
type FSAuditMetadata struct {
	RunID     string
	Workflow  string
	Step      string
	Provider  string
	StoreRoot string
}

// NewFSHandler returns an FSHandler scoped to the given working directory.
func NewFSHandler(cwd string, logger *slog.Logger, audit ...FSAuditMetadata) *FSHandler {
	if logger == nil {
		logger = slog.Default()
	}
	handler := &FSHandler{
		cwd:    filepath.Clean(cwd),
		logger: logger,
	}
	if len(audit) > 0 {
		handler.audit = audit[0]
	}
	return handler
}

func (h *FSHandler) logAudit(ctx context.Context, operation string, requestedPath string, allowed bool, reason string) {
	if h.audit.RunID == "" {
		return
	}

	entry := map[string]any{
		"run_id":   h.audit.RunID,
		"workflow": h.audit.Workflow,
		"step":     h.audit.Step,
		"provider": h.audit.Provider,
		"event":    "acp_filesystem_" + operation,
		"path":     requestedPath,
		"allowed":  allowed,
	}
	if reason != "" {
		entry["reason"] = reason
	}

	data, err := json.Marshal(entry)
	if err != nil {
		h.logger.WarnContext(ctx, "acp_file_audit_marshal_failed", slog.String("error", err.Error()))
		return
	}

	logPath := filepath.Join(h.audit.StoreRoot, ".orq", "runs", h.audit.RunID, "logs", "run.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		h.logger.WarnContext(ctx, "acp_file_audit_write_failed", slog.String("error", err.Error()))
		return
	}

	fh, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		h.logger.WarnContext(ctx, "acp_file_audit_write_failed", slog.String("error", err.Error()))
		return
	}
	defer func() {
		if closeErr := fh.Close(); closeErr != nil {
			h.logger.WarnContext(ctx, "acp_file_audit_write_failed", slog.String("error", closeErr.Error()))
		}
	}()

	if _, err := fh.Write(append(data, '\n')); err != nil {
		h.logger.WarnContext(ctx, "acp_file_audit_write_failed", slog.String("error", err.Error()))
	}
}

// ReadTextFile reads the file at the requested path and returns its content.
// Returns ErrPathOutsideWorkDir when the path escapes the CWD.
func (h *FSHandler) ReadTextFile(ctx context.Context, params acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error) {
	absPath, err := h.resolvePath(params.Path)
	if err != nil {
		h.logger.WarnContext(ctx, "acp_file_rejected",
			slog.String("path", params.Path),
			slog.String("reason", err.Error()),
		)
		h.logAudit(ctx, "read", params.Path, false, err.Error())
		return acpsdk.ReadTextFileResponse{}, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return acpsdk.ReadTextFileResponse{}, fmt.Errorf("acp read file %q: %w", absPath, err)
	}

	h.logger.InfoContext(ctx, "acp_file_read",
		slog.String("path", params.Path),
	)
	h.logAudit(ctx, "read", params.Path, true, "")
	return acpsdk.ReadTextFileResponse{Content: string(data)}, nil
}

// WriteTextFile writes content to the requested path.
// Returns ErrPathOutsideWorkDir when the path escapes the CWD.
func (h *FSHandler) WriteTextFile(ctx context.Context, params acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error) {
	absPath, err := h.resolvePath(params.Path)
	if err != nil {
		h.logger.WarnContext(ctx, "acp_file_rejected",
			slog.String("path", params.Path),
			slog.String("reason", err.Error()),
		)
		h.logAudit(ctx, "write", params.Path, false, err.Error())
		return acpsdk.WriteTextFileResponse{}, err
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
		return acpsdk.WriteTextFileResponse{}, fmt.Errorf("acp write file %q: create dirs: %w", absPath, err)
	}
	if err := os.WriteFile(absPath, []byte(params.Content), 0o600); err != nil {
		return acpsdk.WriteTextFileResponse{}, fmt.Errorf("acp write file %q: %w", absPath, err)
	}

	h.logger.InfoContext(ctx, "acp_file_write",
		slog.String("path", params.Path),
	)
	h.logAudit(ctx, "write", params.Path, true, "")
	return acpsdk.WriteTextFileResponse{}, nil
}

// resolvePath normalises the requested path and validates it stays within CWD.
func (h *FSHandler) resolvePath(rawPath string) (string, error) {
	// Build absolute path relative to CWD if not already absolute.
	var absPath string
	if filepath.IsAbs(rawPath) {
		absPath = filepath.Clean(rawPath)
	} else {
		absPath = filepath.Clean(filepath.Join(h.cwd, rawPath))
	}

	// Ensure the resolved path starts with the CWD prefix.
	rel, err := filepath.Rel(h.cwd, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("%w: %q", ErrPathOutsideWorkDir, rawPath)
	}

	return absPath, nil
}
