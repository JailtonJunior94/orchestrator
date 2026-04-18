---
name: us-to-prd
description: |
  Lê uma User Story completa do Jira via MCP Atlassian e prepara um contexto consolidado
  para iniciar a criação de um PRD pelo fluxo definido em .claude/commands/create-prd.md.
  Use quando o usuário informar uma issue do Jira e pedir PRD, ou mencionar "US para PRD"
  ou "história para PRD". Não use para criar PRD sem issue do Jira, nem para apenas
  consultar issue sem gerar contexto para PRD.
---

# US para PRD

<critical>Ler a US completa antes de invocar create-prd: description, comments, sub-tasks e links relevantes</critical>
<critical>Preservar fatos, decisões, restrições e dependências sem inventar conteúdo</critical>
<critical>Ao final, invocar .claude/commands/create-prd.md com o contexto consolidado</critical>
<critical>Preferir contexto mínimo suficiente para PRD; não transportar ruído operacional, duplicações ou navegação sem impacto funcional</critical>

## Entrada Obrigatória
- Issue key do Jira no formato `PROJ-123`.
- Opcional: `cloudId` do Atlassian. Se ausente, descobrir automaticamente.

## Procedimentos
1. Validar a entrada mínima antes de qualquer chamada no Atlassian.
   Exigir a issue key no formato `PROJ-123`.
   Executar `python3 scripts/validate-issue-key.py "<issue-key>"`.
   Descobrir o `cloudId` com `atlassian-getAccessibleAtlassianResources` quando ele não tiver sido informado.
   Selecionar um recurso acessível com Jira disponível.
   Encerrar com `blocked` se nenhum recurso válido estiver acessível.

2. Ler a issue principal antes de consolidar qualquer contexto.
   Usar `atlassian-getJiraIssue` para obter a issue.
   Extrair, quando disponíveis, `summary`, `description`, `status`, `priority`, `labels`, `reporter`, `assignee`, `epic link`, `sprint` e `Acceptance Criteria`.
   Extrair também campos customizados não vazios que alterem escopo, regra de negócio, dependência ou critério de aceite.
   Ler todos os comentários da issue.
   Encerrar com `blocked` se a issue não existir ou não estiver acessível.

3. Coletar apenas o contexto relacionado que puder alterar o PRD.
   Buscar sub-tarefas via `atlassian-searchJiraIssuesUsingJql` com `parent = <issue-key>`.
   Para cada sub-tarefa, coletar ao menos `summary` e `description`.
   Obter links de issue e links remotos via `atlassian-getJiraIssueRemoteIssueLinks` e campos de link da issue.
   Ler `references/jira-context-rules.md` antes de decidir o que incluir de comentários, links, páginas do Confluence e lacunas.
   Seguir apenas relações e links que adicionem requisito, decisão, dependência, exceção ou contexto funcional.
   Se houver páginas do Confluence diretamente ligadas à definição da US, ler apenas os trechos necessários para preservar contexto funcional.

4. Consolidar o contexto em formato estável para handoff.
   Ler `assets/context-template.md` para montar a estrutura de handoff.
   Preservar o conteúdo original com o mínimo de reescrita.
   Não incluir timestamps, narrativa operacional nem texto sem impacto no PRD.
   Registrar ausências de campos opcionais em `Lacunas Observadas`.

5. Invocar o fluxo final de PRD.
   Chamar `.claude/commands/create-prd.md` com o contexto consolidado.
   Encerrar a skill após a invocação.

## Critérios de Seleção
- Incluir comentários apenas quando mudarem requisito, escopo, regra, prioridade técnica, dependência ou exceção.
- Incluir links externos apenas quando forem fonte direta de requisito ou decisão.
- Ignorar conteúdo duplicado, navegação, ruído operacional e discussões sem impacto funcional.

## Decisões Operacionais
1. Ler tudo antes de resumir qualquer parte da US.
2. Tratar a issue principal como fonte de verdade; usar comentários, sub-tarefas, links e Confluence apenas para complementar ou esclarecer.
3. Preferir ausência explícita em `Lacunas Observadas` a inferência fraca.
4. Preservar termos de negócio, nomes próprios, siglas e decisões exatamente como aparecem na origem quando forem relevantes ao PRD.
5. Não promover comentário, link ou página externa a requisito central sem evidência textual.
6. Preferir contexto consolidado curto e fiel a um dump extenso de conteúdo bruto.

## Estados Finais
- `done`: contexto consolidado e `create-prd` invocado com sucesso.
- `blocked`: issue inexistente, sem acesso ao Jira ou sem recurso Atlassian válido.
- `needs_input`: issue key ausente, ambígua ou dependência externa sem evidência suficiente para continuar com segurança.
- `failed`: falha repetida no MCP que impeça a extração após tentativas razoáveis.

## Tratamento de Erros
- Se `scripts/validate-issue-key.py` falhar, informar o formato esperado: `PROJ-123`.
- Se o MCP Atlassian não responder, informar a falha de acesso e retornar `blocked`.
- Se comentários, subtarefas, Acceptance Criteria ou Confluence não estiverem disponíveis, continuar e registrar a ausência em `Lacunas Observadas`.
