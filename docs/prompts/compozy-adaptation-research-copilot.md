# Prompt Enriquecido: Alinhamento Arquitetural com Compozy (Visão 2026)

## Prompt Original
"Eu quero saber efetivamente, como esse projeto: https://github.com/compozy/compozy conversa com os modelos (LLM) (mandatório) com base no meu codebase . o que falta pra eu fazer igual? crie um plano expecifico para adaptar o meu projeto (mandatório) siga boas práticas implementadas em: https://github.com/compozy/compozy/tree/main (mandatório) Foco no app copilot-cli em 2026 (mandatório)"

---

## Prompt Enriquecido

### Objetivo
Realizar uma análise técnica comparativa entre o projeto `ai-spec-harness` (Orchestrator) e o `compozy` para definir um roadmap de evolução arquitetural. O foco é elevar a integração com LLMs para o padrão "AI-native" de 2026, utilizando o GitHub Copilot CLI como runtime principal via protocolo ACP (Agent Control Protocol).

### Contexto do Sistema (Orchestrator)
- **Status Atual:** Já possui um `internal/runtime` baseado em ACP (ADR-009) e governança via `spec-hash` (SHA-256).
- **Invariantes:** Protocolo PRD-First, âncoras de confiança e evidências obrigatórias.
- **Limitação:** Atualmente opera mais como um "runner" sequencial do que como um orquestrador de ciclo de vida completo com observabilidade em tempo real.

### Contexto de Referência (Compozy)
- **Consulta Mandatória:** Utilizar o GitHub CLI (`gh`) para navegar e extrair informações técnicas diretamente do repositório `compozy/compozy` (ex: `gh repo view compozy/compozy --web` para contexto ou `gh api` para inspeção de arquivos específicos).
- **Interação LLM:** Abstração total via ACP (invoca agentes como `copilot --acp`).
- **Protocolo de Ferramentas:** Integração nativa com MCP (Model Context Protocol) e injeção de servidor MCP reservado para orquestração.
- **Memória:** Sistema de dois níveis (Global `MEMORY.md` e Local `task_N.md`) com compactação automática para gestão de contexto.
- **Extensibilidade:** Sistema de hooks baseado em subprocessos (JSON-RPC 2.0) cobrindo 32 pontos de ciclo de vida.
- **UX:** Daemon-first com interface TUI (Bubble Tea) para visualização do progresso.

### Requisitos do Plano de Adaptação (Mandatórios)

1.  **Exploração via GitHub CLI:**
    - É mandatório o uso de comandos `gh` para validar as suposições sobre a implementação do `compozy`.
    - Analisar a estrutura de arquivos e o código-fonte (especialmente integrações ACP/MCP) usando `gh repo view` e `gh api`.

2.  **Análise de Fluxo ACP/MCP:**
    - Como o `compozy` utiliza o `copilot-cli` como um runtime ACP e como ele injeta ferramentas via MCP.
    - Mapear o que falta no `internal/runtime/runner.go` atual para suportar o servidor MCP "reservado" do `compozy`.

2.  **Sistema de Memória e Contexto:**
    - Propor a migração do nosso modelo atual para o sistema de dois níveis com "compactação automática" do `compozy`.
    - Definir como manter a invariante de `spec-hash` dentro desse modelo de memória evolutiva.

3.  **Ciclo de Vida e Hooks (2026 Vision):**
    - Evoluir os hooks atuais em `.agents/hooks/` (shell scripts) para um sistema formal de plugins/hooks (JSON-RPC ou similar) inspirado no `compozy`.
    - Identificar os 6 estágios da pipeline (Idea, PRD, TechSpec, Tasks, Implementation, Validation) e como o `copilot-cli` se encaixa em cada um.

4.  **Interface e Observabilidade:**
    - Plano para adotar uma TUI (Bubble Tea) que permita "anexar" (attach) a sessões do `copilot-cli` em execução, mantendo a transparência que o `compozy` oferece.

### Formato de Saída Esperado
1.  **Relatório de Gaps:** Tabela comparativa `Feature | Status Orchestrator | Padrão Compozy | Gap Técnico`.
2.  **Plano de Adaptação (Roadmap):** Dividido em fases (Fundação ACP/MCP, Memória Evolutiva, Extensibilidade, UX/TUI).
3.  **Exemplos de Configuração:** Sugestão de como o `COPILOT.md` e `.github/copilot-instructions.md` devem evoluir para suportar essa visão.

---

## Justificativa das Adições

- **Foco em ACP/MCP:** Identificado como o "segredo" da conversa do `compozy` com modelos, permitindo que ele seja agnóstico e extensível.
- **Memória de Dois Níveis:** Crucial para o foco em 2026, onde janelas de contexto grandes ainda precisam de gestão inteligente (compactação) para economia e eficiência.
- **Extensibilidade (JSON-RPC):** Eleva o projeto de um conjunto de scripts para uma plataforma de orquestração profissional.
- **Foco no Copilot CLI:** Atende o requisito mandatório, conectando a arquitetura ACP do `compozy` com a implementação atual do `ai-spec-harness`.
