# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Rótulo:** MD-004
- **Título:** Ativação opt-in com paridade byte-a-byte e fim da degradação silenciosa na persistência de memória
- **Data:** 2026-09-10
- **Status:** Proposta
- **Decisores:** JailtonJunior94 (mantenedor do harness)
- **Relacionados:** [PRD](prd.md) RF-28, RF-29, RF-30, RF-33, RF-34 · [techspec](techspec.md) · ADR-016 do repositorio (cascata de configuracao; documentada em [docs/config-hierarchy.md](../../docs/config-hierarchy.md)) · MD-001, MD-005

## Contexto

O PRD subordina toda decisão à restrição de zero regressão: RF-29 exige que, com a feature desativada, o prompt final seja byte-idêntico ao produzido pela versão atual para os mesmos insumos. O comportamento a preservar está concentrado em três pontos de `internal/runtime/runner.go`: a linha 463 retorna store nulo quando o diretório de tasks está vazio, a linha 476 devolve o prompt original sem injeção nesse caso, e a linha 547 deixa de registrar o hook de persistência. O formato do bloco injetado é a concatenação literal montada entre as linhas 490 e 520, incluindo a diretiva de compactação da linha 516.

A cascata de configuração do repositório já resolve precedência de forma determinística — `internal/config/resolver.go:61` implementa as quatro camadas, e `internal/config/resolver.go:166` faz o merge campo a campo onde cada camada só sobrescreve valores não-zero. O levantamento identificou que **o ponto de merge é o mais fácil de esquecer**: não há reflexão nem teste genérico que force uma chave nova a aparecer ali, e esquecê-lo produz uma chave que nunca propaga entre camadas, com falha silenciosa.

Há um segundo achado, mais grave, que muda o escopo desta decisão. O despacho do ponto de fim de sessão em `internal/runtime/runner.go:428` descarta o erro retornado — `_ = disp.Dispatch(...)`. O mesmo ocorre no ponto de pós-revisão em `internal/runtime/runner.go:221`. Isso significa que, hoje, uma falha ao gravar memória é completamente invisível: não aborta, não avisa, não registra. Além disso, o hook de persistência ignora silenciosamente eventos cujo campo de resumo não seja do tipo esperado, conforme a asserção de tipo em `internal/runtime/hooks/memory_persist.go:57`.

RF-30 proíbe degradação silenciosa em termos diretos, e o `AGENTS.md` do repositório registra a lição de um defeito real: um gate que se desliga sozinho é indistinguível de um gate que aprovou. O comportamento atual da persistência de memória é exatamente esse padrão.

## Decisão

A ativação é opt-in por **flag de execução mais chave na cascata de configuração existente**, com zero-value preservando o comportamento atual. A flag vence a configuração para uma execução específica, conforme a precedência do ADR-016 do repositório.

A paridade é provada por **teste dedicado que compara o prompt final byte a byte** com a saída da implementação atual, cobrindo quatro casos: ausência total de memória, somente memória de workflow, workflow e task na ordem correta, e diretiva de compactação anexada. Esse teste é gate de merge, não verificação opcional. A suíte existente de memória, hooks e runner permanece verde **sem alteração de expectativa** — qualquer necessidade de ajustar um teste existente é sinal de regressão, não de teste obsoleto.

O descarte silencioso do erro de despacho é **corrigido no escopo desta feature**, porque RF-30 é inalcançável sem isso. A correção é conservadora e não altera controle de fluxo: o erro deixa de ser descartado e passa a ser registrado de forma explícita e a compor a evidência da sessão, sem abortar a sessão. Abortar seria mudança de comportamento não pedida pelo PRD e criaria uma regressão nova — uma falha de gravação de memória passaria a derrubar trabalho concluído.

**A ancoragem do teste de paridade é parte desta decisão.** O teste não ancora em `injectMemoryContext` nem no ponto de montagem de `internal/runtime/runner.go:490`: o wiring da porta substitui o call site de `internal/runtime/runner.go:149`, e um teste ancorado no seam interno continuaria verde enquanto o prompt real mudasse — falso positivo perfeito, e o pior possível, porque é o gate que sustenta toda a estratégia de risco. A ancoragem é um hook de captura em `hooks.PointPromptPostBuild`, cujo evento carrega ponteiro para o prompt final e é despachado em `internal/runtime/runner.go:290`. Mede-se o byte na fronteira externa, depois de toda montagem.

Consequência de sequenciamento: na tarefa que cria o teste ele é um retrato do comportamento atual; **o critério de aceite da tarefa de wiring é que esse arquivo passe sem nenhuma alteração de expectativa**. É essa regra que converte retrato em gate.

A asserção de tipo defensiva do hook permanece, mas deixa de ser silenciosa: um evento de tipo inesperado passa a registrar o descompasso, porque hoje um refactor do campo de resumo quebraria o hook sem erro de compilação e sem erro de execução.

A migração de formato ocorre **somente por comando explícito**, com backup verificável gravado antes de qualquer conversão, e recusa de migração já aplicada (RF-33). Nenhuma conversão acontece dentro de uma sessão de agente.

## Alternativas Consideradas

**Ativação por default, com flag de desligar.** Vantagens: entrega valor sem ação do usuário, evidência de campo acumula mais rápido. Desvantagens: maximiza a exposição a regressão em produção, contra a restrição dominante declarada no PRD e reafirmada na modelagem de domínio. Não escolhida — o PRD registra explicitamente que tornar-se default é decisão de uma major futura.

**Ativação apenas por variável de ambiente.** Vantagens: prática em integração contínua e em scripts. Desvantagens: fica fora da cascata de configuração do ADR-016 e invisível na ajuda do comando, o que torna a ativação difícil de auditar.

**Ativação apenas por chave de configuração.** Vantagens: persistente por projeto e versionável. Desvantagens: não oferece escape por execução — desativar exigiria editar arquivo, o que é ruim como plano de mitigação de incidente.

**Migração automática na primeira execução, com backup.** Vantagens: conveniência, é o modelo do projeto de referência. Desvantagens: altera estado dentro de uma sessão de agente e acopla o sucesso da tarefa ao sucesso da migração — uma falha de conversão passaria a se confundir com uma falha de implementação.

**Sem migração, lendo os dois formatos indefinidamente.** Vantagens: risco zero de conversão. Desvantagens: dois formatos coexistindo para sempre, com dobra de caminhos de leitura e de teste. Não escolhida, mas registrada porque tem mérito real: o conteúdo do formato atual é um template de quatro campos de valor durável quase nulo, o que torna a conversão quase trivial e a coexistência desnecessária.

**Manter o descarte do erro de despacho fora do escopo.** Vantagens: menor raio de alteração, mantém a feature contida. Desvantagens: RF-30 seria inalcançável, e a feature entregaria uma persistência de memória cuja falha permanece invisível — o defeito que ela existe para corrigir, em outra forma. Não escolhida.

**Abortar a sessão quando a persistência de memória falha.** Vantagens: garantia forte de que memória gravou. Desvantagens: uma falha de gravação passaria a derrubar trabalho já concluído, criando regressão nova e transformando a memória em ponto único de falha da sessão.

## Consequências

### Benefícios Esperados

- Qualquer regressão em produção é mitigável por flag, sem downgrade de versão.
- A paridade deixa de ser afirmação e passa a ser fato verificado por gate.
- Falha de gravação de memória deixa de ser invisível, o que é pré-requisito para confiar na feature em produção.
- Descompasso de tipo no hook deixa de ser falha silenciosa, protegendo refactors futuros do campo de resumo.
- A migração é uma decisão consciente e reversível, não um efeito colateral da primeira execução.

### Trade-offs e Custos

- O valor só chega a quem ativar. Adoção mais lenta é o preço da restrição de zero regressão.
- A feature altera dois pontos fora do seu escopo natural — os dois despachos que descartam erro. Isso amplia o raio de revisão, e é justificado por RF-30 ser inalcançável sem a alteração.
- O teste de paridade byte-a-byte precisa congelar o formato atual como referência, o que significa que qualquer mudança futura desse formato passa a ser mudança de contrato explícita.

### Riscos e Mitigações

- **Risco:** esquecer o ponto de merge da cascata, produzindo chave que nunca propaga. **Impacto:** a feature parece ativada na configuração e não ativa. **Mitigação:** teste que exercita a chave nova em cada camada da cascata, incluindo o caso de camada superior não definir a chave e não apagar a inferior.
- **Risco:** o teste de paridade congelar um formato que na verdade tem defeito, tornando o defeito contratual. **Impacto:** dívida perpetuada. **Mitigação:** o formato congelado é o do caminho desativado; o caminho ativado é livre para melhorar, e a paridade só vale para quem não ativou.
- **Risco:** registrar o erro de despacho gerar ruído em execuções normais. **Mitigação:** o registro é condicionado a erro real, não a ausência de hooks — o despacho em ponto sem hook registrado retorna nulo por construção, conforme `internal/runtime/hooks/dispatcher.go:131`.
- **Plano de rollback:** a flag desativa a feature inteira. As correções de degradação silenciosa permanecem, porque melhoram o comportamento independentemente da feature.

## Plano de Implementação

1. Adicionar a chave à estrutura de configuração de runtime, **incluindo o ponto de merge da cascata**, com teste por camada.
2. Adicionar a flag de execução, seguindo o padrão de detecção de override explícito já usado nas flags de memória em `cmd/ai_spec_harness/task_loop.go:97`.
3. Atualizar o contrato declarado de linha de comando, exigido pelo gate bidirecional em `cmd/ai_spec_harness/cli_contract_test.go:189`, que compara flags nas duas direções.
4. Implementar o teste de paridade byte-a-byte com os quatro casos, antes de ligar a fachada ao consumidor.
5. Corrigir o descarte de erro nos dois pontos de despacho, com registro explícito e composição da evidência.
6. Tornar o descompasso de tipo do hook observável.
7. Implementar o comando de migração com backup verificável e recusa de reaplicação.

Nota de reconciliação: o passo 6 modifica `internal/runtime/hooks/memory_persist.go`, arquivo que uma versão anterior da especificação técnica declarava intocado. A contradição foi corrigida na techspec. A alteração é restrita à observabilidade do descompasso de tipo — `buildMemoryContent` e o modo de escrita permanecem byte-idênticos, então a paridade de RF-29 continua sustentada por esse arquivo.

Dependências: passo 4 precede o passo em que a fachada é ligada ao consumidor (MD-001, passo 5).

Critério de adoção concluída: com a feature desativada, o prompt é byte-idêntico nos quatro casos, e a suíte existente passa sem alteração de expectativa.

## Monitoramento e Validação

- **Sinais:** contagem de execuções com a feature ativada e desativada; contagem de falhas de despacho registradas; contagem de descompassos de tipo no hook.
- **Logs:** ativação e desativação registradas de forma explícita; toda falha de gravação registrada.
- **Critério de sucesso:** zero alteração de expectativa em teste existente, e paridade byte-a-byte verde.
- **Critério para revisar:** quando houver evidência de campo suficiente para promover a feature a default, esta ADR é revisada e possivelmente substituída.

## Impacto em Documentação e Operação

- `docs/config-hierarchy.md`: chave nova documentada — o levantamento confirmou que não existe gate de CI comparando esse documento com a estrutura de configuração, então a atualização é manual e obrigatória por disciplina.
- `docs/cli-schema.json`: flag e comando novos, sob pena de falha no gate bidirecional de contrato.
- `docs/task-loop-reference.md` e `CLAUDE.md`: comportamento de ativação.
- `docs/troubleshooting.md`: como confirmar se a feature está ativa e como desativá-la.

## Revisão Futura

Revisar na primeira major posterior à acumulação de evidência de campo, para decidir sobre ativação por default. Revisar imediatamente se a paridade byte-a-byte falhar em qualquer execução de integração contínua, porque isso invalida a premissa central da decisão.
