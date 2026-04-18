---
name: pull-request
description: Gera, revisa, cria ou atualiza pull requests no GitHub para a branch atual com título e body em pt-BR, usando evidência de commits, diff e contexto do repositório para definir base branch, resumo das mudanças e modo de publicação. Use quando precisar abrir PR, atualizar uma PR existente, preparar título e descrição de PR, revisar se a PR atual representa corretamente as mudanças ou produzir um rascunho antes de publicar. Não use para commit semântico isolado, code review profundo, release notes completas, reescrita de histórico Git ou publicação automática sem contexto suficiente.
---

# Pull Request

<critical>Usar a evidência do branch diff, dos commits e do contexto do repositório para gerar título e body; não inventar escopo, base branch ou impacto.</critical>
<critical>Verificar se já existe PR aberta para a branch atual antes de criar uma nova.</critical>
<critical>Usar pt-BR no título e no body, salvo quando o repositório ou o usuário exigirem explicitamente outro idioma.</critical>
<critical>Quando o usuário não pedir publicação explícita, priorizar rascunho de título e body antes de criar ou editar a PR remotamente.</critical>

## Procedimentos

**Etapa 1: Coletar Contexto Local e Modo de Execução**
1. Obtenha a branch atual com `git branch --show-current`.
2. Se a branch atual estiver vazia, for `HEAD` destacada ou for a branch principal do repositório, retorne `needs_input` com diagnóstico objetivo.
3. Classifique a intenção do usuário em um destes modos:
   - `draft-only`
   - `create`
   - `update`
   - `create-or-update`
4. Se o pedido não deixar claro que a PR deve ser escrita no GitHub, use `draft-only` como padrão.
5. Obtenha o estado resumido do repositório com `git status -sb`.
6. Registre se existem commits locais ainda não enviados ao remoto; não faça `git push` automaticamente sem pedido explícito do usuário.

**Etapa 2: Resolver a Base Branch**
1. Execute `python3 scripts/resolve_pr_base.py --branch "<branch-atual>"`.
2. Leia `references/base-branch-rules.md` apenas se o resultado do script indicar ambiguidade, ausência de regra forte ou necessidade de fallback adicional.
3. Valide a base branch escolhida com evidência do repositório sempre que possível, por exemplo:
   - upstream configurado
   - branch padrão remota
   - relação de merge-base entre base candidata e `HEAD`
4. Se não houver evidência suficiente para determinar uma base confiável, retorne `needs_input` em vez de inferir uma base fraca.

**Etapa 3: Verificar Pré-condições de Publicação**
1. Se o modo for `create`, `update` ou `create-or-update`, verifique se `gh` está disponível e autenticado.
2. Se a branch atual não existir no remoto e o usuário não tiver pedido push explícito, pare em `needs_input` informando que a branch precisa ser publicada antes da criação da PR.
3. Se existir push pendente e o usuário pediu criação ou atualização remota, informe a necessidade de `git push` antes de prosseguir; só execute o push se isso tiver sido solicitado explicitamente.
4. Se o modo for `draft-only`, não exija autenticação do `gh`.

**Etapa 4: Verificar PR Existente**
1. Se o modo for `draft-only`, esta etapa é opcional; só consulte PR existente se isso ajudar a revisar ou alinhar um rascunho já aberto.
2. Se o modo permitir escrita remota, busque PR aberta para a branch atual com:
   - `gh pr list --head "<branch-atual>" --state open --json number,title,body,url,baseRefName,headRefName,isDraft`
3. Se houver uma PR aberta:
   - use `update` quando o modo for `create-or-update`
   - preserve o número e a URL para edição posterior
4. Se não houver PR aberta:
   - use `create` quando o modo for `create-or-update`
   - siga com geração de conteúdo para criação

**Etapa 5: Coletar Evidência das Mudanças**
1. Colete os commits relevantes com `git log <base>..HEAD --oneline --no-merges`.
2. Colete o diff resumido com `git diff <base>...HEAD --stat`.
3. Quando o título ou o body ainda estiverem ambíguos, leia também:
   - `git diff <base>...HEAD --name-only`
   - mensagens de commit completas entre a base e o `HEAD`
4. Extraia:
   - objetivo principal da PR
   - áreas ou módulos afetados
   - riscos ou breaking changes prováveis
   - evidência de testes, docs ou rollout quando existir
5. Leia `references/pr-content-rules.md` quando precisar decidir:
   - como titular a PR
   - quando usar ou não formato tipo Conventional Commit no título
   - como montar checklist e seção de contexto

**Etapa 6: Gerar Título e Body**
1. Leia `assets/pr-body-template.md` para estruturar a resposta.
2. Gere um título curto, específico e fiel ao objetivo principal da branch.
3. Use formato `type(scope-opcional): descrição` apenas quando a evidência e a convenção do repositório sustentarem esse padrão.
4. Se a convenção do repositório não estiver clara, prefira um título descritivo e objetivo em pt-BR, sem forçar Conventional Commits.
5. Monte o body com base na evidência observada, preenchendo apenas seções sustentadas pelos commits e pelo diff.
6. O checklist deve refletir ações reais ou pendências reais; não marcar itens que não possam ser confirmados.
7. Se houver breaking change, declare isso com linguagem objetiva e explicite o impacto esperado.

**Etapa 7: Validar o Conteúdo Antes da Escrita**
1. Verifique se o título:
   - representa um único objetivo principal
   - não contradiz os commits
   - não usa termos vagos como `ajustes`, `melhorias gerais` ou `update`
2. Verifique se o body:
   - resume o que mudou e por quê
   - não inventa testes, tickets ou validações
   - não repete o diff bruto
   - contém contexto suficiente para revisão humana
3. Se o conjunto de mudanças estiver largo demais para uma única PR coesa, aponte isso explicitamente e sugira split antes de publicar.

**Etapa 8: Criar, Atualizar ou Retornar Rascunho**
1. Se o modo final for `draft-only`, retorne o título e o body prontos sem escrever no GitHub.
2. Se o modo final for `create`, crie a PR com:
   - `gh pr create --base "<base>" --title "<titulo>" --body-file "<arquivo-temporario-ou-equivalente>"`
3. Se o modo final for `update`, edite a PR existente com:
   - `gh pr edit <number> --title "<titulo>" --body-file "<arquivo-temporario-ou-equivalente>"`
4. Se houver suporte a draft e o usuário pedir explicitamente, use a modalidade draft apropriada do `gh`.
5. Após a operação, informe:
   - se a PR foi criada, atualizada ou apenas preparada
   - base branch usada
   - título final
   - URL da PR, quando existir

## Formato de Saída
Use o esqueleto definido em `assets/pr-body-template.md` para o conteúdo da PR e retorne um resumo operacional curto neste formato:

```text
PR: <rascunho|criada|atualizada>
URL: <url-ou-n/a>
Base: <base-branch>
Título: <titulo-final>
```

## Tratamento de Erros
* Se `scripts/resolve_pr_base.py` indicar ambiguidade ou falta de evidência suficiente, não force uma base branch; peça confirmação do usuário.
* Se `gh` não estiver autenticado, informe `gh auth login` e pare sem tentar criar ou editar remotamente.
* Se a branch local não existir no remoto e o usuário pedir escrita remota, informe que é necessário publicar a branch antes da PR.
* Se não houver commits entre a base e o `HEAD`, retorne `needs_input` com o diagnóstico de que não existe delta útil para PR.
* Se o diff estiver muito grande ou misturar objetivos independentes, sugira dividir a branch ou reduzir o escopo da PR antes de publicar.
* Se a criação ou edição falhar por política do repositório, branch protegida ou template obrigatório, informe a falha e preserve o título e o body gerados para uso manual.
