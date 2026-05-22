//go:build integration

package install

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

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
