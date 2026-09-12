# Tarefa 4.0: Lease de bastão de continuidade e detecção de processo vivo

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o bastão de continuidade como lease com prazo e referência de processo, mais a detecção de processo vivo separada por build tag. **Esta tarefa subiu da posição 11 para a 4** porque o lock de camada, e não apenas o bastão, carrega prazo e referência de processo: sem ela, 5.0 herdaria o lock órfão permanente documentado em `internal/taskloop/orchestrator_lock_windows.go:11`, e a precedência de invariantes da fachada em 7.0 nasceria com o passo "dono único de bastão" vazio.

Não existe hoje no repositório nenhum código de detecção de processo vivo, lease ou TTL — é greenfield sobre um padrão existente de separação por plataforma.

<requirements>
- RF-23: lease com prazo padrão de 30 minutos, renovado enquanto a sessão detentora estiver ativa; exatamente um detentor; segunda reivindicação recusada explicitamente; reivindicável por prazo vencido ou dono inexistente; tomada registrada; prazo configurável.
- Detecção de processo vivo em arquivos separados por build tag, replicando o padrão de `internal/taskloop/orchestrator_lock_unix.go` e `_windows.go`.
- Onde a verificação de processo é frágil, o prazo prevalece — assimetria assumida e exercitada por teste.
- Lock órfão nunca sobrescrito em silêncio.
</requirements>

## Subtarefas

- [x] 4.1 Definir `LeaseDeBastao` (dono, prazo, referência de processo) e `BastaoDeContinuidade` como agregado próprio.
- [x] 4.2 Implementar detecção de processo vivo em arquivo com build tag para plataformas tipo Unix.
- [x] 4.3 Implementar detecção de processo vivo em arquivo com build tag para Windows, com fallback explícito por prazo.
- [x] 4.4 Implementar a política de lease: conceder, recusar, transferir, sempre registrando o resultado.
- [x] 4.5 Adicionar a chave de prazo à cascata: `internal/config/runtime.go`, `mergeInto` em `internal/config/resolver.go` **e** `optionsToConfigOverrides` em `internal/taskloop/runtimeconfig.go`.
- [x] 4.6 Documentar em `docs/config-hierarchy.md` e em `docs/troubleshooting.md` (diagnóstico de bastão retido e lock órfão).

## Detalhes de Implementação

Ver techspec.md, seções "Visão Geral dos Componentes" (detecção de processo vivo) e "Riscos Conhecidos" (build tag e segundo ponto da cascata). Ver MD-003 (`adr-003-escrita-atomica-lock-camada-lease.md`), seção Decisão, terceiro e quarto parágrafos.

## Critérios de Sucesso

- Quatro casos cobertos por teste: bastão livre concede; dono vivo dentro do prazo recusa informando dono e prazo restante; prazo vencido permite tomada; dono inexistente permite tomada.
- Toda tomada aparece registrada, nunca silenciosa.
- O fallback por prazo é exercitado explicitamente, não apenas documentado.
- A chave de prazo propaga em **ambos** os pontos da cascata, provado por teste que exercita a flag de CLI ponta a ponta até o `Job`, e por teste por camada de configuração.
- Compila e passa nas duas plataformas da matriz de CI.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
- [x] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Arquivos de lease e de detecção de processo no pacote criado em 2.0 — ver techspec.md
- `internal/taskloop/orchestrator_lock_unix.go` e `internal/taskloop/orchestrator_lock_windows.go` — padrão de build tag, apenas leitura
- `internal/config/runtime.go`, `internal/config/resolver.go`, `internal/taskloop/runtimeconfig.go` — cascata, dois pontos de propagação
- `docs/config-hierarchy.md`, `docs/troubleshooting.md`
