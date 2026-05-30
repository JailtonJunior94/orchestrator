# Protocolo de Múltipla Escolha

<!-- TL;DR
Protocolo canônico para resolver ambiguidade material com o humano: 2–5 opções numeradas, primeira marcada "(Recomendado)", uma pergunta por turno, gatilho = ambiguidade material (não decisões triviais).
Keywords: múltipla escolha, ambiguidade, decisão, recomendado, esclarecimento, pergunta, opções, fatiamento, severidade
Load complete when: a skill precisa decidir entre caminhos materialmente distintos (escopo, arquitetura, fatiamento, severidade de borda) e a escolha errada gera retrabalho ou regressão.
-->

- Rule ID: R-MC-001
- Severidade: hard
- Escopo: Skills de planejamento e revisão que confrontam decisões materialmente ambíguas com o humano.

## Objetivo

Reduzir retrabalho e suposições silenciosas: quando uma decisão é materialmente ambígua,
oferecer opções estruturadas em vez de assumir um caminho ou fazer perguntas abertas vagas.

## Gatilho (quando aplicar)

Aplicar **somente** em **ambiguidade material** — quando caminhos distintos levam a resultados
significativamente diferentes e a escolha errada custa retrabalho, regressão ou re-planejamento.

Exemplos de ambiguidade material:
- Escopo/fronteira de uma funcionalidade (o que entra/sai).
- Decisão de arquitetura com trade-offs (ex.: armazenamento, estratégia de cache).
- Fatiamento de tarefas (granularidade, ordem, paralelismo).
- Severidade de borda em revisão (um achado é `[HARD]` bloqueante ou `soft`?).

**Não** aplicar em decisões triviais, reversíveis e de baixo custo (gera ruído).
Nesses casos, decidir, registrar a suposição e seguir.

## Formato (obrigatório)

1. **2 a 5 opções**, numeradas.
2. A primeira opção é a recomendada e leva o sufixo **"(Recomendado)"**.
3. Cada opção em uma linha, com uma frase curta de consequência/trade-off.
4. **Uma pergunta por turno** — nunca empacotar múltiplas decisões na mesma pergunta.
5. Texto curto e objetivo; sem catch-all "Outro" (o humano pode responder livremente).

### Template

```
<pergunta materialmente ambígua em uma frase>

1. <opção A> (Recomendado) — <consequência/trade-off>
2. <opção B> — <consequência/trade-off>
3. <opção C> — <consequência/trade-off>
```

## Integração por skill

| Skill | Ponto de decisão |
|-------|------------------|
| `create-prd` | escopo/fronteira e objetivos quando o pedido é ambíguo |
| `create-technical-specification` | trade-off de arquitetura/interface material |
| `create-tasks` | fatiamento (granularidade/ordem/paralelismo) ambíguo |
| `review` | severidade de borda de um achado (`[HARD]` vs `soft`) |

## Anti-padrões

- Perguntas abertas vagas ("o que você quer fazer?") sem opções.
- Empacotar várias decisões em uma só pergunta.
- Oferecer múltipla escolha para decisões triviais (ruído).
- Omitir a recomendação — sempre indicar a opção recomendada e por quê.
