# Rodadas Obrigatorias de Clarificacao

Toda execucao DEVE usar multipla escolha e DEVE confrontar o pedido do usuario com custo, robustez, operacao e risco de ambiguidade. Rodadas 1, 2, 3 e 4 sao obrigatorias. Rodadas adicionais sao abertas enquanto existir risco material sem decisao.

## Regras Gerais
- Formular de 2 a 4 opcoes mutuamente exclusivas por pergunta.
- Manter cada chamada com no maximo 4 perguntas.
- Explicitar em cada opcao a consequencia principal: mais custo, mais robustez, menor prazo, maior risco, menor flexibilidade ou maior simplicidade.
- Referenciar o pedido atual do usuario no enunciado.
- Se a ferramenta estruturada nao existir, renderizar a mesma pergunta no texto com opcoes `A`, `B`, `C`, `D`.
- Solicitar arquivos, links ou caminhos logo apos a classificacao, mas nao substituir a classificacao por pergunta aberta.

## Rodada 1 - Linguagem ubiqua, objetivo e fronteiras

Cobertura minima:
1. Objetivo dominante da modelagem.
2. Termo canonico do fluxo ou objeto de negocio principal.
3. Fronteira ou bounded context principal.
4. Restricao dominante: prazo, custo, compliance, legado, time ou dependencia externa.

Exemplos de perguntas:
- O pedido deve priorizar clareza de regra, velocidade de entrega, reducao de erro operacional ou compatibilidade com legado?
- O termo principal representa pedido, solicitacao, tentativa, processo ou caso?
- O fluxo mora em um unico contexto, cruza dois contextos fortes ou depende de integracao externa dominante?

## Rodada 2 - Workflow, comandos, eventos e estados

Cobertura minima:
1. Gatilho principal do workflow.
2. Comando ou intencao dominante.
3. Evento relevante do dominio.
4. Estado ou transicao critica.

Exemplos de perguntas:
- O fluxo nasce por acao humana, automacao agendada, evento externo ou reconciliacao operacional?
- A intencao principal e criar, aprovar, recalcular, reprocessar ou cancelar?
- O que o dominio precisa anunciar como fato: recebido, validado, aprovado, rejeitado, expirado ou compensado?
- O risco maior esta na criacao indevida, duplicidade, transicao irreversivel ou falha de sincronizacao?

## Rodada 3 - Regras, invariantes, politicas e erros

Cobertura minima:
1. Regra de negocio mais sensivel.
2. Invariante que nao pode ser violada.
3. Politica ou decisao calculada.
4. Estrategia de erro de dominio.

Exemplos de perguntas:
- A regra dominante e elegibilidade, limite, janela temporal, sequenciamento ou autorizacao?
- O estado ilegal mais perigoso e duplicidade, aprovacao indevida, perda de ownership ou inconsistencia entre artefatos?
- A decisao do dominio e deterministica, parametrizada por politica ou dependente de excecao humana?
- Em caso de violacao, o fluxo deve bloquear, degradar, registrar para analise ou compensar?

## Rodada 4 - Tipos conceituais, integracoes, custo e operacao

Cobertura minima:
1. Ownership transacional ou agregado.
2. Fronteira externa dominante.
3. Consistencia/persistencia necessaria.
4. Postura de custo e operacao.

Exemplos de perguntas:
- O ownership precisa ficar em um agregado central, em mais de um agregado coordenado ou em contexto separado com integracao explicita?
- A fronteira mais sensivel e API sincrona, evento/fila, persistencia compartilhada ou operacao humana?
- O fluxo exige consistencia forte, eventual ou modelo hibrido?
- O pedido favorece simplicidade e baixo custo, equilibrio, ou robustez adicional mesmo com maior custo operacional?

## Rodadas Adicionais

Abrir nova rodada quando faltar decisao sobre qualquer um dos pontos abaixo:
- termo principal ainda ambiguo;
- estado ilegal nao explicitado;
- regra de negocio sem politica clara;
- ownership ou agregado ainda indefinido;
- erro de dominio tratado genericamente;
- contrato externo dirigindo o modelo interno sem traducao explicita;
- confronto com codebase ausente, suspeito ou conflitante sem risco registrado;
- operacao, observabilidade ou custo sem postura minimamente defensavel.

## Criterio de Encerramento
- Encerrar apenas quando o modelo puder ser preenchido sem placeholder proibido nas secoes criticas.
- Nao encerrar se a solucao ainda nao responder claramente: que problema e resolvido, qual e o termo canonico, qual e o workflow central, quais comandos e eventos existem, quais estados sao validos, quais invariantes protegem o fluxo, quem e dono de cada fronteira, como falha, como opera e quanto custa sustentar.
