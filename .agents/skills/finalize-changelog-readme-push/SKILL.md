---
name: finalize-changelog-readme-push
version: 1.0.0
depends_on: [github-diff-changelog-publisher, semantic-commit]
description: Atualiza CHANGELOG.md, revisa README.md quando necessario, prepara staging, cria commit semantico e publica no remoto. Use quando um lote de mudancas ja estiver pronto para fechamento e o usuario quiser consolidar documentacao, commit e push em sequencia. Nao use para gerar release versionada, revisar codigo ou publicar apenas parte do workspace sem confirmacao explicita.
---

# Finalizar Changelog README Commit e Push

## Procedimentos

**Etapa 1: Validar o escopo de publicacao**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Antes de invocar outras skills, executar `source scripts/lib/check-invocation-depth.sh || { echo "failed: depth limit exceeded"; exit 1; }`.
3. Executar `git status --short` e `git diff --stat` para mapear o lote atual.
4. Se nao houver mudancas locais, retornar `blocked` sem editar arquivos.
5. Se existir risco de misturar trabalho alheio, explicitar isso antes de continuar, porque esta skill opera com `git add .` e publica o workspace inteiro.

**Etapa 2: Atualizar o changelog**
1. Ler `CHANGELOG.md` e preferir a secao `[Unreleased]` quando ela existir.
2. Se o usuario tiver informado um intervalo de refs ou uma release-alvo, usar a skill `github-diff-changelog-publisher` como apoio para estruturar os itens.
3. Se a mudanca ainda nao estiver em commits separados, resumir o lote atual a partir do diff sem inventar numero de versao.
4. Registrar apenas mudancas relevantes para usuario, mantenedor ou fluxo operacional; excluir ruido cosmetico sem impacto.
5. Se o repositório mantiver convencao de categorias, preservar a taxonomia local.

**Etapa 3: Revisar o README apenas se necessario**
1. Ler o `README.md` atual e comparar com as capacidades, comandos ou fluxos alterados no lote.
2. Atualizar apenas as secoes que ficarem desatualizadas por causa da mudanca.
3. Se o README continuar correto, nao editar por zelo excessivo; registrar explicitamente que nenhuma mudanca documental foi necessaria.
4. Manter a alteracao pequena e coerente com a voz e a estrutura existentes.

**Etapa 4: Preparar staging e commit**
1. Reexecutar `git status --short` depois das atualizacoes documentais.
2. Mostrar ao usuario quais arquivos entrarao no commit e confirmar que a intencao e mesmo publicar todo o estado atual com `git add .`.
3. Se a confirmacao nao existir ou o usuario quiser apenas parte do diff, retornar `needs_input` em vez de stagear seletivamente por conta propria.
4. Executar `git add .` somente apos essa confirmacao explicita.
5. Invocar a skill `semantic-commit` para definir o tipo, escopo e a mensagem final do commit.
6. Respeitar a regra da skill `semantic-commit`: confirmar o texto final da mensagem com o usuario antes de executar `git commit`.
7. Nao usar `--no-verify`, `--no-gpg-sign` ou `--amend` sem pedido explicito.

**Etapa 5: Publicar no remoto**
1. Identificar a branch atual com `git branch --show-current` e o upstream com `git rev-parse --abbrev-ref --symbolic-full-name @{u}`.
2. Se nao houver upstream configurado, retornar `needs_input` com a sugestao de `git push -u <remote> <branch>`.
3. Executar `git push` sem `--force` e sem reescrever historico.
4. Ao concluir, informar remoto, branch e `HEAD` publicado.

**Etapa 6: Encerrar com estado explicito**
1. Resumir se houve atualizacao em `CHANGELOG.md` e `README.md`.
2. Informar se `git add .`, commit e push foram executados ou se a skill parou em `needs_input`.
3. Retornar `done`, `needs_input`, `blocked` ou `failed`.

## Tratamento de Erros

* Se `CHANGELOG.md` nao existir, registrar a ausencia e prosseguir apenas se a convencao do repositorio nao exigir changelog.
* Se o `README.md` estiver em conflito ou muito divergente do lote atual, limitar-se a apontar as secoes obsoletas e retornar `needs_input` antes de reescrever em massa.
* Se hooks, testes ou o proprio `git commit` falharem, diagnosticar e corrigir a causa antes de tentar novamente; nao contornar validacoes.
* Se o `push` falhar por divergir do remoto, parar e reportar a causa; nao fazer `pull --rebase` nem `push --force` sem instrucao explicita.
