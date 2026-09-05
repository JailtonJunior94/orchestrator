---
name: semantic-commit
version: 1.1.0
description: Cria commits semanticos no padrao Conventional Commits (feat, fix, chore, docs, refactor, test, ci). Use quando precisar commitar mudancas com mensagem estruturada e rastreavel. Nao use para revisar ou implementar codigo — apenas para a acao de commit.
---

# Commit Semantico

## Procedimentos

**Etapa 1: Analisar mudancas**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Executar `git diff --cached --stat` para ver o que esta staged.
2. Se nada estiver staged, executar `git status` e listar arquivos nao staged.
3. Identificar o tipo da mudanca: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `perf`.
4. Identificar o escopo (modulo, pacote, servico afetado).

**Etapa 2: Compor a mensagem**
1. Formato: `<tipo>(<escopo>): <descricao em imperativo>`
2. Descricao: ≤72 caracteres, sem ponto final, em ingles ou portugues conforme padrao.
3. Body opcional: separado por linha em branco, explicando *por que* (nao *o que*).
4. Footer: `BREAKING CHANGE:` quando houver quebra de compatibilidade; `Closes #nn` para issues.

**Etapa 3: Executar o commit**
1. Confirmar com o usuario o texto final da mensagem antes de commitar.
2. Executar `git commit -m "$(cat <<'EOF'\n<mensagem>\nEOF\n)"`.
3. Nunca usar `--no-verify` ou `--no-gpg-sign` a menos que explicitamente solicitado.

**Etapa 4: Selar evidencia SDD pendente (RF-14)**
1. Aplicavel apenas quando o commit contem o trabalho de uma tarefa SDD cujo
   `execution-result` ainda nao tem `commit_sha`.
2. A prova de execucao e verificada contra a arvore de trabalho viva, que acabou
   de deixar de existir com este commit. Sem selo, a evidencia da tarefa nao e
   re-auditavel depois do merge.
3. Executar, com o SHA recem-criado:
   `ai-spec seal-evidence <result.json> --prd-dir .specs/prd-<slug> --commit HEAD`
4. Conferir com `--verify`. A verificacao nao toca a arvore de trabalho e
   permanece valida indefinidamente.
5. Sem resultado SDD associado ao commit, pular esta etapa sem erro.

## Tratamento de Erros

* Se pre-commit hook falhar, diagnosticar a causa e corrigir antes de retentar — nao contornar o hook.
* Se nada estiver staged e o usuario quiser commitar tudo, perguntar quais arquivos incluir antes de `git add`.
* Nao fazer amend de commits ja publicados no remoto.
