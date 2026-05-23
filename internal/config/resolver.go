package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// projectCandidateNames lista os nomes de arquivo de config de projeto consultados, em ordem de preferencia.
// O primeiro existente em cada diretorio candidato vence.
var projectCandidateNames = []string{
	filepath.Join(".aispec", "config.yaml"),
	filepath.Join(".claude", "config.yaml"),
	filepath.Join(".agents", "config.yaml"),
}

// projectMarkers sao os marcadores de limite de repositorio para o upward-walk.
// A caminhada para quando algum desses diretórios for encontrado no diretorio atual.
var projectMarkers = []string{".git", ".aispec", ".claude", ".agents"}

// Resolver resolve a configuracao de runtime em cascata:
// defaults built-in < global (~/.aispec/config.yaml) < projeto (upward-walk) < overrides explícitos.
type Resolver interface {
	// Resolve aplica precedencia built-in < global < projeto < overrides.
	// cwd e o diretorio de partida para o upward-walk (pode ser "").
	// overrides contem valores de flags CLI; campos zero-value sao ignorados.
	Resolve(cwd string, overrides Runtime) (Runtime, error)
}

// DefaultResolver e a implementacao concreta de Resolver.
// HomeDir e o diretorio home do usuario; quando vazio, a config global e ignorada (RF-16).
type DefaultResolver struct {
	// HomeDir: diretorio home para resolver ~/.aispec/config.yaml.
	// Quando vazio, config global e ignorada sem erro (RF-16).
	HomeDir string
	// readFile: hook para leitura de arquivo (substituivel em testes).
	readFile func(path string) ([]byte, error)
	// isDir: hook para verificar se path e diretorio (substituivel em testes).
	isDir func(path string) bool
}

// NewDefaultResolver cria um Resolver padrao usando os.UserHomeDir e os reais.
func NewDefaultResolver() *DefaultResolver {
	homeDir, _ := os.UserHomeDir()
	return &DefaultResolver{
		HomeDir:  homeDir,
		readFile: os.ReadFile,
		isDir: func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && info.IsDir()
		},
	}
}

// Resolve implementa a cascata de configuracao:
// built-in < global < projeto < overrides.
func (r *DefaultResolver) Resolve(cwd string, overrides Runtime) (Runtime, error) {
	result := DefaultRuntime()

	// Camada 2: config global (~/.aispec/config.yaml).
	if r.HomeDir != "" {
		globalPath := filepath.Join(r.HomeDir, ".aispec", "config.yaml")
		globalCfg, err := r.loadFile(globalPath)
		if err != nil {
			return result, err
		}
		if globalCfg != nil {
			mergeInto(&result, *globalCfg)
		}
	}

	// Camada 3: config de projeto (upward-walk a partir do cwd).
	if cwd != "" {
		projPath, err := r.findProjectConfig(cwd)
		if err != nil {
			return result, err
		}
		if projPath != "" {
			projCfg, err := r.loadFile(projPath)
			if err != nil {
				return result, err
			}
			if projCfg != nil {
				mergeInto(&result, *projCfg)
			}
		}
	}

	// Camada 4: overrides explícitos (flags CLI).
	mergeInto(&result, overrides)

	return result, nil
}

// loadFile le e parseia um arquivo YAML de config.
// Retorna nil sem erro se o arquivo nao existir.
// Retorna erro descritivo se existir mas estiver malformado ou ilegivel.
func (r *DefaultResolver) loadFile(path string) (*Runtime, error) {
	data, err := r.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		rel := path
		return nil, fmt.Errorf("ler %s: %w", rel, err)
	}
	var cfg Runtime
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// findProjectConfig realiza upward-walk a partir de cwd procurando o arquivo de config
// de projeto mais proximo. Para ao encontrar um marcador de projeto ou ao atingir o
// limite do sistema de arquivos.
// Retorna o path absoluto do arquivo encontrado, ou "" se nao houver.
func (r *DefaultResolver) findProjectConfig(cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolver cwd %s: %w", cwd, err)
	}

	current := abs
	for {
		// Verificar candidatos de arquivo de config neste diretorio.
		for _, name := range projectCandidateNames {
			candidate := filepath.Join(current, name)
			data, err := r.readFile(candidate)
			if err == nil {
				_ = data // existente; retornar o path
				return candidate, nil
			}
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("ler %s: %w", candidate, err)
			}
		}

		// Verificar marcadores de projeto: se este diretorio tem um marcador,
		// nao subir mais (ja procuramos candidatos aqui).
		for _, marker := range projectMarkers {
			markerPath := filepath.Join(current, marker)
			if r.isDir(markerPath) {
				return "", nil
			}
		}

		// Subir um nivel.
		parent := filepath.Dir(current)
		if parent == current {
			// Atingiu a raiz do FS.
			break
		}
		current = parent
	}

	return "", nil
}

// mergeInto aplica merge campo-a-campo: cada campo nao-zero de src sobrescreve dst.
// Strings: sobrescreve se nao vazia. Numeros: sobrescreve se != 0.
func mergeInto(dst *Runtime, src Runtime) {
	if src.TasksRoot != "" {
		dst.TasksRoot = src.TasksRoot
	}
	if src.PRDPrefix != "" {
		dst.PRDPrefix = src.PRDPrefix
	}
	if src.EvidenceDir != "" {
		dst.EvidenceDir = src.EvidenceDir
	}
	if src.CoverageThreshold != 0 {
		dst.CoverageThreshold = src.CoverageThreshold
	}
	if src.LanguageDefault != "" {
		dst.LanguageDefault = src.LanguageDefault
	}
	if src.Timeout != "" {
		dst.Timeout = src.Timeout
	}
	if src.MaxRetries != 0 {
		dst.MaxRetries = src.MaxRetries
	}
	if src.RetryBackoffMultiplier != 0 {
		dst.RetryBackoffMultiplier = src.RetryBackoffMultiplier
	}
	if src.Concurrent != 0 {
		dst.Concurrent = src.Concurrent
	}
	if src.BatchSize != 0 {
		dst.BatchSize = src.BatchSize
	}
	if src.DefaultTool != "" {
		dst.DefaultTool = src.DefaultTool
	}
}
