# ADR-008: Paridade multi-tool com invariantes semanticas

**Status:** Aceita  
**Data:** 2026-04-20  
**Autores:** -

---

## Contexto

O projeto suporta 4 agentes (Claude, Gemini, Codex, Copilot) com artefatos de configuracao em formatos distintos. Diff textual entre artefatos geraria falsos positivos por causa de diferencas de formato, estrutura e convencoes especificas de cada ferramenta. E necessario detectar drift semantico — quando um agente perde uma regra ou referencia que os demais possuem — sem exigir formato identico.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Diff textual entre artefatos | Implementacao simples, deteccao direta | Falsos positivos por diferencas de formato, inviavel na pratica |
| Checksums de arquivos | Detecta qualquer mudanca em arquivo | Nao detecta drift semantico, qualquer edicao dispara alerta |
| Nenhuma validacao de paridade | Zero overhead, zero complexidade | Drift silencioso entre agentes, inconsistencia nao detectada |

## Decisao

Decidimos implementar 29 invariantes semanticas com 3 niveis de enforcement:

- **Common:** obrigatorio para todos os agentes — falha bloqueia
- **ToolSpecific:** obrigatorio para ferramenta especifica — falha bloqueia apenas o agente afetado
- **BestEffort:** registrado mas nao bloqueia — usado para limitacoes tecnicas conhecidas (ex: Copilot CLI)

Cada invariante verifica a presenca de um conceito semantico (ex: "referencia a error-handling carregada") no artefato do agente, independente do formato textual.

## Consequencias

### Positivas
- Cada agente evolui independentemente sem falsos positivos
- Invariantes detectam drift semantico real (regra ausente, referencia removida)
- 3 niveis permitem pragmatismo com limitacoes tecnicas conhecidas

### Negativas / Riscos
- Novos artefatos de agente requerem definicao de novos invariantes
- Manutencao do conjunto de 29 invariantes a medida que governanca evolui
- Risco de invariantes ficarem desatualizadas se artefatos mudarem sem atualizar parity

### Neutras / Observacoes
- Complementa ADR-003 que define o principio; este ADR detalha a implementacao
- Numero de invariantes pode crescer ou encolher conforme necessidade

## Referencias

- `internal/parity/parity.go` — implementacao das invariantes
- `internal/parity/parity_test.go` — testes das invariantes
- `docs/adr/003-paridade-semantica.md` — ADR fundacional sobre paridade semantica
