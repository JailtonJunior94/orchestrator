# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Harness Contract v1 em arquivo próprio com parse estrito
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório (JailtonJunior94)
- **Relacionados:** [`.specs/prd-harness-portatil-vendor-neutral/prd.md`](prd.md) (RF-02, RF-03, RF-04, RF-05, RF-06, RF-07, RF-08)

## Contexto

O harness hoje não tem contrato de política. A configuração existente é exclusivamente operacional:
`internal/config/runtime.go:9-26` declara 13 chaves (`tasks_root`, `timeout`, `max_retries`,
`concurrent`, `default_tool`, `durable_memory_enabled` e correlatas) e **zero** chaves de política —
não há onde declarar política de Git, de aprovação de operação destrutiva, de qualidade, de evidência
ou de descoberta de skills. RF-02 exige exatamente essas cinco famílias.

Além de não cobrir política, o pipeline atual de `config.yaml` é estruturalmente incompatível com o
requisito de fail-closed de RF-03:

1. **Campo desconhecido é silenciosamente ignorado.** `internal/config/resolver.go:88` chama
   `yaml.Unmarshal` sem `KnownFields(true)`. Um `auto_push: true` digitado como `auto_pussh: true`
   é aceito, descartado e o fluxo segue com a política inversa da pretendida. Para configuração
   operacional isso é tolerância deliberada; para política de segurança é falha silenciosa.
2. **A cascata de projeto é "primeiro encontrado vence", não merge.**
   `internal/config/resolver.go:104-112` percorre `.aispec/config.yaml` → `.claude/config.yaml` →
   `.agents/config.yaml` e retorna no primeiro acerto. Inserir um arquivo novo nessa lista **não**
   cria uma camada nova: cria competição dentro da mesma camada, onde a presença de um arquivo
   anterior anula por completo o posterior.
3. **A herança entre camadas é manual e frágil.** `internal/config/resolver.go:135-178` é um
   `mergeInto` com 14 `if` escritos à mão. Toda chave nova exige um `if` novo; o esquecimento não
   gera erro de compilação, apenas uma chave que nunca é herdada — a classe exata do BUG-08.
4. **Distinguir "ausente" de "zero-value" já custa código dedicado.**
   `internal/config/runtime.go:28-52` faz um segundo decode numa struct-sonda de ponteiros só para
   dois campos. Política é majoritariamente booleana (`auto_commit`, `auto_push`), então esse custo
   se multiplicaria por campo em vez de ficar contido.

Do outro lado, a infraestrutura para fazer isso certo já existe e está paga. O módulo
`github.com/santhosh-tekuri/jsonschema/v6 v6.0.2` já é dependência **direta** (`go.mod:7`), e o
repositório já tem um padrão canônico de validação por schema, aplicado duas vezes:
`internal/sdd/result_schema.go:40-46` e `internal/agents/schema.go:24-38` — `//go:embed` do schema,
compilação sob demanda via `sync.OnceValues`, validador stateless. `additionalProperties: false` já
é praticado em `internal/agents/agent-frontmatter.schema.json` e em
`internal/sdd/execution-result.schema.json:5`. RF-08 exige verificação estática pura (sem rede, sem
processo de provedor, sem LLM), que é precisamente o que esse padrão entrega.

Premissas assumidas: só existe `version: 1` (RF-04); nenhum projeto instalado pode passar a falhar
por causa desta entrega (RF-06, RF-07); política e operação não podem compartilhar nome de chave
(RF-05).

## Decisão

O **Harness Contract v1** vive em arquivo próprio, `.agents/harness.yaml` (planejado), separado do
`config.yaml` operacional, e é lido por um parser **estrito**.

**Escopo da decisão:**

- **Arquivo dedicado.** `.agents/harness.yaml` (planejado) carrega apenas política: Git, aprovação de
  operação destrutiva, qualidade, evidência e descoberta de skills (RF-02). Nenhuma das 13 chaves
  operacionais de `internal/config/runtime.go:9-26` aparece nele, e nenhuma chave de política migra
  para `config.yaml` (RF-05).
- **`config.yaml` permanece leniente e intocado.** A cascata de `internal/config/resolver.go:104-112`,
  o `mergeInto` de `internal/config/resolver.go:135-178` e a tolerância a campo desconhecido de
  `internal/config/resolver.go:88` continuam exatamente como estão (RF-06). Esta ADR não os altera;
  ela evita depender deles.
- **Parse estrito com erro tipado.** Campo desconhecido é erro tipado. Tipo inválido é erro tipado.
  Versão diferente de `1` é erro tipado que nomeia a versão encontrada, a suportada e a ação
  corretiva (RF-03).
- **Somente `version: 1`.** Nenhuma maquinaria de migração `vN → vN+1` é construída (RF-04). O campo
  `version` existe para que uma futura incompatibilidade seja detectável, não para ser traduzida.
- **Default embarcado quando o arquivo não existe.** Ausência de `.agents/harness.yaml` (planejado)
  aplica o contrato v1 default compilado no binário e o diagnóstico o reporta como `default`. Projeto
  instalado antes desta entrega continua funcionando sem ação do usuário — preservação F1 (RF-07).

**Como a decisão é aplicada.** Reusando o padrão já canônico no repositório: schema JSON embarcado
por `//go:embed`, compilado uma vez por `sync.OnceValues`, validador stateless — igual a
`internal/sdd/result_schema.go:40-46` e `internal/agents/schema.go:24-38` — com
`additionalProperties: false` em todo objeto do schema, como em
`internal/sdd/execution-result.schema.json:5`. A validação é dividida em duas camadas, seguindo o
precedente de `internal/sdd/result_schema.go:57-73`: o schema cobre forma, tipo e chave desconhecida;
invariantes semânticos que schema não expressa ficam em Go. O erro é formatado a partir de
`*jsonschema.ValidationError` no molde de `internal/skills/schema.go:100-121`.

**Armadilha crítica que a implementação deve evitar — a ponte YAML→JSON não pode projetar em struct.**
`internal/skills/schema.go:52-64` parseia o YAML, projeta o resultado num struct com tags `json:` e
só então serializa e valida. Esse caminho **descarta campo desconhecido antes do schema enxergá-lo**:
o struct não tem onde guardá-lo, o JSON gerado já sai limpo, e `additionalProperties: false` valida
o vácuo. Se `.agents/harness.yaml` (planejado) seguir esse caminho, RF-03 vira falso positivo —
o gate passa a reportar sucesso justamente no caso que deveria bloquear. A ponte obrigatória é
preservar todas as chaves: `yaml.Unmarshal` em `map[string]any` e normalizar as chaves para string
antes de serializar, já que YAML admite chave não-string e `encoding/json` não. A projeção em struct
tipado, quando necessária, acontece **depois** da validação, nunca antes.

**Partes impactadas:** um pacote novo de contrato (planejado); os fluxos de `install`, `verify` e
`task-loop`, que passam a consultar o contrato; a documentação de configuração. `internal/config/`
não é impactado.

## Alternativas Consideradas

### (a) Estender `config.yaml` com um bloco `harness:`

- **Descrição:** manter um único arquivo de configuração, acrescentando a política sob uma chave
  aninhada `harness:` em `.aispec/config.yaml`, `.claude/config.yaml` ou `.agents/config.yaml`.
- **Vantagens:** um arquivo a menos; nenhuma mudança na experiência de instalação; a cascata já
  existente seria reaproveitada sem código novo de descoberta.
- **Desvantagens:** exigiria um parser híbrido — leniente fora do bloco (para não quebrar RF-06) e
  estrito dentro dele (para cumprir RF-03) —, sobre um `yaml.Unmarshal` que hoje é global e sem
  `KnownFields` (`internal/config/resolver.go:88`). Herdaria também o "primeiro encontrado vence" de
  `internal/config/resolver.go:104-112`: um `.aispec/config.yaml` sem bloco `harness:` anularia por
  completo o `.agents/config.yaml` que o define, sem nenhum sinal. E o `mergeInto` manual de
  `internal/config/resolver.go:135-178` precisaria de um `if` por campo de política, reabrindo a
  classe do BUG-08 exatamente sobre chaves de segurança.
- **Motivo da rejeição:** um parser com duas disciplinas dentro do mesmo decode é origem estrutural
  de falso positivo (bloco ignorado por posição na cascata, reportado como válido) e de falso negativo
  (campo operacional legítimo rejeitado pelo modo estrito). Separar os arquivos torna a disciplina uma
  propriedade do arquivo, não um estado do parser.

### (b) Contrato exclusivamente embutido no binário, com overlay por allowlist

- **Descrição:** não existir arquivo de contrato no repositório; o contrato v1 viveria compilado, e o
  projeto só poderia sobrescrever um conjunto fechado de chaves via flag ou variável de ambiente.
- **Vantagens:** máximo fail-closed possível — nada que o usuário escreva pode afrouxar a política
  não prevista; zero superfície de parse; RF-07 satisfeito por construção.
- **Desvantagens:** configurabilidade insuficiente para RF-02, que exige política de Git, aprovação,
  qualidade, evidência e descoberta **declaradas pelo projeto**. Política declarada fora do repositório
  não é versionável, não é revisável em PR e não viaja com o projeto — o oposto do objetivo portátil
  e vendor-neutral do PRD.
- **Motivo da rejeição:** resolve o risco errado. O risco desta entrega é política silenciosamente
  ignorada, e isso é resolvido pelo parse estrito; suprimir a declaração explícita sacrifica o valor
  central do contrato para mitigar um risco que o schema já cobre.

### (c) Migrador automático `vN → vN+1` desde já

- **Descrição:** construir, junto com a v1, a maquinaria de detecção de versão e transformação
  incremental de contrato entre versões.
- **Vantagens:** a primeira mudança incompatível não exigiria trabalho de infraestrutura; o usuário
  nunca veria erro de versão.
- **Desvantagens:** só existe `version: 1`. Nenhum caminho de migração seria exercitado por teste
  real, apenas por fixture inventada; a abstração seria desenhada contra uma v2 hipotética cujo
  formato ninguém conhece, e o custo de mantê-la recairia sobre toda alteração do contrato v1.
- **Motivo da rejeição:** abstração sem caminho exercitável é código morto, contrariando diretamente
  RF-04 e o modo de trabalho do repositório (menor mudança segura, sem abstração sem demanda
  concreta). Erro tipado de versão incompatível é a resposta correta enquanto só houver uma versão.

## Consequências

### Benefícios Esperados

- Política de Git, aprovação, qualidade, evidência e descoberta passa a ser declarável e versionada
  no repositório, atendendo RF-02 sem tocar em `internal/config/`.
- Erro de digitação em chave de política vira erro tipado, não comportamento invertido silencioso —
  a falha classe `internal/config/resolver.go:88` deixa de alcançar decisões de segurança.
- A disciplina de parse vira propriedade do arquivo: `config.yaml` leniente, `harness.yaml` estrito.
  Não há modo, flag ou estado de parser que possa ser configurado errado.
- Custo de implementação baixo: `jsonschema/v6` já é dependência direta (`go.mod:7`) e o padrão de
  validação já tem duas implementações de referência no repositório
  (`internal/sdd/result_schema.go:40-46`, `internal/agents/schema.go:24-38`).
- RF-08 satisfeito por construção: validação por schema embarcado não faz rede, não executa binário
  de provedor e não chama LLM.
- Nenhum projeto existente quebra: sem o arquivo, aplica-se o default embarcado (RF-07); com
  `config.yaml` intacto, o comportamento operacional é idêntico (RF-06).

### Trade-offs e Custos

- Um arquivo de configuração a mais no repositório instalado, com a carga cognitiva de saber qual
  chave mora em qual arquivo. Mitigado por RF-05: nenhum nome de chave pode existir nos dois schemas,
  então a pergunta "onde mora esta chave" tem sempre uma resposta única.
- Duas disciplinas de parse coexistindo no produto exigem que a diferença seja documentada; um usuário
  que assume tolerância universal será surpreendido pelo primeiro erro estrito.
- Fail-closed converte erro de digitação em interrupção de fluxo. É o comportamento desejado, mas
  troca "resultado errado silencioso" por "parada ruidosa", e a mensagem de erro passa a ser parte do
  contrato de usabilidade.
- A ponte YAML→JSON via `map[string]any` custa uma passagem de normalização de chaves que a projeção
  em struct não teria — custo aceito deliberadamente, porque é ele que preserva o campo desconhecido.

### Riscos e Mitigações

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Implementação copiar o caminho de `internal/skills/schema.go:52-64` e projetar em struct antes de validar | **Crítico.** `additionalProperties: false` validaria o vácuo; RF-03 viraria falso positivo e o gate reportaria sucesso no caso que deveria bloquear | Teste obrigatório com um `.agents/harness.yaml` (planejado) contendo chave desconhecida, exigindo erro. Esse teste é o critério de aceite de RF-03, não um caso complementar |
| Chave YAML não-string (numérica, booleana, sequência) quebrar a serialização JSON | Erro de parse obscuro em vez de erro de contrato | Normalizar chaves para string na ponte; chave não normalizável produz erro tipado nomeando a chave |
| Nome de chave duplicado entre `config.yaml` e `harness.yaml` | Ambiguidade de fonte da verdade, violando RF-05 | Gate de build que compara os conjuntos de nomes dos dois schemas e falha na interseção |
| Default embarcado divergir do arquivo publicado no `install` | Projeto com e sem arquivo comportando-se diferente, quebrando a premissa de RF-07 | Uma única fonte: o asset embarcado é o mesmo bytes-a-bytes que o `install` escreve, coberto pelos gates de sincronia de assets já existentes |
| Adoção pesada demais por causa do fail-closed | Usuário evita declarar política | Mensagem de erro no molde de `internal/skills/schema.go:100-121`: nomear campo, valor e ação corretiva |

**Rollback:** como a ausência do arquivo aplica o default embarcado (RF-07) e `config.yaml` não é
alterado (RF-06), reverter significa remover `.agents/harness.yaml` (planejado) do projeto ou reverter
o pacote de contrato. Nenhum estado persistido precisa de desfazimento.

## Plano de Implementação

1. **Schema v1.** Definir o schema JSON do contrato cobrindo as cinco famílias de política de RF-02,
   com `version` obrigatório e constante `1`, e `additionalProperties: false` em **todo** objeto —
   molde de `internal/sdd/execution-result.schema.json:5`. Verificar contra as 13 chaves de
   `internal/config/runtime.go:9-26` que nenhum nome colide (RF-05).
2. **Pacote de contrato.** `//go:embed` do schema, compilação por `sync.OnceValues`, validador
   stateless — estrutura idêntica a `internal/sdd/result_schema.go:40-46` e
   `internal/agents/schema.go:24-38`.
3. **Ponte YAML→JSON preservadora.** `yaml.Unmarshal` em `map[string]any`, normalização recursiva de
   chaves para string, serialização e validação. **Não** reusar o caminho de
   `internal/skills/schema.go:52-64`.
4. **Erros tipados.** Campo desconhecido, tipo inválido e versão incompatível como tipos distintos,
   formatados a partir de `*jsonschema.ValidationError` no molde de
   `internal/skills/schema.go:100-121`. O erro de versão nomeia encontrada, suportada e ação.
5. **Camada semântica.** Invariantes que o schema não expressa, em Go, seguindo
   `internal/sdd/result_schema.go:57-73`.
6. **Default embarcado.** Asset v1 default no binário, aplicado quando o arquivo não existe e
   reportado como `default` no diagnóstico (RF-07).
7. **Gate de não-duplicação.** Comparação automatizada dos nomes de chave dos dois schemas (RF-05).
8. **Testes.** Contrato válido; chave desconhecida rejeitada (caso guardião de RF-03); tipo inválido
   rejeitado; `version: 2` rejeitado com erro nomeado; ausência do arquivo aplicando default;
   `config.yaml` com campo desconhecido continuando a passar (guardião de RF-06).

**Dependências:** `jsonschema/v6` (já satisfeita, `go.mod:7`); PRD aprovado; nenhuma dependência sobre
`internal/config/`.

**Sequência recomendada:** 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8, com os testes 8 escritos junto de cada
etapa correspondente.

**Adoção concluída quando:** os oito itens estiverem entregues, o teste de chave desconhecida falhar
sem a ponte preservadora, o gate de não-duplicação estiver em CI, e este repositório declarar o
próprio `.agents/harness.yaml` (planejado) em self-dogfooding.

## Monitoramento e Validação

- **Sinal primário (fail-closed real):** o teste de chave desconhecida deve falhar quando a ponte
  preservadora é removida. Um teste que passa nos dois modos está validando o vácuo e não prova nada.
- **Guardião de RF-06:** teste de regressão garantindo que `config.yaml` com campo desconhecido
  continua sendo aceito. Quebra desse teste significa que o modo estrito vazou para a configuração
  operacional.
- **Guardião de RF-07:** teste de projeto sem `.agents/harness.yaml` (planejado) exercitando o fluxo
  completo com o default embarcado.
- **Cobertura:** o pacote de contrato entra nos gates existentes (75% total, 70% por pacote crítico).
- **Diagnóstico:** `ai-spec verify` deve distinguir contrato `default` (embarcado) de contrato
  declarado pelo projeto; a origem é informação operacional, não detalhe interno.
- **Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`):** frequência de rejeição por categoria de erro.
  Predominância de "campo desconhecido" sobre schema estável sugere problema de nomenclatura ou de
  documentação, não de disciplina.

**Critérios de sucesso:** zero regressão em projeto instalado antes da entrega; toda configuração de
política inválida produzindo erro tipado acionável; nenhuma chave duplicada entre os dois schemas.

**Critérios para revisar ou reverter:** falso positivo confirmado (contrato inválido aceito) invalida
a premissa central e exige revisão imediata; falso negativo recorrente sobre contrato legítimo indica
schema restritivo demais e exige ajuste do schema, não relaxamento do parse.

## Impacto em Documentação e Operação

- [`docs/config-hierarchy.md`](../../docs/config-hierarchy.md): acrescentar a separação entre
  configuração operacional (leniente, cascata "primeiro encontrado vence") e contrato de política
  (estrito, arquivo único), deixando explícito que `.agents/harness.yaml` (planejado) **não** participa
  da cascata de `internal/config/resolver.go:104-112`.
- [`AGENTS.md`](../../AGENTS.md): registrar o contrato entre os artefatos de governança e seu papel
  como fonte da verdade sobre política.
- [`docs/troubleshooting.md`](../../docs/troubleshooting.md): entradas para os três erros tipados
  (campo desconhecido, tipo inválido, versão incompatível), cada uma com a ação corretiva.
- [`docs/guia-instalacao-universal.md`](../../docs/guia-instalacao-universal.md): comportamento do
  `install` quanto ao contrato e o que ocorre na ausência do arquivo.
- [`docs/degradation-matrix.md`](../../docs/degradation-matrix.md): contrato ausente é degradação
  esperada e nomeada (`default`), não falha.
- **Onboarding:** regra de decisão em uma linha — política vai para `harness.yaml`, operação vai para
  `config.yaml`, e nenhum nome de chave existe nos dois.
- **CI:** o gate de não-duplicação de chaves entra no conjunto de gates obrigatórios.

## Revisão Futura

**Revisar quando ocorrer o primeiro destes eventos:**

- Surgir a primeira demanda concreta de mudança incompatível no contrato. É o marco que reabre a
  decisão (c): o migrador só passa a ter caminho exercitável quando existir uma v2 real, e a política
  de evolução (o que é aditivo, o que é incompatível) deve ser registrada em ADR nova antes de
  qualquer código de migração.
- Uma chave precisar existir nos dois arquivos por demanda legítima. Isso invalida a premissa de
  RF-05 e exige reavaliar a separação, não apenas afrouxar o gate.
- `internal/config/` passar a usar `KnownFields(true)` na cascata operacional. Com as duas disciplinas
  convergindo, a alternativa (a) deixa de exigir parser híbrido e volta a ser candidata.
- Falso positivo confirmado na validação — contrato inválido aceito em produção. Revisão imediata,
  independentemente de prazo.

**Substituição:** esta ADR é substituída por uma ADR de política de evolução do contrato quando
`version: 2` for necessária, ou por uma ADR de unificação caso a configuração operacional adote parse
estrito.
