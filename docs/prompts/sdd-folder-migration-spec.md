# Prompt Enriquecido: Migração Mandatória de `tasks/` para `.spec/` no SDD

## Prompt Original
> "eu não quero usar mais a pasta tasks e sim .spec em TODOS os lugares do SDD de forma mandatória, não deixe nada para trás. NAO IMPLEMENTE NADA, SOMENTE ENRIQUEÇA O PROMPT"

---

## Prompt Enriquecido

### Persona e Objetivo
**Atue como um Engenheiro de Staff e Arquiteto de Sistemas Especialista em Automação de Workflows e Governança de IA.**

Seu objetivo é planejar e executar uma migração exaustiva e mandatória da convenção de diretórios do SDD (Software Development Design). A pasta raiz de artefatos, atualmente denominada `tasks/`, deve ser renomeada para `.spec/` em todo o ecossistema do `ai-spec-harness` (Orchestrator). Esta mudança visa alinhar o projeto com padrões modernos de diretórios de metadados/sistema (dot-folders) e garantir que a estrutura de governança seja tratada como infraestrutura do projeto.

### 1. Contexto e Escopo da Migração
A mudança deve ser aplicada de forma "hard break", sem deixar referências legadas para trás, abrangendo:

- **Skills de Governança:** Atualizar todos os arquivos `SKILL.md` (em `.agents/skills/`) que referenciam caminhos como `tasks/prd-<slug>/`. Isso inclui `create-prd`, `create-technical-specification`, `create-tasks`, `execute-task`, `execute-all-tasks` e `bugfix`.
- **Configuração e Scripts de Suporte:**
    - Alterar o valor default da variável `AI_TASKS_ROOT` de `tasks` para `.spec` em `.agents/lib/check-invocation-depth.sh` e scripts relacionados.
    - Atualizar referências em `.agents/config.yaml` ou equivalentes.
- **Hooks e Automação:** Revisar todos os scripts em `.agents/hooks/` e `.gemini/hooks/` que manipulam ou verificam a existência da pasta `tasks/`.
- **Documentação de Governança:** Atualizar `AGENTS.md`, `GEMINI.md`, `README.md` e todos os arquivos em `docs/` que descrevam o fluxo de trabalho do SDD.
- **Templates e Assets:** Atualizar caminhos nos templates de tarefas (`assets/tasks-template.md`) e exemplos de uso.

### 2. Diretrizes Técnicas (Invariantes)
- **Preservação do spec-hash:** A lógica de vinculação via SHA-256 entre `prd.md`, `techspec.md` e `tasks.md` deve permanecer intacta, mudando apenas o prefixo do caminho de arquivo.
- **Resolução em Cascata:** Garantir que a lógica de resolução de caminhos (`AI_TASKS_ROOT`) continue respeitando overrides de ambiente, mas com o novo padrão `.spec` como base absoluta.
- **Consistência de Naming:** Onde houver variáveis internas chamadas `tasks_path` ou similares, avaliar se devem ser renomeadas para `spec_path` para manter a coerência semântica do código.

### 3. Plano de Execução Sugerido
O executor deve seguir estas etapas:
1.  **Refatoração Global de Strings:** Localizar e substituir referências literais de `tasks/` para `.spec/` em arquivos de texto e scripts.
2.  **Atualização de Lógica de Caminhos:** Modificar os defaults no runtime do orquestrador (Go) e nos scripts de auxílio.
3.  **Renomeação Física (Reflexo):** Ajustar instruções de criação de diretórios para que novos projetos iniciem com `.spec/`.
4.  **Validação de Integridade:** Executar `ai-spec check-spec-drift` (ou equivalente) para garantir que a mudança de caminho não quebrou a validação de hashes.

### 4. Critérios de Aceitação (Definition of Done)
- **Zero Legado:** Nenhuma referência à pasta `tasks/` (no contexto de SDD) deve permanecer no repositório.
- **Funcionalidade Preservada:** Os comandos `create-prd`, `create-tasks`, etc., devem funcionar perfeitamente criando e lendo da pasta `.spec/`.
- **Documentação Sincronizada:** Todos os diagramas, exemplos e descrições de fluxo no `README.md` e `docs/` refletem a nova estrutura.
- **Compatibilidade de Script:** O script de instalação (`ai-spec install`) configura o novo diretório `.spec/` corretamente em novos ambientes.

---

## Justificativa do Enriquecimento

1.  **Padronização Semântica:** Eleva o pedido de uma simples troca de nome para uma mudança de padrão arquitetural (dot-folder), justificando o "porquê" técnico.
2.  **Identificação de Pontos Críticos:** Mapeia especificamente onde a pasta `tasks/` está "escondida" (variáveis de ambiente, scripts de runtime, templates), evitando que a migração seja apenas cosmética.
3.  **Foco em Invariantes:** Garante que o mecanismo de segurança (`spec-hash`) não seja afetado pela mudança de infraestrutura.
4.  **Estrutura de "Definition of Done":** Fornece critérios objetivos para que um agente executor saiba exatamente quando a tarefa foi concluída com sucesso.
5.  **Abrangência:** Garante que a migração cubra tanto o código Go (runtime) quanto as definições de skills (Markdown) e automação (Shell).
