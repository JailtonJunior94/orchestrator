package precondition

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type CopilotConfigReader interface {
	ReadConfig() ([]byte, error)
}

type FileCopilotConfigReader struct {
	Path string
}

func NewFileCopilotConfigReader(path string) FileCopilotConfigReader {
	return FileCopilotConfigReader{Path: path}
}

func (r FileCopilotConfigReader) ReadConfig() ([]byte, error) {
	return os.ReadFile(r.Path)
}

func DefaultCopilotConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".copilot", "config.json"), nil
}

func EvaluateCopilotTrustedFolder(reader CopilotConfigReader, projectDir string) (specs.PreconditionState, error) {
	raw, err := reader.ReadConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return specs.PreconditionInert, nil
		}
		return specs.PreconditionUnknown, err
	}

	stripped := stripLineCommentsOutsideStrings(raw)

	var doc struct {
		TrustedFolders []string `json:"trustedFolders"`
	}
	if err := json.Unmarshal(stripped, &doc); err != nil {
		return specs.PreconditionUnknown, err
	}

	absProject, err := filepath.Abs(projectDir)
	if err != nil {
		return specs.PreconditionUnknown, err
	}
	absProject = filepath.Clean(absProject)

	for _, folder := range doc.TrustedFolders {
		absFolder, err := filepath.Abs(folder)
		if err != nil {
			continue
		}
		absFolder = filepath.Clean(absFolder)
		if absProject == absFolder || strings.HasPrefix(absProject, absFolder+string(filepath.Separator)) {
			return specs.PreconditionCurrent, nil
		}
	}
	return specs.PreconditionInert, nil
}

func stripLineCommentsOutsideStrings(data []byte) []byte {
	out := make([]byte, 0, len(data))
	inString := false
	escaped := false

	for i := 0; i < len(data); i++ {
		c := data[i]

		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}

		if c == '/' && i+1 < len(data) && data[i+1] == '/' {
			for i < len(data) && data[i] != '\n' {
				i++
			}
			if i < len(data) {
				out = append(out, '\n')
			}
			continue
		}

		out = append(out, c)
	}
	return out
}
