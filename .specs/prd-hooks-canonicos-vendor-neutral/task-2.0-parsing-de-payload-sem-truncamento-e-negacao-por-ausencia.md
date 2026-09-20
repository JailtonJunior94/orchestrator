# Tarefa 2.0: Parsing de payload sem truncamento e negacao por ausencia de alvo

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar o bypass **verificado end-to-end** do gate de operação Git e alinhar a postura de comando
vazio à regra que o próprio repositório já enuncia em outro ponto: ausência de alvo é negação, nunca
aprovação.

Cobre **RF-57** (parsing estrutural sem truncamento, falha fechada) e **RF-68** (negação por ausência
de alvo). Sem dependências, paralelizável com 1.0 e 3.0 — e, na prática, **deve entrar primeiro**:
é correção de segurança com prova de exploração, não depende de nenhuma outra tarefa e não deve
esperar por nenhuma.

<requirements>
- Eliminar qualquer limite de tamanho na leitura do payload de hook. O truncamento é a causa raiz do
  bypass e não pode ser substituído por um limite maior.
- Eliminar o fallback textual por `grep` sobre o payload bruto. Falha estrutural de parse passa a ser
  **falha fechada**, nunca aprovação silenciosa.
- Alinhar o comando vazio para negação (RF-68), removendo o `exit 0` de
  `.agents/scripts/git-operation-gate.sh:39-41`.
- Espelhar obrigatoriamente em `.claude/`, `.codex/`, `.github/` e `internal/embedded/assets/`
  (ADR-001); espelho divergente é falha de gate, não detalhe de entrega.
- Registrar o risco de hosts sem `python3` nem `jq` e implementar a mitigação: `ai-spec doctor`
  passa a verificar a presença do interpretador.
- Documento em PT-BR; código em inglês, zero comentários (R-STYLE-001, hard).
- Zero regressão: todos os gates da seção de Critérios de Sucesso passam ao fim da tarefa.
</requirements>

## Subtarefas

- [ ] 2.1 Reproduzir e registrar a prova do bypass como teste automatizado antes de qualquer
      correção. **Prova já executada:** payload de **70.077 bytes**, com padding antes de
      `tool_input.command` e comando `echo hi ; git push origin main`, produz **exit 0**; o controle
      `{"tool_input":{"command":"git push origin main"}}` produz **exit 2**. A cadeia causal é:
      `head -c 65536` trunca o JSON → `json.load` lança → o `except` engole a exceção → `command_text`
      volta vazio → `git-operation-gate.sh` faz `exit 0`.
- [ ] 2.2 Criar `.agents/lib/hook-payload.sh` como substituto canônico de
      `.agents/lib/parse-hook-input.sh`: leitura integral do stdin, **sem `head -c`**, parse
      estrutural por `python3` e, na ausência dele, por `jq`.
      Pontos corrigidos: `.agents/lib/parse-hook-input.sh:10` (`parse_file_path`) e `:53`
      (`parse_command_text`), ambos com `input="$(head -c 65536)"`.
- [ ] 2.3 Remover o fallback por `grep` de `.agents/lib/parse-hook-input.sh:80`
      (`grep -o '"command":"[^"]*"'`). O padrão casa em **qualquer posição do texto bruto** — inclui
      corpo de string, campo homônimo aninhado e conteúdo de arquivo carregado no payload —, o que o
      torna simultaneamente burlável e gerador de falso positivo. Sem substituto textual: parse
      estrutural falhou, o gate bloqueia.
- [ ] 2.4 Tornar a falha de parse **fechada**: ausência de interpretador, JSON inválido ou exceção do
      parser resultam em bloqueio com mensagem em stderr identificando a causa, nunca em `exit 0`.
- [ ] 2.5 Alinhar RF-68 em `.agents/scripts/git-operation-gate.sh:39-41`: o bloco
      `if [[ -z "$command_text" ]]; then exit 0; fi` passa a negar.
      A regra correta já existe no repositório e serve de referência literal:
      `.agents/hooks/validate-preload.sh:109` — *"governanca nao carregada e nenhum alvo extraivel do
      payload — ausencia de alvo e negacao, nunca aprovacao"*. A tarefa alinha o gate de Git à postura
      que o gate de preload já pratica.
- [ ] 2.6 Reapontar os consumidores de `parse-hook-input.sh` para `hook-payload.sh`, preservando a
      cascata de resolução de caminho existente: `.agents/hooks/validate-governance.sh:30-33`
      (as três tentativas de `source` seguidas do aviso) e
      `.agents/hooks/validate-preload.sh:50-52,61-63`.
- [ ] 2.7 Propagar os espelhos obrigatórios e sincronizar as listas: `.claude/`, `.codex/`, `.github/`
      e `internal/embedded/assets/` via `scripts/sync-skills.sh`; verificar com
      `scripts/check-scripts-sync.sh` e `scripts/check-skills-sync.sh:102-112` (paridade
      `.agents/lib/` ↔ `scripts/lib/`). Incluir o espelho embarcado
      `internal/embedded/assets/scripts/lib/`, hoje **divergente do canônico e sem gate**.
- [ ] 2.8 Estender `ai-spec doctor` com o check de presença de `python3` ou `jq`, na camada de
      validação, junto aos checks existentes de `internal/doctor/doctor.go`. É a mitigação do risco
      de falhar fechado em host sem interpretador: o operador descobre a lacuna pelo `doctor`, não
      por um bloqueio inexplicado em tempo de hook.
- [ ] 2.9 Criar os casos de fronteira em `scripts/test-validators.sh` com payloads de **1 KiB,
      64 KiB, 65 KiB e 1 MiB**, cada um em duas variantes: comando benigno (deve passar) e
      `git push origin main` em posição arbitrária (deve bloquear). A fronteira de 65 KiB é a que
      expõe a regressão do truncamento. **Hoje `scripts/test-validators.sh` e `scripts/test-hooks.sh`
      têm zero referências ao gate de operação Git** — esta tarefa cria essa cobertura.
- [ ] 2.10 Criar `tests/integration/hook_payload_boundary_test.go` com a mesma matriz de tamanhos,
      mais os cenários de falha fechada: JSON inválido, interpretador ausente e campo `command`
      homônimo aninhado (regressão do fallback removido).
- [ ] 2.11 Garantir que o pacote de integração novo caia em um job de `.github/workflows/test.yml` —
      armadilha de gate órfão ativa no repositório.
- [ ] 2.12 Executar todos os gates declarados em Critérios de Sucesso e persistir as saídas como
      evidência, incluindo a re-execução da prova de 2.1 mostrando **exit 2** onde antes havia
      **exit 0**.

## Detalhes de Implementação

Referenciar, sem duplicar:

- `adr-003-parsing-estrutural-sem-truncamento.md` desta pasta — decisão, alternativas consideradas e,
  em `### Riscos e Mitigações`, o trade-off explícito de falhar fechado em host sem `python3` nem
  `jq`. É a fonte canônica da decisão; esta tarefa a executa, não a redecide.
- `techspec.md` — `### Interfaces Chave` (linha 116) para o contrato de `hook-payload.sh`;
  `### Pontos de Integração` (linha 295) para os consumidores reapontados; `### Testes Unitários`
  (linha 318) e `### Testes de Integração` (linha 377) para a matriz de fronteira;
  `### Rastreabilidade — RF × decisão × arquivo × teste` (linha 409) para RF-57 e RF-68;
  `### Riscos Conhecidos` (linha 586).
- `tasks.md` — `## Dependências Críticas`, bloco "2.0 é independente e deve ir primeiro na prática".
- `AGENTS.md` — tabela de gates por área; ADR-001 (`go:embed`) para os espelhos.
- `.claude/rules/code-style.md` — R-STYLE-001: shebang é permitido; nenhum comentário novo em shell.

Convenção de regex nos validadores, já firmada no repositório: nunca `Crit[eé]rios` — usar
`Crit(e|é)rios`, sob pena de falha silenciosa em `mawk`. `scripts/test-validators.sh` trava isso no
caso `a2` sob `LC_ALL=C`.

## Critérios de Sucesso

- A prova de exploração de 2.1, re-executada sem alteração, retorna **exit 2**. O controle continua
  em **exit 2**. Nenhum payload, em nenhum tamanho da matriz, produz `exit 0` com `git push`.
- `.agents/lib/hook-payload.sh` não contém `head -c` nem qualquer limite de leitura, e não contém
  fallback textual por `grep`.
- Parse estrutural falho bloqueia com mensagem de causa em stderr; nenhum caminho leva a `exit 0`.
- `.agents/scripts/git-operation-gate.sh` nega com comando vazio, coerente com
  `.agents/hooks/validate-preload.sh:109`.
- Os espelhos em `.claude/`, `.codex/`, `.github/` e `internal/embedded/assets/` são byte-idênticos
  ao canônico.
- `ai-spec doctor` reporta a ausência de `python3`/`jq` como check próprio.
- `scripts/test-validators.sh` cobre as quatro fronteiras (1 KiB, 64 KiB, 65 KiB, 1 MiB) — cobertura
  que hoje não existe.
- **Gates de não-regressão (inegociáveis), todos verdes:**
  - `make test lint vet` (mínimo transversal)
  - `make check-hooks-sync` — a tarefa toca `.agents/lib/` e `.agents/hooks/`
  - `make check-scripts-sync` — a tarefa toca `.agents/scripts/`
  - `make test-validators` — novos casos de fronteira
  - `make test-hooks` — comportamento dos hooks shell preservado
  - `make coverage` — 75% total e 70% por pacote crítico

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
  - `scripts/test-validators.sh` — matriz de fronteira 1 KiB / 64 KiB / 65 KiB / 1 MiB, variantes
    benigna e maliciosa; JSON inválido bloqueia; ausência de interpretador bloqueia; campo `command`
    homônimo aninhado não é extraído (regressão do fallback por `grep`).
  - `internal/doctor/` — check de presença de `python3`/`jq`, table-driven com `FakeFileSystem`.
- [ ] Testes de integração
  - `tests/integration/hook_payload_boundary_test.go` — build tag `integration`, `t.TempDir()`,
    mesma matriz de tamanhos mais os cenários de falha fechada, com asserção de **exit code**
    (0 vs 2), não apenas de stderr.
  - Regressão do bypass: payload de 70.077 bytes com `; git push origin main` ao fim.
  - Asserção de que o novo pacote de integração está referenciado em `.github/workflows/test.yml`.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

A criar:
- `.agents/lib/hook-payload.sh` — parsing estrutural sem limite de tamanho, falha fechada
- `tests/integration/hook_payload_boundary_test.go`
- Espelhos: `.claude/lib/`, `.codex/`, `.github/`, `internal/embedded/assets/.agents/lib/`,
  `internal/embedded/assets/scripts/lib/`

A modificar:
- `.agents/lib/parse-hook-input.sh:10,53` (`head -c 65536`) e `:80` (fallback por `grep`)
- `.agents/scripts/git-operation-gate.sh:39-41` — `exit 0` com comando vazio
- `.agents/hooks/validate-governance.sh:30-33,38` — resolução do novo lib
- `.agents/hooks/validate-preload.sh:50-52,61-63` — resolução do novo lib
- `internal/doctor/doctor.go` — check de interpretador
- `scripts/test-validators.sh` — casos de fronteira (zero referências ao gate hoje)
- `scripts/test-hooks.sh` — cenários do gate de operação Git (zero referências hoje)
- `scripts/sync-skills.sh`, `scripts/check-scripts-sync.sh`, `scripts/check-skills-sync.sh:102-112`
- `.github/workflows/test.yml` — job do pacote de integração novo

Referência de postura correta (não modificar):
- `.agents/hooks/validate-preload.sh:109` — "ausencia de alvo e negacao, nunca aprovacao"

Decisão de referência:
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-003-parsing-estrutural-sem-truncamento.md`
