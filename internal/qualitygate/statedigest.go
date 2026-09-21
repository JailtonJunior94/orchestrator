package qualitygate

import (
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"strings"
)

func ComputeStateDigest(workDir string) string {
	head, headErr := runGit(workDir, "rev-parse", "HEAD")
	status, statusErr := runGit(workDir, "status", "--porcelain=v1")
	if headErr != nil && statusErr != nil {
		return ""
	}

	diff, diffErr := runGit(workDir, "diff", "HEAD")
	if diffErr != nil {
		diff = ""
	}

	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(head))
	builder.WriteByte('\n')
	builder.WriteString(status)
	builder.WriteByte('\n')
	builder.WriteString(diff)

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func runGit(workDir string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", workDir}, args...)
	cmd := exec.Command("git", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
