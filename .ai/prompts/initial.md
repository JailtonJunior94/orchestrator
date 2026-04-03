# 🚀 PRD — ORQ (AI Workflow Orchestrator CLI)

---

## 🧠 Visão Geral

O ORQ é um CLI desenvolvido em Go que atua como um:

> **Orquestrador de workflows de desenvolvimento assistido por IA, integrando múltiplos agentes (Claude, Gemini, Codex e Copilot) em um fluxo único, controlado e auditável.**

O sistema permite executar pipelines completos:

```text
PRD → TechSpec → Tasks → Execução
```

Com controle humano em cada etapa e execução real no código.

---

## 🎯 Objetivo

Criar uma ferramenta que permita:

* Orquestrar workflows completos de desenvolvimento
* Integrar múltiplos agentes de IA de forma padronizada
* Controlar execução com previsibilidade
* Garantir outputs estruturados e reutilizáveis
* Permitir validação humana (HITL)
* Executar código real automaticamente

---

## 🚨 Problema

Ferramentas atuais apresentam:

* Execução isolada (Copilot, Claude, Gemini, Codex)
* Falta de orquestração entre etapas
* Outputs inconsistentes e não estruturados
* Baixa confiabilidade para automação
* Falta de controle humano intermediário

Além disso, o cenário moderno (2026) mostra:

* Múltiplos agentes coexistindo (Claude, Codex, Copilot, etc.) ([The Verge][1])
* Necessidade de escolher o melhor agente por tarefa
* Falta de uma camada de orquestração unificada

---

## 💡 Solução Proposta

O ORQ introduz:

* Engine de workflow declarativo (YAML)
* Runtime determinístico controlando execução
* Abstração de providers (multi-agent)
* Validação estruturada (JSON + retry)
* Human-in-the-loop (HITL)
* Execução direta no filesystem

---

## 🧩 Escopo Funcional

---

### 1. CLI (Go + Cobra)

```bash
orq run <workflow>
orq run <workflow> --input "texto"
orq run <workflow> -f input.md
orq continue
orq list
```

---

### 2. Workflow Declarativo

```yaml
name: dev-workflow

steps:
  - name: prd
    provider: claude
    input: "{{input}}"

  - name: techspec
    provider: gemini
    input: "{{steps.prd.output}}"

  - name: tasks
    provider: codex
    input: "{{steps.techspec.output}}"

  - name: execute
    provider: copilot
    input: "{{steps.tasks.output}}"
```

---

### 3. Engine de Execução

* Execução sequencial de steps
* Resolução de variáveis (`{{steps.x.output}}`)
* Contexto isolado por step
* Encadeamento determinístico

---

### 4. Providers (Multi-Agent 2026)

O sistema suporta:

#### 🔹 Claude CLI

* Forte em reasoning e documentação

#### 🔹 Gemini CLI

* Forte em análise e estruturação

#### 🔹 Codex CLI

* Agente de código local capaz de:

  * ler código
  * modificar arquivos
  * executar comandos ([OpenAI Developers][2])

#### 🔹 Copilot CLI

* Forte em:

  * sugestões de código
  * comandos shell
  * integração com GitHub workflows

---

### 5. Output Híbrido

Cada step retorna:

* Markdown (legível)
* JSON estruturado

---

### 6. Validação e Retry

Pipeline:

1. Extrair JSON
2. Corrigir automaticamente
3. Validar schema
4. Retry com LLM

---

### 7. Human-in-the-Loop (HITL)

Cada step requer aprovação:

```text
[A] Aprovar
[E] Editar
[R] Refazer
[S] Sair
```

---

### Persistência

```bash
.orq/
  prd.md
  techspec.md
  tasks.json
  state.json
```

---

### 8. Execução de Tasks

Permitido:

* Criar arquivos
* Editar arquivos
* Executar comandos

Proibido:

* ❌ git commit
* ❌ git push
* ❌ abrir PR

---

## 🧱 Arquitetura

### Camadas

* CLI (cobra)
* Engine (orquestração)
* Workflow (definição)
* Provider (multi-agent CLI)
* Parser (validação)
* Executor (filesystem)
* Storage (estado)

---

## 📂 Estrutura do Projeto

```bash
/cmd
/internal
  /engine
  /workflow
  /provider
  /parser
  /executor
  /storage
/main.go
```

---

## 🔄 Fluxo de Execução

```text
Input
 → PRD (Claude)
   → aprovação
 → TechSpec (Gemini)
   → aprovação
 → Tasks (Codex)
   → aprovação
 → Execução (Copilot)
```

---

## 🔐 Regras do Sistema

* Runtime controla tudo (não o provider)
* Cada step é stateless
* JSON sempre válido
* Outputs determinísticos
* Sem memória implícita
* Sem commit automático

---

## 📊 Critérios de Aceitação

* CLI executa workflow completo
* Steps pausam para aprovação
* Outputs persistidos corretamente
* JSON válido em todos steps
* Integração funcional com:

  * Claude CLI
  * Gemini CLI
  * Codex CLI
  * Copilot CLI
* Execução real no filesystem funcionando

---

## 🧪 Escopo da V1 (ATUALIZADO)

### Inclusões obrigatórias:

* ✅ 1 workflow completo (dev-workflow)
* ✅ Integração real com:

  * Claude CLI
  * Gemini CLI
  * Codex CLI
  * Copilot CLI
* ✅ Engine com HITL
* ✅ Parser + validação básica
* ✅ Execução de tasks simples

---

## 🚀 Roadmap Futuro

* Execução paralela multi-agente
* Benchmark automático entre modelos
* Registry de skills
* Cache de respostas
* Sandbox de execução
* Planejamento automático via IA

---

## 🎯 Resultado Esperado

```bash
orq run dev-workflow --input "criar API de login"
```

Resultado:

* PRD estruturado
* TechSpec consistente
* Tasks geradas via Codex
* Execução assistida via Copilot
* Controle humano em todas etapas

---

## 🧠 Conclusão

O ORQ representa um novo paradigma:

> **Orquestração de múltiplos agentes de IA como um sistema único de desenvolvimento**

Permitindo:

* Escolher o melhor agente por etapa
* Garantir consistência entre ferramentas
* Automatizar com segurança
* Manter controle humano

---

[1]: https://www.theverge.com/news/873665/github-claude-codex-ai-agents?utm_source=chatgpt.com "GitHub adds Claude and Codex AI coding agents"
[2]: https://developers.openai.com/codex/cli/?utm_source=chatgpt.com "Codex CLI"
