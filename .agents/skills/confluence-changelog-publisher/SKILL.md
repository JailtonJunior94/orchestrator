---
name: confluence-changelog-publisher
description: Publica changelogs e resumos técnicos em páginas do Confluence por meio de Atlassian MCP, com coleta guiada de space, título, localização da página e modo de publicação. Exige aprovação explícita do usuário antes de consultar páginas existentes e antes de criar ou atualizar conteúdo. Use quando precisar publicar changelogs, release notes técnicas, resumos de PR ou relatórios de mudança no Confluence com rastreabilidade e confirmação humana. Não use para publicação automática sem confirmação explícita, para destinos fora do Confluence ou para geração de conteúdo sem um texto-base já preparado.
---

# Publicador de Changelog no Confluence

## Procedimentos

**Etapa 1: Validar Entradas de Publicação**
1. Extraia o conteúdo a ser publicado ou a referência para o texto final já aprovado.
2. Extraia ou solicite os campos mínimos de destino:
   - `space`
   - `title`
   - modo de publicação: `create`, `update` ou `decide-after-search`
3. Extraia, quando disponível, um destes localizadores:
   - URL da página existente
   - ID da página existente
   - título da página-pai
   - indicação explícita de publicação na raiz do espaço
4. Execute `python3 scripts/validate-confluence-target.py --space "[space]" --title "[title]" --mode "[mode]" [--page-id "..."] [--parent-title "..."] [--root]`.
5. Se a validação falhar, corrija os campos com o usuário antes de continuar.

**Etapa 2: Confirmar Antes de Consultar o Confluence**
1. Resuma a operação pretendida em um bloco curto:
   - space
   - título da página
   - modo de publicação
   - localização desejada: página existente, página-pai ou raiz
2. Peça aprovação explícita do usuário antes de consultar páginas existentes, pesquisar destinos ou chamar Atlassian MCP.
3. Não consulte o Confluence nem chame Atlassian MCP até o usuário confirmar.

**Etapa 3: Resolver o Destino da Página**
1. Se o usuário forneceu URL ou ID da página, trate esse alvo como destino prioritário para `update`.
2. Se o modo for `decide-after-search`, pesquise páginas com o mesmo título no `space` informado.
3. Se houver múltiplas páginas candidatas, mostre opções curtas ao usuário e peça que escolha antes de continuar.
4. Se o usuário forneceu página-pai, resolva essa página antes de criar o conteúdo.
5. Se o usuário escolheu publicar na raiz do espaço, registre isso explicitamente no plano de publicação.

**Etapa 4: Preparar o Conteúdo para Publicação**
1. Leia `assets/confluence-page-template.md` para estruturar o conteúdo quando o texto-base precisar de padronização mínima.
2. Preserve o conteúdo aprovado pelo usuário sem reescrever desnecessariamente.
3. Garanta que o corpo final contenha, quando aplicável:
   - título
   - resumo
   - changelog ou corpo principal
   - breaking changes
   - notas de migração ou ação
   - metadados de origem
4. Leia `references/publishing-rules.md` quando precisar decidir entre criar, atualizar ou pedir confirmação adicional.

**Etapa 5: Revisar com o Usuário Antes de Escrever**
1. Mostre um preview curto com:
   - título final
   - destino resolvido
   - modo de publicação
   - início do corpo ou resumo do conteúdo
2. Peça aprovação explícita do usuário antes de criar ou atualizar a página.
3. Não escreva no Confluence até o usuário confirmar.

**Etapa 6: Publicar pelo Atlassian MCP**
1. Se o modo for `create`, crie a página no `space` e no local resolvido.
2. Se o modo for `update`, atualize somente a página confirmada pelo usuário.
3. Se o modo for `decide-after-search`, só crie ou atualize depois que o usuário escolher o destino final.
4. Preserve o título confirmado e use o corpo final aprovado.
5. Após a publicação, informe:
   - título publicado
   - space
   - página criada ou atualizada
   - URL ou referência da página, quando disponível

**Etapa 7: Encerrar com Rastreabilidade**
1. Informe se o conteúdo foi criado, atualizado ou mantido como rascunho.
2. Registre qualquer limitação observada, como página-pai não encontrada, múltiplos resultados ambíguos ou campos não resolvidos.
3. Se a publicação não acontecer, retorne o conteúdo final pronto para uso manual.

## Tratamento de Erros
* Se `scripts/validate-confluence-target.py` falhar, corrija espaço, título, modo ou localização com o usuário antes de continuar.
* Se o usuário não fornecer conteúdo final, solicite ou reutilize um changelog já aprovado antes de seguir para publicação.
* Se o Confluence retornar múltiplas páginas com títulos semelhantes, não escolha automaticamente; peça confirmação do usuário.
* Se a página-pai não puder ser resolvida, ofereça publicar na raiz do espaço ou pedir outro destino.
* Se a criação ou atualização falhar, informe a falha com objetividade, preserve o corpo final e pare sem repetir tentativas destrutivas.
* Se o usuário aprovar a consulta, mas não aprovar a escrita, encerre com o conteúdo preparado sem publicar.
