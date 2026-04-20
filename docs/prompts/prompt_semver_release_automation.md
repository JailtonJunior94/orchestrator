# Prompt: Automacao de SemVer e Tags

## Papel

Atue como um engenheiro senior de release automation e GitHub Actions. Trabalhe de forma mandataria, objetiva e verificavel.

## Objetivo

Definir e implementar a melhor estrategia de versionamento aderente ao `https://semver.org/`, usando `commits`, `branches`, `changelog` ou combinacao deles, para gerar tags automaticamente em `.github/workflows` e usar a versao calculada como tag oficial do repositorio.

## Entradas

- Repositorio local atual.
- Branch principal atual: `main`.
- Historico de commits, tags e workflows existentes.
- `README.md` e qualquer `CHANGELOG*`, se existirem.

## Constraints

- A estrategia deve mapear `MAJOR`, `MINOR` e `PATCH` de forma deterministica.
- A automacao deve rodar em `.github/workflows`.
- A versao calculada deve virar a tag Git oficial do release.
- Escolha uma unica estrategia principal e cite no maximo 2 alternativas.
- Preserve compatibilidade com fluxo baseado em `main`, salvo evidencia contraria no repositorio.
- Nao invente tags, arquivos, convencoes ou workflows sem verificar.
- Nao proponha processo manual se houver automacao confiavel.
- Evite respostas vagas e feche uma decisao principal.

## Processo

1. Inspecione branch principal, tags, padrao de commits, changelog e workflows.
2. Compare `commits`, `branches` e `changelog` como fontes de verdade para bump de versao.
3. Escolha a melhor abordagem para este repositorio e explique:
   - o que gera `MAJOR`, `MINOR` e `PATCH`;
   - em que evento a tag e criada;
   - como evitar tag duplicada;
   - como tratar ausencia de tag anterior.
4. Se o padrao atual de commits nao permitir automacao segura, proponha e implemente uma convencao objetiva, preferencialmente alinhada a Conventional Commits.
5. Defina se o changelog sera fonte primaria, secundaria ou artefato derivado.
6. Implemente o workflow necessario com protecao contra reexecucao insegura e criacao indevida de tag.
7. Valide a estrategia com exemplos para correcao, feature compativel, breaking change e repositorio sem tag anterior.

## Criterios de Aceite

- Existe regra documentada para `MAJOR`, `MINOR` e `PATCH`.
- Existe workflow funcional em `.github/workflows` para calcular e/ou publicar a tag.
- O workflow usa a versao calculada como tag Git oficial.
- O resultado explica por que a fonte de verdade escolhida e a melhor para este repositorio.
- O resultado informa arquivos criados ou alterados.

## Output Contract

- Formato: Markdown.
- Inclua, nesta ordem:
  1. `Diagnostico do repositorio`
  2. `Estrategia recomendada`
  3. `Mapeamento SemVer`
  4. `Fluxo de branches e releases`
  5. `Papel do changelog`
  6. `Implementacao em .github/workflows`
  7. `Arquivos alterados`
  8. `Riscos e mitigacoes`
  9. `Exemplos de versao gerada`
- Em `Implementacao em .github/workflows`, forneca YAML completo pronto para uso.
- Em `Arquivos alterados`, liste caminhos exatos e motivo.
- Em `Estrategia recomendada`, declare a opcao principal antes das alternativas.

## Nao Faca

- Nao entregue apenas teoria sobre SemVer.
- Nao use heuristicas opacas para `MAJOR`, `MINOR` e `PATCH`.
- Nao gere tags em qualquer branch sem regra explicita.
- Nao ignore o estado real do repositorio.

## Tratamento de Falhas

- Se faltar historico suficiente, declare a premissa minima e continue.
- Se nao houver tags anteriores, trate a primeira versao de forma explicita.
- Se commits, branches e changelog entrarem em conflito, priorize a fonte de verdade escolhida e justifique.
