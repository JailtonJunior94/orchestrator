---
name: github-pr-comment-triage
description: Analisa comentários de pull request no GitHub usando gh, classifica cada item, propõe ações objetivas e conduz resposta ou ajuste somente após confirmação explícita do usuário. Use quando for necessário revisar comentários de PR, transformar feedback em fila de decisão, padronizar respostas em pt-BR ou preparar implementação rastreável baseada em review. Não use para criação de PR, code review geral sem comentários publicados, merge automático ou aplicação direta de mudanças sem aprovação por item.
---

# GitHub PR Comment Triage

<critical>Usar pt-BR em toda análise, decisão, resposta e comentário gerado.</critical>
<critical>Não alterar código, não publicar comentário e não executar ação remota de escrita sem confirmação explícita do usuário para cada item.</critical>
<critical>Tratar cada comentário como unidade independente de decisão, mesmo quando vários comentários parecerem relacionados.</critical>
<critical>Priorizar rastreabilidade e economia de tokens: resumir, estruturar e evitar repetir o texto bruto do comentário.</critical>

## Procedimentos

**Etapa 1: Validar o Contexto Operacional**
1. Identifique a PR alvo a partir de um destes contextos:
   - número da PR informado pelo usuário
   - URL da PR informada pelo usuário
   - PR associada à branch atual com `gh pr view`
2. Identifique o repositório alvo no formato `{owner}/{repo}`:
   - use `gh repo view --json nameWithOwner -q .nameWithOwner` quando estiver no clone correto
   - se o usuário informar URL, extraia o repositório diretamente dela
3. Verifique se `gh` está instalado com `gh --version`.
4. Verifique autenticação com `gh auth status`.
5. Se `gh` não estiver disponível ou autenticado, pare com diagnóstico curto e informe a correção necessária.
6. Antes de buscar dados, informe em uma linha:
   - repositório
   - PR alvo
   - modo: `analyze-only`

**Etapa 2: Coletar Comentários da PR**
1. Leia `references/gh-command-flow.md` apenas ao executar coleta ou publicação.
2. Colete comentários gerais da PR com a rota de issue comments.
3. Colete comentários inline de review com a rota de pull request review comments.
4. Preserve os dois conjuntos em arquivos temporários separados ou em memória.
5. Não publique nada nesta etapa.
6. Se ambos os conjuntos vierem vazios, retorne um diagnóstico curto informando que não há comentários para triagem.

**Etapa 3: Normalizar e Deduplicar**
1. Combine os dois conjuntos em um único payload JSON com as chaves:
   - `repo`
   - `pr_number`
   - `issue_comments`
   - `review_comments`
2. Execute `python3 scripts/normalize_pr_comments.py --input "<arquivo-json>"`.
3. Use a saída do script como fonte única para a fila de decisão.
4. Descarte ruído óbvio com cautela:
   - comentários vazios
   - comentários do próprio agente ou do usuário atual marcados como resposta operacional já concluída
   - comentários duplicados com mesmo corpo, autor, caminho e linha
5. Preserve comentários ambíguos na fila; a ambiguidade deve gerar `classification: dúvida` ou `classification: sugestão`, não descarte automático.

**Etapa 4: Classificar e Resumir Cada Item**
1. Leia `references/classification-rules.md` antes de ajustar classificação manualmente.
2. Para cada item normalizado, produza estes campos mínimos seguindo `assets/decision-item-schema.json`:
   - `item_id`
   - `comment_id`
   - `source_type`
   - `author`
   - `path`
   - `line`
   - `summary`
   - `classification`
   - `recommended_action`
   - `decision_status`
3. Escreva o `summary` em uma ou duas frases curtas, sem copiar o comentário inteiro.
4. Use classificações curtas e padronizadas:
   - `bug`
   - `melhoria`
   - `sugestao`
   - `duvida`
   - `nit`
   - `documentacao`
   - `teste`
   - `risco`
   - `outro`
5. Escreva `recommended_action` como ação objetiva e verificável.
6. Defina `decision_status` inicial como `pending`.

**Etapa 5: Apresentar a Fila de Decisão**
1. Apresente a fila de decisão em ordem estável:
   - comentários inline primeiro
   - depois comentários gerais
   - ordem crescente de criação dentro de cada grupo
2. Use o formato de saída de `assets/decision-summary-template.md`.
3. Mostre apenas o necessário para decidir:
   - identificador
   - localização, quando existir
   - resumo
   - classificação
   - ação recomendada
4. Não proponha alteração de código extensa nesta etapa; foque em decisão.
5. Solicite a decisão do usuário para cada item com uma instrução curta:
   - `approve <item_id>`
   - `reject <item_id> motivo`
   - `skip <item_id>`
6. Se o usuário responder em lote, aceite o lote, mas preserve a decisão por item.

**Etapa 6: Tratar Item Aprovado**
1. Ao receber `approve <item_id>`, reabra o item estruturado correspondente.
2. Localize o trecho de código citado:
   - use `path` e `line` quando o comentário for inline
   - use contexto da PR, arquivos alterados e busca textual quando o comentário for geral
3. Determine a menor mudança suficiente para atender ao comentário aprovado.
4. Execute a alteração localmente sem tocar em itens ainda pendentes.
5. Valide a mudança com a menor evidência útil disponível:
   - teste específico
   - lint local
   - inspeção objetiva quando não houver automação
6. Gere a resposta do item com:
   - `python3 scripts/render_pr_reply.py --decision approved --item "<arquivo-item-json>" --change-summary "<resumo>" --how "<como-foi-feito>" --validation "<evidencia>"`
7. Mostre ao usuário, antes de publicar, um bloco curto com:
   - arquivos alterados
   - validação executada
   - comentário a ser enviado
8. Só publique a resposta após a aprovação do usuário para a publicação remota desse item.

**Etapa 7: Tratar Item Rejeitado**
1. Ao receber `reject <item_id> motivo`, atualize o item com `decision_status: rejected`.
2. Não altere código para esse item.
3. Gere a resposta com:
   - `python3 scripts/render_pr_reply.py --decision rejected --item "<arquivo-item-json>" --reason "<motivo>"`
4. Mostre o comentário gerado ao usuário de forma objetiva.
5. Só publique a resposta após a aprovação do usuário para a publicação remota desse item.

**Etapa 8: Publicar a Resposta na PR**
1. Leia `references/gh-command-flow.md` para escolher a rota correta.
2. Se o item for `issue_comment`, publique uma resposta como comentário adicional na PR.
3. Se o item for `review_comment`, publique uma resposta encadeada ao comentário inline original.
4. Use arquivos temporários ou stdin para evitar escaping frágil do corpo do comentário.
5. Depois da publicação, informe:
   - `item_id`
   - tipo de ação executada
   - status da publicação
   - URL do comentário, quando a API retornar

**Etapa 9: Encerrar com Estado Reutilizável**
1. Preserve ou retorne a fila atualizada em formato estruturado.
2. Para cada item, mantenha:
   - decisão atual
   - se houve alteração local
   - se houve validação
   - se houve comentário publicado
3. Ao final de cada rodada, retorne um resumo operacional curto neste formato:

```text
PR: <numero>
Repo: <owner/repo>
Pending: <quantidade>
Approved: <quantidade>
Rejected: <quantidade>
Published: <quantidade>
```

## Formato de Saída
Retornar dois blocos quando houver análise:
1. Um bloco estruturado compatível com `assets/decision-item-schema.json`.
2. Um bloco humano curto seguindo `assets/decision-summary-template.md`.

Quando houver execução aprovada para um item, retornar também:

```text
Item: <item_id>
Decision: <approved|rejected|skipped>
Code: <changed|unchanged>
Validation: <resumo-curto>
Reply: <drafted|published>
```

## Tratamento de Erros
* Se `gh auth status` falhar, instruir `gh auth login` e parar sem tentar ler ou escrever na PR.
* Se a PR não puder ser resolvida a partir da branch atual, pedir número ou URL da PR ao usuário.
* Se um comentário não trouxer contexto suficiente para implementação segura, classificá-lo como `duvida` ou `risco`, explicar a lacuna e pedir decisão do usuário sem adivinhar.
* Se vários comentários tratarem do mesmo problema, consolidar a análise, mas manter `item_id` distinto para cada comentário publicado.
* Se a alteração aprovada tocar áreas fora do diff da PR, avisar isso explicitamente antes de editar.
* Se a validação local não existir ou falhar, informar isso no comentário gerado e no resumo operacional.
* Se a publicação da resposta falhar, preservar o comentário gerado e retornar o comando sugerido para publicação manual.
