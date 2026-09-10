# DOMAIN MODEL

## Titulo
Ciclo de Aprovacao e Catalogo de Agentes

## Resumo Executivo
Contexto:
O `ai-spec-harness` orquestra desenvolvimento orientado a especificacao usando CLIs de IA. Duas
capacidades hoje falham silenciosamente: (a) o conjunto de agentes suportados nao reflete mais o
ecossistema oficial e carrega uma excecao de politica exclusiva do Gemini; (b) o encerramento de uma
tarefa afirma aprovacao sem prova, porque o veredito de producao e derivado de texto sintetico e
porque remarks nao-criticos fecham a tarefa.

Decisao central:
O modelo organiza dois comportamentos distintos e ate hoje entrelacados: o `Catalogo de Agentes`,
que declara quais CLIs existem e como cada um aplica governanca, e o `Ciclo de Aprovacao`, que
conduz rodadas de revisao e correcao ate um veredito aprovado e comprovado, ou ate uma parada
explicita.

Status de prontidao:
done

## Problema e Objetivo
Problema atual:
1. "Tarefa concluida" nao significa "requisito verificado". `internal/runtime/runner_autoreview.go:233`
   monta um texto sintetico (`buildReviewOutputFromSummary`) que alimenta `parseReviewStatus`
   (`:60`) — o parser de producao nunca ve a saida real do revisor. O veredito e ficcao.
2. Nao ha ciclo: `internal/runtime/runner.go:216-228` executa a revisao uma unica vez.
   `internal/taskloop/bugfix.go:83` tem um loop, mas as duas camadas nao se conhecem, e ele encerra
   assim que os achados criticos esvaziam (`:131`), aceitando qualquer veredito nao-critico.
3. O criterio de aceite pode ser declarado "nao verificavel pelo diff" e, hoje, isso apenas vira
   risco residual — nunca bloqueia.
4. Adicionar ou remover um agente custa ~215 arquivos, porque a enumeracao dos CLIs esta espalhada
   por enums, catalogos duplicados, invariantes de paridade, snapshots e mensagens de erro.
5. Gates que parecem ativos podem estar inertes: a chave de evento `stop` usada para o Copilot nao
   existe no CLI (o correto e `agentStop`), e os hooks do Codex nao rodam sem `trusted_hash`.

Objetivo de negocio:
Permitir trocar de agente de IA sem perder gates, observabilidade ou rigor, e poder confiar que uma
tarefa marcada como concluida teve cada regra de negocio e cada criterio de aceite verificado contra
evidencia real.

Objetivo de modelagem:
Tornar inequivoco (a) o que e um Agente e quem e dono do conjunto fechado deles; (b) o que e uma
aprovacao, quando ela pode ser afirmada e como um estado de "aprovado sem prova" se torna
inconstruivel; (c) como um ciclo de remediacao termina — por aprovacao, por esgotamento, por
nao-convergencia ou por ausencia de mudanca.

## Materiais e Evidencias
Materiais usados:
- `.specs/prd-harness-quatro-clis-loop-aprovacao/prd.md` (PRD com 51 requisitos funcionais)
- Codebase local `/Users/jailtonjunior/Git/orchestrator` (Go 1.27.1)
- Documentacao embarcada nos binarios `opencode` 1.18.30, `codex` 0.154.0, `copilot` 1.0.83
- Tipos publicados de `@opencode-ai/plugin` 1.18.30
- Testes de disparo real de hooks executados nesta sessao

Confronto com codebase:
- Escopo analisado: path local completo, incluindo `internal/`, `cmd/`, `.agents/`, assets embarcados,
  testes, snapshots e scripts de sincronizacao.
- Status do confronto: realizado; 26 achados classificados, 25 `confirmado` e 1 `refutado`.
- Evidencias: internal/taskloop/bugfix.go:83 (loop de remediacao ja existente),
  internal/runtime/runner_autoreview.go:233 (veredito derivado de texto sintetico),
  internal/runtime/runner.go:216-228 (auto-review one-shot),
  .agents/lib/check-invocation-depth.sh:40 (teto de profundidade default 2),
  internal/metrics/metrics.go:214 (`ToolBudgetsLarge` com entrada unica),
  .agents/normalization-rules.yaml:20-21 (`inherit_common` com item unico),
  internal/skills/skills.go:13 (ordem canonica de tools),
  cmd/ai_spec_harness/cli_contract_test.go:181-184 (gate de CI acoplado ao agente removido),
  internal/uninstall/uninstall.go:124-137 (limpeza parcial do que internal/install/install.go:816-878 cria).
  Lista integral em `transcript.md`, secao `## Confronto com Codebase`.
- Riscos de compatibilidade: internal/metrics/metrics.go:214 (`ToolBudgetsLarge` com entrada unica)
  e .agents/normalization-rules.yaml:20-21 (`inherit_common` com item unico) ficam vazios ao remover
  o agente descontinuado, degradando comportamento sem que nenhum teste falhe. Duas ordens canonicas
  de tools coexistem (internal/skills/skills.go:13 vs cmd/ai_spec_harness/verify.go:154-155), e
  cmd/ai_spec_harness/cli_contract_test.go:181-184 falha o build se a string do agente removido
  desaparecer do schema.

## Escopo e Fora de Escopo
Inclui:
- Modelagem do `Catalogo de Agentes` como agregado dono do conjunto fechado de CLIs suportados.
- Modelagem do `Ciclo de Aprovacao` como agregado dono da regra de parada e do veredito.
- Traducao explicita entre a saida em linguagem natural do revisor e o veredito tipado.
- Mapa 1:1 entre criterio de aceite e evidencia verificavel como parte da definicao de aprovacao.
- Motivos canonicos de encerramento e deteccao de nao-convergencia.

Exclui:
- Como cada CLI e invocado tecnicamente (protocolo, flags, launchers) — detalhe de infraestrutura.
- Formato fisico dos artefatos de evidencia em disco.
- Politica de versionamento e release.
- Estrategia de migracao de usuarios do agente descontinuado.
- Modelagem do fluxo de planejamento (PRD, techspec, tasks), que ja existe e nao muda.

## Linguagem Ubiqua
| Termo | Definicao | Sinonimos proibidos/ambiguous | Observacoes |
| --- | --- | --- | --- |
| Ciclo de Aprovacao | Processo que conduz rodadas de revisao e correcao ate veredito aprovado ou parada explicita | "loop", "loop de review", "ciclo de remediacao" | Nomeado pela invariante que protege, nao pela mecanica |
| Rodada | Uma iteracao completa do Ciclo: revisar, e havendo achados, corrigir | "tentativa", "iteracao de bugfix" | Numerada a partir de 1; cada uma produz evidencia propria |
| Veredito | Resultado canonico de uma revisao | "status", "resultado", "ok/blocked" | Conjunto fechado; `APPROVED` e o unico que aprova |
| Achado | Defeito ou lacuna identificada pela revisao, com severidade | "comentario", "observacao", "remark" | "remark" e proibido: sugere algo que nao exige acao |
| Criterio de Aceite | Condicao declarada na tarefa que precisa ser comprovada | "requisito", "checklist" | Cada um exige uma Linha de Evidencia |
| Linha de Evidencia | Prova verificavel de que um criterio foi atendido | "justificativa", "explicacao" | So vale comando com saida, `arquivo:linha` do diff, ou teste nomeado |
| Motivo de Parada | Razao canonica pela qual o Ciclo encerrou | "erro", "falha" | Conjunto fechado; nao-convergencia nao e erro |
| Fingerprint | Identidade estavel do conjunto de achados de uma rodada | "hash do review" | Calculada sobre `{severidade, arquivo, regra}` normalizados |
| Politica de Aprovacao | Parametros que governam o Ciclo | "config", "settings" | Imutavel; injetada, nao lida de dentro do agregado |
| Agente | Um CLI de IA suportado pelo harness | "tool", "ferramenta", "IDE" | Value Object; identidade + spec + mecanismo de enforcement + janela |
| Catalogo de Agentes | Agregado dono do conjunto fechado de Agentes | "lista de tools", "registry" | Fonte unica; adicionar ou remover agente e uma mudanca aqui |
| Mecanismo de Enforcement | Como um Agente especifico nega uma acao proibida | "hook" | Difere por CLI; o efeito e que precisa ser identico |
| Pre-condicao de Enforcement | Estado externo necessario para o gate de um Agente funcionar | — | Ex.: pasta confiavel, hash de hook confiado, ausencia de interruptor |

## Bounded Contexts e Fronteiras
Contextos:
- Contexto: Catalogo de Agentes
  Objetivo: declarar quais CLIs existem, como cada um e invocado e como cada um aplica governanca.
  Ownership: o conjunto fechado de Agentes, suas specs, seus mecanismos de enforcement e as
  pre-condicoes de cada mecanismo.
  Fronteiras: filesystem (deteccao de sinais), binarios externos (verificacao de pre-condicoes,
  apenas sob comando de diagnostico explicito), arquivos de configuracao de terceiros.

- Contexto: Ciclo de Aprovacao
  Objetivo: conduzir rodadas ate aprovacao comprovada ou parada explicita, e ser dono da regra de parada.
  Ownership: Rodada, Veredito, Achado, Criterio de Aceite, Linha de Evidencia, Fingerprint,
  Motivo de Parada e a Politica de Aprovacao aplicada.
  Fronteiras: o texto em linguagem natural produzido pelo revisor (entrada mais sensivel), o
  repositorio git (diff por rodada) e o armazenamento de evidencia (saida).

Mapa de contexto:
- `Ciclo de Aprovacao` e **downstream** de `Catalogo de Agentes`: o Ciclo recebe o Agente que vai
  conduzir cada rodada, mas nao conhece nada sobre protocolo, flags ou hooks. A relacao e
  **conformist** em uma unica direcao — o Ciclo aceita a identidade do Agente como dado opaco.
- Nenhum dos dois contextos depende do outro para definir suas invariantes. O comportamento do Ciclo
  e identico qualquer que seja o Agente; e essa independencia que torna a paridade verificavel.
- A traducao entre o texto do revisor e o `Veredito` e uma **camada anticorrupcao** explicita: o
  vocabulario do modelo de linguagem nunca entra no dominio sem passar por ela.

## Workflow Principal
Gatilho:
Uma tarefa teve sua implementacao concluida e e submetida para aprovacao.
O Ciclo conduz rodadas ate Veredito APPROVED com mapa 1:1 completo, ou para com motivo canonico.

Passos:
1. O Ciclo e aberto com a Politica de Aprovacao vigente e o Agente designado, no estado `EmRevisao`,
   com a Rodada 1.
2. A Rodada revisa o alvo. Na Rodada 1 o alvo e o diff completo da tarefa; a partir da Rodada 2 e
   apenas o delta produzido pela correcao anterior.
3. O texto produzido pelo revisor e traduzido para um `Veredito` e um conjunto de `Achados`. Se o
   texto nao declarar um veredito canonico, o resultado e `BLOCKED` — nunca aprovacao por ausencia
   de marcadores.
4. Cada Criterio de Aceite da tarefa e confrontado, produzindo o mapa 1:1. Criterio sem Linha de
   Evidencia valida conta como nao atendido.
5. Se o Veredito for `APPROVED` e o mapa 1:1 estiver completo, o Ciclo encerra como `Aprovado` com
   motivo `aprovado`.
6. Caso contrario, calcula-se a Fingerprint dos Achados. Se ela repetir a da rodada anterior, o Ciclo
   encerra com motivo `nao_convergiu`.
7. Se ainda ha rodadas disponiveis, os Achados sao convertidos para o formato canonico de bugs e a
   correcao e executada em sessao nova.
8. Se a correcao nao produzir mudanca alguma, o Ciclo encerra com motivo `sem_mudanca`.
9. Registra-se o ponto de corte do repositorio, incrementa-se a Rodada e volta-se ao passo 2.
10. Esgotadas as rodadas sem aprovacao, o Ciclo encerra com motivo `limite_de_rodadas`.

Ponto de decisao:
- O passo 5 e o unico caminho para "aprovado", e exige **duas** condicoes simultaneas: veredito
  `APPROVED` e mapa 1:1 completo. Nenhuma das duas sozinha basta.
- Os passos 6 e 8 sao checagens baratas colocadas deliberadamente **antes** da correcao cara.

## Comandos
- Comando: SubmeterParaAprovacao
  Intencao: obter uma decisao de aprovacao comprovada sobre o trabalho de uma tarefa.
  Pre-condicoes: existe tarefa ativa com criterios de aceite declarados; existe alvo de revisao
  (diff nao vazio); existe Politica de Aprovacao valida; existe Agente designado.
  Resultado esperado: um Ciclo encerrado, com Motivo de Parada canonico e historico completo de Rodadas.
  Falhas de negocio: tarefa sem criterios declarados; alvo de revisao inexistente; politica invalida
  (teto de rodadas menor que 1).

- Comando: RegistrarAgente
  Intencao: declarar um CLI como suportado pelo harness.
  Pre-condicoes: identidade nao pertence ao conjunto; spec de invocacao declarada; mecanismo de
  enforcement declarado para os tres pontos canonicos; janela de contexto declarada ou derivavel.
  Resultado esperado: Catalogo com o novo Agente e paridade preservada.
  Falhas de negocio: identidade duplicada; cobertura incompleta dos pontos canonicos.

- Comando: RemoverAgente
  Intencao: descontinuar um CLI.
  Pre-condicoes: identidade pertence ao conjunto; o Catalogo nao fica vazio.
  Resultado esperado: Catalogo sem o Agente; invocacao por aquela identidade passa a produzir erro
  dirigido de migracao.
  Falhas de negocio: remover o ultimo Agente; remover agente inexistente.

## Eventos de Dominio
- Evento: RodadaIniciada
  Quando ocorre: ao abrir cada Rodada.
  Quem observa: a persistencia de evidencia e a telemetria opt-in.
  Impacto: cria o espaco de evidencia daquela rodada e registra o ponto de corte do repositorio.

- Evento: RodadaConcluida
  Quando ocorre: apos a traducao do texto do revisor e o confronto de criterios.
  Quem observa: persistencia, telemetria e o proprio Ciclo, que decide a transicao.
  Impacto: carrega Veredito, contagem de Achados por severidade, Fingerprint e o mapa 1:1.
  E o unico insumo capaz de explicar por que um Ciclo nao convergiu.

- Evento: CicloEncerrado
  Quando ocorre: em qualquer transicao terminal.
  Quem observa: o orquestrador de tarefas, os gates de encerramento de sessao e o relatorio.
  Impacto: carrega o Motivo de Parada canonico e determina se a tarefa pode ser marcada como concluida.

## Regras, Politicas e Invariantes
Regras de negocio:
- Regra: apenas o Veredito `APPROVED` aprova.
  Motivo: `APPROVED_WITH_REMARKS` significa que existem achados conhecidos; fechar com eles e aceitar
  debito sem decisao humana. Foi o caminho pelo qual o falso positivo entrou.

- Regra: todo Criterio de Aceite exige uma Linha de Evidencia valida.
  Motivo: afirmar cumprimento sem prova e a definicao operacional de falso positivo.

- Regra: Criterio marcado como nao verificavel proibe a aprovacao.
  Motivo: registra-lo apenas como risco transfere ao futuro uma decisao que e do revisor agora.

- Regra: o texto do revisor sem veredito canonico declarado resulta em `BLOCKED`.
  Motivo: inferir aprovacao pela ausencia de marcadores negativos e presumir sucesso; a postura
  dominante do modelo e fail-closed.

- Regra: nao-convergencia e resultado, nao falha.
  Motivo: falha convida retry; nao-convergencia exige decisao humana. Tratar as duas igual faz o
  sistema reprocessar o que ja provou nao convergir.

- Regra: cada Rodada executa em sessao nova.
  Motivo: contexto acumulado atraves de rodadas e o vetor direto de aprovacao alucinada.

- Regra: um Agente so integra o Catalogo se cobrir os tres pontos canonicos de enforcement.
  Motivo: paridade parcial e indistinguivel de paridade, e mente na matriz.

- Regra: um Mecanismo de Enforcement cuja Pre-condicao nao esta satisfeita nao conta como gate ativo.
  Motivo: um gate inerte e indistinguivel de um gate que aprovou.

Politicas:
- Politica: Politica de Aprovacao
  Entradas: teto de rodadas, criterio de aprovacao, regras de aborto antecipado.
  Saida/decisao: dado o estado do Ciclo e o resultado da Rodada, decide continuar ou encerrar, e com
  qual motivo.

- Politica: Traducao de Veredito
  Entradas: texto em linguagem natural do revisor.
  Saida/decisao: `Veredito` tipado + conjunto de `Achados`, ou recusa explicita.

- Politica: Deteccao de Nao-Convergencia
  Entradas: Fingerprint da rodada corrente e da anterior; existencia de mudanca apos a correcao.
  Saida/decisao: continuar, ou encerrar com `nao_convergiu` ou `sem_mudanca`.

Invariantes:
- Invariante: um Ciclo `Aprovado` sempre possui Veredito `APPROVED` e mapa 1:1 completo.
  Como impedir estado ilegal: o estado `Aprovado` so e alcancavel por uma transicao que recebe ambos
  como argumentos obrigatorios; nao existe construtor publico que produza `Aprovado` sem eles.

- Invariante: todo Ciclo encerrado possui exatamente um Motivo de Parada canonico.
  Como impedir estado ilegal: Motivo e Value Object de conjunto fechado, sem zero-value valido; o
  estado terminal exige um.

- Invariante: o numero de Rodadas nunca excede o teto da Politica.
  Como impedir estado ilegal: o agregado e dono da iteracao; nao ha comando externo capaz de abrir
  uma Rodada.

- Invariante: o Catalogo de Agentes nunca fica vazio.
  Como impedir estado ilegal: `RemoverAgente` recusa a remocao do ultimo.

- Invariante: identidade de Agente e unica no Catalogo.
  Como impedir estado ilegal: `RegistrarAgente` recusa duplicata.

- Invariante: evidencia de Rodada e imutavel apos escrita.
  Como impedir estado ilegal: escrita append-only, endereco derivado do numero da Rodada; nunca
  reutilizar o mesmo endereco entre rodadas.

## Estados e Transicoes
Estado inicial:
`EmRevisao` (Rodada 1)

Estados validos:
- `EmRevisao` — a Rodada corrente esta sendo revisada.
- `EmCorrecao` — houve achados e a correcao da Rodada corrente esta em curso.
- `Aprovado` — terminal.
- `Bloqueado` — terminal.

Transicoes permitidas:
- `EmRevisao` -> `Aprovado`: Veredito `APPROVED` **e** mapa 1:1 completo. Motivo `aprovado`.
- `EmRevisao` -> `Bloqueado`: Veredito `BLOCKED`, ou Fingerprint repetida, ou teto de rodadas
  atingido. Motivos `entrada_bloqueada`, `nao_convergiu`, `limite_de_rodadas`.
- `EmRevisao` -> `EmCorrecao`: ha achados acionaveis e ainda ha rodadas disponiveis.
- `EmCorrecao` -> `EmRevisao`: a correcao produziu mudanca; a Rodada e incrementada.
- `EmCorrecao` -> `Bloqueado`: a correcao nao produziu mudanca. Motivo `sem_mudanca`.

Transicoes proibidas:
- `EmRevisao` -> `Aprovado` com Veredito `APPROVED_WITH_REMARKS`: existem achados conhecidos.
- `EmRevisao` -> `Aprovado` com mapa 1:1 incompleto: aprovacao sem prova.
- `Aprovado` -> qualquer estado: terminal, e o selo de aprovacao nao se reabre.
- `Bloqueado` -> `EmRevisao`: retomar exige um novo Ciclo, para que o historico de parada permaneca intacto.
- `EmCorrecao` -> `Aprovado`: aprovacao so pode vir de uma revisao, nunca de uma correcao.

## Tipos Conceituais
Entidades:
- Ciclo de Aprovacao: identidade e a tarefa submetida; responsavel por conduzir rodadas, aplicar a
  politica e ser dono da decisao de parada.
- Rodada: identidade e o par (Ciclo, numero); responsavel por registrar alvo revisado, veredito,
  achados, fingerprint e evidencia produzida.

Value Objects:
- Veredito: conjunto fechado de quatro valores; construtor validante; zero-value invalido.
- MotivoDeParada: conjunto fechado — `aprovado`, `limite_de_rodadas`, `nao_convergiu`, `sem_mudanca`,
  `entrada_bloqueada`.
- PoliticaDeAprovacao: teto de rodadas, criterio de aprovacao e regras de aborto; imutavel.
- Fingerprint: identidade estavel do conjunto de achados; derivada de `{severidade, arquivo, regra}`
  normalizados e ordenados; ignora numero de linha, identificador atribuido pelo agente e texto livre.
- Achado: severidade, localizacao e impacto.
- LinhaDeEvidencia: uma das tres formas validas; qualquer outra forma e invalida por construcao.
- MapaDeCriterios: associacao total entre Criterios de Aceite e Linhas de Evidencia; "total" e a
  invariante — nao existe mapa parcial valido, apenas mapa incompleto, que reprova.
- Agente: identidade, spec de invocacao, mecanismo de enforcement por ponto canonico e janela de contexto.
- PontoCanonico: conjunto fechado — pre-ferramenta, pos-ferramenta, encerramento de sessao.
- PreCondicaoDeEnforcement: o que precisa ser verdade no ambiente para o gate daquele Agente funcionar.

Agregados:
- Ciclo de Aprovacao: consistencia forte sobre suas Rodadas; dono exclusivo da regra de parada.
  Limite: nao conhece Agente alem da identidade, nao conhece protocolo, nao escreve em disco.
- Catalogo de Agentes: consistencia forte sobre o conjunto fechado de Agentes.
  Limite: nao conhece Ciclo, nao executa binarios, nao decide aprovacao.

## Erros de Dominio
- Erro: TarefaSemCriterios
  Quando ocorre: a tarefa submetida nao declara criterios de aceite.
  Impacto no fluxo: o Ciclo nao abre.
  Acao esperada: devolver ao humano; sem criterios nao existe o que comprovar.

- Erro: VereditoIndecifravel
  Quando ocorre: o texto do revisor nao declara veredito canonico.
  Impacto no fluxo: a Rodada resulta em `BLOCKED` e o Ciclo encerra com `entrada_bloqueada`.
  Acao esperada: devolver ao humano; nunca inferir aprovacao.

- Erro: PoliticaInvalida
  Quando ocorre: teto de rodadas menor que 1.
  Impacto no fluxo: o Ciclo nao abre.
  Acao esperada: corrigir a configuracao; falhar explicitamente em vez de aplicar um default silencioso.

- Erro: CatalogoVazio
  Quando ocorre: tentativa de remover o ultimo Agente.
  Impacto no fluxo: remocao recusada.
  Acao esperada: registrar outro Agente antes.

- Erro: AgenteDuplicado
  Quando ocorre: registro de identidade ja existente.
  Impacto no fluxo: registro recusado.
  Acao esperada: usar a identidade existente ou escolher outra.

- Erro: EnforcementIncompleto
  Quando ocorre: Agente nao cobre os tres pontos canonicos.
  Impacto no fluxo: registro recusado.
  Acao esperada: declarar o mecanismo faltante.

Nota: esgotar rodadas, nao convergir e nao produzir mudanca **nao** sao erros. Sao resultados
terminais legitimos, distinguidos por Motivo de Parada. Essa separacao existe para que o codigo de
retry, que trata erro, nunca reprocesse uma reprovacao legitima.

## Fronteiras Externas e Traducao
Entradas externas:
- Texto em linguagem natural produzido pelo agente revisor — a entrada mais sensivel do modelo.
- Estado do repositorio git — alvo de revisao e prova de que a correcao produziu mudanca.
- Configuracao em camadas — origem da Politica de Aprovacao.
- Ambiente de execucao — origem das Pre-condicoes de Enforcement.

Saidas externas:
- Evidencia por Rodada, imutavel e numerada.
- Sumario consolidado do Ciclo no relatorio de execucao.
- Eventos observaveis para telemetria opt-in.
- Estado terminal consumido pelos gates de encerramento de sessao dos quatro Agentes.

Traducao entre dominio e contrato externo:
- Camada anticorrupcao obrigatoria entre o texto do revisor e o `Veredito`: o vocabulario do modelo
  de linguagem nao entra no dominio sem traducao validada. A traducao e **fail-closed** — recusa
  explicita em vez de default permissivo.
- A severidade de quatro niveis usada na revisao e traduzida para os tres niveis do formato canonico
  de bugs ao alimentar a correcao, preservando o nivel original no campo de impacto.
- A identidade do Agente atravessa a fronteira como dado opaco; nenhuma caracteristica de CLI
  (protocolo, flag, formato de hook) entra no `Ciclo de Aprovacao`.

## Persistencia, Consistencia e Auditoria
Persistencia necessaria:
- Evidencia de cada Rodada: alvo revisado, veredito, achados por severidade, mapa 1:1 e fingerprint.
- Sumario do Ciclo: rodadas executadas, veredito de cada uma e Motivo de Parada.
- Nao e necessario persistir o estado intermediario do Ciclo — retomada apos crash esta fora de escopo.

Consistencia requerida:
- Forte e em memoria durante a execucao do Ciclo, para rodada corrente, conjunto de fingerprints
  vistas e ponto de corte anterior.
- A evidencia e append-only e imutavel: o endereco de escrita deriva do numero da Rodada, de modo que
  uma rodada nunca sobrescreve outra.

Auditoria/rastreabilidade:
- Cadeia completa requisito funcional -> tarefa -> criterio de aceite -> linha de evidencia.
- Para cada Ciclo: por que parou, em que rodada, com quais achados e sobre qual alvo.
- Um Ciclo que nao convergiu precisa ser tao auditavel quanto um aprovado — e o caso operacionalmente
  mais dificil, e o unico insumo para diagnostica-lo e o historico por rodada.

## Observabilidade e Operacao
Sinais minimos:
- Metricas: rodadas por Ciclo; distribuicao de Motivos de Parada; taxa de aprovacao na Rodada 1;
  criterios reprovados por falta de evidencia.
- Logs: por Rodada — numero, veredito, contagem por severidade, fingerprint, decisao tomada.
- Alertas: crescimento de `nao_convergiu` ou `limite_de_rodadas` indica revisor mal calibrado ou
  criterios de aceite mal escritos — nao necessariamente codigo ruim.

Falhas operacionais relevantes:
- Revisor produzindo texto sem veredito declarado: o Ciclo bloqueia corretamente, mas em volume
  significa que o contrato de saida do revisor regrediu.
- Pre-condicao de Enforcement nao satisfeita: gate inerte. Precisa ser visivel como estado, nunca
  como sucesso silencioso.
- Interruptor de ambiente desligando plugins ou skills de um Agente: a sessao deve ser recusada.

Rollback/contingencia:
- O Ciclo nao altera codigo por conta propria: sua saida e uma decisao. Reverter significa descartar
  o resultado e reabrir um novo Ciclo.
- Reduzir o rigor e uma decisao de configuracao explicita (teto de rodadas), nunca um escape silencioso.

## Economia, Eficiencia e Custos
Decisoes para reduzir custo:
- Checagens baratas antes das caras: fingerprint repetida e ausencia de mudanca abortam antes de
  gastar uma sessao de revisao.
- Revisao incremental: a partir da Rodada 2 revisa-se apenas o delta.
- Reuso do loop ja existente em vez de uma segunda implementacao.
- Reuso do padrao de Value Object com construtor validante ja presente no codebase.

Custo cognitivo:
- Evitou-se um terceiro bounded context so para evidencia, que nao carrega decisao de negocio propria.
- Evitou-se interface de estrategia para a politica: nao ha fronteira consumidora real hoje.
- Evitou-se persistencia de estado intermediario e sua semantica de retomada.

Drivers de custo residual:
- O pior caso continua sendo cinco rodadas completas, cada uma com revisao e correcao.
- O mapa 1:1 obriga o revisor a percorrer todos os criterios mesmo quando o diff e pequeno.
- Sessao nova por rodada paga custo de contexto inicial repetido — preco deliberado do isolamento.

## Trade-offs e Decisoes
Alternativas rejeitadas:
- Nomear o fluxo como "loop" ou "ciclo de remediacao": vocabulario de mecanismo; "remediacao" mente
  no caso de aprovacao na primeira rodada.
- Contexto unico abarcando agentes e aprovacao: reproduz o acoplamento que hoje faz remover um agente
  custar ~215 arquivos.
- Terceiro contexto so para evidencia: fronteira que cobra traducao sem decisao de negocio propria.
- Aceitar `APPROVED_WITH_REMARKS` sem tag critica: e exatamente o caminho do falso positivo atual.
- Fingerprint sobre o texto integral dos achados: falso negativo garantido.
- Fingerprint sobre contagem por severidade: falso positivo que abortaria ciclos em progresso.
- Politica como interface substituivel: indirecao sem fronteira consumidora real.
- Comando por rodada controlado de fora: devolve a regra de parada para fora do dominio — e como o
  fluxo falha hoje.
- Estado do Ciclo persistido por rodada: custo de versionamento e corrupcao sem requisito que peca.
- Orcamento de tokens como motivo de parada: introduz parada nao-deterministica.
- Agregado morando em `internal/taskloop` ou `internal/runtime`: acopla dominio a orquestracao ou a
  protocolo, e quebra a paridade entre os dois modos de execucao.

Trade-offs aceitos:
- Mais resultados `Bloqueado` devolvidos ao humano, em troca de zero aprovacao sem prova.
- Crash durante um Ciclo perde o progresso e ele recomeca, em troca de nao versionar estado em disco.
- Sessao nova por rodada custa mais tokens de contexto inicial, em troca de eliminar o vetor de
  aprovacao alucinada por contexto acumulado.
- Um pacote de dominio novo, em troca de uma unica implementacao compartilhada pelas duas camadas.

Decisoes consolidadas:
- Termo canonico: `Ciclo de Aprovacao`.
- Dois bounded contexts, com evidencia como saida do Ciclo.
- Restricao dominante: correcao fail-closed.
- Agregado `Catalogo de Agentes`, com Agente como Value Object imutavel.
- Eventos: `RodadaIniciada`, `RodadaConcluida`, `CicloEncerrado`.
- Estado ilegal alvo: Ciclo encerrado com sucesso sem veredito e sem mapa 1:1.
- Nao-convergencia como transicao terminal explicita, nao erro.
- Comando unico `SubmeterParaAprovacao`; o agregado e dono das rodadas.
- `Veredito` como Value Object com construtor validante.
- Resultado tipado + sentinelas para falha de infraestrutura.
- Fingerprint sobre `{severidade, arquivo, regra}` normalizados.
- `PoliticaDeAprovacao` como Value Object injetado.
- Agregado em pacote de dominio proprio, sem dependencia de ACP, CLI ou filesystem.
- Fronteira mais sensivel: o texto do revisor, com camada anticorrupcao fail-closed.
- Consistencia forte em memoria; evidencia append-only imutavel por rodada.
- Postura de custo: abortar cedo e barato.

## Itens em Aberto
Nenhum item aberto bloqueante.

Duas verificacoes empiricas de infraestrutura permanecem em andamento e **nao afetam este modelo**,
porque dizem respeito ao contexto `Catalogo de Agentes` no nivel de mecanismo, nao de regra: (a) se o
bloqueio por excecao do agente OpenCode se comporta igual sob o runtime ACP; (b) quais interruptores
de ambiente desligam plugins de projeto. Ambas afetam a implementacao do Mecanismo de Enforcement e
das Pre-condicoes de Enforcement daquele Agente — conceitos que o modelo ja declara — sem alterar
invariante, estado, comando ou evento.

## Proximo Passo Recomendado
technical-discovery-production com o objetivo de traduzir este modelo em especificacao tecnica: desenho
do pacote de dominio, contratos Go dos Value Objects e do agregado, plano de migracao do loop existente
em `internal/taskloop/bugfix.go` para o novo agregado, e estrategia de teste que trave as invariantes
de transicao proibida.
