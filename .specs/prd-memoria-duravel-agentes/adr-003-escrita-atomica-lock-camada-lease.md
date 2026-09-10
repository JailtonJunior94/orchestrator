# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Rótulo:** MD-003
- **Título:** Escrita atômica na abstração de filesystem, lock por camada e lease de bastão com detecção de dono morto
- **Data:** 2026-09-10
- **Status:** Proposta
- **Decisores:** JailtonJunior94 (mantenedor do harness)
- **Relacionados:** [PRD](prd.md) RF-16, RF-23, RF-25 · [techspec](techspec.md) · [ADR-002 do repositorio](../../docs/adr/002-fake-filesystem-testes.md) · MD-001, MD-002 desta feature

## Contexto

A feature grava conhecimento em arquivos versionados sob concorrência de múltiplos processos: RF-16 exige que escritas concorrentes não corrompam páginas nem perdam fatos, e o caso de uso central é duas CLIs abertas no mesmo projeto. RF-23 exige um bastão de continuidade com dono único, liberável quando o prazo vence ou quando o processo dono não existe mais.

O levantamento do código de produção revelou quatro fatos que delimitam a decisão.

Primeiro: a escrita atômica **existe** no repositório, em `internal/sdd/state.go:299` até `internal/sdd/state.go:321`, com o padrão correto de arquivo temporário no mesmo diretório, `Sync`, fechamento e `Rename`. Mas ela usa chamadas diretas de sistema operacional e vive fora da abstração de filesystem, portanto não é reutilizável nem testável pelos outros pacotes. Na abstração, `internal/fs/fs.go:162` faz escrita direta, sem temporário e sem rename — se o processo morrer no meio, o arquivo fica truncado.

Segundo: o subsistema de memória atual **diverge do padrão declarado do repositório**. `internal/runtime/memory/store.go:104`, `:121` e `:153` usam chamadas diretas de sistema operacional, enquanto o ADR-002 do repositório determina que todo serviço receba a abstração de filesystem por construtor. `internal/runtime/persistence/` segue o padrão corretamente. A divergência no pacote de memória não está registrada como decisão — é desvio de aderência, e herdá-la silenciosamente significaria herdar a impossibilidade de teste unitário com filesystem falso.

Terceiro: o lock por plataforma existe e é reutilizável, mas as duas implementações têm **semântica de recuperação divergente**. Em `internal/taskloop/orchestrator_lock_unix.go:17` o lock é advisory via `flock` não bloqueante, e o kernel o libera automaticamente quando o processo morre. Em `internal/taskloop/orchestrator_lock_windows.go:15` o lock é criação exclusiva de arquivo, e o próprio comentário do código registra que um processo morto deixa o arquivo órfão bloqueando permanentemente todas as tentativas futuras, exigindo remoção manual após confirmação operacional. Reutilizar esse lock sem tratar a divergência produziria uma feature que funciona em duas plataformas e travava na terceira.

Quarto: **não existe no repositório nenhum código de detecção de processo vivo, lease ou TTL**. A busca por PID-liveness retornou apenas gestão de subprocessos do agente.

## Decisão

Três decisões acopladas, todas na camada de infraestrutura do subsistema.

**Escrita atômica é promovida à abstração de filesystem.** O padrão de `internal/sdd/state.go:299` é portado para `internal/fs`, com implementação real e implementação falsa, e passa a ser a única forma de gravar página. Nenhum arquivo parcial pode ficar visível em nenhum momento. O erro de `Sync` e de `Close` é verificado, porque o linter `errcheck` do repositório só tolera fechamento via `io.Closer` e um conjunto nomeado de funções, e `Sync` não está nessa lista.

**O subsistema de memória durável usa a abstração de filesystem injetada por construtor**, corrigindo o desvio observado no store atual em vez de propagá-lo. Isso é pré-condição de testabilidade: sem a abstração, não há teste unitário com filesystem falso, e o repositório usa filesystem falso em 46 arquivos de teste.

**O lock é por camada**, reutilizando o lock por plataforma existente. A granularidade coincide com o agregado definido no modelo de domínio: escrita de memória de task não bloqueia leitura de memória de projeto.

**A divergência de recuperação entre plataformas é tratada explicitamente por lease.** O bastão e o lock de camada carregam prazo e referência de processo. Um lock ou bastão é reivindicável quando o prazo vence **ou** quando o processo dono não existe mais. A detecção de processo vivo é implementada com arquivos separados por build tag, replicando exatamente o padrão que o repositório já usa para o lock. Em plataforma onde a verificação de processo é frágil, o prazo é o critério que prevalece — e o prazo padrão é de 30 minutos, configurável. Lock órfão nunca é sobrescrito em silêncio: a tomada é registrada na evidência.

## Alternativas Consideradas

**Reaproveitar o lock existente sem tratar o lock órfão.** Vantagens: nenhum código novo. Desvantagens: em plataforma sem liberação automática, um processo morto travaria a memória do projeto até intervenção manual — para uma feature que roda a cada sessão, isso é indisponibilidade recorrente. Não escolhida.

**Detecção apenas por processo vivo, sem prazo.** Vantagens: liberação imediata após falha. Desvantagens: identificador de processo pode ser reciclado por outro processo, e a verificação não funciona quando as sessões rodam em contêineres ou máquinas distintas. Não escolhida.

**Prazo apenas, sem verificação de processo.** Vantagens: simples e portátil, sem introspecção de processo nem build tags novas. Desvantagens: obriga esperar o prazo inteiro depois de uma falha, mesmo sabendo que o dono morreu. Não escolhida, mas é o comportamento de fallback adotado na plataforma onde a verificação de processo é frágil.

**Liberação apenas manual.** Vantagens: totalmente determinístico e auditável. Desvantagens: uma falha à noite bloqueia a próxima sessão até intervenção humana.

**Escrita atômica local ao pacote de memória, sem promover à abstração.** Vantagens: menor raio de alteração, não toca `internal/fs`. Desvantagens: perpetua o desvio de aderência, mantém a impossibilidade de teste unitário com filesystem falso, e deixa o repositório com duas implementações de escrita atômica em pacotes diferentes. Não escolhida.

**Lock global cobrindo as três camadas.** Vantagens: uma única fronteira, mais simples de raciocinar. Desvantagens: serializaria sessões concorrentes no mesmo projeto, penalizando exatamente o caso de uso de duas CLIs abertas.

**Lock por Fato.** Desvantagens: as invariantes de não duplicidade e de consolidação perderiam ponto de aplicação transacional.

## Consequências

### Benefícios Esperados

- Nenhum arquivo parcial visível, em nenhuma plataforma, mesmo com falha durante a escrita.
- Teste unitário do subsistema com filesystem falso, alinhando a feature ao ADR-002 do repositório em vez de ampliar o desvio.
- Escrita atômica disponível para o restante do repositório, incluindo a possibilidade futura de corrigir o gap de concorrência multi-processo do registro de eventos, hoje reescrito por inteiro a cada evento em `internal/runtime/persistence/jsonl.go`.
- Falha de processo não bloqueia a próxima sessão em nenhuma plataforma.
- Paridade real entre as quatro CLIs suportadas (RF-25), porque a semântica de recuperação passa a ser a mesma.

### Trade-offs e Custos

- Dois arquivos novos separados por build tag para detecção de processo, com o custo de manutenção de código específico por plataforma.
- A verificação de processo é intrinsecamente menos confiável em uma das plataformas, e o prazo passa a carregar mais peso lá — isso é assimetria assumida, não resolvida.
- Alterar `internal/fs` amplia o raio de impacto da feature para um pacote consumido por praticamente todo o repositório, o que exige que a mudança seja estritamente aditiva.
- Um teste de concorrência com processos reais é mais caro de escrever e mais lento que o molde existente com goroutines.

### Riscos e Mitigações

- **Risco:** alterar a abstração de filesystem quebrar consumidores existentes. **Impacto:** regressão ampla. **Mitigação:** a mudança é estritamente aditiva — um método novo na interface e nas duas implementações; nenhuma assinatura existente muda. O compilador prova a completude, e o gate de mocks força a regeneração.
- **Risco:** o teste de concorrência existente dar falsa confiança. **Impacto:** exclusão inter-processo não provada. **Mitigação:** o único molde concorrente do repositório, em `internal/taskloop/orchestrator_test.go:215`, usa goroutines no mesmo processo, o que não exercita o mecanismo real. RF-16 exige teste com processos separados, e essa lacuna está declarada, não escondida.
- **Risco:** identificador de processo reciclado ser interpretado como dono vivo. **Impacto:** bastão preso até o prazo vencer. **Mitigação:** o prazo é o limite superior do dano, e é configurável.
- **Plano de rollback:** desativar a feature restaura o comportamento anterior, que não usa lock nem lease. A escrita atômica adicionada à abstração permanece, porque é aditiva e não altera comportamento de quem não a chama.

## Plano de Implementação

1. Adicionar escrita atômica à abstração de filesystem, portando o padrão existente, com implementação real e falsa e teste que prove que arquivo parcial nunca fica visível.
2. Regenerar os mocks da abstração de filesystem e confirmar o gate de mocks.
3. Implementar detecção de processo vivo em arquivos separados por build tag, replicando o padrão do lock existente.
4. Implementar o lease com prazo configurável, prazo padrão de 30 minutos.
5. Implementar o agregado de camada com lock por camada e escrita atômica.
6. Implementar o teste de concorrência com processos reais, sob a marcação de teste de integração.

Dependências: passo 1 precede tudo. MD-002 desta feature precede o passo 5, porque o agregado serializa páginas.

Critério de adoção concluída: dois processos reais competindo pela mesma camada não corrompem página nem perdem fato, e lock órfão é detectado e reportado sem sobrescrita silenciosa.

## Monitoramento e Validação

- **Sinais:** número de tomadas de bastão por prazo vencido; número de tomadas por dono inexistente; número de recusas de reivindicação por dono vivo.
- **Logs:** toda tomada de bastão e toda detecção de lock órfão devem aparecer na evidência da sessão.
- **Critério de sucesso:** nenhuma sessão bloqueada por lock órfão em nenhuma plataforma, e nenhuma página corrompida sob concorrência.
- **Critério para revisar:** se tomadas por prazo vencido forem frequentes, o prazo padrão está curto demais para as sessões reais e precisa aumentar.

## Impacto em Documentação e Operação

- `docs/troubleshooting.md`: como diagnosticar bastão retido e lock órfão, e o que significa cada mensagem.
- `docs/config-hierarchy.md`: chave de prazo do lease.
- `mockery.yml` e mocks da abstração de filesystem: regeneração.
- [`docs/adr/002-fake-filesystem-testes.md`](../../docs/adr/002-fake-filesystem-testes.md): vale anotar que o subsistema de memória passou a aderir ao padrão, encerrando o desvio.

## Revisão Futura

Revisar quando o gap de concorrência multi-processo do registro de eventos for endereçado, porque a escrita atômica introduzida aqui é a peça que falta para corrigi-lo. Revisar também se o suporte a sessões em contêineres ou máquinas distintas entrar em escopo, porque a verificação de processo vivo deixa de funcionar e o prazo passaria a ser o único critério.
