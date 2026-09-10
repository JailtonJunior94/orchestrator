<!-- spec-hash-prd: 0a9ad37a14dece6109b909750abfb8ec3c61f4a66181934758687842764737ce -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica

**PRD consumido:** `.specs/prd-harness-quatro-clis-loop-aprovacao/prd.md` (spec-version 3, 63 RFs)
**Modelo de domínio:** `discoveries/domain-loop-de-aprovacao-e-catalogo-de-agentes-cli/`
**Decisão de padrão:** `pattern-decisions/ciclo-de-aprovacao-maquina-de-estados/`

## Resumo Executivo

A entrega consolida quatro CLIs oficiais (Claude Code, Codex, GitHub Copilot CLI, OpenCode), remove o
Gemini e substitui a revisão de rodada única por um Ciclo de Aprovação determinístico que só encerra com
prova. A estratégia técnica tem três eixos.

**Primeiro**, um pacote de domínio novo e isolado — `internal/approval` — passa a ser a autoridade única
sobre a regra de parada. Ele não importa ACP, CLI nem filesystem, e torna o falso positivo
*inconstruível por tipo*: o estado Aprovado só é alcançável mediante um valor `ProvaDeAprovacao`, cujo
único construtor exige simultaneamente veredito `APPROVED` e mapa 1:1 completo. Hoje o repositório tem
duas implementações divergentes do mesmo ciclo — um loop em `internal/taskloop/bugfix.go:83` e uma
revisão one-shot em `internal/runtime/runner.go:216-228` — e ambas passam a consumir o mesmo agregado.

**Segundo**, o conjunto de agentes vira um catálogo com Agente como Value Object imutável, de modo que
adicionar ou remover um CLI deixe de custar centenas de arquivos. O enforcement é de **efeito**
igualitário, não de implementação: cada CLI usa seu mecanismo nativo, todos delegando aos mesmos
scripts canônicos, e toda pré-condição de funcionamento (pasta confiável no Copilot, hash de hook
confiado no Codex, ausência de interruptor no OpenCode) passa a ser verificada e reportada.

**Terceiro**, quatro defeitos confirmados por evidência são corrigidos porque sem eles a entrega
descreveria comportamento que o código não teria — com destaque para
`internal/runtime/runner_autoreview.go:233`, que faz o parser de veredito ler texto sintético em vez da
saída real do revisor, tornando todo veredito de produção ficção.

A execução é sequenciada em seis fases por dependência dura. A ordem importa: o gate de contrato em
`cmd/ai_spec_harness/cli_contract_test.go:181-184` *exige* a presença da string `gemini` no schema do
CLI, então ele bloqueia a própria remoção e precisa ser o primeiro item alterado.

## Arquitetura do Sistema

### Visão Geral dos Componentes

**Componentes novos**

| Componente | Responsabilidade |
|---|---|
| `internal/approval` (pacote novo) | Agregado `Ciclo`, entidade `Rodada`, Value Objects de conjunto fechado, tabela de transições, tradução anticorrupção do texto do revisor e política de parada. Zero dependência de ACP, CLI ou filesystem |
| `internal/approval/portas.go` | Três interfaces declaradas no consumidor — `Revisor`, `Corretor`, `Repositorio` — que dão ao agregado o que ele não pode fazer sozinho |
| `internal/approval/mocks/` | Mocks gerados por `mockery.yml` para as três portas |
| Spec do OpenCode em `internal/runtime/specs/` | Runtime ACP por subcomando, com launcher de fallback e janela derivada do modelo |
| Plugin de governança do OpenCode | Hook de pré-ferramenta que bloqueia por exceção e sinaliza carga via sentinela |
| Gate de encerramento canônico | Script tool-neutro que bloqueia o fim da sessão sem veredito aprovado, registrado nos 4 CLIs |
| Gates de build novos | Não-vazio de estruturas críticas, ordem canônica única, sincronia dos catálogos ACP e paridade de hooks com disparo real |

**Componentes modificados**

| Componente | Mudança |
|---|---|
| `internal/taskloop/bugfix.go` | O loop é promovido: passa a delegar ao agregado, fechando as quatro lacunas (ressalvas realimentam, não-convergência, ausência de mudança, ponto de corte por rodada) |
| `internal/runtime/runner_autoreview.go` | Deixa de sintetizar o texto do revisor e de sobrescrever a evidência; passa a alimentar o agregado com a saída real |
| `internal/runtime/runner.go` | O ponto de chamada da revisão passa a conduzir o Ciclo |
| `internal/detect/`, `internal/install/`, `internal/uninstall/`, `internal/manifest/` | Catálogo de 4 agentes, rastreamento de arquivos instalados e limpeza fiel |
| `internal/config/`, `cmd/ai_spec_harness/task_loop.go` | Primeira exposição do teto de rodadas, com merge campo a campo |
| Skills `execute-task`, `review`, `bugfix`, `agent-governance` | Alinhadas ao comportamento do agregado, que é a fonte de verdade |

### Fluxo de dados

O orquestrador monta a política a partir da hierarquia de configuração e abre o Ciclo com a identidade
da tarefa, a identidade opaca do agente e os critérios de aceite. O agregado conduz as rodadas: pede ao
`Repositorio` o ponto de corte e o alvo, pede ao `Revisor` a revisão em sessão nova, traduz o texto para
veredito e achados de forma fail-closed, confronta os critérios, e decide. A decisão de aprovar exige a
prova; qualquer outro caminho consulta a política, que aplica as checagens baratas antes da correção
cara. Os eventos emitidos pelo agregado são consumidos pelo adaptador para escrever evidência numerada
por rodada e alimentar telemetria opt-in.

Nenhum tipo de git, ACP ou sistema de arquivos atravessa a fronteira do domínio: `AlvoDeRevisao`,
`PontoDeCorte` e `SaidaDoRevisor` são valores opacos.

## Design de Implementação

### Pacote de domínio `internal/approval`

Pacote sem dependência de ACP, CLI ou filesystem. Imports permitidos: `context`, `crypto/sha256`,
`encoding/hex`, `errors`, `fmt`, `iter`, `slices`, `cmp`, `strings`.

| Arquivo | Responsabilidade |
|---|---|
| `identidades.go` | `IdentidadeDaTarefa`, `IdentidadeDoAgente` (dado opaco vindo do Catálogo) |
| `veredito.go` | `Veredito` — conjunto fechado de 4, construtor validante, zero-value inválido |
| `motivo.go` | `MotivoDeParada` — conjunto fechado de 5 |
| `achado.go` | `Severidade` (`iota+1`), `Achado`, tradução 4→3 níveis para o schema de bugs |
| `evidencia.go` | `LinhaDeEvidencia` (3 formas), `CriterioDeAceite`, `MapaDeCriterios` |
| `prova.go` | `ProvaDeAprovacao` — o tipo que torna "Aprovado sem prova" inconstruível |
| `fingerprint.go` | `Fingerprint` + calculadora SHA-256 sobre conjunto ordenado |
| `politica.go` | `PoliticaDeAprovacao` imutável + Functional Options + `Decidir` |
| `estado.go` | `Estado` (`iota+1`) + `TabelaDeTransicoes` |
| `rodada.go` | Entidade `Rodada`, imutável após concluída |
| `tradutor.go` | Camada anticorrupção fail-closed: texto do revisor → veredito |
| `portas.go` | `Revisor`, `Corretor`, `Repositorio` + DTOs opacos |
| `eventos.go` / `resultado.go` | Eventos de domínio e resultado do comando |
| `erros.go` | Sentinelas (`errors.Is`) + tipos customizados (`errors.As`) |
| `ciclo.go` | Agregado `Ciclo` |

### Interfaces Chave

As três portas são declaradas **no consumidor** e devolvem valores opacos — nenhum tipo de git ou ACP
atravessa a fronteira:

```go
// Revisor devolve TEXTO BRUTO, nunca Veredito. Se o adaptador pudesse devolver
// Veredito, reintroduziria o defeito atual (runner_autoreview.go:233).
type Revisor interface {
    Revisar(ctx context.Context, pedido PedidoDeRevisao) (SaidaDoRevisor, error)
}

// Corretor devolve apenas error: "houve mudança" é responsabilidade do
// Repositorio — o corretor não é fonte confiável sobre si mesmo.
type Corretor interface {
    Corrigir(ctx context.Context, pedido PedidoDeCorrecao) error
}

type Repositorio interface {
    PontoDeCorte(ctx context.Context) (PontoDeCorte, error)
    AlvoCompleto(ctx context.Context) (AlvoDeRevisao, error)
    Delta(ctx context.Context, desde PontoDeCorte) (AlvoDeRevisao, error)
}
```

### A invariante central, garantida por tipo

O falso positivo deixa de ser condição a verificar e passa a ser estado inconstruível:

```go
// ProvaDeAprovacao é a ÚNICA chave que abre o estado Aprovado.
type ProvaDeAprovacao struct {
    veredito Veredito
    mapa     MapaDeCriterios
    valida   bool
}

// Exige as DUAS condições simultaneamente. Nenhuma sozinha basta.
func NewProvaDeAprovacao(veredito Veredito, mapa MapaDeCriterios) (ProvaDeAprovacao, error) {
    if !veredito.Aprova() {
        return ProvaDeAprovacao{}, fmt.Errorf("%w: veredito %q nao aprova",
            ErrProvaInsuficiente, veredito.String())
    }
    if !mapa.Completo() {
        return ProvaDeAprovacao{}, fmt.Errorf("%w: mapa 1:1 incompleto (%d/%d criterios)",
            ErrProvaInsuficiente, mapa.Associados(), mapa.Total())
    }
    return ProvaDeAprovacao{veredito: veredito, mapa: mapa, valida: true}, nil
}
```

`aprovar` só aceita `ProvaDeAprovacao` e recusa o zero-value. Não existe sobrecarga sem prova.
`Veredito.Aprova()` é verdadeiro **apenas** para `APPROVED` (RF-33), e `MapaDeCriterios.Completo()` é
falso quando qualquer critério está sem evidência **ou** declarado não verificável (RF-49).

### Tabela de transições

Ausência da tabela é proibição. As transições proibidas do modelo têm teste dedicado:

```go
var _transicoesPermitidas = map[Estado][]Estado{
    EstadoEmRevisao:  {EstadoAprovado, EstadoBloqueado, EstadoEmCorrecao},
    EstadoEmCorrecao: {EstadoEmRevisao, EstadoBloqueado},
    EstadoAprovado:   nil, // terminal: o selo não reabre
    EstadoBloqueado:  nil, // retomar exige novo Ciclo
}
```

### Fingerprint estável

SHA-256 sobre o conjunto **ordenado e deduplicado** de `{severidade, arquivo, regra}`. Ignora
deliberadamente número de linha e identificador atribuído pelo agente: hash do texto integral daria
falso negativo garantido; contagem por severidade daria falso positivo que abortaria ciclo em progresso.
Duas fingerprints não-calculadas **nunca** são iguais — ausência de achados não pode ser lida como
não-convergência.

### Ordem deliberada das checagens

`PoliticaDeAprovacao.Decidir` aplica as checagens baratas antes da correção cara (RF-37, O-05): veredito
bloqueado, fingerprint repetida, teto de rodadas. A aprovação **não** passa por ali — só a prova abre o
estado Aprovado.

## Sequenciamento de Desenvolvimento

### Restrição de ordem descoberta na análise

O `opencode` ainda não existe em nenhuma linha de Go. Nas células onde o Gemini é **ocupante único**
(`internal/metrics/metrics.go:214` e `inherit_common` nos dois arquivos de regras de normalização), o
OpenCode precisa entrar **antes** da remoção — caso contrário a estrutura fica vazia, o comportamento
muda e nenhum teste falha. Por isso a Fase 3 precede a Fase 4.

### Fases

| Fase | Escopo | Depende de |
|---|---|---|
| **F0 — Gates operáveis** | Tornar os gates capazes de rodar e de dizer a verdade: portabilidade do gate de referências de caminho para shell sem recursos de versão 4; notação de caminho planejado; desarme do gate de contrato que exige a string do agente a remover | — |
| **F1 — Domínio** | Pacote de domínio da aprovação completo, com mocks e testes de invariante. Nenhum consumidor ainda | F0 |
| **F2a — Mapa 1:1 (pré-requisito bloqueante)** | Seção de critérios no template de artefato de revisão, asserção nos validadores canônicos, rotina de revisão no validador em Go, restrição do escape legado, propagação aos espelhos | F0 |
| **F2b — Ciclo** | Promoção do loop existente; correção dos defeitos de veredito sintético e de sobrescrita de evidência; evidência numerada por rodada; reset de profundidade; revisão por delta | F1, F2a |
| **F2c — Critério estrito** | Cadeia de propagação do teto de rodadas e virada do critério de encerramento | F2b |
| **F3a — Catálogo** | Registro único de agentes, ordem canônica, sincronia dos catálogos, separação entre detecção e diagnóstico | F1 |
| **F3b — OpenCode** | Spec por subcomando, detecção, instalação de pegada mínima, janela derivada do modelo, entradas nas células de ocupante único | F3a |
| **F3c — Enforcement** | Plugin que bloqueia por exceção, permissões declarativas, sanitização de ambiente, handshake | F3b |
| **F5 — Hooks e paridade** | Gate de encerramento nos 4 agentes, correção da chave de evento, verificadores de pré-condição, matriz de paridade com disparo real | F3c |
| **F4 — Remoção do Gemini** | Etapas topológicas restantes, cada uma deixando o repositório verde; manifesto por arquivo e desinstalação fiel | F5 |
| **F6 — Fechamento** | Rastreabilidade ancorada por hash, release major, suíte de não-regressão dos agentes remanescentes | F2c, F4 |

**Sobre o posicionamento de F2a.** A ordenação anterior colocava o mapa de critérios junto dos demais
gates, ao final. Isso contradizia a própria seção de Riscos Conhecidos deste documento e a ADR-001, que
declaram o item **pré-requisito bloqueante** da fase do Ciclo: ligar o critério estrito de aprovação
antes de o mapa existir como dado converte o falso positivo atual em **falso negativo total** — todo
ciclo terminaria bloqueado. A fase foi reposicionada para eliminar a contradição.

**Sobre F0.** Não constava da versão anterior porque os dois defeitos que a motivam foram descobertos
depois, por verificação empírica: o gate de referências de caminho usa um recurso de shell indisponível
na versão instalada em ambientes de desenvolvimento, saindo com sucesso sem verificar nada; e a notação
de caminho planejado que este documento promete na seção de arquivos relevantes não tem implementação.
Sem F0, o requisito de manter o gate verde é vazio, porque o gate não roda.

### Ordem de build da Fase 4 (remoção)

Cada etapa é um commit que mantém verde `go build ./... && go vet ./... && go test ./... -count=1` mais
os gates de sincronia.

1. **Desarmar o gate auto-bloqueante.** `cmd/ai_spec_harness/cli_contract_test.go:181-184` exige a
   string do agente no schema — é a única aresta sem predecessora. O bloco literal é removido e o loop
   existente ganha guarda de vacuidade. A direção inversa (nenhum agente aposentado pode aparecer no
   schema) só é ligada na etapa 10.
2. **Unificar a ordem canônica.** `internal/skills/skills.go:13` vira fonte única; `verify.go:154-155`
   passa a consumi-la; `ParseTool` deriva da mesma lista, eliminando a terceira enumeração implícita.
3. **Gate de ordem canônica** (verde por construção após a etapa 2).
4. **Unificar os dois catálogos ACP** e substituir o fallback silencioso de `taskloop.go:938-940` por
   erro explícito. É a armadilha de regressão mais cara e precisa estar armada antes de qualquer
   remoção de entrada.
5. **Gate de sincronia dos catálogos.**
6. **Gates de não-vacuidade**, escritos com o Gemini ainda presente — ficam vermelhos exatamente se a
   remoção sem substituto ocorrer.
7. **Desfragilizar asserts posicionais e contagens fixas**, de modo que a etapa de remoção não precise
   tocar nesses arquivos.
8. **Gate de paridade de hooks** com as células ainda completas.
9. **Remover as folhas de teste primeiro.** Cinco arquivos de teste fora de `specs` consomem as
   constantes do agente removido; inverter a ordem quebra a compilação de quatro pacotes ao mesmo
   tempo e impede bissecção.
10. **Remover a camada de runtime/specs**, atômica com as seis linhas do schema do CLI e com a ligação
    da direção inversa do gate da etapa 1.
11. **Remover o valor do enum de agentes** — etapa grande e inevitavelmente atômica, guiada pelo
    compilador. Inclui a remoção completa da política de detecção opt-in.
12. **Decisões de conteúdo** que os gates da etapa 6 forçam a explicitar.
13. **Artefatos físicos e espelhos**, com ordem interna crítica: parametrizar os arrays dos scripts de
    sincronia, remover a entrada, e só então apagar os diretórios.
14. **CI e release**, removendo o empacotamento do diretório do agente.
15. **Histórico preservado**: a ADR do runtime removido recebe `Status: Substituída`; corpo intocado.

### Regeneração de snapshots

Nove golden files precisam ser regenerados com revisão manual do diff arquivo a arquivo. A regeneração
automática é um cheque em branco: nada distingue "mudou porque o agente saiu" de "mudou porque o
gerador quebrou".

### Catálogo de Agentes: um registro, tudo derivado

A enumeração dos agentes está hoje espalhada por **onze** pontos independentes: catálogo ACP do CLI,
catálogo ACP do taskloop, `specForTool` do instalador, `allEntries` da detecção, `resolveAgentSpec` do
servidor MCP, `mapIDEToSpec` do perfil, `ParseDriverID`, `ParseTool` mais `AllTools`, mapa de ADRs do
probe, mapas de orçamento e a ordem de exibição do `verify`. É a causa mecânica do custo de ~215
arquivos.

O desenho substitui os onze por um registro único, em que cada `Agente` é Value Object imutável
carregando identidade, spec, enforcement por ponto canônico, sinais de detecção, política de ambiente,
ADR e orçamentos. Todos os call sites passam a derivar dele. Adicionar ou remover agente passa a ser
editar o registro mais o arquivo de spec e o asset de hooks — e o gate de paridade falha se faltar
qualquer um dos três.

Dois tipos novos merecem destaque porque o modelo os exige e o código atual não os tem:

- `PontoCanonico` — conjunto fechado (pré-ferramenta, pós-ferramenta, encerramento). O construtor de
  `Enforcement` **recusa** cobertura incompleta, tornando a invariante do modelo verificável em
  compilação do registro em vez de em revisão humana.
- `PreCondicaoDeEnforcement` — conceito de primeira classe, com o remédio acionável embutido. É o que
  permite distinguir "gate ausente" de "gate presente porém inerte".

### ACP por flag e por subcomando, sem caso especial

`FixedArgs` já é prefixo posicional no argv, e o subcomando é apenas um `FixedArgs` sem hífen. A
invariante que faz isso funcionar é hoje **acidental**, então passa a ser travada: nenhum item de
`FixedArgs` pode aparecer depois de um argumento iniciado por hífen. Resultado:

| Agente | argv |
|---|---|
| Copilot (flag) | `copilot --acp` |
| OpenCode (subcomando) | `opencode acp --cwd /repo --log-level error` |
| OpenCode via fallback | `npx --yes opencode-ai@<versão> acp --cwd /repo` |

### Janela derivada do modelo

A spec ganha uma função opcional de resolução de janela. Quando ausente — caso dos três agentes
atuais — o comportamento é byte-idêntico ao de hoje, o que preserva a não-regressão. Para o OpenCode,
a janela vem de tabela versionada com casamento exato e depois por maior prefixo (cobrindo sufixos de
data em identificadores de modelo), e **qualquer** caminho não resolvido cai no fallback conservador.
Remover uma entrada da tabela é seguro por construção: o modelo cai no conservador, nunca em janela
maior.

### Enforcement do OpenCode

O plugin cobre os três pontos canônicos, mas **só o de pré-ferramenta bloqueia** — é o único que impede
algo ainda não feito. Três decisões de desenho que vêm direto da evidência:

- **Custo**: apenas ferramentas que mutam o repositório pagam o shell-out; um único `stat` por sessão
  resolve a existência do validador; e o resultado é memoizado por `(ferramenta, arquivos)`. O custo
  assintótico é de um shell-out por arquivo distinto tocado, não por chamada de ferramenta.
- **Timeout é negação**, não aprovação. Tratar timeout como aprovação seria o mesmo defeito do gate
  inerte.
- **Validador ausente** bifurca por modo: em execução orquestrada é falha fechada; em uso interativo
  avisa uma vez e segue, porque ali o usuário está no comando.

### Sanitização de ambiente e handshake

O harness não passa a flag de modo puro e remove do ambiente **do processo filho que ele mesmo cria**
os interruptores que desligam o gate — sem tocar no ambiente do usuário. O agente cujo `EnvPolicy` é
zero-value tem o ambiente herdado intacto, o que mantém os três agentes atuais sem regressão.

O handshake encaixa no único ponto correto do ciclo de vida da sessão: **depois** da negociação do
protocolo, quando o subprocesso já carregou plugins, e **antes** do primeiro prompt. Falhar ali mata o
processo sem que o modelo veja uma única palavra — é a diferença entre "sessão recusada" e "sessão sem
gate". O caminho do sentinela é único por sessão e removido antes do spawn, porque sentinela de
execução anterior faria o handshake passar com o plugin desligado.

### Verificação de pré-condições sem violar a regra de detecção

A regra permanece intacta porque as responsabilidades ficam separadas por estado e por flag:

| Pré-condição | Verificável sem executar binário? | `verify` default | Com opt-in explícito |
|---|---|---|---|
| Nenhuma | — | `current` | `current` |
| Pasta confiável (Copilot) | **sim** — lê arquivo | `current` / `inert` | idem |
| Sem interruptor (OpenCode) | **sim** — lê ambiente | `current` / `inert` | idem |
| Handshake (OpenCode) | não — exige sessão | `unknown` | `unknown` |
| Hash confiado (Codex) | **não** — exige RPC | `unknown` | executa RPC → `current` / `inert` |

Dois estados novos, ambos exigidos pelo modelo: **`inert`** (artefato existe e está atualizado, mas a
pré-condição não está satisfeita — conta como falha) e **`unknown`** (ausência de informação, não
sucesso). O arquivo de configuração do Copilot **não é JSON estrito** — traz comentários de linha — e
exige remoção de comentários fora de strings, sem expressão regular, porque uma regex sobre `//`
engoliria URLs dentro de valores.

## Abordagem de Testes

### Testes Unitários

O pacote de domínio é testado como caixa-branca, porque as transições proibidas só são exercitáveis com
acesso ao método de transição. Cobertura obrigatória:

- **A invariante central**: prova recusada com veredito de ressalvas; com mapa incompleto; com critério
  não verificável; com veredito zero-value. O zero-value da prova nunca é válido.
- **Toda transição proibida do modelo**, uma asserção por linha: correção não pode aprovar; aprovado é
  terminal; bloqueado não retoma; não há auto-transição.
- **Fingerprint**: estável sob reordenação, duplicata, mudança de número de linha e mudança de
  identificador atribuído pelo agente; instável sob mudança de severidade, arquivo ou regra. Duas
  fingerprints não-calculadas nunca são iguais.
- **Tradução fail-closed**: veredito com ressalvas não é lido como aprovado (prefixo comum — a ordem de
  teste importa); texto sem declaração canônica resulta em bloqueio; variantes em português e inglês.
- **Política**: teto menor que 1 recusado na construção; ordem das checagens baratas antes da cara,
  verificada pela ausência de chamada ao corretor.

Testes de fingerprint e de tradução recebem também fuzzing, seguindo o padrão já usado no repositório
para parsers e validadores.

### Testes de Integração

O projeto já tem fronteiras de IO críticas e já sofreu o caso em que testes unitários passaram e a
integração real falhou — a chave de evento inválida do Copilot é exatamente isso. Integração é,
portanto, justificada, e o repositório já tem a infraestrutura: um servidor ACP falso in-process que
elimina a dependência do CLI real.

Três camadas de prova de enforcement, com honestidade sobre o que cada uma cobre:

1. **Matriz obrigatória** (unitário, sempre roda): produto cartesiano agentes × pontos canônicos, com
   verificação de que o arquivo de registro escrito contém a chave nativa declarada **e** invoca o
   validador declarado. Nenhum agente pode ter cobertura menor que outro. É a camada que teria pego a
   chave de evento inválida do Copilot.
2. **Disparo simulado** (integração, roda no CI): executa o script instalado com o contrato de entrada
   documentado de cada CLI e afirma bloqueio real — entrada que viola governança produz saída não-zero.
   Não exige CLI instalado. É a diferença entre "o arquivo existe" e "o arquivo, executado, nega".
3. **Disparo verdadeiro pelo CLI** (nightly, fora do gate de merge): exige os quatro binários
   instalados e autenticados. O repositório **já tem** esse padrão — build tag dedicada, alvo próprio no
   Makefile e workflow noturno separado. A camada 3 espelha o padrão existente e **não** é vendida como
   bloqueante de merge; o job noturno falha se qualquer célula for pulada, para não virar teste vazio.

### Testes E2E

Fluxo completo com o servidor ACP falso: ciclo que aprova na primeira rodada; ciclo que aprova na
terceira após duas correções; ciclo que esgota o teto; ciclo que aborta por fingerprint repetida sem
gastar a rodada seguinte; ciclo que aborta por ausência de mudança. Verificação de que as evidências de
rodadas distintas coexistem e de que reescrever endereço existente é erro.

## Monitoramento e Observabilidade

Os eventos do agregado alimentam a telemetria opt-in existente, sem novo mecanismo de consentimento:

- **Métricas**: rodadas por ciclo; distribuição de motivos de parada; taxa de aprovação na primeira
  rodada; critérios reprovados por falta de evidência; contagem de sessões recusadas por pré-condição.
- **Logs**: por rodada — número, veredito, contagem por severidade, fingerprint e decisão tomada. Por
  sessão — pré-condições avaliadas e seu estado real.
- **Alertas**: crescimento de não-convergência ou de esgotamento de teto indica revisor mal calibrado
  ou critérios de aceite mal escritos, não necessariamente código ruim. Crescimento de sessões recusadas
  por interruptor indica ambiente de CI mal configurado.

O bloqueio efetivo do OpenCode é observável no próprio fluxo do protocolo: a atualização da chamada de
ferramenta chega com estado de falha e o texto da exceção. Isso permite registrar evidência de gate
acionado sem depender de ler o log do plugin.

## Considerações Técnicas

### Decisões Chave

| # | Decisão | ADR |
|---|---|---|
| 1 | Ciclo de Aprovação como agregado em pacote de domínio isolado, com a aprovação garantida por tipo | `adr-001-ciclo-de-aprovacao-agregado.md` |
| 2 | Catálogo de Agentes como registro único, com Agente como Value Object e enforcement com cobertura validada | `adr-002-catalogo-de-agentes-registro-unico.md` |
| 3 | OpenCode via ACP por subcomando, com janela derivada do modelo | `adr-003-opencode-acp-subcomando.md` |
| 4 | Enforcement do OpenCode por exceção no hook de pré-ferramenta, com sanitização de ambiente e handshake ativo | `adr-004-enforcement-opencode-handshake.md` |
| 5 | Pré-condições de enforcement como conceito de primeira classe, com estados `inert` e `unknown` | `adr-005-precondicoes-de-enforcement.md` |
| 6 | Máquina de estados por tabela de transição, sem padrão formal | `pattern-decisions/ciclo-de-aprovacao-maquina-de-estados/` |

### Riscos Conhecidos

| Risco | Mitigação |
|---|---|
| **O mapa 1:1 não existe como dado.** O template de artefato de revisão não tem seção para ele e o validador não o cobra. Ligar o critério estrito sem isso transforma falso positivo em **falso negativo total** — todo ciclo terminaria bloqueado | Pré-requisito bloqueante da fase do Ciclo: seção no template, asserção no validador e propagação aos espelhos **antes** de ligar o critério estrito |
| **O caminho de produção não é o que parecia.** O loop existente só é alcançável por uma função sem chamador de produção; o caminho real usa revisão one-shot | Os três caminhos migram para o agregado. Sem isso a paridade declarada seria falsa |
| **Critérios de aceite não chegam ao caminho ACP.** A extração vive numa dependência do caminho legado | O campo de nome do arquivo de tarefa já existe no job e é o gancho natural para o plumbing |
| Regeneração de golden files pode congelar regressão junto | Revisão manual do diff arquivo a arquivo; a regeneração automática é um cheque em branco |
| Bug pré-existente na geração de governança emite a tabela de capacidades ignorando os agentes selecionados, e afirma que o Copilot não tem hooks nativos — hoje comprovadamente falso | Corrigido antes da regeneração, para que os golden files passem a refletir a verdade |
| Disparo real pelo CLI não é gate de merge | Explicitado; job noturno falha se houver célula pulada |
| Manifestos antigos no disco de usuários não rastreiam arquivos instalados | Campo aditivo com omissão, caindo em caminho conservador anunciado e migrando no próximo upgrade |
| A semântica de merge "não-zero vence" torna zero indistinguível de ausente | A validação de teto vive na camada de linha de comando, a única que sabe distinguir presença; valor negativo falha explicitamente em vez de ser normalizado em silêncio |

### Conformidade com Padrões

- **Governança transversal** (`.claude/rules/governance.md`, R-GOV-001): precedência respeitada; nenhuma
  aprovação sem evidência; sem ação destrutiva de git ou publicação remota sem pedido explícito.
- **Regras Go [HARD] R0–R7**: função de inicialização implícita proibida; toda função é método de
  struct, com as exceções exaustivas; enums começando em um, com o zero-value reservado; sentinelas e
  tipos de erro conforme o uso do chamador; sem panic em produção; capacidade de slices declarada;
  ordenação canônica dentro do arquivo; globais não exportados com prefixo; recursos modernos da versão
  declarada no módulo. Sem `interface{}`.
- **Interface apenas com fronteira consumidora real**: três portas no domínio, mais os pontos de
  extensão do cliente, que seguem precedente já existente no runner para evitar churn nos fakes.
- **Mocks** gerados pela configuração vigente; **testes** em suíte com tabela.
- **Segurança operacional**: detecção não executa binários; o harness não concede trust nem altera o
  ambiente do usuário; telemetria permanece opt-in.
- **ADR-023 (classe de janela)**: estendida, não violada — a janela estática permanece o caminho dos
  agentes atuais.

### Arquivos Relevantes e Dependentes

Caminhos ainda não existentes são marcados como `(planejado)` — notação que os distingue de referência a
caminho inexistente por erro, mantendo o gate de referências útil.

**Novos (planejados)**: pacote de domínio de aprovação com seus quinze arquivos e o diretório de mocks;
registro de agentes, tipos de enforcement e spec do OpenCode em `internal/runtime/specs/`; pacote de
pré-condições; pacote de handshake; plugin de governança do OpenCode; adaptadores em `internal/taskloop`
e `internal/runtime`; gates de paridade, de ordem canônica, de sincronia de catálogos e de cobertura de
orçamento.

**Modificados**: `internal/runtime/{runner.go, runner_autoreview.go, summary.go, types.go, options.go}`;
`internal/runtime/client/client.go`; `internal/runtime/specs/{spec.go, driver.go, claude.go, codex.go,
copilot.go}`; `internal/runtime/probe/probe.go`; `internal/taskloop/{taskloop.go, runloop.go, bugfix.go,
reviewer.go, acpinvoker.go, profile.go, acceptance.go, runtimeconfig.go}`; `internal/detect/agent.go`;
`internal/skills/skills.go`; `internal/install/install.go`; `internal/uninstall/uninstall.go`;
`internal/manifest/manifest.go`; `internal/metrics/{metrics.go, flow.go}`;
`internal/contextgen/contextgen.go`; `internal/evidence/evidence.go`; `internal/invocation/invocation.go`;
`internal/config/{runtime.go, resolver.go}`; `cmd/ai_spec_harness/{task_loop.go, verify.go, flags.go,
install.go, root.go, lint.go, scaffold.go, wrapper.go, cli_contract_test.go}`; `docs/cli-schema.json`;
`mockery.yml`; as skills `execute-task`, `review`, `bugfix`, `agent-governance` e seus espelhos; os
scripts de sincronia e o workflow de release.

**Removidos**: os quarenta e oito artefatos do agente descontinuado, listados no mapa de remoção.
