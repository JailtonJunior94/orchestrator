package config

import "fmt"

// Runtime agrupa configuracao de runtime consumida por skills, scripts e
// pelo orquestrador internal/taskloop. Carregada de .claude/config.yaml
// (fonte canonica) ou .agents/config.yaml (alias) na raiz do projeto.
//
// Defaults usam o layout canonico atual (.specs/prd-<slug>).
// Projetos que precisam de outro root podem sobrescrever tasks_root.
//
// Chaves operacionais opcionais (zero-value => default/F1):
//   - Timeout: duracao de inatividade (string parseable por time.ParseDuration); "" = sem limite.
//   - MaxRetries: numero maximo de retentativas; 0 = uma tentativa (F1).
//   - RetryBackoffMultiplier: multiplicador exponencial; <=0 = sem espera.
//   - Concurrent: grau de paralelismo; <=0 = 1 (sequencial, F1).
//   - BatchSize: tamanho do lote; <=0 = 1 (F1).
//   - DefaultTool: ferramenta padrao quando nao especificada; "" = sem padrao.
type Runtime struct {
	TasksRoot              string  `yaml:"tasks_root"`
	PRDPrefix              string  `yaml:"prd_prefix"`
	EvidenceDir            string  `yaml:"evidence_dir"`
	CoverageThreshold      float64 `yaml:"coverage_threshold"`
	LanguageDefault        string  `yaml:"language_default"`
	Timeout                string  `yaml:"timeout"`
	MaxRetries             int     `yaml:"max_retries"`
	RetryBackoffMultiplier float64 `yaml:"retry_backoff_multiplier"`
	Concurrent             int     `yaml:"concurrent"`
	BatchSize              int     `yaml:"batch_size"`
	DefaultTool            string  `yaml:"default_tool"`
}

// RuntimeProvider fornece configuracao de runtime stateless.
type RuntimeProvider struct{}

// NewRuntimeProvider cria um RuntimeProvider stateless.
func NewRuntimeProvider() *RuntimeProvider {
	return &RuntimeProvider{}
}

// DefaultRuntime retorna a configuracao com defaults compativeis com o layout atual.
func (p *RuntimeProvider) DefaultRuntime() Runtime {
	return Runtime{
		TasksRoot:         ".specs",
		PRDPrefix:         "prd-",
		EvidenceDir:       "",
		CoverageThreshold: 70.0,
		LanguageDefault:   "",
	}
}

// LoadRuntime e um wrapper fino sobre DefaultResolver para compatibilidade retroativa.
// Resolve a configuracao a partir de repoRoot como CWD, sem overrides e sem config global.
// Quando nenhum arquivo existir, retorna DefaultRuntime sem erro.
// Quando o arquivo existir mas estiver malformado, propaga erro descritivo.
func (p *RuntimeProvider) LoadRuntime(repoRoot string) (Runtime, error) {
	r := NewDefaultResolver()
	r.HomeDir = "" // sem config global: compatibilidade F1 (RF-16)
	return r.Resolve(repoRoot, Runtime{})
}

// EnvVars projeta a configuracao em variaveis de ambiente exportadas pelo
// script scripts/lib/check-invocation-depth.sh para consumo de skills e validators.
// O caller decide se aplica via os.Setenv ou se gera linhas `export FOO=bar`.
func (r Runtime) EnvVars() map[string]string {
	return map[string]string{
		"AI_TASKS_ROOT":         r.TasksRoot,
		"AI_PRD_PREFIX":         r.PRDPrefix,
		"AI_EVIDENCE_DIR":       r.EvidenceDir,
		"AI_COVERAGE_THRESHOLD": fmt.Sprintf("%g", r.CoverageThreshold),
		"AI_LANGUAGE_DEFAULT":   r.LanguageDefault,
	}
}
