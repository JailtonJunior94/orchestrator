<!-- spec-hash-prd: 223b55df6ca415d48c7af39dcc748d59746d9543cd76739ea21ad31d1dc59a27 -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Memória Durável de Agentes

**PRD consumido:** [prd.md](prd.md) (spec-version 3, 37 requisitos funcionais)
**Modelo de domínio:** [domain-model.md](../../discoveries/domain-memoria-duravel-de-agentes/domain-model.md) (validado, `SUCCESS`)
**Decisão de padrão:** [decision.md](../../pattern-decisions/memoria-duravel-fachada/decision.md) (validado, `SUCCESS`)
**ADRs:** [MD-001](adr-001-fachada-porta-unica-memoria.md) · [MD-002](adr-002-fato-pagina-roundtrip-lossless.md) · [MD-003](adr-003-escrita-atomica-lock-camada-lease.md) · [MD-004](adr-004-optin-paridade-byte-a-byte.md) · [MD-005](adr-005-evidencia-metricas-memoria.md)

> **Convenção de caminhos neste documento.** Caminhos entre backticks são caminhos **que já existem** no repositório e foram verificados. Caminhos de componentes **a criar** aparecem em texto plano, sem backticks — deliberadamente, porque `scripts/check-spec-paths.sh` extrai apenas tokens entre backticks e reprova caminho inexistente quando o PRD entra em gestão SDD. Tratar caminho planejado como caminho existente é exatamente o defeito que originou aquele gate.

> **Rótulo das ADRs locais.** As cinco ADRs desta feature são rotuladas `MD-001` a `MD-005`, seguindo a convenção do repositório para ADR local de PRD (`PP-001`/`PP-002` em `AGENTS.md`), e não `ADR-001`..`ADR-005` — esses rótulos já designam ADRs do repositório (`docs/adr/001-go-embed-baseline.md`, `docs/adr/002-fake-filesystem-testes.md`) e a colisão de namespace tornaria toda citação ambígua.
>
> **Duas ADRs do repositório citadas neste documento não têm arquivo.** `ADR-016` (cascata de configuração) e `ADR-023` (política de janela) são citadas por comentários do código de produção — `internal/runtime/runner.go:141`, `internal/runtime/types.go:123`, `internal/metrics/metrics.go:211` entre outros — mas seus arquivos foram removidos do repositório no commit `91f8cb9`, e o índice de ADRs do `AGENTS.md` aponta para um diretório inexistente. O comportamento que elas descrevem **existe e está implementado**; apenas o registro da decisão desapareceu. Esta especificação cita, para esses dois casos, o código e a documentação que existem (`docs/config-hierarchy.md`, `internal/runtime/memory/window_policy.go`). Restaurar os dois arquivos é defeito de documentação pré-existente, fora do escopo desta feature.

## Resumo Executivo

A implementação introduz um subsistema de memória durável **atrás do ponto de integração que o runtime já usa**, sem alterar o contrato atual. O consumidor passa a depender de uma porta estreita de dois métodos, e todo o resto — parse de página, agregado de camada, cinco políticas stateless, lease de bastão — vive atrás dela (MD-001). A unidade de conhecimento é o Fato, identificado por chave semântica mais hash de conteúdo, persistido como seção de uma Página Markdown cujo round-trip é invariante de escrita (MD-002).

A estratégia de risco é assimétrica de propósito. Toda a feature é opt-in por flag mais chave na cascata de configuração existente, e o caminho desativado é provado byte-idêntico ao atual por teste dedicado (MD-004). As três primeiras entregas — escrita atômica, round-trip de página, políticas stateless — não tocam nenhuma linha do consumidor, o que significa que a feature pode ser mais de metade implementada com risco de regressão estruturalmente nulo. O consumidor só é alterado na sexta entrega, quando a paridade já está travada por teste.

Três correções de defeito pré-existente entram no escopo porque requisitos do PRD são inalcançáveis sem elas: o subsistema de memória passa a usar a abstração de filesystem injetada, encerrando o desvio de `internal/runtime/memory/store.go:104` em relação ao ADR-002 do repositório; a escrita atômica de `internal/sdd/state.go:299` é promovida à abstração, virando reutilizável e testável; e o descarte silencioso do erro de despacho em `internal/runtime/runner.go:428` e `internal/runtime/runner.go:221` é corrigido, porque RF-30 proíbe degradação silenciosa e hoje uma falha de gravação de memória é completamente invisível.

## Arquitetura do Sistema

### Visão Geral dos Componentes

**Componentes novos** (todos a criar; caminhos em texto plano por convenção declarada acima):

| Componente | Local | Responsabilidade |
|---|---|---|
| Porta de memória | internal/runtime/memory_port.go | Interface de dois métodos declarada **no pacote consumidor**, satisfeita pela fachada. Segue a convenção de interface no consumidor já usada em `internal/runtime/runner.go:32` |
| Fachada | internal/runtime/memory/durable/facade.go | Único ponto de contato do runtime. Ordem de chamada, precedência de invariantes, tradução de erro. Sem decisão de domínio |
| Tipos de domínio | internal/runtime/memory/durable/fato.go | `Fato`, `Durabilidade`, `ChaveSemantica`, `HashConteudo`, `OrigemDeFato`, `Ligacao`, `EstadoFato` |
| Leitor/serializador de Página | internal/runtime/memory/durable/pagina.go | Parse e serialização com round-trip lossless. Frontmatter via `gopkg.in/yaml.v3`, já dependência direta |
| Agregado de camada | internal/runtime/memory/durable/camada.go | Fronteira transacional. Lock por camada, consolidação, idempotência, contradição, escrita atômica |
| Política de relevância | internal/runtime/memory/durable/politica_relevancia.go | Ordenação determinística dos candidatos |
| Política de orçamento | internal/runtime/memory/durable/politica_orcamento.go | Teto global por `specs.WindowClass` e cotas por camada |
| Política de sanitização | internal/runtime/memory/durable/politica_sanitizacao.go | Catálogo mínimo, redação de trecho, recusa quando não isolável |
| Política de compactação | internal/runtime/memory/durable/politica_compactacao.go | Conjunto a arquivar, preservando bloco humano |
| Política de lease | internal/runtime/memory/durable/politica_lease.go | Concessão, recusa e tomada do bastão |
| Detecção de processo vivo | internal/runtime/memory/durable/processo_unix.go e processo_windows.go | Separação por build tag, replicando o padrão de `internal/taskloop/orchestrator_lock_unix.go` |
| Eventos de domínio | internal/runtime/memory/durable/eventos.go | Sete tipos satisfazendo `hooks.Event` |
| Comando de operação | cmd/ai_spec_harness/memory.go | `ai-spec memory show|search|export|compact|migrate|handoff` |

**Componentes existentes modificados:**

| Arquivo | Alteração | Natureza |
|---|---|---|
| `internal/fs/fs.go` | Método de escrita atômica na interface `FileSystem` e em `OSFileSystem` | Estritamente aditiva |
| `internal/fs/fake.go` | Mesmo método em `FakeFileSystem` | Estritamente aditiva |
| `internal/config/runtime.go` | Campos de configuração de memória durável | Aditiva; zero-value preserva comportamento |
| `internal/config/resolver.go` | Merge dos campos novos em `mergeInto` | Aditiva — **ponto de maior risco de esquecimento** |
| `internal/runtime/types.go` | Campos no `Job` para a feature | Aditiva; zero-value preserva |
| `internal/runtime/runner.go` | Resolução da porta; correção do descarte de erro nas linhas 428 e 221 | Aditiva mais correção de defeito |
| `internal/runtime/persistence/report.go` | Injeção da seção de evidência **antes** da seção de métricas | Aditiva, com invariante de posição |
| `cmd/ai_spec_harness/root.go` | Registro do comando novo | Aditiva |
| `cmd/ai_spec_harness/task_loop.go` | Flag de ativação | Aditiva |
| `docs/cli-schema.json` | Contrato do comando e flag novos | Obrigatória — gate bidirecional |
| `internal/taskloop/runtimeconfig.go` | Propagação da chave nova em `optionsToConfigOverrides` | Aditiva — **segundo ponto de esquecimento da cascata** |
| `internal/taskloop/taskloop.go` e `internal/taskloop/acpinvoker.go` | `Options` e option de invoker para a flag nova | Aditiva |
| `internal/runtime/hooks/memory_persist.go` | Descompasso de tipo passa a ser observável (MD-004, passo 6) | Correção de defeito; template de saída inalterado |
| `Makefile` | Alvos `integration` e `bench` enumeram diretórios fixos e precisam incluir o pacote novo | Obrigatória — sem isso os testes novos **nunca rodam** |
| `mockery.yml` | Interfaces novas (`Pagina`, `Camada`, `MemoryPort`) | Aditiva |

**Componentes deliberadamente não modificados:** `internal/runtime/memory/store.go` permanece íntegro, porque é o caminho executado quando a feature está desativada e a paridade de RF-29 depende dele. `.claude/hooks/*.sh` não são tocados. O enum fechado de `internal/runtime/events/kinds.go` não é estendido (MD-005).

`internal/runtime/hooks/memory_persist.go` **é** modificado, e a versão anterior desta especificação o declarava intocado — contradição direta com o passo 6 da MD-004, que exige tornar observável o descompasso de tipo de `internal/runtime/hooks/memory_persist.go:57`. A alteração é restrita: o `buildMemoryContent` e o modo de escrita permanecem byte-idênticos, de modo que a paridade de RF-29 continua sustentada por esse arquivo; apenas a asserção de tipo silenciosa deixa de ser silenciosa.

### Fluxo de Dados

```
                        ┌─────────────────────────────────┐
  ACPRunner.Run() ──────▶│  MemoryPort (2 métodos)         │  interface no consumidor
                        └───────────────┬─────────────────┘
                    ┌───────────────────┴───────────────────┐
                    │                                       │
        feature OFF │                                       │ feature ON
                    ▼                                       ▼
      memory.Store (atual, intocado)              Facade (durable)
      → prompt byte-idêntico                        │
                                    ┌───────────────┼───────────────┬──────────────┐
                                    ▼               ▼               ▼              ▼
                              Pagina          Camada          Politicas×5     Lease
                            (round-trip)   (lock+atômico)    (stateless)   (TTL+PID)
                                    │               │
                                    └───────┬───────┘
                                            ▼
                                    fs.FileSystem (injetada)
                                            │
                              ┌─────────────┴─────────────┐
                              ▼                           ▼
                    .aispec/memory/            .specs/<prd>/memory/
                    (camada projeto)          (camadas PRD e task)

  Operação humana: ai-spec memory ──▶ Camada + Politicas  (NÃO passa pela Facade)
```

## Design de Implementação

### Interfaces Chave

**Porta declarada no pacote consumidor** (internal/runtime/memory_port.go):

```go
// MemoryPort é a porta estreita do subsistema de memória (MD-001).
// Declarada no consumidor conforme a convenção de internal/runtime/runner.go:32.
type MemoryPort interface {
	MontarContexto(ctx context.Context, esc MemoryScope) (MemoryContext, error)
	RegistrarSessao(ctx context.Context, in SessionFacts) (MemoryReport, error)
}

var _ MemoryPort = (*durable.Facade)(nil)
```

**Agregado de camada** (internal/runtime/memory/durable/camada.go) — interface pequena, consumida pela fachada e pelo comando de operação:

```go
// Camada é a fronteira transacional do conjunto de Fatos ativos de um escopo.
type Camada interface {
	Ler(ctx context.Context, c Escopo) ([]Fato, BlocoHumano, error)
	Consolidar(ctx context.Context, c Escopo, novos []Fato) (ResultadoConsolidacao, error)
	Arquivar(ctx context.Context, c Escopo, ids []Identidade) error
	Promover(ctx context.Context, de, para Escopo, id Identidade) error
}
```

**Leitor e serializador de Página** (internal/runtime/memory/durable/pagina.go):

```go
// Pagina traduz entre Fatos e Markdown preservando round-trip (RF-36).
type Pagina interface {
	Parse(conteudo []byte) ([]Fato, BlocoHumano, error)
	Serializar(fatos []Fato, humano BlocoHumano) ([]byte, error)
}
```

**Escrita atômica na abstração existente** — adição estritamente aditiva a `internal/fs/fs.go`:

```go
// WriteFileAtomic grava via temporário no mesmo diretório, Sync e Rename.
// Nenhum arquivo parcial fica visível. Porta do padrão de internal/sdd/state.go:299.
WriteFileAtomic(path string, data []byte) error
```

**Políticas** — structs stateless com métodos, **sem interface**, no idioma de `internal/runtime/memory/window_policy.go:37`, porque cada uma tem implementação única (MD-001):

```go
type PoliticaOrcamento struct{}

var DefaultPoliticaOrcamento = PoliticaOrcamento{}

// Resolver devolve teto global e cotas por camada a partir da classe de janela.
// Cotas default: projeto 50%, PRD 30%, task 20%; sobra de uma camada é cedida às outras.
func (p PoliticaOrcamento) Resolver(class specs.WindowClass, cfg OrcamentoConfig) Orcamento
```

**Erros de domínio** — sentinelas de pacote mais envelopamento por contexto, forma usada em ~50 pontos do repositório e a única que suporta `errors.Is` como o repositório pratica:

```go
var (
	ErrChaveSemanticaAusente = errors.New("durable: chave semantica ausente")
	ErrDurabilidadeAusente   = errors.New("durable: durabilidade ausente")
	ErrSegredoNaoRedigivel   = errors.New("durable: trecho sensivel nao isolavel")
	ErrPromocaoSemMarcacao   = errors.New("durable: promocao exige marcacao explicita")
	ErrBastaoJaReivindicado  = errors.New("durable: bastao detido por processo vivo")
	ErrRoundTripNaoPreservado = errors.New("durable: serializacao nao preserva conteudo humano")
	ErrPaginaIlegivel        = errors.New("durable: pagina ilegivel")
	ErrOrcamentoExcedido     = errors.New("durable: orcamento de contexto excedido")
	ErrLimiteInalcancavel    = errors.New("durable: limite inalcancavel por conteudo humano")
	ErrMigracaoJaAplicada    = errors.New("durable: migracao ja aplicada")
)
```

Os quatro primeiros mais `ErrRoundTripNaoPreservado` e `ErrPromocaoSemMarcacao` são **falha fechada na escrita**. `ErrPaginaIlegivel`, `ErrOrcamentoExcedido` e `ErrLimiteInalcancavel` são **degradação explícita na leitura ou na compactação** — reportados, nunca silenciosos.

**Options** — se a fachada precisar de configuração opcional, seguem o molde do repositório: `type Option func(*Facade)` com as options declaradas como métodos de `*Catalog`, conforme `internal/runtime/options.go:11`. Uma option standalone violaria a regra R1.

### Modelos de Dados

```go
// Identidade compõe a identidade do Fato (MD-002): chave semântica + hash.
type Identidade struct {
	Chave ChaveSemantica // sobre o que o Fato afirma algo
	Hash  HashConteudo   // sha256 do conteúdo normalizado
}

// Durabilidade é enum fechado. Zero-value é inválido de propósito:
// Fato sem durabilidade declarada é escrita inválida (RF-07).
type Durabilidade uint8

const (
	DurabilidadeInvalida Durabilidade = iota // zero-value: recusado
	DurabilidadeEfemera                      // camada task; nunca promove
	DurabilidadeDePRD                        // camada PRD
	DurabilidadeDuravel                      // elegível a promoção
)

type Fato struct {
	Identidade  Identidade
	Conteudo    string
	Durabilidade Durabilidade
	Origem      OrigemDeFato // sessão, CLI, task, data — sustenta RF-31
	Estado      EstadoFato   // Proposto|Ativo|Contradito|Promovido|Arquivado
	Ligacoes    []Ligacao    // substitui|causa|corrige|contradiz
}
```

`BlocoHumano` carrega o conteúdo de autoria humana verbatim, contado no orçamento e nunca reescrito (RF-37). `MemoryReport` agrega escritas, redações, compactação, omissões por orçamento, páginas isoladas e tomada de bastão — é o valor estruturado que RF-34 exige, e é devolvido como valor, não como argumentos de saída (critério de contenção do MD-001).

**Formato de Página** — Markdown válido; cada Fato é uma seção `###` com frontmatter YAML delimitado; o `BlocoHumano` é todo conteúdo fora das seções reconhecidas, preservado na posição original.

### Endpoints de API

Não aplicável: a feature não expõe rede (RF-10). A superfície externa é a linha de comando:

| Subcomando | Função |
|---|---|
| `memory show` | Fatos ativos por camada, com origem e contradições sinalizadas |
| `memory search` | Busca textual e por entidade, scan determinístico sem índice |
| `memory export` | Artefato autocontido e portátil |
| `memory compact` | Compactação determinística sob demanda |
| `memory migrate` | Conversão do formato atual, com backup verificável |
| `memory handoff` | Estado, reivindicação e liberação do bastão |

Molde: comando pai sem `RunE` mais subcomandos, seguindo `cmd/ai_spec_harness/telemetry.go`, com o wiring de `output.New` e `fs.NewOSFileSystem()` de `cmd/ai_spec_harness/skills.go`. Erro com código de saída específico via `newExitError`, conforme `cmd/ai_spec_harness/exit_error.go`.

## Pontos de Integração

Não há integração externa. As três fronteiras são internas e todas já existem:

1. **Ciclo de vida do runtime** — `internal/runtime/hooks/dispatcher.go`. Os sete eventos de domínio satisfazem `hooks.Event`, que exige um único método. O fan-out é sequencial com aborto no primeiro erro, então os hooks de memória tratam o próprio erro e o compõem na evidência em vez de propagá-lo, para não impedir hooks alheios (MD-005).
2. **Filesystem** — `internal/fs.FileSystem`, injetada por construtor. Antes de gravar em caminho derivado de configuração, `fs.RefuseExternalSymlink` é chamada para resistir a traversal por symlink.
3. **Configuração** — cascata de `internal/config/resolver.go:61`, sem redefinir precedência.

**Tratamento de erro na fronteira:** a fachada traduz erro interno em erro de fronteira envelopado com contexto (`fmt.Errorf("durable: contexto: %w", err)`), preservando a sentinela para `errors.Is`. O consumidor distingue falha fechada de degradação pela sentinela, não por string.

## Abordagem de Testes

### Testes Unitários

Forma: suite `testify/suite` mais tabela de cenários mais `s.Run`, com o objeto sob teste instanciado **dentro** do loop, seguindo `internal/detect/detect_test.go:20` e `internal/agents/registry_test.go:75`. Nomes de cenário em PT-BR, frase imperativa começando com "deve". `SetupTest` só se a suite ganhar campos — o repositório não tem nenhuma ocorrência de `SetupTest` hoje, e replicar essa ausência é intencional: o isolamento vem de instanciar dentro do `s.Run`.

Substituição de dependência: `fs.NewFakeFileSystem()`, usado em 46 arquivos de teste do repositório. As interfaces novas são declaradas em `mockery.yml` para satisfazer a regra R3 e o gate `make check-mocks`, mas **os testes não importam os mocks gerados** — nenhum teste do repositório importa mock gerado hoje, e prometer o contrário seria especificar algo sem precedente.

Cenários críticos por componente:

- **Página:** round-trip em corpus de páginas reais do repositório, não apenas fixtures sintéticas; falha se qualquer byte de `BlocoHumano` mudar.
- **Identidade:** idempotência com chave e hash iguais; candidatura a contradição com chave igual e hash diferente; derivação determinística de chave a partir do mesmo sinal estruturado em duas execuções.
- **Durabilidade:** zero-value recusado; resolução de camada para os três valores; promoção de efêmero recusada.
- **Orçamento:** cotas respeitadas; cessão de sobra; omissões declaradas ao atingir o teto.
- **Sanitização:** cada padrão do catálogo mínimo; recusa quando o trecho não é isolável; redação registrada.
- **Compactação:** camada volta ao limite; bloco humano preservado; limite inalcançável reportado.
- **Lease:** concessão em bastão livre; recusa com dono vivo dentro do prazo; tomada por prazo vencido; tomada por dono inexistente.
- **Fachada:** ordem de invariantes — teste que falha se a persistência ocorrer antes da sanitização.
- **Escrita atômica:** arquivo parcial nunca visível; erro de `Sync` e de `Close` verificado, porque `errcheck` só tolera a allowlist de `.golangci.yml:14`, e `Sync` não está nela.

**Paridade (RF-29):** teste dedicado comparando o prompt final byte a byte com a saída atual, nos quatro casos — ausência total de memória, somente workflow, workflow e task na ordem, e diretiva de compactação anexada.

**Ancoragem, decidida contra falso positivo.** O teste **não** deve chamar `injectMemoryContext` nem ancorar em `internal/runtime/runner.go:490`, porque o wiring da porta substitui o call site de `internal/runtime/runner.go:149` e o gate continuaria verde enquanto o prompt real mudasse — falso positivo perfeito. A ancoragem é um hook de captura registrado em `hooks.PointPromptPostBuild`, cujo `PromptBuildEvent.Prompt` é ponteiro para o prompt final e é despachado em `internal/runtime/runner.go:290`. Isso mede o byte na fronteira externa, depois de toda montagem, e sobrevive a qualquer refactor interno. O texto de referência é a concatenação literal produzida hoje por `internal/runtime/runner.go:490` até `internal/runtime/runner.go:520`, capturada como golden.

**O golden só vira gate na tarefa de wiring.** Na tarefa que o cria, o teste é um retrato do comportamento atual. O critério de aceite da tarefa de wiring é que esse arquivo de teste passe **sem nenhuma alteração de expectativa** — é essa regra que converte retrato em gate.

### Testes de Integração

**Decisão: sim, são necessários** — dois dos três critérios do template são atendidos com folga, e sem testcontainers.

- Existe fronteira de entrada e saída crítica onde substituto não garante correção: exclusão mútua inter-processo. O único molde concorrente do repositório, `internal/taskloop/orchestrator_test.go:215`, usa goroutines **no mesmo processo**, o que não exercita `flock` nem criação exclusiva de arquivo de verdade.
- Já houve incidente da mesma classe: `internal/taskloop/orchestrator_lock_windows.go:11` documenta no próprio código que processo morto deixa lock órfão bloqueando permanentemente, com semântica divergente da outra plataforma.
- Custo é proporcional: **nenhum container é necessário**. Basta `exec.Command` reinvocando o próprio binário de teste, mais `t.TempDir()`.

Sob a marcação `//go:build integration`, alvo `make integration`:

1. Dois processos reais competindo pela mesma camada: nenhuma página corrompida, nenhum Fato perdido, e a falha, quando ocorre, é exclusivamente a sentinela de lock.
2. Lock órfão: processo morto sem liberar; o próximo detecta por prazo vencido ou por dono inexistente, e a tomada aparece na evidência.
3. Bootstrap em repositório vazio, confirmando que ausência total de memória não falha (RF-21).
4. Migração com backup verificável e recusa de reaplicação (RF-33).
5. Handoff cross-CLI: sessão em uma CLI, sessão seguinte em outra, com os fatos duráveis presentes no contexto injetado (RF-25, objetivo O-1).

### Testes E2E

O harness não tem interface de usuário. O papel de ponta a ponta é cumprido por:

- `tests/integration/` com o servidor ACP falso de `internal/runtime/acpfake/`, exercitando sessão completa com memória ativada e agente que **ignora toda diretiva textual** — é assim que RF-13 e o objetivo O-5 se provam.
- **Extensão obrigatória do teste de validadores de evidência** (`make test-validators`): um relatório contendo a seção de memória, provando que os gates de `.agents/scripts/` continuam capturando corretamente. Sem isso, a seção nova pode fechar prematuramente a captura de uma seção anterior e desligar um gate em silêncio — o padrão de defeito que o `AGENTS.md` documenta. Nenhuma expressão nova pode usar classe de bracket com caractere multibyte.
- Benchmark de recuperação com 1.000 páginas, sob `make bench`, para RF-20.

## Sequenciamento de Desenvolvimento

### Ordem de Build

A ordem abaixo **corrige quatro inversões** da versão anterior desta especificação, encontradas por auditoria adversarial de sequenciamento e confirmadas no código. As correções estão registradas ao final da seção.

As doze entregas foram consolidadas em **dez fatias**, respeitando o teto de tarefas por PRD. A ordem é escolhida para que o risco de regressão só apareça depois de a paridade estar travada.

| # | Fatia | Toca arquivo consumido por outros pacotes? | Risco | Por que nesta posição |
|---|---|---|---|---|
| T1 | Escrita atômica na abstração de filesystem | Sim (`internal/fs/fs.go`, 40 consumidores) | Médio | Aditiva; o compilador prova completude, e os dublês de teste embutem `*fs.FakeFileSystem` (ex.: `internal/runtime/persistence/jsonl_test.go:18`), logo herdam o método novo e continuam compilando |
| T2 | Fato, identidade, durabilidade, sentinelas de erro e Página com round-trip | Não | Baixo | Fusão obrigatória: a assinatura de `Pagina` usa `Fato` e `BlocoHumano`, então tipos e Página não compilam separados. Declara as dez sentinelas aqui, para que nenhuma fatia posterior dispute o mesmo arquivo |
| T3 | Quatro políticas stateless (relevância, orçamento, sanitização, compactação) | Não | Baixo | Funções puras sobre fatos e classe de janela; dependem apenas dos tipos de T2 |
| T4 | Lease de bastão e detecção de processo vivo | Sim (cascata de config) | Médio | **Subiu de 11 para 4.** O lock de camada carrega prazo e referência de processo (MD-003), então T5 e T7 dependem desta fatia. Código por build tag: o compilador só prova a plataforma compilada |
| T5 | Agregado de camada com lock e escrita atômica | Não | Baixo-médio | Depende de T1, T2 e T4. Risco vem de corretude concorrente, não de consumidor externo |
| T6 | Trava de regressão: golden byte-a-byte e fim da degradação silenciosa | Sim (`internal/runtime/runner.go`) | Médio | **Precede o wiring.** Instala o próprio gate que protege as fatias seguintes, e corrige o descarte de erro antes de existir feature — do contrário todo o desenvolvimento do wiring aconteceria com falha de despacho invisível |
| T7 | Fachada, porta e wiring por configuração | Sim (sete arquivos existentes) | **Alto** | Pico de risco, com a rede de T6 montada. Único ponto que altera runner, `Job`, cascata, cadeia do taskloop, flag de CLI, schema de CLI e mocks ao mesmo tempo |
| T8 | Evidência, métricas e telemetria | Sim (`report.go`, ordem de `Run()`) | **Alto** | Depende de T6 e T7. Alto por consequência: mexe com o invariante de posição da seção de métricas e com gates de evidência por expressão regular |
| T9 | Comando `memory` com seis subcomandos, incluindo migração | Sim (`root.go`, schema de CLI) | Médio | Comando novo, sem impacto no caminho existente; `migrate` converte estado em disco, mitigado por backup verificável |
| T10 | Integração multi-processo, e2e com agente não colaborativo, benchmark e alvos de Make | Sim (`Makefile`) | Baixo | Fecha as lacunas que teste unitário não cobre. Dono único do `Makefile`, para que os alvos não sejam disputados |

**Ordem de execução, com paralelismo seguro:** `T1 ∥ T6` → `T2` → `T3 ∥ T4` → `T5` → `T7` → `T8 ∥ T9` → `T10`.

**Pares que parecem paralelos e não são**, por colisão de arquivo verificada: T6 e T7 (ambas em `internal/runtime/runner.go`); T7 e T9 (ambas em `docs/cli-schema.json` e no gate bidirecional); T7 e T4 (ambas adicionam chave na mesma cascata); T2 e T3 (dependência de compilação); e qualquer par que execute `make mocks`, porque a regeneração reescreve o conjunto inteiro e o gate compara tudo.

### Inversões corrigidas nesta revisão

1. **Página antes dos tipos que ela usa.** Inversão de compilação. Corrigida pela fusão em T2.
2. **Lease na posição 11, depois do agregado de camada e da fachada.** A MD-003 decide que o lock de camada carrega prazo e referência de processo, e o plano da própria MD-003 ordena liveness antes do agregado — a ordem anterior contradizia a ADR que ela implementa. Sem a correção, `camada.go` e `facade.go` seriam reabertos, junto com o teste de ordem de invariantes que a MD-001 declara como critério de contenção. Lease subiu para T4.
3. **`memory handoff` antes do lease que ele opera.** Corrigida automaticamente ao subir T4.
4. **Aresta fantasma entre escrita atômica e Página.** A interface `Pagina` opera sobre `[]byte` nas duas direções e não toca filesystem; quem precisa de escrita atômica é o agregado de camada. A dependência declarada na versão anterior era falsa e custava paralelismo. Removida: T1 e T2 correm em paralelo.

### Dependência não declarada anteriormente: reordenação de `Run()`

Em `internal/runtime/runner.go:209`, `persistSummary` — que chama `EnrichReport` — executa **antes** de `dispatchSessionPostEnd` em `internal/runtime/runner.go:213`. O relatório é escrito antes de a memória operar. RF-34 exige no relatório o que a memória leu e gravou, e RF-30 exige a falha de despacho na evidência: ambos são inalcançáveis sem que o despacho anteceda o enriquecimento. Isso é **mudança de ordem de controle**, não injeção de seção.

O defeito já é latente hoje, independentemente desta feature: `summary.ReviewStatus` e `summary.ReviewPath` são atribuídos em `internal/runtime/runner.go:219` e `:220`, depois de `persistSummary`, então o relatório atual já não vê esses dois campos.

A reordenação é decidida em **T7**, protegida pelo golden de T6: trocar a ordem de `persistSummary` e do despacho não altera nenhum byte do prompt, então a paridade permanece válida. O que ela altera é o conteúdo do relatório, e o guarda disso é `make test-validators` estendido, em T8.

### Dependências Técnicas

Nenhuma infraestrutura nova, nenhum serviço externo, nenhuma dependência direta nova — o frontmatter usa `gopkg.in/yaml.v3`, já presente.

Dependências internas entre fatias: T2 não depende de T1; T3 e T5 dependem de T2; T4 depende apenas dos tipos de T2; T5 depende de T1, T2 e T4; T7 depende de T2, T3, T4, T5 e da ordem de T6; T8 depende de T6 e T7; T9 depende de T2, T3, T4 e T5; T10 depende de todas.

**Regenerações obrigatórias:** `make mocks` após T1 (interface `FileSystem`), T2 (`Pagina`), T5 (`Camada`) e T7 (`MemoryPort`), sempre com `make check-mocks` verde. Como a regeneração reescreve o conjunto inteiro de mocks, duas fatias não podem regenerar em paralelo.

**Documentação com dono declarado**, para não ficar órfã: documento do formato de Página em T2; `docs/config-hierarchy.md` em T4 e T7 (chave de prazo do lease e chave de ativação); `docs/troubleshooting.md` em T4, T8 e T9; `docs/task-loop-reference.md` e `CLAUDE.md` em T7; `docs/telemetry-feedback-cycle.md` em T8; índice de ADRs de `AGENTS.md` em T7.

## Monitoramento e Observabilidade

Não há Prometheus nem Grafana no escopo: o harness é processo local de linha de comando, e o próprio modelo de domínio registra que alerta automatizado não se aplica por não haver operador de plantão. A contrapartida é que **toda degradação precisa ser visível no relatório da sessão**.

**Métricas** — pelo mapa de campos extra de `events.MetricSet`, cujos campos são renderizados genericamente por `internal/runtime/persistence/report.go:71`, sem tocar template (MD-005). Chaves declaradas como constantes nomeadas, para que erro de digitação não produza métrica silenciosamente separada: fatos ativos e arquivados por camada, orçamento consumido por camada, contradições abertas, redações aplicadas, compactações executadas, tomadas de bastão, latência de recuperação.

**Logs** — em PT-BR, explícitos e nunca silenciosos: ativação e desativação da feature; página isolada por invalidez; degradação por orçamento com omissões nomeadas; limite inalcançável por conteúdo humano; recusa de reivindicação de bastão com dono e prazo restante; recusa de escrita por segredo não redigível; descompasso de tipo no hook de sessão, que hoje é ignorado em silêncio em `internal/runtime/hooks/memory_persist.go:57`.

**Telemetria** — opt-in via `GOVERNANCE_TELEMETRY`, append-only, formato de pares chave e valor, com campos adicionados condicionalmente para não poluir sessões sem memória, conforme o padrão de `internal/telemetry/acp.go`. Nenhum dado sai da máquina (ADR-006 do repositório).

## Considerações Técnicas

### Decisões Chave

Cada decisão material tem ADR própria:

| ADR | Decisão | Alternativa principal rejeitada |
|---|---|---|
| [MD-001](adr-001-fachada-porta-unica-memoria.md) | Fachada como porta única do runtime; operação humana acessa colaboradores diretamente; políticas sem interface | Duas funções de alto nível — perdeu por perda de isolamento de teste na fronteira e por replicar a precedência de invariantes |
| [MD-002](adr-002-fato-pagina-roundtrip-lossless.md) | Fato identificado por chave semântica mais hash; Página como persistência; round-trip como invariante de escrita | Identidade só por hash — tornaria contradição indetectável sem modelo de linguagem |
| [MD-003](adr-003-escrita-atomica-lock-camada-lease.md) | Escrita atômica promovida à abstração; lock por camada; lease com prazo e verificação de processo | Reaproveitar o lock sem tratar órfão — travaria a memória em uma plataforma até intervenção manual |
| [MD-004](adr-004-optin-paridade-byte-a-byte.md) | Opt-in por flag mais configuração; paridade byte-a-byte como gate; fim do descarte silencioso de erro | Default-on — maximiza exposição a regressão, contra a restrição dominante |
| [MD-005](adr-005-evidencia-metricas-memoria.md) | Eventos pelo dispatcher existente; métricas pelo mapa de campos extra; seção antes das métricas | Estender o enum fechado de tipos de evento — ampliaria contrato lido por ferramentas externas |

### Riscos Conhecidos

| Risco | Impacto | Mitigação |
|---|---|---|
| Esquecer o merge da chave nova em `internal/config/resolver.go:166` | Chave parece configurada e nunca ativa; falha silenciosa. Não há reflexão nem teste genérico que force isso | Teste por camada da cascata, incluindo camada superior que não define a chave e não apaga a inferior |
| Esquecer o **segundo** ponto de propagação, `optionsToConfigOverrides` em `internal/taskloop/runtimeconfig.go`, que hoje propaga apenas `Timeout`, `Concurrent` e `BatchSize` | A flag de CLI não chega à cascata; a chave de config funciona e a flag não. Falha assimétrica e difícil de diagnosticar | Teste que exercita a flag ponta a ponta, de `cmd/ai_spec_harness/task_loop.go` até o `Job`; a cadeia completa é flag → `Options` → option de invoker → `Job` |
| Reordenação de `Run()` descoberta tarde | Reabre `internal/runtime/runner.go` depois de T7 tê-lo estabilizado | A reordenação é decidida em T7, não em T8, e é coberta pelo golden de T6 (não altera bytes de prompt) e por `make test-validators` em T8 |
| Não estender os alvos do `Makefile` | Os testes de integração e o benchmark **nunca rodam** — `integration` e `bench` enumeram diretórios fixos (`Makefile:26` e `Makefile:61`). O risco migra em silêncio para as outras fatias | T10 é dona única do `Makefile`; critério de aceite inclui provar que o pacote novo aparece na saída dos dois alvos |
| Código por build tag: o compilador só prova a plataforma compilada | Detecção de processo vivo pode não compilar ou não funcionar na plataforma não exercitada localmente | CI cobre duas plataformas; o teste de lease usa fallback por prazo quando a verificação de processo não é confiável, e esse fallback é exercitado explicitamente |
| Alterar `internal/fs/fs.go`, consumido por quase todo o repositório | Regressão ampla | Mudança estritamente aditiva; o compilador prova completude; `make check-mocks` força regeneração |
| Gate bidirecional de contrato de CLI | Falha de build ao adicionar comando ou flag sem atualizar o schema | `docs/cli-schema.json` atualizado na mesma entrega; `cmd/ai_spec_harness/cli_contract_test.go:81` e `:189` comparam nas duas direções |
| Seção nova no relatório desligar um gate de evidência em silêncio | Gate aprovado sem verificar | `make test-validators` estendido com relatório contendo a seção; alternação em vez de bracket multibyte |
| Seção de métricas deixar de ser a última | Seção nova apagada na próxima injeção | Injeção posicionada antes; `internal/runtime/persistence/report.go:104` substitui do cabeçalho de métricas até o fim |
| `errcheck` reprovar erro ignorado fora da allowlist | Lint vermelho | `Sync`, `Close` de temporário e `WriteFile` têm erro verificado; a allowlist está em `.golangci.yml:14` |
| `gosec` G115 em conversão de inteiro com perda | Lint vermelho | Contadores em `int`; conversões explícitas com verificação de faixa |
| Cobertura por pacote crítico | Gate vermelho | Threshold total 75% no CI e 70% por pacote em `scripts/check-package-coverage.sh`; o pacote novo pode entrar na lista de críticos |
| Teste concorrente intra-processo dar falsa confiança | Exclusão inter-processo não provada | Teste com `exec.Command` reinvocando o binário de teste; a lacuna do molde existente está declarada, não escondida |
| Derivação instável de chave semântica | Fatos duplicados e contradições espúrias | Derivação determinística do sinal estruturado; teste com duas sessões consecutivas |
| Fachada virar objeto-deus | Perda da fronteira | Critério de contenção: sem decisão de domínio, apenas ordem e tradução de erro |
| Ausência de `-race` nos gates | Corrida não detectada | O subsistema não usa goroutines próprias; se passar a usar, `-race` deve ser adicionado ao alvo de teste como parte da entrega |

**Área que precisa de pesquisa antes da entrega 11:** verificação de processo vivo é intrinsecamente frágil em uma das plataformas suportadas, e não há precedente no repositório. O prazo do lease é o critério que prevalece lá, e essa assimetria é assumida, não resolvida (MD-003).

### Conformidade com Padrões

**`.claude/rules/governance.md` (R-GOV-001, severidade hard).** Precedência aplicada: a governança transversal vence; `go-implementation` vence `object-calisthenics-go` em conflito. Política de evidência atendida — cada decisão desta techspec é justificável pelo PRD, por regra explícita ou por necessidade técnica demonstrada com `path:linha`. Segurança operacional: nenhuma ação de git destrutiva nem publicação remota está no escopo.

**Regras estritas de `go-implementation` (R0–R7, todas `[HARD]`), com o molde real do repositório:**

| Regra | Como esta especificação a satisfaz |
|---|---|
| R0 — `init()` proibida | Nenhuma. Tabelas estáticas como `var _nome` não exportada; parse caro via `go:embed` mais `sync.OnceValues`, molde de `internal/skills/schema.go:21` |
| R1 — função como método | Cada pacote novo ganha `r1_catalog.go` com `type Catalog struct{}` e `func NewCatalog() *Catalog`; operações puras como `func (c *Catalog) Nome(...)`. Políticas como structs stateless com instância compartilhada, molde de `internal/runtime/memory/window_policy.go:20` |
| R2 — sem alias de campo | Campos usados diretamente; variável local só com transformação real |
| R3 — mocks via mockery | Interfaces novas declaradas em `mockery.yml`; `make mocks` mais `scripts/normalize-mocks.sh`; `make check-mocks` verde |
| R4 — `testify/suite` table-driven | Suite mais tabela mais `s.Run`, objeto sob teste dentro do loop. `SetupTest` apenas se a suite ganhar campos |
| R5 — Uber Go Style | Sentinelas de pacote com `errors.New`; envelopamento com `%w`; `var _ Iface = (*impl)(nil)` em cada implementação; nenhum `panic`; nenhum `os.Exit` fora de `main.go` |
| R6 — interface no consumidor, DI, sem estado global | Porta declarada no consumidor; todas as dependências por construtor; nenhuma variável exportada mutável nova |
| R7 — recursos modernos | `any` em vez de `interface{}`; genéricos onde reduzirem duplicação real |

**ADR-001 e ADR-002 do repositório:** binário único com `go:embed` preservado, sem daemon e sem CGO; a abstração de filesystem passa a ser injetada no subsistema de memória, encerrando o desvio de `internal/runtime/memory/store.go:104`.

**Linters realmente habilitados** (`.golangci.yml`): `errcheck`, `staticcheck`, `gosec`, `govet`, `ineffassign`, `unused`. Não há linter de complexidade ciclomática, tamanho de função, comprimento de linha nem `godot` — esta especificação não impõe limites que o repositório não verifica, para não criar restrição fictícia. Linha de base verificada antes de qualquer alteração: `make vet` sem saída e `make lint` com zero problemas, sendo que os 40 problemas brutos vêm integralmente dos mocks gerados e são filtrados pelo processador de arquivos gerados.

### Mapeamento Requisito → Decisão → Teste

| RF | Decisão | Teste |
|---|---|---|
| RF-01 | Três camadas; durabilidade resolve destino (MD-002) | Unit: resolução de camada para os três valores |
| RF-02 | Camada de projeto em `.aispec/memory/`; Markdown puro (MD-002) | Unit: parse e serialização; Integração: arquivo legível e versionável |
| RF-03 | Caminho de PRD preservado, derivado como hoje | Unit: caminho igual ao de `internal/runtime/memory/store.go:88` |
| RF-04 | Camada de task escrita pela fachada | Unit: escrita efetiva; fecha a lacuna de `store.go:188` sem chamador |
| RF-05 | Frontmatter YAML em Markdown válido (MD-002) | Unit: round-trip; Markdown válido |
| RF-06 | Ligações tipadas em quatro relações | Unit: parse e serialização de cada relação |
| RF-07 | Enum fechado de três valores; zero-value inválido (MD-002) | Unit: zero-value recusado com `ErrDurabilidadeAusente` |
| RF-08 | Captura em `session.post_end`, qualquer condição de saída | Integração: sucesso, timeout, cancelamento, permissão negada |
| RF-09 | Fonte dupla; parte estruturada independe do agente | E2E: agente que ignora toda diretiva ainda produz fatos |
| RF-10 | Sem rede e sem LLM no caminho default | Teste que falha se socket, DNS ou CLI externa for usado |
| RF-11 | Consolidação sem remoção de fato não contradito | Unit: fato não mencionado permanece ativo |
| RF-12 | Identidade por chave mais hash (MD-002) | Unit: regravação idêntica não duplica |
| RF-13 | Compactação determinística pela fachada (MD-001) | E2E: agente não colaborativo; nenhuma página acima do limite |
| RF-14 | Arquivamento append-only, reversível | Unit: arquivado fora do contexto e presente no repositório; reversão |
| RF-15 | Catálogo mínimo; redação com registro; recusa se não isolável | Unit: cada padrão; `ErrSegredoNaoRedigivel` |
| RF-16 | Lock por camada mais escrita atômica (MD-003) | Integração: dois processos reais |
| RF-17 | Seleção por relevância, não injeção integral | Unit: seleção menor que o conjunto ativo |
| RF-18 | Teto por classe de janela com cotas e cessão (MD-001) | Unit: cotas, cessão, omissões declaradas |
| RF-19 | Scan determinístico, sem índice (MD-002) | Unit: busca textual e por entidade |
| RF-20 | Alvo de p95 com degradação reportada | Bench: 1.000 páginas |
| RF-21 | Ausência de memória não falha | Unit e Integração: repositório vazio |
| RF-22 | Página inválida isolada e reportada | Unit: sessão prossegue; `ErrPaginaIlegivel` reportado |
| RF-23 | Lease com prazo de 30 min mais processo vivo (MD-003) | Unit: quatro casos; Integração: dono morto |
| RF-24 | Promoção só por marcação explícita | Unit: `ErrPromocaoSemMarcacao`; efêmero recusado |
| RF-25 | Memória neutra de CLI (MD-003) | Integração: handoff cross-CLI |
| RF-26 | Camada de projeto funciona sem PRD ativo | Unit: escopo sem diretório de tasks |
| RF-27 | Contradição por colisão de chave com hash divergente (MD-002) | Unit: detecção sem LLM; ambos permanecem |
| RF-28 | Opt-in por flag mais configuração (MD-004) | Teste por camada da cascata |
| RF-29 | Paridade byte-a-byte como gate (MD-004) | Teste dedicado, quatro casos |
| RF-30 | Fim do descarte silencioso de erro (MD-004) | Unit: falha de despacho registrada e composta na evidência |
| RF-31 | Comando `memory` com seis subcomandos | Gate de contrato de CLI; teste por subcomando |
| RF-32 | Origem nos metadados do Fato (MD-005) | Unit: rastreio até a sessão |
| RF-33 | Migração por comando com backup verificável (MD-004) | Integração: backup e recusa de reaplicação |
| RF-34 | Evidência antes da seção de métricas (MD-005) | `make test-validators` estendido; posição das seções |
| RF-35 | Métricas pelo mapa de campos extra (MD-005) | Unit: chaves presentes; opt-in respeitado |
| RF-36 | Round-trip como invariante de escrita (MD-002) | Unit: corpus real; `ErrRoundTripNaoPreservado` |
| RF-37 | Bloco humano contado e sinalizado (MD-002) | Unit: limite inalcançável reportado sem reescrita |

### Arquivos Relevantes e Dependentes

**Lidos e verificados durante esta especificação** (todos existentes):

`internal/runtime/memory/store.go` · `internal/runtime/memory/window_policy.go` · `internal/runtime/memory/r1_catalog.go` · `internal/runtime/memory/mocks/store.go` · `internal/runtime/runner.go` · `internal/runtime/runner_autoreview.go` · `internal/runtime/types.go` · `internal/runtime/options.go` · `internal/runtime/summary.go` · `internal/runtime/errors.go` · `internal/runtime/hooks/dispatcher.go` · `internal/runtime/hooks/memory_persist.go` · `internal/runtime/hooks/token_budget.go` · `internal/runtime/hooks/governance.go` · `internal/runtime/hooks/spec_drift.go` · `internal/runtime/persistence/report.go` · `internal/runtime/persistence/session.go` · `internal/runtime/persistence/jsonl.go` · `internal/runtime/events/event.go` · `internal/runtime/events/kinds.go` · `internal/runtime/events/metricset.go` · `internal/runtime/specs/window.go` · `internal/config/resolver.go` · `internal/config/runtime.go` · `internal/fs/fs.go` · `internal/fs/fake.go` · `internal/fs/symlink_guard.go` · `internal/sdd/state.go` · `internal/taskloop/orchestrator.go` · `internal/taskloop/orchestrator_lock_unix.go` · `internal/taskloop/orchestrator_lock_windows.go` · `internal/taskloop/acpinvoker.go` · `internal/telemetry/acp.go` · `internal/telemetry/telemetry.go` · `internal/output/output.go` · `cmd/ai_spec_harness/root.go` · `cmd/ai_spec_harness/task_loop.go` · `cmd/ai_spec_harness/telemetry.go` · `cmd/ai_spec_harness/skills.go` · `cmd/ai_spec_harness/exit_error.go` · `cmd/ai_spec_harness/cli_contract_test.go` · `docs/cli-schema.json` · `docs/config-hierarchy.md` · `docs/adr/002-fake-filesystem-testes.md` · `docs/adr/006-telemetria-feedback-cycle.md` · `mockery.yml` · `.golangci.yml` · `Makefile` · `AGENTS.md` · `.claude/rules/governance.md` · `.agents/scripts/validate-task-evidence.sh` · `.agents/scripts/validate-review-evidence.sh`

**Testes que precisam permanecer verdes sem alteração de expectativa:** `internal/runtime/memory/store_test.go` · `internal/runtime/memory/window_policy_test.go` · `internal/runtime/hooks/memory_persist_test.go` · `internal/runtime/runner_memory_test.go` · `cmd/ai_spec_harness/cli_contract_test.go` · `cmd/ai_spec_harness/task_loop_test.go` · `internal/config/resolver_test.go`

Qualquer necessidade de alterar a expectativa de um desses testes é sinal de regressão, não de teste obsoleto.
