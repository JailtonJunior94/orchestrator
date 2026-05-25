# Enriquecimento de Prompt: Gerenciamento de Governança no MacOS

Este documento contém o prompt enriquecido para a tarefa de atualização e reinstalação da governança `ai-spec` no projeto `financialcontrol-api`.

## Etapa 1: Análise do Prompt Original

- **Intenção Principal:** Atualizar, desinstalar e reinstalar a governança `ai-spec` em um repositório específico no MacOS.
- **Contexto Identificado:** MacOS, caminho do projeto (`financialcontrol-api`), necessidade de ser "sem efeito colateral" e "100% funcional".
- **O que falta/Ambiguidades:** 
    - Especificação de quais ferramentas de IA instalar (Claude, Gemini, etc.).
    - Definição do modo de instalação (`copy` vs `symlink`).
    - Comandos de validação para garantir o estado "100% funcional".
    - Detalhes sobre o que documentar no README.

## Etapa 2: Prompt Enriquecido (Recomendado)

```markdown
# Role: Engenheiro de Plataforma AI / DevOps

## Objetivo
Realizar a manutenção do ciclo de vida da governança `ai-spec` no repositório `/Users/jailtonjunior/Git/financialcontrol-api`, garantindo uma instalação limpa, atualizada e verificada no MacOS.

## Contexto do Sistema
- **OS:** MacOS.
- **Ferramenta:** `ai-spec` CLI (binário instalado).
- **Fonte de Governança:** `/Users/jailtonjunior/Git/orchestrator` (Diretório atual).
- **Projeto Alvo:** `/Users/jailtonjunior/Git/financialcontrol-api`.

## Tarefas Procedurais

### 1. Auditoria e Limpeza (Safety First)
- Liste a versão atual do `ai-spec` e os artefatos presentes no projeto alvo.
- Execute `ai-spec uninstall /Users/jailtonjunior/Git/financialcontrol-api --dry-run` para prever a remoção.
- Execute a desinstalação real. Remova manualmente qualquer diretório oculto legado relacionado (`.claude`, `.gemini`, `.agents`) se persistirem, para garantir "zero efeitos colaterais".

### 2. Nova Instalação (Fresh Setup)
- Instale a governança utilizando o orchestrator local como fonte:
  ```bash
  ai-spec install /Users/jailtonjunior/Git/financialcontrol-api \
    --source . \
    --tools all \
    --langs go \
    --mode copy
  ```
- *Nota:* O modo `copy` é preferido para garantir que o projeto seja auto-contido e funcional após atualizações do orchestrator.

### 3. Validação de Integridade (100% Funcional)
- Execute a sequência de validação mandatória e capture os resultados:
  1. `ai-spec inspect /Users/jailtonjunior/Git/financialcontrol-api`
  2. `ai-spec doctor /Users/jailtonjunior/Git/financialcontrol-api`
  3. `ai-spec lint /Users/jailtonjunior/Git/financialcontrol-api`
- Não considere a tarefa concluída até que o `doctor` retorne status verde (0 missing, 0 drifted).

### 4. Documentação de Governança
- Edite o arquivo `README.md` do `financialcontrol-api`.
- Adicione ou atualize a seção "AI Governance" descrevendo:
  - Data da última atualização de baseline.
  - Versão do `ai-spec` utilizada.
  - Como os desenvolvedores devem invocar skills (ex: `execute-task`).

## Critérios de Aceite
- O comando `ai-spec doctor` deve passar sem erros.
- O arquivo `.ai_spec_harness.json` deve estar presente com as configurações corretas.
- O README.md local deve conter instruções claras de uso da governança instalada.
```

## Justificativa das Adições

1. **Modo de Instalação (`--mode copy`):** Adicionado para garantir o requisito de "sem efeito colateral" e estabilidade futura, evitando dependência de links simbólicos que podem quebrar se o orchestrator for movido.
2. **Ciclo de Validação (`inspect`, `doctor`, `lint`):** Incluído para satisfazer o requisito de "100% funcional" de forma mensurável e auditável.
3. **Fase de Desinstalação Explícita:** Adicionada para garantir que a nova instalação não herde resíduos de versões anteriores (limpeza total).
4. **Estrutura de Critérios de Aceite:** Fornece um checklist claro para o agente que for executar este prompt.
