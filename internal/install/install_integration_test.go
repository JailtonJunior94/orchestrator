//go:build integration

package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// fakeLookPatherInt implementa probe.LookPather para testes de integracao.
type fakeLookPatherInt struct {
	found map[string]string
}

func newFakeLookPatherInt(available ...string) *fakeLookPatherInt {
	f := &fakeLookPatherInt{found: make(map[string]string)}
	for _, name := range available {
		f.found[name] = "/usr/bin/" + name
	}
	return f
}

func (f *fakeLookPatherInt) LookPath(name string) (string, error) {
	if p, ok := f.found[name]; ok {
		return p, nil
	}
	return "", errors.New("not found: " + name)
}

// newRealService cria um Service com OSFileSystem para testes de integracao.
func newRealService(t *testing.T) *Service {
	t.Helper()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	return NewService(fsys, printer, mfst, adpt, ctxg)
}

// createMinimalSource cria um diretorio fonte com uma skill minima para testes.
func createMinimalSource(t *testing.T, sourceDir string, skills []string) {
	t.Helper()
	for _, sk := range skills {
		skillDir := filepath.Join(sourceDir, ".agents", "skills", sk)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("criar skill dir %s: %v", sk, err)
		}
		content := []byte("---\nversion: 1.0.0\n---\n# " + sk + "\nconteudo de teste")
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0o644); err != nil {
			t.Fatalf("criar SKILL.md para %s: %v", sk, err)
		}
	}
}

// TestIntegration_Bootstrap_LessThan30s verifica que o bootstrap em repo vazio
// com agentes detectados conclui em menos de 30 segundos (RF-11).
func TestIntegration_Bootstrap_LessThan30s(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Criar skill minima na fonte.
	createMinimalSource(t, sourceDir, []string{"review", "execute-task"})

	svc := newRealService(t)

	start := time.Now()
	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("install falhou: %v", err)
	}

	if elapsed > 30*time.Second {
		t.Errorf("bootstrap demorou %v, limite e 30s (RF-11)", elapsed)
	}

	t.Logf("bootstrap concluido em %v", elapsed)
}

// TestIntegration_Idempotency_InstallTwice_VerifyAllCurrent verifica que install
// executado duas vezes resulta em 100% current para as skills instaladas (RF-09).
// Verifica apenas as skills presentes na fonte (nao o catalogo completo de AllSkills).
func TestIntegration_Idempotency_InstallTwice_VerifyAllCurrent(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	installedSkills := []string{"review", "execute-task"}
	createMinimalSource(t, sourceDir, installedSkills)

	svc := newRealService(t)
	opts := config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	}

	// Primeira instalacao.
	if err := svc.Execute(opts); err != nil {
		t.Fatalf("primeira instalacao falhou: %v", err)
	}

	// Segunda instalacao (idempotencia: convergencia).
	if err := svc.Execute(opts); err != nil {
		t.Fatalf("segunda instalacao falhou: %v", err)
	}

	// Verify somente as skills instaladas (nao o catalogo completo).
	// Verificar diretamente usando verifySkillFile para cada skill instalada.
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	svcVerify := NewService(fsys, printer, mfst, adpt, ctxg)

	for _, sk := range installedSkills {
		state := svcVerify.verifySkillFile(sourceDir, projectDir, sk)
		if state != VerifyStateCurrent {
			t.Errorf("idempotencia violada: skill %s got %v, want current", sk, state)
		}
	}
	t.Logf("idempotencia confirmada para skills: %v", installedSkills)
}

// TestIntegration_GlobalScope_HOME_Controlled verifica que install com ScopeGlobal
// materializa assets em ~/.aispec (HOME controlado via TempDir) (RF-07).
func TestIntegration_GlobalScope_HOME_Controlled(t *testing.T) {
	// t.Setenv nao pode ser usado com t.Parallel()
	tmpHome := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSource(t, sourceDir, []string{"review"})

	// Controlar HOME para o teste (R-SEC-001: paths via os.UserHomeDir).
	t.Setenv("HOME", tmpHome)

	expectedGlobalDir := filepath.Join(tmpHome, ".aispec")

	svc := newRealService(t)
	err := svc.Execute(config.InstallOptions{
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		Scope:       config.ScopeGlobal,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("install global falhou: %v", err)
	}

	// Verificar que ~/.aispec/.agents/skills/review/SKILL.md existe.
	installedSkill := filepath.Join(expectedGlobalDir, ".agents", "skills", "review", "SKILL.md")
	if _, err := os.Stat(installedSkill); os.IsNotExist(err) {
		t.Errorf("skill 'review' nao foi instalada globalmente em %s", installedSkill)
	}

	// Verify deve retornar current para review.
	items, err := svc.Verify(config.InstallOptions{
		SourceDir: sourceDir,
		Tools:     []skills.Tool{skills.ToolClaude},
		Scope:     config.ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("Verify global falhou: %v", err)
	}

	found := false
	for _, item := range items {
		if item.Skill == "review" {
			found = true
			if item.State != VerifyStateCurrent {
				t.Errorf("skill review global: got %v, want current", item.State)
			}
		}
	}
	if !found {
		t.Error("skill 'review' nao encontrada nos resultados de verify global")
	}
}

// TestIntegration_GlobalScope_NoHome_ExplicitError verifica que escopo global sem
// $HOME retorna erro explicito (nao silencioso) (ADR-019).
func TestIntegration_GlobalScope_NoHome_ExplicitError(t *testing.T) {
	// Limpar HOME para simular ausencia.
	t.Setenv("HOME", "")

	svc := newRealService(t)
	err := svc.Execute(config.InstallOptions{
		SourceDir: t.TempDir(),
		Tools:     []skills.Tool{skills.ToolClaude},
		Scope:     config.ScopeGlobal,
	})

	if err == nil {
		t.Fatal("esperava erro quando HOME ausente com ScopeGlobal")
	}
	t.Logf("erro esperado com HOME ausente: %v", err)
}

// TestIntegration_Idempotency_VerifyAfterMutation verifica que mutar skill
// resulta em drifted, e install corrige -> verify current.
func TestIntegration_Idempotency_VerifyAfterMutation(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSource(t, sourceDir, []string{"review"})

	svc := newRealService(t)
	opts := config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("instalacao inicial falhou: %v", err)
	}

	// Mutar skill instalada.
	installedSkill := filepath.Join(projectDir, ".agents", "skills", "review", "SKILL.md")
	if err := os.WriteFile(installedSkill, []byte("conteudo-mutado"), 0o644); err != nil {
		t.Fatalf("falha ao mutar skill: %v", err)
	}

	// Verify deve reportar drifted.
	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
	})
	if err != nil {
		t.Fatalf("Verify pos-mutacao falhou: %v", err)
	}

	foundDrifted := false
	for _, item := range items {
		if item.Skill == "review" && item.State == VerifyStateDrifted {
			foundDrifted = true
		}
	}
	if !foundDrifted {
		t.Error("skill mutada deveria estar drifted, mas nao foi detectada como tal")
	}

	// Re-instalar para corrigir.
	if err := svc.Execute(opts); err != nil {
		t.Fatalf("re-instalacao falhou: %v", err)
	}

	// Verify deve retornar current apos correcao.
	items, err = svc.Verify(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
	})
	if err != nil {
		t.Fatalf("Verify pos-correcao falhou: %v", err)
	}

	for _, item := range items {
		if item.Skill == "review" && item.State != VerifyStateCurrent {
			t.Errorf("skill corrigida: got %v, want current", item.State)
		}
	}
}

// --- Testes Tarefa 7.0: Cenarios P/M/G (subtarefa 7.5) ---

// newRealServiceWithLookPather cria Service real com LookPather injetavel.
func newRealServiceWithLookPather(t *testing.T, lp *fakeLookPatherInt) *Service {
	t.Helper()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	agentDet := detect.NewBinaryAgentDetector(detect.OSLookPather{}, detect.OSHomeDir{}, detect.NewFileDetector(fsys))
	langDet := detect.NewFileDetector(fsys)
	return NewServiceWithOptions(fsys, printer, mfst, adpt, ctxg, agentDet, langDet, lp)
}

// TestIntegration_7_5_ScenarioP_EmptyRepo verifica install em repo vazio (cenario P).
// Sem go.mod/package.json/pyproject.toml -> sem skill de linguagem auto-detectada.
// install + verify deve convergir sem erros. ADR-024.
func TestIntegration_7_5_ScenarioP_EmptyRepo(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSource(t, sourceDir, []string{"review", "execute-task"})

	lp := newFakeLookPatherInt() // sem binario ACP
	svc := newRealServiceWithLookPather(t, lp)

	start := time.Now()
	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		Langs:       nil, // auto-detect deve retornar vazio (repo P)
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("cenario P: install falhou: %v", err)
	}

	if elapsed > 30*time.Second {
		t.Errorf("cenario P: bootstrap demorou %v, limite e 30s (RF-11)", elapsed)
	}

	// Segunda execucao: idempotente.
	err = svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("cenario P: segunda instalacao falhou: %v", err)
	}

	// Verify deve reportar skills current.
	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
	})
	if err != nil {
		t.Fatalf("cenario P: verify falhou: %v", err)
	}

	// Verificar skills instaladas estao current.
	for _, item := range items {
		if item.Kind == VerifyKindBinary {
			continue // binary pode ser missing (sem PATH real neste cenario)
		}
		if item.Skill == "review" || item.Skill == "execute-task" {
			if item.State != VerifyStateCurrent {
				t.Errorf("cenario P: skill %s: got %v, want current", item.Skill, item.State)
			}
		}
	}

	t.Logf("cenario P concluido em %v", elapsed)
}

// TestIntegration_7_5_ScenarioM_GoRepo verifica install em repo Go existente (cenario M).
// Com go.mod -> go-implementation deve ser detectada e instalada automaticamente.
// ADR-024 RI-01.
func TestIntegration_7_5_ScenarioM_GoRepo(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Criar marcador de stack Go.
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/test\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	createMinimalSource(t, sourceDir, []string{"review", "go-implementation"})

	lp := newFakeLookPatherInt()
	svc := newRealServiceWithLookPather(t, lp)

	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		Langs:       nil, // RI-01: auto-detect deve encontrar Go
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("cenario M (Go): install falhou: %v", err)
	}

	// go-implementation deve ter sido instalada.
	goSkill := filepath.Join(projectDir, ".agents", "skills", "go-implementation", "SKILL.md")
	if _, err := os.Stat(goSkill); os.IsNotExist(err) {
		t.Error("cenario M (Go): RI-01 falhou — go-implementation nao instalada sem --langs")
	}

	// Reexecucao: idempotente.
	err = svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		Langs:       nil,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("cenario M (Go): segunda instalacao falhou: %v", err)
	}

	// Verify skills.
	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
	})
	if err != nil {
		t.Fatalf("cenario M (Go): verify falhou: %v", err)
	}

	foundGoSkill := false
	for _, item := range items {
		if item.Kind == VerifyKindBinary {
			// Verificar que item binary existe (RI-04).
			if item.Tool == skills.ToolClaude {
				// Com LookPather sem binario -> deve ser missing.
				if item.State != VerifyStateMissing {
					t.Errorf("cenario M: binary claude sem PATH -> esperado missing, got %v", item.State)
				}
			}
			continue
		}
		if item.Skill == "go-implementation" {
			foundGoSkill = true
			if item.State != VerifyStateCurrent {
				t.Errorf("cenario M: go-implementation got %v, want current", item.State)
			}
		}
	}
	if !foundGoSkill {
		t.Error("cenario M: go-implementation nao apareceu no verify")
	}

	// Confirmar que ha pelo menos um item binary (RI-04).
	hasBinary := false
	for _, item := range items {
		if item.Kind == VerifyKindBinary {
			hasBinary = true
		}
	}
	if !hasBinary {
		t.Error("cenario M: RI-04 falhou — nenhum item binary nos resultados de verify")
	}

	t.Logf("cenario M (Go) concluido com sucesso")
}

// TestIntegration_7_5_ScenarioG_Monorepo verifica install em monorepo multi-agente (cenario G).
// Projeto com go.mod + package.json -> Go + Node detectados; todos os agentes instalados.
// ADR-024 RI-01/RI-04.
func TestIntegration_7_5_ScenarioG_Monorepo(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Criar marcadores multi-stack (monorepo G: Go + Node).
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/mono\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(`{"name":"mono"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	createMinimalSource(t, sourceDir, []string{"review", "go-implementation", "node-implementation", "execute-task"})

	lp := newFakeLookPatherInt()
	svc := newRealServiceWithLookPather(t, lp)

	// Instalar para todos os 4 CLIs.
	allTools := skills.AllTools
	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       allTools,
		Langs:       nil, // auto-detect: deve encontrar Go + Node
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("cenario G: install falhou: %v", err)
	}

	// Skills de linguagem devem estar instaladas.
	for _, sk := range []string{"go-implementation", "node-implementation"} {
		p := filepath.Join(projectDir, ".agents", "skills", sk, "SKILL.md")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("cenario G: RI-01 falhou — %s nao instalada sem --langs", sk)
		}
	}

	// Reexecucao: idempotente.
	err = svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       allTools,
		Langs:       nil,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("cenario G: segunda instalacao falhou: %v", err)
	}

	// Verify: deve haver itens binary por cada CLI (RI-04).
	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      allTools,
	})
	if err != nil {
		t.Fatalf("cenario G: verify falhou: %v", err)
	}

	binaryCount := 0
	for _, item := range items {
		if item.Kind == VerifyKindBinary {
			binaryCount++
		}
	}
	// Deve haver 1 item binary por CLI (4 CLIs).
	if binaryCount != len(allTools) {
		t.Errorf("cenario G: RI-04 — esperado %d itens binary (1/CLI), got %d", len(allTools), binaryCount)
	}

	t.Logf("cenario G concluido com %d itens (incluindo %d binary)", len(items), binaryCount)
}

// TestIntegration_7_5_ProbeNonFatal_BinaryAbsent verifica que probe nao-fatal
// permite install concluir mesmo sem binario ACP. ADR-024 RI-02.
func TestIntegration_7_5_ProbeNonFatal_BinaryAbsent(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSource(t, sourceDir, []string{"review"})

	// Nenhum binario disponivel (probe falhara para todos).
	lp := newFakeLookPatherInt()
	svc := newRealServiceWithLookPather(t, lp)

	start := time.Now()
	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude, skills.ToolGemini, skills.ToolCodex, skills.ToolCopilot},
		Langs:       nil,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	elapsed := time.Since(start)

	// Install deve concluir sem erro (probe e nao-fatal).
	if err != nil {
		t.Fatalf("RI-02: install com binarios ausentes nao deve abortar, got: %v", err)
	}

	// Bootstrap < 30s mesmo com probe para 4 CLIs.
	if elapsed > 30*time.Second {
		t.Errorf("RI-02: bootstrap com 4 CLIs sem binarios demorou %v, limite e 30s (RF-11)", elapsed)
	}

	t.Logf("RI-02 probe nao-fatal verificado em %v", elapsed)
}
