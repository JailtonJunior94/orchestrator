package hookinventory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

type Entry struct {
	Name           string `json:"name"`
	Location       string `json:"location"`
	Event          string `json:"event"`
	Provider       string `json:"provider"`
	Purpose        string `json:"purpose"`
	PolicyGate     string `json:"policy_gate"`
	Blocking       string `json:"blocking"`
	Cost           string `json:"cost"`
	FailureMode    string `json:"failure_mode"`
	TestCoverage   string `json:"test_coverage"`
	Duplication    string `json:"duplication"`
	Classification string `json:"classification"`
	Justification  string `json:"justification"`
	IntegrityHash  string `json:"integrity_hash,omitempty"`
}

type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewService(filesystem fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: filesystem, printer: printer}
}

func (s *Service) Generate(root string) ([]Entry, error) {
	entries, err := s.Inventory(root)
	if err != nil {
		return nil, err
	}
	markdown, err := s.markdown(entries)
	if err != nil {
		return nil, err
	}
	data, err := s.json(entries)
	if err != nil {
		return nil, err
	}
	if err := s.fs.WriteFileAtomic(filepath.Join(root, "docs", "hook-inventory.md"), markdown); err != nil {
		return nil, fmt.Errorf("write hook inventory markdown: %w", err)
	}
	if err := s.fs.WriteFileAtomic(filepath.Join(root, "testdata", "hook-inventory.json"), data); err != nil {
		return nil, fmt.Errorf("write hook inventory json: %w", err)
	}
	if err := s.syncSkillsLock(root, entries); err != nil {
		return nil, err
	}
	s.printer.Info("Inventario de hooks gerado: %d entradas", len(entries))
	return entries, nil
}

func (s *Service) Check(root string) error {
	entries, err := s.Inventory(root)
	if err != nil {
		return err
	}
	markdown, err := s.markdown(entries)
	if err != nil {
		return err
	}
	data, err := s.json(entries)
	if err != nil {
		return err
	}
	if err := s.matches(filepath.Join(root, "docs", "hook-inventory.md"), markdown); err != nil {
		return err
	}
	if err := s.matches(filepath.Join(root, "testdata", "hook-inventory.json"), data); err != nil {
		return err
	}
	if err := s.checkSkillsLock(root, entries); err != nil {
		return err
	}
	return nil
}

func (s *Service) Inventory(root string) ([]Entry, error) {
	locations := []string{
		".agents/hooks", ".claude/hooks", ".codex/hooks", ".github/hooks", ".agents/scripts",
		".claude/scripts", "internal/runtime/hooks", ".opencode/plugin", ".agents/lib", "scripts/git-hooks",
	}
	entries := make([]Entry, 0)
	for _, location := range locations {
		found, err := s.scan(root, location)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Location < entries[j].Location })
	return entries, nil
}

func (s *Service) scan(root, location string) ([]Entry, error) {
	entries, err := s.fs.ReadDir(filepath.Join(root, location))
	if err != nil {
		return nil, fmt.Errorf("read hook source %s: %w", location, err)
	}
	items := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), "_test.go") || entry.Name() == "dispatcher.go" {
			continue
		}
		path := filepath.Join(location, entry.Name())
		if location == "internal/runtime/hooks" && filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		data, err := s.fs.ReadFile(filepath.Join(root, path))
		if err != nil {
			return nil, fmt.Errorf("read hook %s: %w", path, err)
		}
		items = append(items, s.newEntry(filepath.ToSlash(path), data))
	}
	return items, nil
}

func (s *Service) newEntry(location string, data []byte) Entry {
	profile := s.profile(location)
	entry := Entry{
		Name:           filepath.Base(location),
		Location:       location,
		Event:          profile.Event,
		Provider:       s.provider(location),
		Purpose:        profile.Purpose,
		PolicyGate:     profile.PolicyGate,
		Blocking:       s.blocking(location, data),
		Cost:           "baixo",
		FailureMode:    "FAIL-CLOSED",
		TestCoverage:   "COM TESTE",
		Duplication:    s.duplication(location),
		Classification: "KEEP",
		Justification:  profile.Justification,
	}
	if location == ".agents/scripts/validate-governance-references.sh" || location == ".claude/scripts/validate-governance-references.sh" {
		entry.TestCoverage = "SEM TESTE"
		entry.Classification = "REMOVE"
		entry.Justification = "REMOVE (diagnostico; RF-04 impede a remocao nesta tarefa): orfao de invocacao verificado por grep repo-wide — nenhum call-site executavel, apenas o proprio arquivo, o espelho de provedor, as listas de copia (scripts/sync-skills.sh:173, scripts/check-scripts-sync.sh:27, internal/install/install.go) e a mencao documental em docs/evidence-gates.md:20. O invariante que o script protege (referencias de governanca coerentes) nao tem cobertura equivalente no core; remocao so pode ocorrer apos essa cobertura existir."
	}
	if strings.Contains(location, ".claude/hooks/post-wave.sh") {
		entry.TestCoverage = "SEM TESTE"
	}
	if strings.Contains(location, "scripts/git-hooks/pre-commit") {
		entry.TestCoverage = "COM TESTE (parcial: apenas bloco 3; scripts/test-hooks.sh:538)"
		entry.FailureMode = "FAIL-OPEN (permissivo; linhas 41,54,58,61,66)"
	}
	if strings.Contains(location, "validate-token-budget.sh") {
		entry.Classification = "ON-DEMAND"
		entry.Justification = "ON-DEMAND (LOCAL-ONLY inerte): allowlist CLAUDE_LOCAL_ONLY_HOOKS (scripts/check-hooks-sync.sh:154,158) impede o espelhamento deste hook para outros provedores e ele nunca e registrado em .claude/settings.json — decisao registrada aqui em vez de tratada como divergencia de sync."
	}
	if failureMode, ok := s.failOpen(location); ok {
		entry.FailureMode = failureMode
	}
	if s.critical(location) {
		hash := sha256.Sum256(data)
		entry.IntegrityHash = hex.EncodeToString(hash[:])
	}
	return entry
}

type hookProfile struct {
	Event         string
	Purpose       string
	PolicyGate    string
	Justification string
}

func (s *Service) profile(location string) hookProfile {
	if profile, ok := uniqueHookProfiles[location]; ok {
		return profile
	}
	if profile, ok := basenameHookProfiles[filepath.Base(location)]; ok {
		return s.adaptProfileToProvider(location, profile)
	}
	return hookProfile{
		Event:         "NAO MAPEADO",
		Purpose:       "Script de governanca sem perfil individual mapeado ainda; requer classificacao dedicada na proxima rodada de inventario.",
		PolicyGate:    "Nenhuma policy associada mapeada.",
		Justification: "KEEP: presente no repositorio sem classificacao individual detalhada; revisar em rodada dedicada de inventario.",
	}
}

func (s *Service) adaptProfileToProvider(location string, profile hookProfile) hookProfile {
	if !strings.HasPrefix(location, ".agents/") {
		return profile
	}
	canonicalNote := " Fonte canonica em " + location + "; distribuida para os demais provedores por instalacao/sync, nao registrada diretamente em nenhum settings/config de provedor."
	profile.Event = profile.Event + " (fonte canonica)"
	profile.Justification = profile.Justification + canonicalNote
	return profile
}

var basenameHookProfiles = map[string]hookProfile{
	"validate-preload.sh": {
		Event:         "PreToolUse",
		Purpose:       "Ponto de entrada do gate de governanca pre-edicao: exige que AGENTS.md e a skill de governanca tenham sido carregados antes de Bash/Edit/Write/NotebookEdit/apply_patch, e encadeia hook-prereq-gate.sh e git-operation-gate.sh.",
		PolicyGate:    "R-GOV-001 (governanca transversal) + hook-prereq-gate.sh + git-operation-gate.sh; nenhuma policy em .agents/policies/ e lida diretamente pelo script.",
		Justification: "KEEP: unico ponto de entrada PreToolUse do harness (validate-preload.sh:5 PRELOAD_BLOCK_EXIT=2); remove-lo elimina o gate anti-edicao-sem-governanca inteiro.",
	},
	"validate-governance.sh": {
		Event:         "PostToolUse",
		Purpose:       "Avisa e, por padrao, bloqueia (GOVERNANCE_HOOK_MODE=fail) a edicao direta de AGENTS.md e SKILL.md apos Edit/Write, para impedir drift silencioso do contrato de governanca.",
		PolicyGate:    "R-GOV-001 (protecao de AGENTS.md/SKILL.md); modo warn via GOVERNANCE_HOOK_MODE e opt-out explicito.",
		Justification: "KEEP: unico gate PostToolUse que impede edicao nao revisada dos arquivos de contrato de governanca (validate-governance.sh:44-49).",
	},
	"subagent-stop-wrapper.sh": {
		Event:         "SubagentStop (matcher task-executor)",
		Purpose:       "Ao encerrar o subagente task-executor, valida se o relatorio de execucao YAML foi persistido com evidencia fisica antes de aceitar a tarefa como concluida.",
		PolicyGate:    "R-GOV-001.4 (Evidencia obrigatoria) + AGENTS.md invariante 4.",
		Justification: "KEEP: fecha F2/F25 (evidencia fisica e checkpoint); sem ele a Invariante 4 de AGENTS.md (evidencia obrigatoria) nao e verificavel no fim da execucao.",
	},
	"validate-session-end.sh": {
		Event:         "Stop",
		Purpose:       "Valida ao final da sessao interativa se ha evidencia de execucao pendente de persistencia antes de permitir o encerramento.",
		PolicyGate:    "R-GOV-001.4 (Evidencia obrigatoria).",
		Justification: "KEEP: ultima barreira de evidencia antes do encerramento de sessao; espelha o mesmo contrato de post-execute-task.sh para o caminho interativo.",
	},
	"post-execute-task.sh": {
		Event:         "skill-lifecycle (invocado via bash pelo procedimento execute-task/execute-all-tasks, SKILL.md:112; nao e um hook declarativo de settings/config)",
		Purpose:       "Validacao programatica pos-execute-task: fecha F2 (evidencia fisica), F13 (path absoluto), F24 (escalonamento de remark critico), F25 (checkpoint) e F35 (revert de git).",
		PolicyGate:    "R-GOV-001.4 (Evidencia obrigatoria) via .agents/scripts/validate-task-evidence.sh encadeado.",
		Justification: "KEEP: execute-task/execute-all-tasks tratam ausencia deste hook como falha ('failed: hook ausente — reinstale via ai-spec install'), sem modo legado silencioso (SKILL.md:112).",
	},
	"post-wave.sh": {
		Event:         "skill-lifecycle (invocado via bash apos cada wave concluida do execute-all-tasks, SKILL.md:126)",
		Purpose:       "Registra o resultado de uma wave concluida em .specs/prd-<slug>/_orchestration_report.partial.md (append-only).",
		PolicyGate:    "Contrato de relatorio append-only de execute-all-tasks; nenhuma policy formal em .agents/policies/ o cobre.",
		Justification: "KEEP: unica fonte do relatorio parcial de orquestracao; ausencia em todos os caminhos falha explicitamente o wave (SKILL.md:130).",
	},
	"pre-execute-all-tasks.sh": {
		Event:         "skill-lifecycle (invocado via bash no inicio do execute-all-tasks, SKILL.md:24)",
		Purpose:       "Valida regex de tasks.md, gaps numericos de tarefas, spec-hash cross-PRD e ciclos de dependencia antes de iniciar a orquestracao de waves.",
		PolicyGate:    "Invariante 2 de AGENTS.md (ancora de confianca / spec-hash).",
		Justification: "KEEP: gate de pre-condicao do DAG de tarefas; SKILL.md:24 trata ausencia total como integridade quebrada, sem modo legado.",
	},
	"validate-token-budget.sh": {
		Event:         "nao registrado (LOCAL-ONLY inerte)",
		Purpose:       "Estimaria consumo de tokens da sessao contra um orcamento configuravel antes de permitir a proxima operacao.",
		PolicyGate:    "Nenhuma; nunca chega a ser avaliado porque nao esta registrado em .claude/settings.json.",
		Justification: "ON-DEMAND: ver Justification especifica aplicada apos este perfil base (allowlist LOCAL-ONLY).",
	},
	"hook-prereq-gate.sh": {
		Event:         "tool_call-lifecycle (encadeado por validate-preload.sh antes de concluir o PreToolUse, validate-preload.sh:61)",
		Purpose:       "Bloqueia a edicao de codigo quando a skill de linguagem correspondente ao arquivo tocado nao foi carregada na sessao (validate-skill-prerequisites.sh) e emite a lista cirurgica de referencias (resolve-references.sh).",
		PolicyGate:    "R-GOV-001 (contrato de carga base de AGENTS.md) aplicado por extensao de arquivo.",
		Justification: "KEEP: unico ponto que impede edicao de codigo sem a skill de linguagem correta carregada; hook-prereq-gate.sh:53 bloqueia com exit 1 quando a skill esperada esta ausente.",
	},
	"git-operation-gate.sh": {
		Event:         "tool_call-lifecycle (encadeado por validate-preload.sh para comandos Bash com git, validate-preload.sh:73)",
		Purpose:       "Analisa o texto do comando Bash proposto e nega operacoes git destrutivas (push --force, reset --hard, checkout -- etc.) sem confirmacao explicita do usuario.",
		PolicyGate:    "Protocolo de Seguranca Git do AGENTS.md/CLAUDE.md (nao executar git destrutivo sem pedido explicito).",
		Justification: "KEEP: unica barreira automatizada contra comandos git destrutivos nao confirmados; alvo dedicado da tarefa 2.0 deste PRD para revisao de profundidade da analise.",
	},
	"resolve-references.sh": {
		Event:         "tool_call-lifecycle (encadeado por hook-prereq-gate.sh para montar a lista cirurgica de references, hook-prereq-gate.sh:58)",
		Purpose:       "Resolve, a partir dos arquivos tocados e de sinais de diff, quais references de uma skill de linguagem (INDEX.yaml) devem ser carregadas nesta edicao.",
		PolicyGate:    "ADR-004 (lazy-loading de references sob demanda).",
		Justification: "KEEP: implementa o carregamento sob demanda de references exigido pelo ADR-004; sem ele hook-prereq-gate.sh perderia a orientacao GUIDANCE emitida em stderr.",
	},
	"validate-bugfix-evidence.sh": {
		Event:         "skill-lifecycle (invocado na Etapa 5 da skill bugfix, fora do ciclo de hooks declarativos de settings/config)",
		Purpose:       "Valida se bugfix_report.md contem o pacote de evidencias exigido (arquivo, teste de regressao, validacao) por bug corrigido, com opt-out via --no-rf.",
		PolicyGate:    "R-GOV-001.4 (Evidencia obrigatoria) aplicada ao fluxo de bugfix.",
		Justification: "KEEP: gate fail-closed desde 0.31.0 (docs/evidence-gates.md); sem ele bugfix_report.md poderia ser aceito sem prova fisica de correcao.",
	},
	"validate-refactor-evidence.sh": {
		Event:         "skill-lifecycle (invocado na Etapa final da skill refactor, fora do ciclo de hooks declarativos de settings/config)",
		Purpose:       "Valida se refactor_report.md contem evidencia de preservacao de comportamento (testes antes/depois) para a refatoracao executada.",
		PolicyGate:    "R-GOV-001.4 (Evidencia obrigatoria) aplicada ao fluxo de refactor.",
		Justification: "KEEP: unico validador de evidencia do fluxo de refatoracao; ausencia reabriria regressao silenciosa de comportamento sem prova de teste.",
	},
	"validate-review-evidence.sh": {
		Event:         "skill-lifecycle (invocado no modo --auto-review, RF-20; espelha validate-task-evidence.sh)",
		Purpose:       "Valida se o relatorio de review (review.md) documenta achados classificados por severidade com evidencia rastreavel antes de encerrar o ciclo de aprovacao.",
		PolicyGate:    "R-GOV-001 (Politica de Evidencia: 'nao aprovar solucao com lacuna critica conhecida').",
		Justification: "KEEP: fecha o Ciclo de Aprovacao (RF-35/RF-38) com prova fisica de revisao, nao apenas afirmacao de que a revisao ocorreu.",
	},
	"validate-skill-prerequisites.sh": {
		Event:         "tool_call-lifecycle (invocado por hook-prereq-gate.sh:47)",
		Purpose:       "Bloqueia quando a skill de linguagem correspondente ao arquivo tocado nao possui SKILL.md + references/INDEX.yaml na arvore .agents/skills/.",
		PolicyGate:    "R-GOV-001 (contrato de carga base) aplicado por extensao de arquivo; PREREQ_MODE=warn e opt-out explicito.",
		Justification: "KEEP: unica verificacao de presenca fisica da skill de linguagem antes de liberar a edicao; hook-prereq-gate.sh depende dele para decidir exit 1 vs exit 0.",
	},
	"validate-task-evidence.sh": {
		Event:         "skill-lifecycle (invocado por post-execute-task.sh e pela Etapa de evidencia de execute-task)",
		Purpose:       "Valida o DoD, os criterios de aceite e a prova fisica de testes de uma tarefa antes de marca-la como done.",
		PolicyGate:    "R-GOV-001.4 (Evidencia obrigatoria); gate de criterios de aceite fail-closed desde 0.31.0 (AI_SDD_STRICT_EVIDENCE).",
		Justification: "KEEP: maior e mais critico validador de evidencia do harness (547 linhas); encadeado diretamente por post-execute-task.sh, sem ele nenhuma tarefa teria prova de DoD verificada.",
	},
	"validate-governance-references.sh": {
		Event:         "on-demand (sem call-site executavel; nenhum evento de hook o invoca)",
		Purpose:       "Verificaria se referencias citadas em documentos de governanca apontam para arquivos existentes.",
		PolicyGate:    "Nenhuma; e o proprio caso descrito na subtarefa 1.8 de policy declarada sem enforcement real.",
		Justification: "REMOVE: ver Justification especifica aplicada apos este perfil base (orfao de invocacao verificado por grep repo-wide).",
	},
	"hook-payload.sh": {
		Event:         "lib (source por validate-governance.sh e validate-preload.sh para parse_file_path/parse_command_text)",
		Purpose:       "Fornece o parser comum de payload JSON de hook (extrai file_path/command_text via python3 com fallback), consumido por todos os hooks shell que leem stdin.",
		PolicyGate:    "Dependencia critica do fail-open documentado em validate-governance.sh:33 e validate-preload.sh:25-28 (ausencia de python3/jq degrada o parsing).",
		Justification: "KEEP: dependencia transitiva de todo hook shell que consome payload JSON; sua ausencia e o proprio gatilho do fail-open registrado em .agents/hooks/validate-governance.sh:33.",
	},
	"parse-hook-input.sh": {
		Event:         "lib (wrapper de uma linha que faz source de hook-payload.sh)",
		Purpose:       "Alias de compatibilidade para hooks/scripts que esperam o nome de arquivo parse-hook-input.sh em vez de hook-payload.sh.",
		PolicyGate:    "Mesma policy de hook-payload.sh, por delegacao direta.",
		Justification: "KEEP: mantido por compatibilidade de nome com consumidores existentes; remove-lo quebraria qualquer script que faca 'source .agents/lib/parse-hook-input.sh'.",
	},
	"check-invocation-depth.sh": {
		Event:         "lib (source pelas skills bugfix/review/execute-task/execute-all-tasks na Etapa 1)",
		Purpose:       "Controla e incrementa AI_INVOCATION_DEPTH para impedir loop infinito na cadeia execute-task -> review -> bugfix.",
		PolicyGate:    "Teto de rodadas do Ciclo de Aprovacao (RF-35, default 5) e limite de profundidade (AI_INVOCATION_MAX, default 2).",
		Justification: "KEEP: unico mecanismo de guarda contra recursao infinita entre skills; divergencia de conteudo entre .agents/lib/ e scripts/lib/ ja esta registrada como lacuna de sync pela subtarefa 1.7.",
	},
}

var uniqueHookProfiles = map[string]hookProfile{
	".github/hooks/governance.json": {
		Event:         "declaracao de registro (chaves agentStop, postToolUse, preToolUse) consumida pelo GitHub Copilot CLI",
		Purpose:       "Registra, em formato JSON nativo do Copilot CLI, quais scripts shell rodam em agentStop (subagent-stop-wrapper.sh + validate-session-end.sh), postToolUse (validate-governance.sh) e preToolUse (validate-preload.sh).",
		PolicyGate:    "Equivalente funcional de .claude/settings.json e .codex/config.toml para o provedor Copilot.",
		Justification: "KEEP: unico arquivo de registro declarativo do provedor Copilot; sem ele nenhum dos quatro scripts mapeados roda no Copilot CLI.",
	},
	".opencode/plugin/governance.js": {
		Event:         "plugin OpenCode (hooks nativos exportados pelo modulo, carregados pelo loader de plugins do OpenCode CLI)",
		Purpose:       "Replica em JavaScript, para o runtime nativo do OpenCode, o mesmo contrato de governanca (preload + validate-governance + sentinelas de sessao) aplicado aos demais provedores via shell.",
		PolicyGate:    "R-GOV-001 (governanca transversal), reimplementada nativamente em vez de shell por restricao de integracao do OpenCode.",
		Justification: "KEEP: unico ponto de paridade de governanca para o OpenCode, que nao consome os hooks shell diretamente (RF-13, .opencode/plugin/ e a integracao nativa documentada em AGENTS.md).",
	},
	"scripts/git-hooks/pre-commit": {
		Event:         "git-hook nativo (pre-commit, instalado em .git/hooks/pre-commit)",
		Purpose:       "Roda blocos de validacao locais antes do commit, incluindo o bloco 3 de spec-drift mencionado como 'permissivo' no proprio arquivo.",
		PolicyGate:    "Invariante 2 de AGENTS.md (ancora de confianca / spec-hash), aplicada apenas no bloco 3.",
		Justification: "KEEP: unico git hook nativo do repositorio com cobertura de teste propria (scripts/test-hooks.sh:538, cenarios B3-*); fora de qualquer lista de sync (subtarefa 1.7), lacuna registrada separadamente.",
	},
	"internal/runtime/hooks/governance.go": {
		Event:         "runtime.pre_open (dispatcher.go:PointRuntimePreOpen)",
		Purpose:       "Replicacao semantica exata de validate-governance.sh para o modo ACP orchestrator: aborta a sessao quando AGENTS.md nao existe no WorkDir.",
		PolicyGate:    "R-GOV-001 aplicada em runtime.pre_open (coexiste com o hook shell no modo interativo).",
		Justification: "KEEP: unico enforcement de AGENTS.md obrigatorio no runtime ACP orquestrado; Run() retorna erro nao-nil que aborta o fan-out sequencial do Dispatcher.",
	},
	"internal/runtime/hooks/spec_drift.go": {
		Event:         "runtime.pre_open (dispatcher.go:PointRuntimePreOpen, registrado ao lado de GovernanceHook)",
		Purpose:       "Aborta a sessao ACP antes de abrir quando o hash do PRD/techspec diverge do registrado em tasks.md (RG-01) ou quando tasks.md nao rastreia hash de PRD (RG-02, PRD-first).",
		PolicyGate:    "Invariante 2 de AGENTS.md (ancora de confianca / spec-hash) e Invariante 1 (protocolo PRD-first).",
		Justification: "KEEP: unico guard de drift de spec-hash no runtime orquestrado; sem TasksDir opera em modo no-op preservando comportamento F1.",
	},
	"internal/runtime/hooks/token_budget.go": {
		Event:         "prompt.post_build (dispatcher.go:PointPromptPostBuild)",
		Purpose:       "Equivalente Go de validate-token-budget.sh: estimaria consumo de tokens do prompt montado contra um orcamento configuravel.",
		PolicyGate:    "Mesma policy pretendida de validate-token-budget.sh, nao aplicada de fato (ver Justification).",
		Justification: "KEEP: coexiste com o hook shell (comentario do arquivo: 'Shell hook permanece ativo para modo interativo Claude Code'); nenhum dos dois esta de fato acionado hoje, mas o codigo Go e o unico caminho para o modo ACP orquestrado caso seja habilitado.",
	},
	"internal/runtime/hooks/quality_gate.go": {
		Event:         "session.post_end (dispatcher.go:PointSessionPostEnd, quality_gate.go:42)",
		Purpose:       "Avalia qualitygate.Gate contra o EvaluationInput da sessao e retorna erro (Reason/PolicyID/GateID) quando a politica de qualidade nao e satisfeita.",
		PolicyGate:    "Policy dinamica resolvida por qualitygate.Gate (por evento, com selecao por risco); nao le .agents/policies/ diretamente.",
		Justification: "KEEP: unico gate de qualidade acionado no encerramento de sessao do runtime orquestrado; entregue pela tarefa 10.0 deste PRD, em desenvolvimento concorrente a esta correcao.",
	},
	"internal/runtime/hooks/memory_events.go": {
		Event:         "definicoes de evento (constantes PointMemoryFactRecorded e demais PointMemory*)",
		Purpose:       "Declara os tipos de evento e pontos canonicos de memoria duravel (fact_recorded, fact_archived, fact_promoted, contradiction_detected, secret_redacted, compaction_executed, baton_transferred); nao implementa a interface Hook.",
		PolicyGate:    "Nenhuma; e um arquivo de tipos/constantes, nao um hook executavel.",
		Justification: "KEEP: contrato de tipos consumido por memory_evidence.go e memory_persist.go; escaneado pelo inventario por estar em internal/runtime/hooks/ apesar de nao implementar Run().",
	},
	"internal/runtime/hooks/memory_evidence.go": {
		Event:         "memory.* (todos os PointMemory* definidos em memory_events.go)",
		Purpose:       "Agrega, via MemoryEvidenceRecorder, um registro append-only de cada evento de memoria emitido na sessao, para uso como evidencia RF-38.",
		PolicyGate:    "R-GOV-001.4 (Evidencia obrigatoria) aplicada aos eventos de memoria duravel.",
		Justification: "KEEP: Run() sempre retorna nil (telemetria, nao bloqueante); mantido por ser a unica trilha de evidencia dos eventos de memoria duravel no runtime orquestrado.",
	},
	"internal/runtime/hooks/memory_persist.go": {
		Event:         "session.post_end (dispatcher.go:PointSessionPostEnd)",
		Purpose:       "Escreve MEMORY.md em .specs/<prd>/memory/ com o resumo da sessao via memory.Store e emite metricas por log.",
		PolicyGate:    "ADR MD-001..MD-005 (memoria duravel de agentes, fachada como porta unica).",
		Justification: "KEEP: unico ponto de persistencia de MEMORY.md no runtime orquestrado; falha de escrita retorna erro que aborta o fan-out (memory_persist.go:63).",
	},
	"internal/runtime/hooks/projection.go": {
		Event:         "nao e hook; mapa auxiliar CanonicalEventFor/AllPoints/PointsWithoutCanonicalEvent consumido pelo runtime para checar cobertura de eventos canonicos",
		Purpose:       "Projeta cada ponto canonico do dispatcher (PointRuntimePreOpen etc.) para o hookcontract.EventKind correspondente, quando existir um.",
		PolicyGate:    "Nenhuma; utilitario de projecao, nao um hook executavel.",
		Justification: "KEEP: escaneado por estar em internal/runtime/hooks/ (arquivo .go sem sufixo _test.go); nao implementa Run() e nao e, em si, um hook, mas e a fonte de verdade de quais pontos tem evento canonico mapeado.",
	},
}

func (s *Service) provider(location string) string {
	switch {
	case strings.HasPrefix(location, ".claude/"):
		return "Claude"
	case strings.HasPrefix(location, ".codex/"):
		return "Codex"
	case strings.HasPrefix(location, ".github/"):
		return "Copilot"
	case strings.HasPrefix(location, ".opencode/"):
		return "OpenCode"
	case strings.HasPrefix(location, "internal/runtime/"):
		return "Runtime ACP"
	default:
		return "Vendor-neutral"
	}
}

var (
	literalExitPattern   = regexp.MustCompile(`exit\s+([1-9][0-9]*)\b`)
	variableExitPattern  = regexp.MustCompile(`exit\s+"?\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?"?`)
	variableAssignRegexp = func(name string) *regexp.Regexp {
		return regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `=([1-9][0-9]*)\b`)
	}
	goHookRunPattern   = regexp.MustCompile(`\)\s*Run\(`)
	goErrorReturnRegex = regexp.MustCompile(`return\s+(fmt\.Errorf|errors\.New|Err[A-Z]\w*)\(`)
)

func (s *Service) blocking(location string, data []byte) string {
	if strings.HasSuffix(location, ".go") {
		if s.goHookBlocks(data) {
			return "BLOQUEANTE"
		}
		return "NAO BLOQUEANTE"
	}
	if s.shellHookBlocks(data) {
		return "BLOQUEANTE"
	}
	return "NAO BLOQUEANTE"
}

func (s *Service) goHookBlocks(data []byte) bool {
	content := string(data)
	return goHookRunPattern.MatchString(content) && goErrorReturnRegex.MatchString(content)
}

func (s *Service) shellHookBlocks(data []byte) bool {
	content := string(data)
	if literalExitPattern.MatchString(content) {
		return true
	}
	for _, match := range variableExitPattern.FindAllStringSubmatch(content, -1) {
		if variableAssignRegexp(match[1]).MatchString(content) {
			return true
		}
	}
	return false
}

func (s *Service) duplication(location string) string {
	if strings.HasPrefix(location, ".agents/") || strings.HasPrefix(location, ".claude/") {
		return "Possui espelhos por provedor ou distribuicao embarcada."
	}
	return "Sem espelho equivalente inventariado."
}

func (s *Service) failOpen(location string) (string, bool) {
	modes := map[string]string{
		".agents/hooks/subagent-stop-wrapper.sh": "FAIL-OPEN (.agents/hooks/subagent-stop-wrapper.sh:30,35,40,87,101)",
		".agents/hooks/validate-governance.sh":   "FAIL-OPEN (.agents/hooks/validate-governance.sh:33,41)",
		".agents/scripts/hook-prereq-gate.sh":    "FAIL-OPEN (.agents/scripts/hook-prereq-gate.sh:50,60)",
		".agents/hooks/validate-preload.sh":      "FAIL-OPEN (.agents/hooks/validate-preload.sh:54,65)",
		".agents/scripts/git-operation-gate.sh":  "FAIL-OPEN (.agents/scripts/git-operation-gate.sh:39-41)",
	}
	mode, ok := modes[location]
	return mode, ok
}

func (s *Service) critical(location string) bool {
	return strings.Contains(location, "validate-") || strings.Contains(location, "gate") || strings.Contains(location, "pre-commit")
}

func (s *Service) markdown(entries []Entry) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("# Inventario de Hooks\n\n")
	builder.WriteString("Gerado por `ai-spec hooks inventory`; nao editar manualmente.\n\n")
	builder.WriteString("| Nome | Localizacao | Evento | Provedor | Objetivo | Policy/gate | Bloqueante | Custo | Falha esperada | Cobertura | Duplicacao | Classificacao | Justificativa | Integridade |\n")
	builder.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, entry := range entries {
		values := []string{entry.Name, entry.Location, entry.Event, entry.Provider, entry.Purpose, entry.PolicyGate, entry.Blocking, entry.Cost, entry.FailureMode, entry.TestCoverage, entry.Duplication, entry.Classification, entry.Justification, entry.IntegrityHash}
		for index, value := range values {
			if index > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString(strings.ReplaceAll(value, "|", "/"))
		}
		builder.WriteString("\n")
	}
	return []byte(builder.String()), nil
}

func (s *Service) json(entries []Entry) ([]byte, error) {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hook inventory: %w", err)
	}
	return append(data, '\n'), nil
}

type skillsLockFile struct {
	Version int                        `json:"version"`
	Skills  map[string]json.RawMessage `json:"skills"`
}

type hookLockEntry struct {
	Source       string `json:"source"`
	SourceType   string `json:"sourceType"`
	Path         string `json:"path"`
	ComputedHash string `json:"computedHash"`
}

func (s *Service) criticalCanonicalEntries(entries []Entry) []Entry {
	result := make([]Entry, 0)
	for _, entry := range entries {
		if entry.IntegrityHash == "" {
			continue
		}
		if !strings.HasPrefix(entry.Location, ".agents/") && entry.Location != "scripts/git-hooks/pre-commit" {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func (s *Service) lockKey(location string) string {
	return "hook:" + location
}

func (s *Service) readSkillsLock(root string) (skillsLockFile, error) {
	path := filepath.Join(root, "skills-lock.json")
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return skillsLockFile{}, fmt.Errorf("read skills-lock.json: %w", err)
	}
	var lock skillsLockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return skillsLockFile{}, fmt.Errorf("parse skills-lock.json: %w", err)
	}
	if lock.Skills == nil {
		lock.Skills = make(map[string]json.RawMessage)
	}
	return lock, nil
}

func (s *Service) syncSkillsLock(root string, entries []Entry) error {
	lock, err := s.readSkillsLock(root)
	if err != nil {
		return err
	}
	for _, entry := range s.criticalCanonicalEntries(entries) {
		encoded, err := json.Marshal(hookLockEntry{
			Source:       "internal",
			SourceType:   "hook",
			Path:         entry.Location,
			ComputedHash: entry.IntegrityHash,
		})
		if err != nil {
			return fmt.Errorf("marshal hook lock entry %s: %w", entry.Location, err)
		}
		lock.Skills[s.lockKey(entry.Location)] = encoded
	}
	out, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal skills-lock.json: %w", err)
	}
	if err := s.fs.WriteFileAtomic(filepath.Join(root, "skills-lock.json"), append(out, '\n')); err != nil {
		return fmt.Errorf("write skills-lock.json: %w", err)
	}
	return nil
}

func (s *Service) checkSkillsLock(root string, entries []Entry) error {
	lock, err := s.readSkillsLock(root)
	if err != nil {
		return err
	}
	for _, entry := range s.criticalCanonicalEntries(entries) {
		raw, ok := lock.Skills[s.lockKey(entry.Location)]
		if !ok {
			return fmt.Errorf("skills-lock drift: hook critico sem entrada em skills-lock.json: %s", entry.Location)
		}
		var locked hookLockEntry
		if err := json.Unmarshal(raw, &locked); err != nil {
			return fmt.Errorf("skills-lock drift: entrada invalida para %s: %w", entry.Location, err)
		}
		if locked.ComputedHash != entry.IntegrityHash {
			return fmt.Errorf("skills-lock drift: hash divergente do hook critico %s", entry.Location)
		}
	}
	return nil
}

func (s *Service) matches(path string, expected []byte) error {
	actual, err := s.fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read generated inventory %s: %w", path, err)
	}
	if string(actual) != string(expected) {
		return fmt.Errorf("hook inventory drift: %s", path)
	}
	return nil
}
