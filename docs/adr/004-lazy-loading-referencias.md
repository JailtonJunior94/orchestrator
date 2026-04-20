# ADR-004: Lazy-loading de referencias em skills

**Status:** Aceita  
**Data:** 2024-01-01  
**Autores:** JailtonJunior94

---

## Contexto

Skills do harness (ex: `go-implementation`, `agent-governance`) possuem documentos de referencia extensos em seus subdiretorios (`references/`). Esses documentos incluem guias de estilo, regras de arquitetura, padroes de teste e politicas de seguranca.

Carregar todos esses documentos no contexto do agente a cada invocacao de skill teria custo elevado de tokens, especialmente para skills com multiplos arquivos de referencia.

A questao e: como disponibilizar referencias ricas sem sobrecarregar o contexto de token por padrao?

Restricoes relevantes:
- Custo de tokens e uma restricao operacional real para usuarios.
- Referencias devem estar disponiveis quando necessarias, sem etapa manual adicional.
- O modelo de instrucao deve ser claro o suficiente para que o agente saiba quando carregar.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Lazy-loading sob demanda (escolhida) | Reduz custo de tokens por padrao; carrega apenas o necessario | Complexidade na instrucao de carregamento; risco de referencia nao carregada |
| Carregamento completo sempre | Garantia total de contexto disponivel | Custo alto de tokens; latencia aumentada; contexto poluido |
| Carregamento por tag/categoria | Granularidade intermediaria | Complexidade de indexacao e selecao de tags |
| Referencias inline no SKILL.md | Sem etapa adicional de carregamento | SKILL.md cresce demais; dificulta manutencao |

## Decisao

Decidimos adotar lazy-loading: o `SKILL.md` de cada skill descreve quais referencias existem e em quais situacoes devem ser carregadas. O agente e responsavel por ler os arquivos de `references/` sob demanda, conforme as instrucoes do SKILL.md.

O carregamento e acionado por instrucoes explicitas no SKILL.md do tipo "antes de implementar X, leia `references/Y.md`".

## Consequencias

### Positivas
- Custo de tokens reduzido por padrao: agentes nao carregam referencias irrelevantes para a tarefa.
- SKILL.md permanece legivel e focado na orquestracao.
- Referencias podem ser atualizadas sem alterar o SKILL.md.

### Negativas / Riscos
- Complexidade na autoria do SKILL.md: instrucoes de carregamento devem ser precisas.
- Risco de referencia nao carregada quando necessaria, caso a instrucao de carregamento seja ambigua.
- Agentes menos capazes podem ignorar a instrucao de lazy-load.

### Neutras / Observacoes
- O modelo e consistente com a politica de precedencia descrita em `.claude/rules/governance.md`.
- Skills com referencias criticas devem incluir instrucao de carregamento obrigatorio no inicio do SKILL.md.

## Referencias

- `.agents/skills/go-implementation/SKILL.md` — exemplo de skill com referencias lazy-loaded
- `.agents/skills/go-implementation/references/` — referencias de implementacao Go
- `.agents/skills/agent-governance/references/` — referencias de governanca
- `.claude/rules/governance.md` — politica de precedencia de regras
