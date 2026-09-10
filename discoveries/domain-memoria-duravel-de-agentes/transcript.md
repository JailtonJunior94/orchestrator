# Transcript da Modelagem de Dominio

## Contexto Inicial

- Problema: o harness orquestra sessoes de agentes em 4 CLIs e persiste evidencia rica por sessao, mas nao possui conhecimento que sobreviva a sessao. O custo aparece como retrabalho silencioso (redescoberta de convencao, rediagnostico, repeticao de erro ja corrigido).
- Objetivo de negocio: eliminar o custo de re-explicar arquitetura a cada sessao, em uso real de producao.
- Objetivo de modelagem: preparar implementacao com dominio explicito de memoria durave (camadas, fatos, promocao, bastao de continuidade), antes de qualquer codigo Go.
- Natureza do pedido: quebra de dominio mal modelado. O subsistema atual colapsa "memoria" em um arquivo sobrescrito por sessao, sem fato, sem durabilidade e sem ownership.
- Escopo de codebase: path local `/Users/jailtonjunior/Git/orchestrator` (Go 1.27.1, 7 dependencias diretas, 182 arquivos de teste). Confronto executado.
- Materiais de apoio: PRD `.specs/prd-memoria-duravel-agentes/prd.md` (spec-version 2, 35 RF, 14 decisoes fechadas), AGENTS.md, ADR-001/006/008/016/023, e o desenho publico do ai-memory 2.0 como referencia conceitual.

## Confronto com Codebase

Classificacao das evidencias confrontadas no codebase local.

| Achado | Classificacao | Evidencia |
|---|---|---|
| Persistencia de memoria sobrescreve o estado anterior a cada sessao | confirmado | `internal/runtime/hooks/memory_persist.go:65` — `WriteWorkflow(ctx, content, memory.WriteModeReplace)` |
| Tier de task existe na interface e nao tem chamador de producao | confirmado | `internal/runtime/memory/store.go:68` (contrato) e `:188` (implementacao); varredura de todo o codigo nao-teste nao encontra chamador |
| Tier de task e lido, mas nunca escrito | confirmado | `internal/runtime/runner.go:482` — `store.ReadTask(ctx, j.TaskFileName)` |
| Compactacao e diretiva textual sem garantia de execucao | confirmado | `internal/runtime/runner.go:516` — `sb.WriteString("\ncompact the flagged memory files before proceeding\n")` |
| Ponto de ciclo de vida disponivel para captura ao fim da sessao | confirmado | `internal/runtime/hooks/dispatcher.go:25` — `PointSessionPostEnd = "session.post_end"` |
| Politica de limites sensivel a janela de contexto ja existe | confirmado | `internal/runtime/memory/window_policy.go:37` — `LimitsForWithOverride(class, base, explicit)` |
| Caminho da memoria de PRD e derivado de `tasksDir` + `memory/` + `MEMORY.md` | confirmado | `internal/runtime/memory/store.go:12-13` (constantes) e `:88` (composicao) |
| `.aispec` e marcador reconhecido de raiz de projeto | confirmado | `internal/config/resolver.go:21` — `_projectMarkers = []string{".git", ".aispec", ".claude", ".agents"}` |
| Padrao de lock por plataforma reutilizavel para escrita concorrente | confirmado | `internal/taskloop/orchestrator_lock_unix.go:12` — `acquireOrchestratorLock(path)` |
| Nocao de fato de memoria com identidade, origem e durabilidade | ausente | nenhum tipo, campo ou constante no codebase representa fato de memoria; hoje o conteudo e string opaca |
| Nocao de bastao de continuidade / ownership de sessao | ausente | nenhuma estrutura de lease, dono ou takeover em `internal/runtime/` |
| Nocao de camada de projeto (conhecimento acima do PRD) | ausente | `store.directory()` e sempre relativo a `tasksDir`; nao existe raiz de memoria de repositorio |
| Deteccao de contradicao entre fatos | ausente | nenhuma ligacao tipada entre conteudos de memoria |

Nota de metodo: nenhum achado foi classificado como `confirmado` com base em teste, mock, fixture ou documentacao. Os quatro achados `ausente` sao a fronteira que esta modelagem precisa cobrir.

## Rodada 1 - Linguagem e Fronteiras

Perguntas em multipla escolha, com consequencia explicita por opcao. Respostas do usuario em negrito.

**P1.1 - Objetivo dominante da modelagem**
- **Robustez em producao com zero regressao (escolhida)** — subordina toda decisao de modelo a preservacao do comportamento atual; custo: mais gates e paridade testada.
- Clareza maxima de regra — risco de sobre-modelagem para subsistema interno.
- Velocidade de entrega — risco de repetir o defeito atual (memoria sem fato).
- Compatibilidade maxima com o formato atual — limita o modelo ao template de 4 campos.

**P1.2 - Termo canonico da unidade principal**
- **Fato (escolhida)** — unidade atomica com origem, data e durabilidade; habilita consolidacao, arquivamento, promocao e deteccao de contradicao por fato.
- Pagina — granularidade grossa demais para promocao seletiva.
- Nota — sem compromisso de verificabilidade nem de origem.
- Registro/Entrada — nao distingue durave de efemero.

**P1.3 - Fronteira principal**
- **Contexto Memoria proprio (escolhida)** — isola conhecimento (mutavel, consolidado) de evidencia (imutavel, selada); integra com Runtime de Sessao pelos pontos de ciclo de vida existentes.
- Subdominio do Runtime — acopla durabilidade ao ciclo de vida da sessao, origem do modo `replace` atual.
- Dentro de Evidencia/Persistencia — quebraria a garantia de `seal-evidence`.
- Dois contextos (Projeto e Sessao) — duplica vocabulario para ganho que a marcacao de durabilidade ja entrega.

**P1.4 - Restricao dominante**
- **Zero regressao no caminho default (escolhida)** — opt-in obrigatorio e paridade byte-a-byte como gate de merge.
- Custo operacional minimo — pressao para omitir invariantes.
- Seguranca de segredo — daria a um falso positivo poder de degradar memoria.
- Prazo — risco de entregar meia camada.

**Consolidado da rodada:** termo canonico `Fato`; documento agrupador `Pagina`; contexto proprio `Memoria`; objetivo e restricao dominantes convergem em robustez com zero regressao. Termos proibidos: `Nota`, `Registro`, `Entrada` como sinonimos de Fato; `Memoria` usado como sinonimo de `Evidencia`.

## Rodada 2 - Workflow e Comportamento

**P2.1 - Gatilho do workflow de captura**
- **Fim de sessao + reconciliacao na abertura seguinte (escolhida)** — captura no ponto `session.post_end` existente; na abertura seguinte, reconcilia sessao orfa lendo o `events.jsonl` anterior. Cobre crash e `kill -9`; custo de um caminho de reconciliacao.
- Somente fim de sessao — perda total em crash.
- Incremental por evento — contencao de lock e ruido.
- Somente acao humana — restaura a cerimonia que a feature elimina.

**P2.2 - Comando dominante**
- **Registrar Fato (escolhida)** — consolidacao, idempotencia e sanitizacao passam a ser regras de dominio executadas antes de I/O.
- Gravar Memoria — colapsa em CRUD de arquivo.
- Resumir Sessao — sem granularidade de fato.
- Sincronizar Memoria — vocabulario de sync sem sync real.

**P2.3 - Conjunto de eventos**
- **Sete eventos alinhados a RF-33 (escolhida)**: `FatoRegistrado`, `FatoArquivado`, `FatoPromovido`, `ContradicaoDetectada`, `SegredoRedigido`, `CompactacaoExecutada`, `BastaoTransferido`. Cada item exigido por RF-33 na evidencia tem fonte propria.
- Cinco eventos com redacao/compactacao como metrica — enfraquece rastreio de RF-31.
- Evento generico unico — evidencia perde significado.
- Nenhum evento — RF-31 e RF-33 sem fonte.

**P2.4 - Estado ilegal de maior perigo (invariante com primazia)**
- **Perda silenciosa de fato durave (escolhida)** — unico dano irrecuperavel por inspecao. `delete` e substituido por arquivamento em todo o modelo.
- Duplicidade — detectavel e corrigivel a posteriori.
- Dois donos de bastao — dano de coordenacao, nao de conhecimento.
- Segredo persistido — irreversivel, mas enderecado por redacao em vez de descarte.

## Rodada 3 - Regras e Invariantes

**P3.1 - Regra de negocio central**
- **Durabilidade governa camada e ciclo de vida (escolhida)** — a durabilidade declarada de um Fato determina camada, elegibilidade a promocao e momento de arquivamento. Consequencia aceita: durabilidade e campo obrigatorio na origem.
- Elegibilidade por relevancia — deixaria a escrita sem regra forte.
- Janela temporal — trataria recencia como validade.
- Autorizacao por sessao — governa quem escreve, nao o que e conhecimento.

**P3.2 - Ordem de precedencia entre invariantes**
- **Segredo > Perda > Dono unico > Orcamento (escolhida)** — a tensao entre as duas primeiras e resolvida por redacao de trecho: o segredo sai, o Fato permanece. Nenhuma das duas e sacrificada.
- Perda > Segredo — permitiria segredo persistido em arquivo versionado.
- Orcamento acima de todas — omissao por teto seria indistinguivel de perda.
- Sem ordem — comportamento nao deterministico em conflito.

**P3.3 - Politica de relevancia**
- **Politica deterministica parametrizada por config (escolhida)** — criterios e pesos explicitos na cascata do ADR-016, resultado reproduzivel para a mesma entrada.
- Heuristica fixa no codigo — calibragem exigiria release.
- Delegada ao agente — destroi determinismo e garantia de orcamento.
- Cronologica inversa — enterraria conhecimento de projeto sob ruido de task.

**P3.4 - Estrategia de erro de dominio**
- **Erros tipados; falha fechada na escrita, degradacao explicita na leitura (escolhida)** — unica combinacao que satisfaz RF-21, RF-22 e RF-29 simultaneamente.
- Falha fechada em tudo — pagina corrompida abortaria a sessao (viola RF-21/RF-22).
- Degradar em tudo — escrita invalida chegaria ao remoto.
- Erro generico unico — evidencia de RF-33 sem semantica.

## Rodada 4 - Tipos e Integracoes

**P4.1 - Ownership transacional**
- **Agregado `CamadaDeMemoria`, com `Fato` como entidade interna (escolhida)** — a consistencia protegida e o conjunto de fatos ativos de uma camada. Lock e escrita atomica por camada, casando com `internal/taskloop/orchestrator_lock_unix.go:12`.
- Agregado por Fato — invariantes sem ponto de aplicacao.
- Agregado unico das tres camadas — escrita de task bloquearia leitura de projeto.
- Repositorio plano — sem fronteira transacional.

**P4.2 - Fronteira externa dominante**
- **Filesystem do repositorio + pontos de ciclo de vida do runtime (escolhida)** — unicas fronteiras reais. Traducao para Markdown ocorre na borda; o modelo interno nunca e dirigido pelo formato de arquivo nem pelo formato de prompt.
- Operacao humana via CLI — inverteria a proposta de valor.
- Contrato de prompt — anti-padrao explicitamente proibido pela skill.
- Git — versionamento e consequencia do local do arquivo, nao contrato de dominio.

**P4.3 - Consistencia**
- **Forte por camada, eventual entre camadas (escolhida)** — escrita atomica sob lock dentro da camada; promocao e operacao explicita desacoplada da sessao. Consequencia aceita: promocao pendente deve ser visivel na inspecao.
- Forte global — serializaria sessoes concorrentes no mesmo projeto.
- Eventual em tudo — nao-duplicidade sem ponto de aplicacao.
- Sem garantia — corrupcao sob concorrencia, proibida por RF-16.

**P4.4 - Postura de custo e operacao**
- **Menor desenho que preserva as invariantes (escolhida)** — sem indice, sem daemon, reuso de lock e hooks existentes. Limitacao de performance ao volume alvo e aceita e declarada.
- Robustez adicional — contraria D-09 e D-12 ja fechadas.
- Equilibrio sem postura — decisoes de custo ad-hoc.
- Otimizar acima do alvo — complexidade por volume inexistente.

## Rodada 5 - Riscos Materiais Remanescentes

**P5.1 - Valores de Durabilidade**
- **Tres niveis alinhados as camadas: Efemero, DePRD, Duravel (escolhida)** — a durabilidade mapeia direto para a camada de destino, sem tabela de traducao; resolucao de camada fica deterministica por construcao. Efemero persiste, apenas nunca promove.
- Dois niveis — exigiria segunda regra para escolher entre task e PRD.
- Quatro niveis com Descartavel — nivel que nao produz Fato e ruido: nao gravar e ausencia de comando.
- Escala numerica 0-100 — threshold arbitrario sem semantica de negocio.

**P5.2 - Identidade do Fato**
- **Chave semantica declarada + hash de conteudo (escolhida)** — a chave diz sobre o que o Fato fala; o hash detecta regravacao identica. Dois fatos de mesma chave com conteudo divergente sao candidatos a contradicao, dando base deterministica a deteccao sem modelo de linguagem.
- Apenas hash — contradicao indetectavel.
- Apenas id gerado — viola idempotencia.
- Caminho + posicao — identidade rompe a cada edicao ou compactacao.

**P5.3 - Natureza da Pagina**
- **Pagina e forma de persistencia; Fato e secao com metadados; parse lossless (escolhida)** — Markdown permanece fonte de verdade e edicao manual e respeitada. Consequencia aceita: round-trip parse/serialize e invariante protegida por teste dedicado permanente.
- Pagina como agregado com Fato sem identidade — reverteria o termo canonico na pratica.
- Um arquivo por Fato — explosao de arquivos e historico ruidoso.
- Pagina como projecao de arquivo de dados — destruiria Markdown como fonte de verdade.

**P5.4 - Conteudo editado a mao sem Fato correspondente**
- **Preservado, contado no orcamento e sinalizado para compactacao humana (escolhida)** — nunca reescrito automaticamente. Quando a pagina excede limites por conteudo humano, a violacao e reportada em vez de silenciada.
- Preservado fora do orcamento e da compactacao — abriria caminho de crescimento ilimitado.
- Convertido em Fato com durabilidade inferida — inventaria semantica nao declarada.
- Quarentena — puniria a edicao manual que a feature promove como valor.

**Encerramento das rodadas:** os nove gates de `references/quality-gates.md` passam a ter decisao. Nenhum risco material permanece sem endereco; nenhuma rodada adicional foi aberta.

Aberta conforme Step 8: os gates 2, 3 e 5 de `references/quality-gates.md` permanecem sem decisao sobre enumeracao de durabilidade, identidade do Fato, natureza da Pagina e tratamento de conteudo editado a mao.

## Decisoes Registradas

Decisoes efetivas desta modelagem, na ordem em que foram fechadas. As decisoes D-01 a D-14 do PRD permanecem validas e nao foram reabertas.

| # | Decisao | Rodada |
|---|---|---|
| M-01 | Objetivo dominante: robustez em producao com zero regressao | 1 |
| M-02 | Termo canonico: `Fato` como unidade; `Pagina` como documento agrupador | 1 |
| M-03 | Contexto `Memoria` proprio, separado de `Evidencia` | 1 |
| M-04 | Restricao dominante: zero regressao no caminho default | 1 |
| M-05 | Gatilho: fim de sessao + reconciliacao de sessao orfa na abertura seguinte | 2 |
| M-06 | Comando dominante: `RegistrarFato` | 2 |
| M-07 | Sete eventos de dominio, alinhados ao que o relatorio de execucao precisa registrar | 2 |
| M-08 | Invariante com primazia: nao perda silenciosa de Fato durave; `delete` substituido por arquivamento | 2 |
| M-09 | Regra central: durabilidade governa camada e ciclo de vida | 3 |
| M-10 | Precedencia entre invariantes: segredo > nao perda > dono unico > orcamento | 3 |
| M-11 | Politica de relevancia deterministica parametrizada por configuracao | 3 |
| M-12 | Erros tipados: falha fechada na escrita, degradacao explicita na leitura | 3 |
| M-13 | Agregado `CamadaDeMemoria` com `Fato` como entidade interna; lock por camada | 4 |
| M-14 | Fronteira externa dominante: filesystem do repositorio + ciclo de vida do runtime | 4 |
| M-15 | Consistencia forte por camada, eventual entre camadas | 4 |
| M-16 | Postura de custo: menor desenho que preserva as invariantes | 4 |
| M-17 | Durabilidade em tres niveis: Efemero, DePRD, Duravel | 5 |
| M-18 | Identidade do Fato: chave semantica declarada + hash de conteudo | 5 |
| M-19 | Pagina e forma de persistencia; Fato e secao com metadados; round-trip lossless como invariante | 5 |
| M-20 | Conteudo de autoria humana: preservado, contado no orcamento, sinalizado para compactacao humana | 5 |

Segundo agregado (`BastaoDeContinuidade`) justificado em Gate 8: ciclo de vida independente das camadas; uni-lo faria a reivindicacao de continuidade bloquear escrita de conhecimento sem necessidade.
