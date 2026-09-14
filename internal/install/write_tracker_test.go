package install

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type trackerHarness struct{}

func (h trackerHarness) install(t *testing.T, projectDir string, tools []skills.Tool) *manifest.Manifest {
	t.Helper()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	store := manifest.NewStore(fsys)
	svc := NewService(fsys, printer, store, adapters.NewGenerator(fsys, printer), contextgen.NewGenerator(fsys, printer))
	err := svc.Execute(config.InstallOptions{
		ProjectDir:   projectDir,
		Tools:        tools,
		Langs:        []skills.Lang{skills.LangGo},
		LinkMode:     skills.LinkCopy,
		GenerateCtx:  true,
		CodexProfile: "full",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	mf, err := store.Load(projectDir)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	return mf
}

func (h trackerHarness) filesOnDisk(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == manifest.ManifestFile {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	sort.Strings(out)
	return out
}

func TestManifestTracksExactlyWhatInstallWrote(t *testing.T) {
	harness := trackerHarness{}
	root := t.TempDir()
	mf := harness.install(t, root, []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode})

	tracked := make(map[string]bool, len(mf.InstalledFiles)+len(mf.MergedFiles))
	for _, rel := range mf.InstalledFiles {
		tracked[rel] = true
	}
	for _, rel := range mf.MergedFiles {
		tracked[rel] = true
	}

	onDisk := harness.filesOnDisk(t, root)
	for _, rel := range onDisk {
		if !tracked[rel] {
			t.Errorf("arquivo criado pelo install nao rastreado no manifesto: %s", rel)
		}
		delete(tracked, rel)
	}
	for rel := range tracked {
		t.Errorf("manifesto rastreia caminho inexistente em disco: %s", rel)
	}
	if len(onDisk) == 0 {
		t.Fatal("install nao escreveu nenhum arquivo")
	}
}

func TestManifestNeverTracksUntouchedUserFiles(t *testing.T) {
	harness := trackerHarness{}
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# claude do usuario\n"), 0o644); err != nil {
		t.Fatalf("seed CLAUDE.md: %v", err)
	}
	pluginDir := filepath.Join(root, ".opencode", "plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "my-user-plugin.js"), []byte("export const p = {}\n"), 0o644); err != nil {
		t.Fatalf("seed plugin: %v", err)
	}

	mf := harness.install(t, root, []skills.Tool{skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode})

	for _, rel := range mf.InstalledFiles {
		if rel == "CLAUDE.md" {
			t.Error("CLAUDE.md nunca foi escrito por este install e nao pode entrar no manifesto")
		}
		if strings.HasSuffix(rel, "my-user-plugin.js") {
			t.Error("plugin do usuario nao pode entrar no manifesto")
		}
		if rel == filepath.Join(".opencode", "plugin") {
			t.Error("manifesto deve rastrear arquivos, nunca o diretorio .opencode/plugin")
		}
	}
	if len(mf.MergedFiles) == 0 {
		t.Error("opencode.json/.github/settings.json deveriam ser rastreados como merged")
	}
}
