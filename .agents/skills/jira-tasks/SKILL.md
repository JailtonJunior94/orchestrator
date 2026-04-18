---
name: jira-tasks
description: Cria tasks no Jira abaixo de uma User Story com base em arquivos de decomposição gerados localmente. Identifica o tipo de issue correto, resolve campos obrigatórios, monta descrições a partir dos arquivos de task, estima horas de forma conservadora e atribui as tasks ao usuário atual. Use quando o usuário pedir para criar tasks no Jira a partir de uma US, sincronizar `tasks.md` com o Jira ou publicar a decomposição local no Jira. Não use para decomposição local sem Jira, leitura isolada de issue ou criação de PRD.
---

# Criação de Tasks no Jira

<critical>Toda task DEVE ser criada como filha da User Story informada.</critical>
<critical>Toda task DEVE incluir assignee, estimativa e campos obrigatórios já na criação.</critical>
<critical>Os arquivos `tasks.md` e `[num]_task.md` são a fonte de verdade para título, descrição e escopo.</critical>
<critical>Não editar a issue após a criação para completar campos que já poderiam ter sido enviados no create.</critical>

## Procedimentos
1. Validar a entrada mínima antes de qualquer chamada no Jira.
   Exigir a issue key da User Story.
   Localizar o bundle em `./tasks/prd-[feature-name]/`.
   Procurar diretamente por `./tasks/prd-[feature-name]/tasks.md` quando a feature estiver explícita.
   Listar diretórios compatíveis com `./tasks/prd-*/` e pedir confirmação apenas quando houver mais de um candidato plausível.
   Executar `python3 scripts/validate-task-bundle.py ./tasks/prd-[feature-name]`.
   Encerrar com `blocked` se o script retornar erro.

2. Carregar o contexto fonte em paralelo.
   Obter o `cloudId` com `atlassian-getAccessibleAtlassianResources` quando ele não tiver sido informado.
   Obter o usuário atual com `atlassian-atlassianUserInfo` para capturar o `account_id` do assignee.
   Ler `tasks.md`.
   Ler todos os arquivos `[num]_task.md` do diretório validado.

3. Construir a lista de tasks candidatas a partir do bundle local.
   Usar `tasks.md` como índice da decomposição.
   Tratar os arquivos `[num]_task.md` como fonte de verdade para descrição e escopo.
   Preservar a ordem numérica do prefixo `[num]`.
   Ignorar qualquer arquivo markdown que não siga o padrão `[num]_task.md`.
   Interromper com `blocked` se o índice e os arquivos detalhados apontarem para conjuntos divergentes de tasks.

4. Ler a User Story e os metadados do projeto antes de montar payloads.
   Ler a US com `atlassian-getJiraIssue`.
   Extrair pelo menos `summary`, `description`, `status`, `priority`, `labels`, `components` e `project key`.
   Listar os tipos de issue do projeto com `atlassian-getJiraProjectIssueTypesMetadata`.
   Selecionar o tipo filho mais compatível, priorizando `Sub-task`, `Subtask` e `Sub-tarefa`.
   Ler os campos do tipo escolhido com `atlassian-getJiraIssueTypeMetaWithFields`.
   Ler `references/jira-field-rules.md` antes de decidir título, descrição, estimativa e campos obrigatórios.

5. Resolver todos os campos obrigatórios antes de iniciar a criação.
   Considerar como já cobertos `project`, `summary`, `description`, `issuetype`, `parent`, `assignee` e `timetracking`.
   Inspecionar cada campo obrigatório restante.
   Reaproveitar contexto explícito da US quando houver valor compatível.
   Reaproveitar contexto explícito da task local quando houver valor compatível.
   Avaliar `allowedValues` quando o Jira expuser opções enumeradas.
   Montar o payload usando `id` para opções enumeradas e a estrutura nativa esperada pelo campo.
   Interromper com `needs_input` quando nenhum valor puder ser inferido com segurança.

6. Montar cada task de forma determinística.
   Ler `assets/task-description-template.md` para manter a estrutura da descrição.
   Extrair do arquivo local apenas conteúdo factual.
   Preservar seções existentes quando o arquivo local já contiver estrutura equivalente.
   Aplicar o formato de título definido em `references/jira-field-rules.md`.
   Estimar esforço conforme as faixas e regras do mesmo arquivo.
   Interromper com `blocked` quando a estimativa ultrapassar o limite máximo recomendado e orientar a divisão da task.

7. Criar as tasks no Jira com o payload completo já no create.
   Criar cada task com `atlassian-createJiraIssue`.
   Enviar `projectKey`, `issueTypeName`, `summary`, `description`, `parent`, `contentFormat`, `assignee_account_id` e `additional_fields` na mesma chamada.
   Incluir `timetracking.originalEstimate` em `additional_fields` quando o projeto aceitar estimativa na criação.
   Incluir todos os `customfield_*` obrigatórios já resolvidos em `additional_fields`.
   Criar em paralelo apenas quando a ferramenta permitir concorrência sem perder rastreabilidade de erro por task.

8. Relatar o resultado com observabilidade suficiente para reconciliação.
   Exibir key, título e estimativa de cada task criada.
   Informar total estimado e assignee aplicado.
   Informar separadamente tasks criadas e tasks que falharam.
   Não reverter issues já criadas com sucesso.

## Decisões Operacionais
1. Preferir criar zero tasks a criar tasks com tipo, parent ou campos obrigatórios incorretos.
2. Preservar o vocabulário técnico do bundle local em vez de reescrever para linguagem genérica.
3. Herdar contexto da User Story apenas quando isso não contradisser o arquivo `[num]_task.md`.
4. Tratar `tasks.md` como índice e não como substituto do conteúdo detalhado dos arquivos individuais.
5. Reexecutar a leitura de metadados do tipo apenas quando o Jira rejeitar o payload por incompatibilidade de campos.

## Estados Finais
- `done`: tasks criadas e relatadas com rastreabilidade suficiente.
- `blocked`: bundle inválido, tipo de issue incompatível ou impedimento estrutural no Jira.
- `needs_input`: campo obrigatório sem inferência segura ou dado essencial ausente.
- `failed`: erro repetido de execução após reconstrução razoável do payload.

## Tratamento de Erros
- Se `scripts/validate-task-bundle.py` indicar ausência de `tasks.md`, ausência de arquivos `[num]_task.md`, numeração duplicada ou arquivos vazios, orientar o ajuste do bundle local e encerrar com `blocked`.
- Se `tasks.md` listar tasks que não existam como arquivos `[num]_task.md`, encerrar com `blocked` e apontar a divergência.
- Se o projeto não expuser um tipo de subtask compatível, encerrar com `blocked` e informar os tipos encontrados.
- Se um campo obrigatório customizado não puder ser resolvido com segurança a partir de `allowedValues`, encerrar com `needs_input`.
- Se o Jira rejeitar a criação por campo ausente ou formato inválido, reler os metadados do tipo com campos, reconstruir o payload e tentar novamente uma vez.
- Se `timetracking` não estiver disponível no projeto, informar a limitação explicitamente e continuar sem estimativa apenas quando o Jira rejeitar esse campo.
