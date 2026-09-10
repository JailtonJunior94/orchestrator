# Tarefa 4.8: Prova de paridade entre os três caminhos e fluxos E2E

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Sexta e última fatia do Bloco D. Com os três caminhos (`Service.Execute` em 4.4, `RunLoop` em 4.6,
`ACPRunner` em 4.7) já conduzindo o `approval.Cycle`, esta fatia entrega a **prova** de que se
comportam de forma idêntica. A techspec registra que o repositório já sofreu caso de unitário verde
com integração real falhando (chave de evento inválida do Copilot), por isso a prova E2E é
obrigatória. Fecha o estágio de paridade — o gate de não-regressão mais importante do plano
(ADR-001, item 4).

<requirements>
- RF-31 / RF-34: os três caminhos produzem veredito, motivo canônico e estrutura de evidência
  idênticos para a mesma entrada; cada rodada em sessão nova.
- A virada do critério estrito **não** acontece aqui (é 5.0). A suíte existente permanece verde.
- T-REV-01/02/04 verdes e sem alteração de asserção.
</requirements>

## Subtarefas

- [ ] 4.8.1 Teste de paridade: mesmo cenário de entrada produz veredito, motivo canônico e estrutura
      de evidência idênticos em `ACPRunner`, `Service.Execute` e `RunLoop`.
- [ ] 4.8.2 Cinco fluxos E2E com o servidor ACP falso in-process já existente no repositório:
      (a) aprova na primeira rodada; (b) aprova na terceira após duas correções; (c) aborta por
      fingerprint repetida **sem gastar a rodada seguinte**; (d) aborta por ausência de mudança;
      (e) remediação sem diff.
- [ ] 4.8.3 Teste dedicado: `APPROVED_WITH_REMARKS` realimenta a correção ponta a ponta.

## Detalhes de Implementação

Seguir a techspec desta pasta, **"Abordagem de Testes" → "Testes E2E"** (servidor ACP falso
in-process), a subseção **"Integração do agregado nos três caminhos (Bloco D)"** e a ADR-001 "Plano
de Implementação" item 4.

## Critérios de Sucesso

- Teste de paridade entre os três caminhos verde.
- Os cinco fluxos E2E verdes.
- Teste de `APPROVED_WITH_REMARKS` realimentando verde.
- `grep -rn 'approval\.' internal/runtime/runner.go internal/taskloop/taskloop.go
  internal/taskloop/runloop.go` retorna ocorrências nos três arquivos.
- T-REV-01/02/04 verdes e sem alteração de asserção.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync check-hooks-sync check-mocks budget`
  verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/runtime/... ./internal/taskloop/... -count=1` verde.
- Golden files de governança byte-idênticos.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória:

- Paridade: `ACPRunner`, `Service.Execute` e `RunLoop` produzem veredito, motivo e evidência
  idênticos para a mesma entrada.
- E2E: os cinco fluxos com o servidor ACP falso in-process.
- `APPROVED_WITH_REMARKS` realimenta a correção ponta a ponta.
- Preservação: T-REV-01/02/04 verdes e não modificados.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/runner.go`, `internal/taskloop/taskloop.go`, `internal/taskloop/runloop.go` —
  os três caminhos.
- `internal/taskloop/*_test.go`, `internal/runtime/*_test.go` — testes de paridade e E2E.
- Servidor ACP falso in-process já existente no repositório.
- `internal/approval/` — agregado e portas.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — "Abordagem de Testes" → "Testes E2E".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md`.
