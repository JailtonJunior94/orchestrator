# Tarefa 10.0: Integração multi-processo, e2e, benchmark e alvos de Make

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar as lacunas que teste unitário não cobre, e garantir que os testes novos **efetivamente executem**. Esta tarefa é dona única do `Makefile`: `integration` em `Makefile:26` e `bench` em `Makefile:61` enumeram diretórios fixos e não incluem o pacote novo — sem estendê-los, os testes de integração e o benchmark nunca rodam e o risco migra em silêncio para as outras fatias.

O único molde concorrente do repositório, `internal/taskloop/orchestrator_test.go:215`, usa goroutines no mesmo processo, o que **não** exercita o mecanismo de exclusão real. RF-16 exige processos separados.

<requirements>
- RF-16: dois processos reais competindo pela mesma camada, sem corromper página e sem perder fato.
- RF-25: handoff cross-CLI — sessão em uma CLI, sessão seguinte em outra, com os fatos duráveis presentes no contexto injetado.
- RF-20: benchmark de recuperação com 1.000 páginas, alvo de p95 abaixo de 200 ms.
- RF-10: teste que falha se o caminho default abrir socket, resolver DNS ou invocar CLI externa.
- Nenhum contêiner é necessário: `exec.Command` reinvocando o binário de teste mais `t.TempDir()`.
- Os alvos do `Makefile` precisam incluir o pacote novo, provado pela saída dos comandos.
</requirements>

## Subtarefas

- [ ] 10.1 Estender `Makefile` nos alvos `integration` e `bench` para incluir o pacote novo.
- [ ] 10.2 Teste de dois processos reais competindo pela mesma camada, sob `//go:build integration`.
- [ ] 10.3 Teste de lock órfão: processo morto sem liberar; o próximo detecta por prazo vencido ou dono inexistente, com a tomada registrada.
- [ ] 10.4 Teste de bootstrap em repositório vazio, confirmando que ausência total de memória não falha.
- [ ] 10.5 Teste de migração com backup verificável e recusa de reaplicação.
- [ ] 10.6 Teste de handoff cross-CLI com fatos duráveis presentes na sessão seguinte.
- [ ] 10.7 Teste e2e com o servidor ACP falso de `internal/runtime/acpfake/` e **agente que ignora toda diretiva textual**, provando RF-13.
- [ ] 10.8 Benchmark de recuperação com 1.000 páginas.
- [ ] 10.9 Teste que falha se o caminho default tocar a rede.

## Detalhes de Implementação

Ver techspec.md, seções "Testes de Integração" (a decisão de adotá-los e os cinco cenários) e "Testes E2E".

## Critérios de Sucesso

- `make integration` e `make bench` **incluem o pacote novo na saída** — provado por execução, não por inspeção do arquivo.
- Dois processos reais não corrompem página nem perdem fato; qualquer falha é exclusivamente a sentinela de lock.
- Lock órfão é detectado e reportado, nunca sobrescrito em silêncio.
- Repositório vazio não falha e não emite bloco de memória vazio.
- Agente não colaborativo: após a sessão, nenhuma página excede seus limites.
- Handoff cross-CLI entrega ao menos 90% dos fatos duráveis da sessão anterior, objetivo O-1 do PRD.
- p95 abaixo de 200 ms para 1.000 páginas.
- Cobertura total do repositório permanece acima de 75%, threshold do CI.

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
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `Makefile` — alvos `integration` (linha 26) e `bench` (linha 61); **dono único**
- `tests/integration/` — testes novos sob build tag
- `internal/runtime/acpfake/` — servidor ACP falso, apenas leitura
- `internal/taskloop/orchestrator_test.go` — molde concorrente insuficiente, apenas leitura
- `scripts/check-package-coverage.sh` — threshold por pacote, apenas leitura
