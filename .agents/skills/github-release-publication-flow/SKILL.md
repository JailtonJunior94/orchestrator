---
name: github-release-publication-flow
description: Orquestra a leitura de diffs no GitHub, a geração de changelog objetivo e a publicação aprovada pelo usuário no próprio GitHub ou no Confluence via Atlassian MCP. Centraliza coleta de entradas, confirmações obrigatórias, escolha de destino e rastreabilidade final, reaproveitando uma abordagem guiada para análise de release, pull request, branch ou compare. Use quando precisar conduzir o fluxo completo de changelog desde a origem no GitHub até a publicação revisada. Não use para publicação automática sem confirmação explícita, para repositórios fora do GitHub ou para conteúdo que já esteja pronto e precise apenas de uma publicação isolada.
---

# Fluxo de Publicacao de Changelog a partir do GitHub

## Procedimentos

**Etapa 1: Coletar e Validar o Escopo**
1. Extraia a origem do GitHub exatamente como o usuário informou:
   - release
   - pull request
   - branch
   - compare
2. Extraia ou solicite o destino de publicação:
   - `github`
   - `confluence`
   - `draft-only`
3. Execute `python3 scripts/validate-publication-flow.py --target "[target]" --destination "[destination]"`.
4. Se a validação falhar, corrija com o usuário antes de continuar.

**Etapa 2: Confirmar a Analise Antes de Buscar Dados**
1. Resuma em um bloco curto:
   - origem alvo
   - destino pretendido
   - estratégia de comparação: `main`, depois `master`
2. Peça aprovação explícita do usuário antes de buscar qualquer dado remoto.
3. Não chame GitHub, web ou Atlassian MCP até o usuário confirmar.

**Etapa 3: Gerar o Changelog a partir do GitHub**
1. Siga `../github-diff-changelog-publisher/SKILL.md` para:
   - classificar a origem
   - priorizar GitHub e só cair para web quando necessário
   - comparar com `main` e fallback para `master`
   - gerar o changelog objetivo
2. Preserve o resultado como rascunho de trabalho.
3. Se o destino for `draft-only`, apresente o resultado e encerre sem publicar.

**Etapa 4: Revisar o Conteudo com o Usuario**
1. Mostre uma prévia curta do changelog com:
   - título
   - resumo da origem
   - principais mudanças
   - breaking changes
2. Peça aprovação explícita do usuário para o conteúdo.
3. Se o usuário solicitar ajustes, revise o changelog e repita esta etapa.
4. Não publique até o usuário aprovar o conteúdo final.

**Etapa 5: Resolver o Destino Final**
1. Se o destino for `github`, determine o alvo mais adequado conforme a origem:
   - corpo da release
   - comentário ou corpo da PR
   - outro alvo explícito no repositório, se o usuário pedir
2. Se o destino for `confluence`, siga `../confluence-changelog-publisher/SKILL.md` para coletar:
   - `space`
   - `title`
   - modo de publicação
   - localização da página
3. Se o destino ainda estiver ambíguo, peça confirmação do usuário antes de seguir.

**Etapa 6: Confirmar a Publicacao**
1. Resuma a ação final em um bloco curto:
   - destino
   - alvo exato de publicação
   - título final
2. Peça aprovação explícita do usuário antes de escrever no GitHub ou no Confluence.
3. Não publique até o usuário confirmar esta etapa.

**Etapa 7: Publicar no Destino Escolhido**
1. Se o destino for `github`, publique no alvo confirmado usando o caminho nativo do GitHub mais adequado ao contexto.
2. Se o destino for `confluence`, execute a publicação conforme `../confluence-changelog-publisher/SKILL.md`.
3. Preserve o conteúdo final aprovado sem reescritas inesperadas.
4. Após publicar, informe:
   - onde foi publicado
   - URL ou referência do alvo
   - se foi criação, atualização ou comentário

**Etapa 8: Encerrar com Rastreabilidade**
1. Retorne um resumo curto do fluxo executado:
   - origem
   - base de comparação usada
   - destino final
   - status da publicação
2. Se a publicação não ocorrer, retorne o changelog final pronto para uso manual.

## Tratamento de Erros
* Se `scripts/validate-publication-flow.py` falhar, corrija origem ou destino com o usuário antes de seguir.
* Se a análise do GitHub não puder ser concluída com fontes nativas, peça aprovação antes de usar fallback web.
* Se o usuário aprovar a análise, mas não aprovar o conteúdo final, mantenha o changelog como rascunho e não publique.
* Se o destino for `confluence` e os dados de página não estiverem completos, siga o fluxo de resolução da skill `confluence-changelog-publisher`.
* Se a publicação falhar no destino escolhido, preserve o changelog final e reporte a falha sem repetir tentativas destrutivas.
