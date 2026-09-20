//go:build integration

package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

func TestAtomicWriteUnderPRDDoesNotAlterPatchSHA256(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatal(err)
	}
	runGitSetupCommand(t, dir, "init")
	runGitSetupCommand(t, dir, "config", "user.email", "integration@example.com")
	runGitSetupCommand(t, dir, "config", "user.name", "Integration")
	runGitSetupCommand(t, dir, "add", ".")
	runGitSetupCommand(t, dir, "-c", "commit.gpgSign=false", "commit", "-m", "base")

	orchestrator := taskloop.NewOrchestrator(store)
	startSnapshot, err := orchestrator.CaptureSnapshotFromGit(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.Start(dir, "run", "1.0", 1, startSnapshot); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "change.go"), []byte("package change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	evidenceDir := filepath.Join(dir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logContent := []byte("PASS\n")
	if err := os.WriteFile(filepath.Join(evidenceDir, "test.log"), logContent, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(logContent)

	result := sdd.ExecutionResult{
		SchemaVersion: sdd.SchemaVersion,
		RunID:         "run",
		TaskID:        "1.0",
		Attempt:       1,
		Status:        sdd.StatusDone,
		BaseSHA:       startSnapshot.BaseSHA,
		PatchRef:      "evidence/patch.diff",
		Tests:         []sdd.TestProof{{Command: "go test ./...", ExitCode: 0, OutputSHA256: hex.EncodeToString(digest[:])}},
		Criteria:      []sdd.CriterionProof{{ID: "AC-01", EvidenceRef: "evidence/test.log#criterio"}},
		Evidence:      []string{"evidence/test.log"},
		ReviewVerdict: "approved",
	}

	excluded := map[string]bool{
		"evidence/test.log":     true,
		result.PatchRef:         true,
		"sdd-state.json":        true,
		".sdd-orchestrate.lock": true,
		".checkpoints/**":       true,
		"**/.tmp-*":             true,
	}

	osFS := fs.NewOSFileSystem()
	targetPath := filepath.Join(dir, "sub", "target.json")
	payload := []byte(`{"race":true}`)
	if err := osFS.WriteFileAtomic(targetPath, payload); err != nil {
		t.Fatalf("preparar arquivo alvo da escrita atomica: %v", err)
	}

	patchWithoutRace, _ := canonicalPatch(t, dir, excluded)
	baselineDigest := sha256.Sum256(patchWithoutRace)
	baselineSHA := hex.EncodeToString(baselineDigest[:])

	stop := make(chan struct{})
	var raceRuns atomic.Int64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				if err := osFS.WriteFileAtomic(targetPath, payload); err != nil {
					t.Errorf("escrita atomica concorrente falhou: %v", err)
					return
				}
				raceRuns.Add(1)
			}
		}
	}()

	sawTmpFileDuringRace := false
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		patchDuringRace, sawTmp := canonicalPatch(t, dir, excluded)
		if sawTmp {
			sawTmpFileDuringRace = true
		}
		duringDigest := sha256.Sum256(patchDuringRace)
		duringSHA := hex.EncodeToString(duringDigest[:])
		if duringSHA != baselineSHA {
			close(stop)
			wg.Wait()
			t.Fatalf("PatchSHA256 divergiu durante escrita atomica concorrente: base=%s durante=%s", baselineSHA, duringSHA)
		}
	}
	close(stop)
	wg.Wait()

	if raceRuns.Load() == 0 {
		t.Fatal("goroutine de escrita atomica concorrente nao chegou a executar nenhuma vez")
	}
	if !sawTmpFileDuringRace {
		t.Fatal("corrida nao observada: nenhuma amostra capturou um arquivo .tmp-* pendente; a corrida foi presumida, nao exercitada")
	}

	finalPatch, _ := canonicalPatch(t, dir, excluded)
	finalSnapshot := taskloop.NewSnapshot(startSnapshot.BaseSHA, string(finalPatch), startSnapshot.BaseSHA+"\n"+string(finalPatch))
	result.PatchSHA256 = finalSnapshot.PatchSHA256
	result.FinalStateSHA256 = finalSnapshot.FinalStateSHA256
	if err := os.WriteFile(filepath.Join(dir, result.PatchRef), finalPatch, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := orchestrator.ValidateExecutionEvidence(dir, result); err != nil {
		t.Fatalf("escrita atomica sob o PRD nao deveria disparar o fail-closed de divergencia: %v", err)
	}
	if _, err := orchestrator.Finish(dir, result); err != nil {
		t.Fatalf("Finish deveria aceitar resultado apos escrita atomica concorrente sob o PRD: %v", err)
	}
}

func runGitSetupCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func canonicalPatch(t *testing.T, dir string, excluded map[string]bool) ([]byte, bool) {
	t.Helper()
	args := []string{"diff", "--binary", "HEAD", "--", "."}
	keys := make([]string, 0, len(excluded))
	for path := range excluded {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	for _, path := range keys {
		if path == "**/.tmp-*" {
			args = append(args, ":(glob,exclude)"+path)
			continue
		}
		args = append(args, ":(exclude)"+path)
	}
	trackedCommand := exec.Command("git", args...)
	trackedCommand.Dir = dir
	tracked, err := trackedCommand.Output()
	if err != nil {
		t.Fatalf("git diff rastreado: %v", err)
	}

	untrackedCommand := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z")
	untrackedCommand.Dir = dir
	untrackedRaw, err := untrackedCommand.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}

	patch := append([]byte(nil), tracked...)
	untracked := strings.Split(strings.TrimSuffix(string(untrackedRaw), "\x00"), "\x00")
	sort.Strings(untracked)
	sawTmp := false
	for _, path := range untracked {
		path = filepath.ToSlash(path)
		if path == "" {
			continue
		}
		if strings.HasPrefix(filepath.Base(path), ".tmp-") {
			sawTmp = true
		}
		if isCanonicallyExcluded(path, excluded) {
			continue
		}
		diffCommand := exec.Command("git", "diff", "--binary", "--no-index", "--", os.DevNull, path)
		diffCommand.Dir = dir
		output, diffErr := diffCommand.CombinedOutput()
		if diffErr != nil {
			var exitErr *exec.ExitError
			if !errors.As(diffErr, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("git diff --no-index %s: %v: %s", path, diffErr, output)
			}
		}
		patch = append(patch, output...)
	}
	return patch, sawTmp
}

func isCanonicallyExcluded(path string, excluded map[string]bool) bool {
	if excluded[path] {
		return true
	}
	for excludedPath := range excluded {
		if strings.HasSuffix(excludedPath, "/**") && strings.HasPrefix(path, strings.TrimSuffix(excludedPath, "**")) {
			return true
		}
		if excludedPath == "**/.tmp-*" && strings.HasPrefix(filepath.Base(path), ".tmp-") {
			return true
		}
	}
	return false
}
