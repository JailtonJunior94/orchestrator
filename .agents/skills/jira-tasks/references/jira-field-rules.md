# Regras de Campos e Estimativa

## Título

Aplicar o formato:

```text
[Backend] <Componente>: <Ação descritiva da task>
```

Usar nomes concretos, curtos e verificáveis. Evitar títulos genéricos como "ajustes", "melhorias" ou "implementação da regra".

Escolher o componente com base no escopo dominante da task.
Preferir `Backend`, `Frontend`, `Infra`, `Data`, `QA` ou o nome do subsistema quando isso tornar o título mais preciso.
Preservar nomes de serviço, módulo, tabela, endpoint, job ou flag quando eles estiverem explícitos no bundle local.

Exemplos:

```text
[Backend] Config: Adicionar env var CRYPTO_REWARD_REFUND_LOOKBACK_DAYS
[Backend] Repository: Implementar HasUnreversedCryptoReward
[Backend] Service: Aplicar branch heurístico no fluxo de refund
```

## Descrição

Montar a descrição da issue a partir do conteúdo real do arquivo `[num]_task.md`.

Ler `assets/task-description-template.md` antes de montar a descrição final.
Preencher apenas seções sustentadas pelo conteúdo do arquivo local.
Omitir uma seção quando o arquivo local não trouxer informação suficiente para preenchê-la sem inferência.
Manter listas técnicas, critérios verificáveis e nomes de arquivos exatamente como aparecem no bundle local quando isso aumentar a precisão.

Preservar, quando existirem, as seções:

* Visão Geral
* Requisitos
* Subtarefas
* Critérios de Sucesso
* Testes
* Dependências
* Arquivos Relevantes

Não inventar requisitos. Não condensar a task a ponto de perder critérios verificáveis.

## Mapeamento entre índice e arquivos

Usar `tasks.md` apenas para confirmar quais tasks fazem parte do bundle.
Usar o prefixo numérico de `[num]_task.md` para ordenar a criação.
Se o índice listar uma task sem arquivo detalhado correspondente, interromper a criação.
Se existir arquivo detalhado sem evidência de que ele faz parte do índice, sinalizar a divergência antes de criar.

## Estimativa

Estimar como dev sênior conservador. Incluir implementação, testes, revisão e edge cases.

Faixas recomendadas:

* Task simples: `2h` a `4h`
* Task média: `4h` a `8h`
* Task complexa: `8h` a `16h`

Regras:

* Não exceder `16h` por task.
* Se exceder `16h`, orientar a divisão antes de criar no Jira.
* Preferir `timetracking.originalEstimate` em horas inteiras.
* Considerar dependências externas, migrações, rollout e testes de regressão quando estiverem explícitos no arquivo local.
* Evitar estimativas de `1h` para tasks de implementação, salvo quando o arquivo local descrever ajuste estritamente pontual.

Sinais de complexidade:

* Simples: mudança localizada, pouco acoplamento, testes diretos.
* Média: integração entre camadas, migração pequena, mais de um fluxo de teste.
* Complexa: regra crítica, alto acoplamento, coordenação entre serviços, backfill ou rollout controlado.

## Campos obrigatórios

Resolver todos os campos obrigatórios antes da criação.

Ordem de decisão:

1. Reutilizar o contexto explícito da US.
2. Reutilizar o contexto explícito da task local.
3. Escolher um valor de `allowedValues` semanticamente compatível.
4. Se não houver correspondência segura, pedir confirmação ao usuário.

Para `customfield_*` com opções enumeradas, preferir payload por `id`.

Campos que costumam ser herdados com segurança quando existirem e forem compatíveis:

* `labels`
* `components`
* campos de squad, stream, domínio ou trilha

Campos que NÃO devem ser copiados automaticamente sem evidência:

* sprint
* fixVersion
* due date
* epic link adicional
* qualquer campo cujo valor mude por task operacional
