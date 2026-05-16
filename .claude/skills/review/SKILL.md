---
name: review
version: 1.2.0
description: Revisa um diff de código quanto a correção, segurança, regressões e testes faltantes usando regras específicas do repositório. Use quando uma branch ou diff local precisar de revisão no estilo dono do código antes de merge ou fechamento de tarefa. Não use para implementação, planejamento de produto ou limpeza apenas de estilo.
---

# Revisar

## Procedimentos

**Etapa 1: Carregar contexto mínimo**

1. Aplicar guard de profundidade quando disponível, resolvendo `check-invocation-depth.sh` em cascata `.agents/lib/` → `scripts/lib/` (B1): `for d in .agents/lib scripts/lib; do [ -r "$d/check-invocation-depth.sh" ] && { source "$d/check-invocation-depth.sh" || true; break; }; done`. Em harness instalado, o script aborta com mensagem própria se o limite for atingido. Se nenhum dos caminhos existir, seguir.
2. Determinar escopo do diff:
   - Se `AI_REVIEW_PRIOR_SHA` estiver definido (rodada pós-`bugfix`), revisar apenas `git diff "$AI_REVIEW_PRIOR_SHA"..HEAD` — somente o delta da remediação, não o PR inteiro.
   - Caso contrário, usar a base apropriada (ex.: `git diff --merge-base origin/main`) restrita aos arquivos efetivamente alterados.
3. Aplicar orçamento de revisão. Se exceder e nenhum recorte for possível, retornar `BLOCKED` pedindo fatiamento:
   - `AI_REVIEW_MAX_FILES` (default 8)
   - `AI_REVIEW_MAX_DIFF_LINES` (default 400)
   - Acima do teto, abrir apenas `git diff --stat` + `git diff --name-only` e amostrar arquivos por categoria de risco antes de carregar conteúdo completo.
4. Ler `prd.md`, `techspec.md` ou arquivo de tarefa **somente quando** o diff toca arquivo citado neles ou a tarefa ativa aponta o documento.

**Etapa 2: Carregar referências sob gatilho**

Carregar cada referência apenas se o gatilho correspondente ocorrer no diff. Sem gatilho, não carregar.

1. Detectar a linguagem majoritária do diff pela extensão dominante dos arquivos alterados:
   - `.go` → `go`
   - `.ts`, `.tsx`, `.js`, `.jsx` → `node`
   - `.py` → `python`
   - Outra ou indefinida → fallback `go`

2. Carregar o arquivo de gatilhos correspondente em `.agents/skills/agent-governance/triggers/<lang>.yaml`.
   O schema é `{triggers: [{ref, patterns: []}]}`. Cada entrada lista a referência a carregar
   e os padrões que a disparam. Fallback para `go.yaml` em linguagem desconhecida ou vazia.

3. Para cada entrada do YAML carregado: se qualquer padrão ocorrer no diff, carregar a referência indicada.
   Sem gatilho, não carregar.

Confirmar o contrato de carga base definido em `AGENTS.md` quando ele existir; quando ausente, seguir com as regras desta skill como fallback explícito.

**Etapa 3: Revisar como dono do código**

1. Priorizar correção, segurança, regressões de comportamento, testes faltantes e lacunas de evidência.
2. Verificar a mudança contra o comportamento pretendido, não apenas o estilo local.
3. Conferir se as validações são suficientes para o nível de risco.
4. Tratar observações apenas de estilo como secundárias, a menos que escondam defeito real.

**Etapa 4: Produzir achados antes do veredito**

1. Atribuir severidade canônica a cada achado: `critical`, `high`, `medium`, `low`.
2. Incluir referência de arquivo, linha quando aplicável, impacto curto e dica de correção.
3. Para bugs acionáveis, emitir lista no formato `.agents/skills/agent-governance/references/bug-schema.json` para consumo da skill `bugfix`.
4. Sem achados: dizer explicitamente e registrar riscos residuais e lacunas de teste.

**Etapa 5: Veredito determinístico**

Mapeamento severidade → veredito (uso obrigatório):

| Condição | Veredito |
|---|---|
| Faltam diff, contexto necessário ou evidência de validação | `BLOCKED` |
| Há ao menos um achado `critical` ou `high` | `REJECTED` |
| Apenas achados `medium` ou `low` | `APPROVED_WITH_REMARKS` |
| Sem achados | `APPROVED` |

Se o chamador estiver em fluxo de remediação (`AI_REMEDIATION=1` ou `AI_REVIEW_PRIOR_SHA` definido) e houver bugs no formato canônico, instruir explicitamente o uso da skill `bugfix` antes de nova rodada de revisão.

**Etapa 6: Output mínimo estruturado**

Retornar bloco contendo, no mínimo:

- `verdict`: um dos quatro valores canônicos
- `files_reviewed`: lista de caminhos efetivamente lidos
- `refs_loaded`: lista de referências carregadas (vazia quando nenhuma foi disparada)
- `findings`: lista de `{severity, file, line, impact, fix_hint}`
- `residual_risks`: lista
- `validations_run`: comandos de validação executados ou consultados

## Tratamento de Erros

* Se nenhum diff ou conjunto de arquivos alterados estiver disponível, retornar `BLOCKED` e solicitar o alvo de revisão faltante.
* Se o orçamento de diff (`AI_REVIEW_MAX_FILES`, `AI_REVIEW_MAX_DIFF_LINES`) for excedido e o fatiamento não for viável, retornar `BLOCKED` com pedido explícito de fatiamento.
* Se o repositório tiver alterações sujas não relacionadas, restringir a revisão ao diff pretendido e explicitar a incerteza quando esse isolamento não for possível.
* Se a revisão depender de comportamento externo ou documentação que possa ter mudado, verificar em fontes primárias (código upstream, docs locais, testes existentes) antes de apontar um defeito.
