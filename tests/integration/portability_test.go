//go:build integration

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// TestPortability_GenerateGovernance valida o caminho critico que o usuario final
// pisa em outro repositorio: copiar uma fixture para t.TempDir(), rodar o
// gerador `generate-governance.sh` no modo 3-CLI (Claude+Codex+Copilot), assertar
// que os 4 arquivos canonicos sao gerados, marcadores de merge presentes, e
// reexecucao e idempotente preservando customizacoes.
//
// Cobre Frentes 1, 2, 5 e parte da 3 do plano production-proof. NAO depende do
// binario ai-spec-harness (testa apenas o gerador shell, que e o caminho mais
// frequente em projetos terceiros que so copiaram .agents/).
func TestPortability_GenerateGovernance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script generator requer bash; pular Windows")
	}
	repoRoot := findRepoRoot(t)
	generator := filepath.Join(repoRoot, ".agents", "skills", "analyze-project", "scripts", "generate-governance.sh")
	if _, err := os.Stat(generator); err != nil {
		t.Fatalf("generator script ausente em %s: %v", generator, err)
	}

	cases := []struct {
		name            string
		fixturePath     string
		wantStackSub    string // substring esperado em "Stack detectada:" no AGENTS.md
		skipSkillsIndex bool   // Java nao tem skill de implementacao (RF-32 nao cobre a skill); sem referencia a INDEX.yaml
	}{
		{name: "go-monolith", fixturePath: filepath.Join("testdata", "go-monolith"), wantStackSub: "Go"},
		{name: "node-api", fixturePath: filepath.Join("testdata", "node-api"), wantStackSub: "Node.js"},
		{name: "polyglot-monorepo", fixturePath: filepath.Join("testdata", "polyglot-monorepo"), wantStackSub: "Go, Node.js, Python"},
		{name: "dotnet-api", fixturePath: filepath.Join("testdata", "dotnet-api"), wantStackSub: "C#/.NET"},
		{name: "java-maven", fixturePath: filepath.Join("testdata", "java-maven"), wantStackSub: "Java/Kotlin", skipSkillsIndex: true},
		{name: "java-gradle", fixturePath: filepath.Join("testdata", "java-gradle"), wantStackSub: "Java/Kotlin", skipSkillsIndex: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workDir := t.TempDir()
			copyDirRecursive(t, filepath.Join(repoRoot, tc.fixturePath), workDir)

			// Execucao 1: cria do zero com 3 CLIs inegociaveis.
			runGenerator(t, generator, workDir, map[string]string{
				"INSTALL_CLAUDE":  "1",
				"INSTALL_CODEX":   "1",
				"INSTALL_COPILOT": "1",
			})

			canonical := []string{
				"AGENTS.md",
				"CLAUDE.md",
				filepath.Join(".codex", "config.toml"),
				filepath.Join(".github", "copilot-instructions.md"),
			}
			for _, f := range canonical {
				path := filepath.Join(workDir, f)
				info, err := os.Stat(path)
				if err != nil {
					t.Fatalf("arquivo canonico ausente %s: %v", f, err)
				}
				if info.Size() == 0 {
					t.Fatalf("arquivo canonico vazio %s", f)
				}
			}

			// Marcadores de merge inteligente presentes em todos os 4.
			assertContains(t, filepath.Join(workDir, "AGENTS.md"), "<!-- ai-spec:generated-start")
			assertContains(t, filepath.Join(workDir, "AGENTS.md"), "<!-- ai-spec:generated-end")
			assertContains(t, filepath.Join(workDir, "CLAUDE.md"), "<!-- ai-spec:generated-start")
			assertContains(t, filepath.Join(workDir, ".codex", "config.toml"), "# ai-spec:generated-start")
			assertContains(t, filepath.Join(workDir, ".github", "copilot-instructions.md"), "<!-- ai-spec:generated-start")

			// Stack detectada bate com a esperada.
			assertContains(t, filepath.Join(workDir, "AGENTS.md"), tc.wantStackSub)

			// Bloco de skills surgical aparece quando ha stack com skill de implementacao.
			if !tc.skipSkillsIndex {
				assertContains(t, filepath.Join(workDir, "AGENTS.md"), "references/INDEX.yaml")
			}

			// Codex tem sandbox + approval (ADR-002 fecha lacuna de route-around).
			assertContains(t, filepath.Join(workDir, ".codex", "config.toml"), "sandbox_mode = \"workspace-write\"")
			assertContains(t, filepath.Join(workDir, ".codex", "config.toml"), "approval_policy = \"on-request\"")

			// Customizacao do usuario FORA dos marcadores precisa sobreviver
			// a reinstalacao (merge inteligente).
			customMark := "## Convencoes Customizadas Test\n\n- Marco unico de teste portabilidade."
			appendToFile(t, filepath.Join(workDir, "AGENTS.md"), "\n"+customMark+"\n")

			// Execucao 2: idempotente, customizacao preservada.
			runGenerator(t, generator, workDir, map[string]string{
				"INSTALL_CLAUDE":  "1",
				"INSTALL_CODEX":   "1",
				"INSTALL_COPILOT": "1",
			})
			assertContains(t, filepath.Join(workDir, "AGENTS.md"), "Convencoes Customizadas Test")
			assertContains(t, filepath.Join(workDir, "AGENTS.md"), "Marco unico de teste portabilidade.")
		})
	}
}

// TestPortability_ResolveReferencesSurgical valida o nucleo de Frente 3: o
// resolver carrega *apenas* references cujo escopo bate com a tarefa, e zero
// quando a tarefa nao toca codigo (economia de tokens).
func TestPortability_ResolveReferencesSurgical(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell resolver requer bash")
	}
	repoRoot := findRepoRoot(t)
	resolver := filepath.Join(repoRoot, ".agents", "scripts", "resolve-references.sh")
	if _, err := os.Stat(resolver); err != nil {
		t.Fatalf("resolver ausente em %s: %v", resolver, err)
	}

	type scenario struct {
		name      string
		skill     string
		files     []string
		diffStdin string
		want      []string // ids que DEVEM aparecer
		notWant   []string // ids que NAO devem aparecer (proteccao economia)
	}
	scenarios := []scenario{
		{
			name:    "go config.yaml carrega so always",
			skill:   "go-implementation",
			files:   []string{"config.yaml"},
			want:    []string{"architecture"},
			notWant: []string{"persistence", "examples-testing", "messaging"},
		},
		{
			name:      "go repository sql carrega persistence",
			skill:     "go-implementation",
			files:     []string{"internal/repository/user.go"},
			diffStdin: "+ rows, err := db.Query(\"SELECT * FROM users\")",
			want:      []string{"architecture", "persistence"},
			notWant:   []string{"messaging", "graceful-lifecycle"},
		},
		{
			name:      "node test file carrega examples-testing",
			skill:     "node-implementation",
			files:     []string{"src/user.test.ts"},
			diffStdin: "describe('UserService', () => { it('works', () => {}) })",
			want:      []string{"architecture", "examples-testing"},
			notWant:   []string{"persistence", "messaging"},
		},
		{
			name:      "dotnet DbContext carrega persistence",
			skill:     "dotnet-csharp-implementation",
			files:     []string{"src/Infrastructure/UserRepo.cs"},
			diffStdin: "+ public class UserRepo : DbContext { }",
			want:      []string{"architecture", "persistence"},
			notWant:   []string{"messaging", "graceful-lifecycle"},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			args := append([]string{resolver, sc.skill}, sc.files...)
			cmd := exec.Command("bash", args...)
			cmd.Env = append(os.Environ(), "AGENTS_ROOT="+repoRoot)
			if sc.diffStdin != "" {
				cmd.Stdin = strings.NewReader(sc.diffStdin)
			}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("resolver exit=%v out=%s", err, string(out))
			}
			outStr := string(out)
			for _, id := range sc.want {
				if !strings.Contains(outStr, sc.skill+"/"+id) {
					t.Errorf("esperava ref %s/%s na saida, got:\n%s", sc.skill, id, outStr)
				}
			}
			for _, id := range sc.notWant {
				if strings.Contains(outStr, sc.skill+"/"+id) {
					t.Errorf("ref %s/%s NAO deveria aparecer (economia), got:\n%s", sc.skill, id, outStr)
				}
			}
		})
	}
}

// TestPortability_PrerequisitesBlocksMissingSkill garante que o gate bloqueia
// quando a skill obrigatoria nao esta instalada — protecao contra alucinacao
// por falta de descoberta.
func TestPortability_PrerequisitesBlocksMissingSkill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell validator requer bash")
	}
	repoRoot := findRepoRoot(t)
	validator := filepath.Join(repoRoot, ".agents", "scripts", "validate-skill-prerequisites.sh")

	// Diretorio TEMPORARIO sem .agents/skills/ → simula projeto sem governanca.
	emptyRoot := t.TempDir()
	cmd := exec.Command("bash", validator, "foo.go")
	cmd.Env = append(os.Environ(), "AGENTS_ROOT="+emptyRoot)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("esperava bloqueio (exit != 0) sem .agents/skills/, mas passou. saida: %s", out)
	}
	if !strings.Contains(string(out), "go-implementation") {
		t.Errorf("mensagem de erro nao menciona skill esperada: %s", out)
	}

	// Mesmo cmd com AGENTS_ROOT do repo (skills presentes) deve passar.
	cmd2 := exec.Command("bash", validator, "foo.go")
	cmd2.Env = append(os.Environ(), "AGENTS_ROOT="+repoRoot)
	if out2, err := cmd2.CombinedOutput(); err != nil {
		t.Fatalf("repo com skills falhou no validador: err=%v out=%s", err, out2)
	}
}

// TestPortability_HookGateCrossCLI valida que o gate de descoberta cirurgical
// se comporta IDENTICAMENTE invocado por Claude, Codex e Copilot — paridade
// inegociavel. Simula o JSON stdin do PreToolUse de cada CLI.
func TestPortability_HookGateCrossCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hooks shell-only")
	}
	repoRoot := findRepoRoot(t)

	// /tmp project com .agents/skills/ via cp do canonico — simula instalacao real
	// num repo terceiro. Hook deve resolver a partir desse AGENTS_ROOT.
	project := t.TempDir()
	copyDirRecursive(t, filepath.Join(repoRoot, ".agents"), filepath.Join(project, ".agents"))

	type hookCase struct {
		name    string
		hook    string
		wantOK  bool   // true = exit 0
		stdin   string // JSON PreToolUse simulado
		mustSee string // substring esperado em stderr
	}

	jsonForFile := func(absPath string) string {
		return `{"tool_input":{"file_path":"` + absPath + `"}}`
	}

	editGo := filepath.Join(project, "internal", "repository", "user.go")
	editPy := filepath.Join(project, "app", "main.py")
	editMd := filepath.Join(project, "README.md")

	cases := []hookCase{
		// Caminho feliz: edita .go, gate emite guidance, exit 0 (com PRELOAD_CONFIRMED).
		{name: "claude-go-ok", hook: filepath.Join(repoRoot, ".claude/hooks/validate-preload.sh"), stdin: jsonForFile(editGo), wantOK: true, mustSee: "go-implementation/persistence"},
		{name: "codex-go-ok", hook: filepath.Join(repoRoot, ".codex/hooks/validate-preload.sh"), stdin: jsonForFile(editGo), wantOK: true, mustSee: "go-implementation/persistence"},
		{name: "codex-apply-patch-go-ok", hook: filepath.Join(repoRoot, ".codex/hooks/validate-preload.sh"), stdin: `{"input":{"arguments":{"patchText":"*** Add File: ` + editGo + `\n+package repository\n"}}}`, wantOK: true, mustSee: "go-implementation/persistence"},
		{name: "copilot-go-ok", hook: filepath.Join(repoRoot, ".github/hooks/validate-preload.sh"), stdin: jsonForFile(editGo), wantOK: true, mustSee: "go-implementation/persistence"},
		// Edita Python: guidance Python.
		{name: "claude-py-ok", hook: filepath.Join(repoRoot, ".claude/hooks/validate-preload.sh"), stdin: jsonForFile(editPy), wantOK: true, mustSee: "python-implementation/architecture"},
		{name: "codex-py-ok", hook: filepath.Join(repoRoot, ".codex/hooks/validate-preload.sh"), stdin: jsonForFile(editPy), wantOK: true, mustSee: "python-implementation/architecture"},
		// Edita .md: zero ruido, sem GUIDANCE header.
		{name: "claude-md-silent", hook: filepath.Join(repoRoot, ".claude/hooks/validate-preload.sh"), stdin: jsonForFile(editMd), wantOK: true, mustSee: ""},
		{name: "copilot-md-silent", hook: filepath.Join(repoRoot, ".github/hooks/validate-preload.sh"), stdin: jsonForFile(editMd), wantOK: true, mustSee: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("bash", tc.hook)
			cmd.Stdin = strings.NewReader(tc.stdin)
			cmd.Env = append(os.Environ(),
				"GOVERNANCE_PRELOAD_CONFIRMED=1",
				"AGENTS_ROOT="+project,
			)
			out, err := cmd.CombinedOutput()
			if tc.wantOK && err != nil {
				t.Fatalf("esperava exit 0; got err=%v out=%s", err, out)
			}
			if !tc.wantOK && err == nil {
				t.Fatalf("esperava bloqueio; got exit 0 out=%s", out)
			}
			if tc.mustSee != "" && !strings.Contains(string(out), tc.mustSee) {
				t.Errorf("esperava %q em stderr; got=%s", tc.mustSee, out)
			}
			if tc.name == "claude-md-silent" || tc.name == "copilot-md-silent" {
				if strings.Contains(string(out), "GUIDANCE") {
					t.Errorf("esperava silencio para .md; got GUIDANCE no output: %s", out)
				}
			}
		})
	}
}

// TestPortability_HookBlocksWhenSkillsMissing garante que sem .agents/skills/
// o hook bloqueia em todos os 3 CLIs. Critico para repos novos.
func TestPortability_HookBlocksWhenSkillsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hooks shell-only")
	}
	repoRoot := findRepoRoot(t)
	emptyProject := t.TempDir() // sem .agents/

	hooks := map[string]string{
		"claude":  filepath.Join(repoRoot, ".claude/hooks/validate-preload.sh"),
		"codex":   filepath.Join(repoRoot, ".codex/hooks/validate-preload.sh"),
		"copilot": filepath.Join(repoRoot, ".github/hooks/validate-preload.sh"),
	}

	stdin := `{"tool_input":{"file_path":"` + filepath.Join(emptyProject, "main.go") + `"}}`

	for cli, hook := range hooks {
		t.Run(cli, func(t *testing.T) {
			cmd := exec.Command("bash", hook)
			cmd.Stdin = strings.NewReader(stdin)
			cmd.Env = append(os.Environ(),
				"GOVERNANCE_PRELOAD_CONFIRMED=1",
				"AGENTS_ROOT="+emptyProject,
			)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("%s: esperava bloqueio sem skills, exit 0. out=%s", cli, out)
			}
			if !strings.Contains(string(out), "go-implementation") {
				t.Errorf("%s: msg de erro nao menciona skill ausente. out=%s", cli, out)
			}
		})
	}
}

// TestPortability_EconomyZeroRefsOnConfigEdit confirma que editar config.yaml
// nao infla o contexto — so refs always:true sao listadas (1 ref carregada).
// Esta e a invariante de economia inegociavel: sem signal nem padrao, zero
// references especificas vazam para o contexto do agente.
func TestPortability_EconomyZeroRefsOnConfigEdit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hooks shell-only")
	}
	repoRoot := findRepoRoot(t)
	resolver := filepath.Join(repoRoot, ".agents", "scripts", "resolve-references.sh")

	cmd := exec.Command("bash", resolver, "go-implementation", "config.yaml")
	cmd.Env = append(os.Environ(), "AGENTS_ROOT="+repoRoot)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("resolver falhou: %v %s", err, out)
	}
	// Conta linhas nao-vazias (cada linha = uma ref carregada).
	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			lines++
		}
	}
	if lines > 1 {
		t.Errorf("config.yaml deveria gerar 1 ref (architecture); got %d:\n%s", lines, out)
	}
	if !strings.Contains(string(out), "architecture") {
		t.Errorf("esperava architecture (always); got:\n%s", out)
	}
}

func TestPortability_CLISwitchWithoutReinstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hooks shell-only")
	}
	repoRoot := findRepoRoot(t)
	projectDir := t.TempDir()

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	installSvc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	allTools := []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode}

	if err := installSvc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		Tools:       allTools,
		Langs:       []skills.Lang{skills.LangGo},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	}); err != nil {
		t.Fatalf("instalacao unica com 4 CLIs falhou: %v", err)
	}

	items, err := installSvc.Verify(config.InstallOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("verify pos-instalacao unica falhou: %v", err)
	}
	seenByTool := map[skills.Tool]bool{}
	for _, it := range items {
		if it.Tool == "" {
			continue
		}
		seenByTool[it.Tool] = true
		if it.State == install.VerifyStateDrifted {
			t.Errorf("item %s/%s drifted logo apos instalacao unica com 4 CLIs", it.Tool, it.Skill)
		}
	}
	for _, tool := range allTools {
		if !seenByTool[tool] {
			t.Errorf("nenhum item verificado para %s; instalacao unica nao serviu os 4 provedores", tool)
		}
	}

	editGo := filepath.Join(projectDir, "internal", "repository", "user.go")
	stdin := `{"tool_input":{"file_path":"` + editGo + `"}}`
	hooks := []struct {
		cli  string
		path string
	}{
		{"claude", filepath.Join(repoRoot, ".claude/hooks/validate-preload.sh")},
		{"codex", filepath.Join(repoRoot, ".codex/hooks/validate-preload.sh")},
		{"copilot", filepath.Join(repoRoot, ".github/hooks/validate-preload.sh")},
	}
	for _, h := range hooks {
		cmd := exec.Command("bash", h.path)
		cmd.Stdin = strings.NewReader(stdin)
		cmd.Env = append(os.Environ(),
			"GOVERNANCE_PRELOAD_CONFIRMED=1",
			"AGENTS_ROOT="+projectDir,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: gate de descoberta falhou na MESMA instalacao (RF-53); err=%v out=%s", h.cli, err, out)
		}
		if !strings.Contains(string(out), "go-implementation") {
			t.Errorf("%s: esperava guidance go-implementation na mesma instalacao; out=%s", h.cli, out)
		}
	}

	pluginPath := filepath.Join(projectDir, ".opencode", "plugin", "governance.js")
	info, err := os.Stat(pluginPath)
	if err != nil {
		t.Fatalf("opencode: plugin de governanca ausente na mesma instalacao: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("opencode: plugin de governanca vazio na mesma instalacao")
	}
}

func TestPortability_NoDaemonRequired(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shebang/systemd sao conceitos POSIX")
	}
	projectDir := t.TempDir()

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	installSvc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	allTools := []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode}
	if err := installSvc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		Tools:       allTools,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	}); err != nil {
		t.Fatalf("instalacao para checagem de daemon falhou: %v", err)
	}

	forbiddenSuffixes := []string{".service", ".plist", ".pid", ".sock"}
	forbiddenNameSubstrings := []string{"daemon", "supervisor", "systemd", "launchd"}
	var scriptCount int
	err := filepath.Walk(projectDir, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() {
			return nil
		}
		name := strings.ToLower(fi.Name())
		for _, suf := range forbiddenSuffixes {
			if strings.HasSuffix(name, suf) {
				t.Errorf("artefato instalado com extensao de processo permanente: %s", path)
			}
		}
		for _, sub := range forbiddenNameSubstrings {
			if strings.Contains(name, sub) {
				t.Errorf("artefato instalado sugere daemon/supervisor: %s", path)
			}
		}
		if strings.HasSuffix(name, ".sh") {
			scriptCount++
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("ler script instalado %s: %v", path, readErr)
			}
			firstLine := strings.SplitN(string(data), "\n", 2)[0]
			if !strings.HasPrefix(firstLine, "#!/") {
				t.Errorf("script instalado sem shebang one-shot: %s (primeira linha: %q)", path, firstLine)
			}
			if strings.Contains(firstLine, "systemd-run") {
				t.Errorf("script instalado invoca supervisor via shebang: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varredura do diretorio instalado falhou: %v", err)
	}
	if scriptCount == 0 {
		t.Fatalf("nenhum script .sh encontrado na instalacao; fixture nao exerce a asserção")
	}
	t.Logf("%d scripts instalados, todos one-shot via shebang, nenhum artefato de daemon", scriptCount)
}

// --- helpers ---

func findRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod nao encontrado em ancestrais de %s", cwd)
		}
		dir = parent
	}
}

func copyDirRecursive(t *testing.T, src, dst string) {
	t.Helper()
	cmd := exec.Command("cp", "-R", src+"/.", dst+"/")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("copyDir %s -> %s: %v\n%s", src, dst, err, out)
	}
}

func runGenerator(t *testing.T, script, workDir string, env map[string]string) {
	t.Helper()
	cmd := exec.Command("bash", script, workDir)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generator falhou em %s: err=%v out=%s", workDir, err, out)
	}
}

func assertContains(t *testing.T, path, needle string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), needle) {
		t.Fatalf("%s nao contem %q", path, needle)
	}
}

func appendToFile(t *testing.T, path, extra string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open append %s: %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(extra); err != nil {
		t.Fatalf("write append %s: %v", path, err)
	}
}
