---
name: github-diff-changelog-publisher
description: Lê diffs do GitHub para releases, pull requests, branches ou compares e produz um changelog claro e objetivo, com passo a passo do que mudou, breaking changes e impacto em relação à branch base padrão do repositório, priorizando main e usando master como fallback. Prioriza fontes e ferramentas nativas do GitHub, com fallback para web apenas quando o acesso ao GitHub for insuficiente. Exige aprovação explícita do usuário antes de buscar dados e exige que o usuário escolha e confirme se a publicação deve acontecer no próprio GitHub ou em uma página do Confluence via Atlassian MCP. Use quando precisar preparar release notes, resumos de PR, comparações entre branches ou relatórios publicáveis baseados em histórico do GitHub. Não use para repositórios fora do GitHub, release notes de marketing sem diff de origem ou publicação automática sem confirmação explícita do usuário.
---

# Publicador de Changelog a partir de Diff do GitHub

## Procedimentos

**Etapa 1: Validar o Escopo e Coletar Entradas**
1. Extraia a origem alvo informada pelo usuário exatamente como foi enviada. Aceite um destes tipos de origem:
   - tag de release ou URL de release
   - URL de pull request ou número da PR com repositório
   - nome de branch com repositório
   - URL de compare ou dois refs do mesmo repositório
2. Extraia o identificador do repositório quando estiver disponível no formato `{owner}/{repo}`.
3. Execute `python3 scripts/classify-github-target.py "[target]"` para classificar a origem e detectar campos ausentes.
4. Se o script reportar uma origem não suportada ou contexto de repositório ausente, peça ao usuário a informação faltante antes de continuar.
5. Determine o destino de publicação solicitado pelo usuário:
   - `github`
   - `confluence`
   - `draft-only`
6. Se o destino não estiver explícito, peça ao usuário para escolher um antes de continuar.

**Etapa 2: Confirmar Antes de Qualquer Execução Remota**
1. Resuma a operação pretendida em um bloco curto:
   - tipo de origem e alvo
   - repositório
   - estratégia da branch base de comparação: `main`, depois `master`
   - destino de publicação pretendido
2. Peça aprovação explícita do usuário antes de buscar qualquer dado no GitHub ou na web.
3. Não chame ferramentas do GitHub, APIs do GitHub, busca web ou Atlassian MCP até o usuário confirmar.

**Etapa 3: Resolver a Melhor Fonte**
1. Priorize acesso nativo ao GitHub nesta ordem:
   - GitHub MCP ou ferramentas conectoras do GitHub, quando disponíveis
   - URLs diretas do GitHub ou endpoints da API do GitHub
   - acesso web público às páginas do GitHub
2. Use busca web genérica apenas se o acesso nativo ao GitHub estiver indisponível, bloqueado ou sem o contexto de diff necessário.
3. Para releases:
   - obtenha as notas da release, a tag da release e o compare range, se existir
   - identifique a tag da release anterior quando a release atual não expuser um link de compare
4. Para pull requests:
   - obtenha título da PR, descrição, arquivos alterados, lista de commits e merge base, quando disponíveis
5. Para branches:
   - compare a branch com a branch base do repositório determinada na Etapa 4
6. Para compares:
   - obtenha o diff exato do compare e preserve os refs de origem e destino exatamente como informados

**Etapa 4: Determinar a Branch Base e a Janela de Comparação**
1. Resolva a branch padrão do repositório usando metadados nativos do GitHub quando possível.
2. Se a branch padrão for `main`, use `main`.
3. Se `main` não existir, use `master`.
4. Se nem `main` nem `master` existirem, use a branch padrão real e explicite esse desvio.
5. Para diffs de release, compare:
   - a tag da release atual com a tag da release anterior para mudanças entre releases
   - o alvo da release atual com `main` ou `master` quando o usuário pedir impacto de baseline em relação à branch principal
6. Para PRs e branches, compare com a branch base resolvida, a menos que o usuário forneça explicitamente outra base.

**Etapa 5: Extrair e Normalizar o Conjunto de Mudanças**
1. Colete a evidência mínima necessária para explicar o conjunto de mudanças:
   - título e resumo
   - commits
   - arquivos alterados
   - adições e remoções
   - arquivos renomeados ou removidos
   - labels, metadados de release ou notas de migração, quando existirem
2. Agrupe as mudanças em blocos claros usando a linguagem nativa do repositório quando possível, por exemplo:
   - funcionalidades
   - correções
   - refatorações
   - infraestrutura
   - dependências
   - documentação
3. Leia `references/breaking-change-rules.md` ao avaliar risco de compatibilidade.
4. Marque candidatos a breaking change de forma conservadora. Só classifique algo como breaking change quando o diff ou os metadados sustentarem essa conclusão.
5. Se a evidência estiver incompleta, rotule o item como `possível breaking change` e diga o que está faltando.

**Etapa 6: Produzir o Rascunho do Changelog**
1. Use `assets/changelog-template.md` como estrutura de saída.
2. Gere um changelog claro, objetivo e curto o suficiente para leitura rápida.
3. Inclua estas seções nesta ordem, salvo quando alguma estiver vazia:
   - título
   - resumo da origem
   - passo a passo do que mudou
   - breaking changes
   - notas de migração ou ação
   - impacto em relação a `main` ou `master`
   - metadados de publicação
4. Em `passo a passo do que mudou`, explique o fluxo das mudanças como uma sequência numerada de deltas concretos, e não como um dump bruto de arquivos.
5. Em `impacto em relação a main ou master`, informe:
   - se o diff está à frente, atrás ou divergente
   - o efeito operacional provável
   - áreas de risco como schema, API, config, auth ou remoções
6. Evite linguagem de marketing, especulação e enchimento.

**Etapa 7: Revisar com o Usuário Antes de Publicar**
1. Apresente o rascunho ou uma prévia concisa ao usuário.
2. Peça aprovação explícita do usuário antes de publicar em qualquer lugar.
3. Se o usuário quiser ajustes, revise o rascunho e pergunte novamente.
4. Não publique até o usuário confirmar ambos:
   - o conteúdo
   - o destino

**Etapa 8: Publicar no Destino Escolhido**
1. Se o usuário escolheu `github`, publique somente após confirmação usando o caminho nativo do GitHub que melhor se encaixar no contexto da origem:
   - atualização do corpo da release para uma release
   - comentário na PR ou atualização do corpo da PR para uma pull request
   - discussion, issue ou wiki do repositório apenas se o usuário pedir explicitamente esse alvo
2. Se o usuário escolheu `confluence`, publique somente após confirmação usando Atlassian MCP na página ou localização-pai selecionada pelo usuário.
3. Se o usuário escolheu `draft-only`, pare após retornar o texto final do changelog.
4. Depois de publicar, informe:
   - onde foi publicado
   - a URL alvo ou referência da página, quando disponível
   - quaisquer campos que não puderam ser publicados automaticamente

## Tratamento de Erros
* Se `scripts/classify-github-target.py` não conseguir classificar a entrada, peça ao usuário uma URL do GitHub, um par de refs ou `{owner}/{repo}` com o identificador da origem antes de continuar.
* Se o acesso nativo ao GitHub falhar, informe a falha brevemente e peça aprovação do usuário antes de cair para acesso web genérico.
* Se a branch padrão do repositório não estiver disponível, tente `main`, depois `master`, depois a branch padrão vinda dos metadados do repositório. Informe exatamente qual branch foi usada.
* Se não houver tag anterior confiável para comparar uma release, compare o alvo da release com a branch base resolvida e informe que a comparação é baseada em branch, não em release anterior.
* Se o diff for grande demais para leitura completa arquivo por arquivo, priorize metadados da release, resumo da PR, histórico de commits, arquivos mais alterados e arquivos relacionados a migração.
* Se a evidência for insuficiente para confirmar um breaking change, marque como possível, cite a evidência ausente e evite afirmar certeza indevida.
* Se o usuário aprovar o conteúdo mas não a publicação, retorne o changelog finalizado e pare sem publicar.
