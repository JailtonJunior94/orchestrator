# Confronto com Codebase

Use este procedimento quando a modelagem tocar sistema existente, migracao, integracao, banco, API, evento, fila, workflow ja implementado, validacao de regra, calculo, status ou padrao operacional ja adotado.

## Objetivo
- Verificar se o dominio proposto e compativel com o codebase real.
- Evitar falso positivo causado por match textual isolado.
- Identificar termos existentes, estados, enums, regras, erros, contratos e ownership ja praticados antes de recomendar mudanca.

## Escopo de Busca
- Preferir o caminho informado pelo usuario.
- Se o usuario nao informar caminho e houver workspace local, usar o repositorio atual como candidato e registrar a premissa.
- Para repo remoto, usar `gh` apenas quando o usuario fornecer `owner/repo` ou quando a origem remota estiver clara.
- Se nao houver codebase aplicavel, registrar `greenfield` ou `nao aplicavel` com justificativa objetiva.

## Evidencia
Classificar cada achado como:

| Status | Criterio |
| --- | --- |
| `confirmado` | Match em codigo de producao com inspecao de contexto e citacao `path:linha`. |
| `suspeito` | Match parcial, nomenclatura parecida, regra incompleta ou evidencia sem contexto suficiente. |
| `ausente` | Busca direcionada sem evidencia em codigo de producao. |
| `refutado` | Evidencia mostra que o termo, fluxo ou regra nao se aplica ao caso. |
| `greenfield` | Nao existe codebase alvo ou a capacidade sera criada do zero. |

## Regra Anti-Falso-Positivo
- Match em teste, mock, fixture, exemplo, snapshot, documentacao ou arquivo gerado nao confirma uso em producao.
- Nome parecido nao confirma conceito de dominio, ownership, contrato ou politica.
- Evidencia sem `path:linha` nao pode ser usada para declarar compatibilidade confirmada.
- Status ou enum isolado nao confirma workflow ou regra de negocio.
- Quando houver duvida entre `confirmado` e `suspeito`, usar `suspeito` e abrir pergunta de refinamento.
- Se o usuario quiser seguir sem confronto, registrar a decisao como risco material e exigir mitigacao no dossie.

## Perguntas Derivadas
Abrir perguntas de multipla escolha quando o confronto apontar:
- dois termos concorrentes para o mesmo conceito;
- regra de negocio espalhada em mais de um modulo;
- status parecido sem garantia de mesmo significado;
- contrato externo parcialmente compativel;
- ownership transacional incerto;
- risco de breaking change;
- ausencia de observabilidade, auditoria ou erro explicito no caminho afetado.

## Registro no Modelo
Preencher `## Materiais e Evidencias` com:
- escopo analisado;
- status do confronto;
- evidencias com `path:linha` quando houver sistema existente;
- riscos de compatibilidade.

Nao usar esta secao para despejar arvore de arquivos, JSON bruto ou resultados de busca sem curadoria.
