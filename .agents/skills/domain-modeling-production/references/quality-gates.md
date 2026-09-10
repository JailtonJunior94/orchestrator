# Gates de Prontidao

O modelo so esta pronto para handoff quando todos os gates abaixo estiverem atendidos ou explicitamente aceitos como risco residual pelo usuario.

## Gate 1 - Problema, objetivo e escopo
- Problema atual descrito com impacto claro.
- Objetivo de negocio separado do objetivo de modelagem.
- Inclui e exclui definidos.

## Gate 2 - Linguagem ubiqua
- Existe termo canonico para o fluxo, objeto ou decisao principal.
- Sinonimos ambiguos foram proibidos ou contextualizados.
- O mesmo termo nao muda de sentido dentro do mesmo contexto.

## Gate 3 - Workflow e comportamento
- Gatilho, passos relevantes e ponto de decisao do workflow principal foram definidos.
- Existe ao menos um comando material e um evento material, quando aplicavel.
- A modelagem nao se reduz a CRUD quando o problema possui regra ou decisao real.

## Gate 4 - Regras e invariantes
- Regra de negocio central esta explicita.
- Existe pelo menos uma invariante nao negociavel.
- Estados ilegais foram descritos e bloqueados por regra, transicao ou erro.

## Gate 5 - Tipos conceituais e ownership
- Entidades, value objects e agregados foram definidos apenas quando necessarios.
- Ownership e limite de consistencia estao claros.
- Nao ha agregado gigante sem justificativa, nem fragmentacao artificial.

## Gate 6 - Fronteiras e compatibilidade
- Bounded contexts e fronteiras externas foram descritos.
- Contratos externos nao foram confundidos com o modelo interno.
- Quando houver codebase existente, evidencias confirmadas citam `path:linha` ou o caso foi marcado como `greenfield`/`nao aplicavel`.

## Gate 7 - Erros, operacao e observabilidade
- Erros de dominio materiais foram nomeados.
- Existe postura minima de observabilidade e contingencia.
- O modelo informa o que precisa ser auditavel ou rastreavel.

## Gate 8 - Economia e custo
- Existe justificativa explicita para novos contextos, eventos, agregados ou politicas.
- O modelo escolheu o menor desenho que preserva as regras relevantes.
- Drivers de custo residual foram identificados.

## Gate 9 - Handoff
- O modelo deixa claro o que pode seguir para discovery tecnico, backlog ou implementacao.
- Itens em aberto nao foram escondidos em secoes cosmeticas.
- Nenhum risco material foi tratado como detalhe futuro sem registro.
