# Prompt: Release Automatica no GitHub a cada Push para main

Atue como um engenheiro senior de release automation e GitHub Actions neste repositorio. Trabalhe de forma objetiva, mandataria, verificavel e com a menor mudanca segura.

Objetivo: implementar ou ajustar a automacao de release para que, a cada `push` na branch `main`, o repositorio:

1. calcule deterministicamente a proxima versao seguindo `SemVer`;
2. gere a tag Git oficial no formato `vMAJOR.MINOR.PATCH`;
3. crie a GitHub Release correspondente;
4. use `Conventional Commits` como fonte primaria para decidir `MAJOR`, `MINOR` e `PATCH`;
5. execute tudo via `.github/workflows`, sem depender de etapa manual.

Referencias obrigatorias:

- `https://www.conventionalcommits.org/pt-br/v1.0.0-beta.4/`
- `https://semver.org/`

Contexto ja confirmado neste repositorio:

- branch principal atual: `main`
- diretorio de automacao alvo: `.github/workflows/`
- workflows existentes: `.github/workflows/release.yml` e `.github/workflows/release-dry-run.yml`
- existem tags previas no formato `v*`, incluindo `v0.5.0`, `v0.4.0` e `v0.3.0`

Instrucoes obrigatorias:

1. Inspecione o estado real do repositorio antes de decidir a implementacao.
2. Reutilize e ajuste os workflows existentes se isso for suficiente; nao reescreva tudo sem necessidade.
3. Use `Conventional Commits` como regra oficial de interpretacao dos commits.
4. Mapeie bump de versao assim:
   - `MAJOR`: commit com breaking change explicito, incluindo `!` no tipo/escopo ou footer `BREAKING CHANGE:`
   - `MINOR`: commit `feat`
   - `PATCH`: commits como `fix`, `perf` ou outro tipo que represente correcao compativel e gere release
5. Defina explicitamente como tratar tipos como `docs`, `style`, `refactor`, `test`, `build`, `ci` e `chore`:
   - se nao gerarem release, diga isso claramente;
   - se algum deles gerar `PATCH`, justifique com base no contexto real do repositorio.
6. Garanta idempotencia:
   - nao criar tag duplicada;
   - nao publicar release duplicada;
   - nao entrar em loop ao commitar artefatos de release.
7. Trate o caso de repositorio sem tag anterior com uma regra explicita para a primeira versao.
8. Considere o uso correto de `permissions`, `fetch-depth: 0`, autenticacao com `GITHUB_TOKEN` e prevencao de reexecucao insegura.
9. Se houver geracao de `CHANGELOG.md`, trate-o como artefato derivado da versao e dos commits, nao como fonte primaria para bump, salvo evidencia forte no repositorio.
10. Preserve compatibilidade com o fluxo atual baseado em `push` para `main`, salvo evidencia tecnica contraria.
11. Se a implementacao atual ja fizer parte do trabalho, identifique precisamente o que falta para cumprir o objetivo completo de tag + GitHub Release automatica.

Escopo minimo de inspecao:

- `.github/workflows/release.yml`
- `.github/workflows/release-dry-run.yml`
- arquivos ou comandos que calculam a proxima versao
- historico recente de commits e tags
- qualquer configuracao de release ja usada pelo repositorio

Criterios de aceite:

- existe uma regra deterministica e documentada para `MAJOR`, `MINOR` e `PATCH`
- a automacao roda a cada `push` em `main`
- a tag Git oficial e criada automaticamente no formato `vMAJOR.MINOR.PATCH`
- a GitHub Release e criada automaticamente para a tag gerada
- a implementacao evita duplicidade de tag e release
- o fluxo informa claramente o que acontece quando nao ha commits elegiveis para release
- o resultado inclui validacao proporcional, preferencialmente com um dry-run ou estrategia equivalente
- os arquivos criados ou alterados sao listados com caminho exato

Contrato de saida:

- formato: Markdown
- seja direto e tecnico
- entregue nesta ordem:
  1. `Diagnostico do estado atual`
  2. `Estrategia recomendada`
  3. `Mapeamento Conventional Commits -> SemVer`
  4. `Fluxo de automacao em main`
  5. `Implementacao em .github/workflows`
  6. `Protecoes de idempotencia e seguranca`
  7. `Arquivos alterados`
  8. `Validacao executada`
  9. `Riscos, gaps ou decisoes assumidas`
- em `Implementacao em .github/workflows`, forneca o YAML completo pronto para uso ou diff suficiente para aplicacao direta
- em `Arquivos alterados`, liste cada caminho com uma frase curta explicando o motivo
- se optar por usar GitHub Release nativa, diga como ela sera criada
- se optar por ferramenta auxiliar, justifique por que ela e necessaria neste repositorio

Nao faca:

- nao entregue apenas explicacao teorica sobre `SemVer` ou `Conventional Commits`
- nao proponha processo manual para criar tag ou release
- nao use heuristicas opacas para calcular versao
- nao ignore workflows e tags que ja existem no repositorio
- nao mude a branch principal, o formato de tag ou a estrategia de disparo sem justificar
- nao trate qualquer commit como elegivel para release sem regra explicita

Tratamento de falhas:

- se os commits atuais nao seguirem `Conventional Commits` de forma suficiente, declare o gap e proponha a menor estrategia segura para transicao
- se houver conflito entre o workflow atual e a regra desejada, priorize a solucao mais simples que preserve o comportamento util existente
- se faltar permissao ou token para publicar a release, explicite a configuracao exata necessaria
- se nao for possivel provar a automacao com execucao real, informe o comando ou procedimento de validacao que ficou pendente

Exemplos minimos que a resposta deve cobrir:

- `fix(parser): corrigir leitura de tag` => bump `PATCH`
- `feat(cli): adicionar comando release-plan` => bump `MINOR`
- `feat(api)!: remover campo legado` => bump `MAJOR`
- `docs: atualizar README` => explicar se gera release ou nao

## Justificativas das Adicoes

- Adicionei contexto do repositorio ja verificado para reduzir ambiguidade e evitar que o agente invente estrutura inexistente.
- Explicitei a relacao entre `Conventional Commits` e `SemVer`, porque esse mapeamento e o nucleo da automacao pedida.
- Forcei verificacao de idempotencia, prevencao de loop e tratamento de primeira tag, que sao os pontos mais sensiveis desse fluxo.
- Restrinji o output para uma ordem objetiva e acionavel, incluindo YAML pronto para uso e validacao proporcional.
- Mantive o escopo centrado em `.github/workflows` e no fluxo de `push` para `main`, sem ampliar a solucao para processos manuais ou estrategias paralelas.
