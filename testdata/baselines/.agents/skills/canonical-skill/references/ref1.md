# Referencia: Modelo de Custo -- Definicoes Formais

## Formula de Estimativa de Tokens

```
tokens_est = round(len(text) / 3.5)
```

Onde `len(text)` e o numero de bytes do arquivo (caracteres ASCII, 1 byte cada).
O arredondamento e matematico (half-up), conforme `math.Round` em Go.

Esta formula e uma estimativa operacional. Nao corresponde ao tokenizador
real do provedor de IA (tiktoken, SentencePiece, etc.).

## Thresholds por Ferramenta

| Ferramenta | Budget (tokens est.) | Janela de Contexto |
|-----------|---------------------|--------------------|
| claude    | 70.000              | Grande             |
| gemini    | 4.000               | Grande             |
| codex     | 13.000              | Medio              |
| copilot   | 2.000               | Restrito           |

Os budgets sao conservadores: representam o limite recomendado para
governanca, nao o limite tecnico maximo do provedor.

## Tolerancia do Gate de Regressao

O gate aceita crescimento de ate 25% acima do baseline sem falhar.
Este valor foi escolhido para:

- Absorver variacao normal de conteudo entre versoes (adicao de exemplos,
  correcoes de texto, expansao de secoes menores).
- Detectar crescimento silencioso maior que 25%, que indica provavel
  adicao nao planejada de artefatos ou expansao sem revisao de custo.

Para alterar a tolerancia, editar o campo `tolerance_pct` no arquivo
`testdata/baselines/cost-baseline.json` e documentar a justificativa.

## Atualizacao do Baseline

O baseline deve ser atualizado quando:
1. Um release e publicado com mudancas intencionais nos artefatos canonicos.
2. A tolerancia se mostrar muito restritiva para o ritmo do projeto.

Nunca atualizar o baseline para suprimir falhas causadas por crescimento
nao planejado. Investigar a causa antes de atualizar.
