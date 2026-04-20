# ADR-003: Paridade semantica vs textual entre artefatos de agentes

**Status:** Aceita  
**Data:** 2024-01-01  
**Autores:** JailtonJunior94

---

## Contexto

O harness suporta multiplos agentes de IA (Claude, Gemini, Codex, Copilot), cada um com seus proprios arquivos de governanca (CLAUDE.md, GEMINI.md, CODEX.md, etc.). Esses artefatos evoluem de forma independente e podem ter formatacoes e convencoes distintas.

O problema e: como verificar que os artefatos de diferentes agentes estao em sincronia, sem exigir que sejam textualmente identicos?

Restricoes relevantes:
- Cada agente tem convencoes proprias de formatacao e instrucao.
- Diff textual produziria falsos positivos constantes.
- A verificacao deve ser automatizavel e executavel em CI.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Paridade semantica por invariantes (escolhida) | Tolerante a evolucao independente; foca no que importa | Invariantes devem ser definidas e mantidas manualmente |
| Diff textual direto | Simples de implementar | Falsos positivos para qualquer diferenca de formatacao |
| Golden files identicos | Garantia total de sincronia | Impossivel manter para agentes com convencoes distintas |
| Ignorar paridade | Zero esforco | Divergencia silenciosa degrada qualidade da governanca |

## Decisao

Decidimos implementar verificacao de paridade por **invariantes semanticas** no pacote `internal/parity`.

Invariantes sao afirmacoes sobre conteudo (ex: "todos os artefatos devem mencionar Conventional Commits") que cada artefato deve satisfazer, independentemente de como exprima essa informacao. A verificacao falha quando um artefato viola uma invariante, nao quando difere textualmente de outro.

## Consequencias

### Positivas
- Cada agente pode evoluir seu artefato de governanca de forma independente sem quebrar paridade.
- CI detecta derivacoes reais (remocao de secoes criticas) sem falsos positivos de formatacao.
- As invariantes servem como especificacao viva da governanca obrigatoria.

### Negativas / Riscos
- Invariantes devem ser definidas e atualizadas manualmente quando a governanca muda.
- Falsos negativos sao possiveis: um artefato pode satisfazer a invariante textualmente sem aplicar o principio de fato.
- A abordagem e mais complexa de implementar que um diff textual simples.

### Neutras / Observacoes
- O pacote `internal/parity` e separado do `internal/specdrift`, que detecta drift entre specs de um mesmo agente ao longo do tempo.

## Referencias

- `internal/parity/` — implementacao de verificacao de paridade semantica
- `internal/specdrift/` — deteccao de drift entre versoes de specs
- `AGENTS.md` — lista de artefatos de governanca por agente
