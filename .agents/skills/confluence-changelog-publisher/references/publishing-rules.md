# Regras de Publicacao

Use este arquivo somente quando houver duvida sobre criar, atualizar ou pedir confirmacao extra.

Escolha `create` quando:

- o usuario pedir uma pagina nova
- o destino for uma pagina-pai ou a raiz do space
- nao existir pagina confirmada para atualizar

Escolha `update` quando:

- o usuario fornecer ID ou URL de pagina existente
- o usuario confirmar explicitamente que quer sobrescrever ou complementar uma pagina existente

Escolha `decide-after-search` quando:

- o usuario souber o `space` e o `title`, mas nao souber se a pagina ja existe
- houver risco de duplicidade e a pesquisa no Confluence for necessaria

Peça confirmacao adicional quando:

- houver mais de um resultado com o mesmo titulo
- o usuario mudar `space`, `title` ou localizacao depois do preview
- a pagina existente tiver escopo amplo e o update puder afetar outros times

Nao publique automaticamente quando:

- o usuario ainda nao aprovou o corpo final
- o destino ainda estiver ambiguo
- a ferramenta nao retornar um alvo de pagina confiavel
