# Principios de Modelagem

Use estes principios para evitar complexidade acidental e modelagem fraca.

## 1. Modelar intencao antes de estrutura
- Comecar por decisao, comando, evento, regra e consequencia.
- Nao comecar por tabela, endpoint, payload, ORM, fila ou classe.

## 2. Tornar estados ilegais explicitamente impossiveis
- Se um estado nao pode existir, registrar isso como invariante, regra de transicao, erro de dominio ou combinacao desses mecanismos.
- Nao esconder regra critica em comentario, runbook ou "o front valida".

## 3. Separar comando de evento
- Comando representa intencao e pode falhar.
- Evento representa fato ocorrido e nao deve carregar semantica de desejo.
- Se a equipe usa o mesmo nome para os dois, renomear explicitamente.

## 4. Preservar linguagem ubiqua
- O mesmo termo deve carregar o mesmo significado dentro do mesmo bounded context.
- Quando houver sinonimo ambiguo, marcar como proibido.
- Quando o mesmo termo mudar de sentido entre contextos, nomear a diferenca.

## 5. Tratar dominio como fluxo de decisao
- Workflow principal precisa mostrar gatilho, passos relevantes, ponto de decisao, efeitos e falhas.
- CRUD puro so e aceitavel quando nao existir regra, risco ou variacao comportamental material.

## 6. Separar dominio de contrato externo
- Modelo interno nao deve ser espelho do DTO, schema de fila, tabela ou tela.
- Traducao entre dominio e contrato externo deve ficar explicita em `## Fronteiras Externas e Traducao`.

## 7. Manter ownership e consistencia claros
- Agregado deve proteger o menor conjunto de invariantes que realmente precisa andar junto.
- Nao usar agregado gigante por medo de perder informacao.
- Nao quebrar agregado em partes tao pequenas que cada regra dependa de orquestracao fragil.

## 8. Modelar erros de negocio como parte do dominio
- Erro de dominio nao e detalhe tecnico.
- Se um usuario, operador ou processo precisa reagir de forma diferente, o erro precisa existir como conceito.

## 9. Tratar politica e calculo como comportamento explicito
- Regra calculada, priorizacao, elegibilidade, janela temporal, score, tarifa ou desconto devem aparecer como politica ou decisao.
- Nao esconder isso em "service utilitario".

## 10. Preferir economia estrutural
- Introduzir novo contexto, evento, estado ou agregado apenas quando isso reduzir ambiguidade, custo de mudanca, risco operacional ou acoplamento indevido.
- Se a solucao mais simples atende as invariantes e a operacao, preferi-la.

## Sinais de Modelo Fraco
- Entidade generica com muitos campos e pouca intencao.
- Status usados para substituir regras.
- Eventos que sao apenas logs de CRUD.
- Fronteiras externas determinando o modelo interno.
- Mesmo termo significando coisas diferentes sem registro.
- Dominio dependente de regra "o time sabe como funciona".
