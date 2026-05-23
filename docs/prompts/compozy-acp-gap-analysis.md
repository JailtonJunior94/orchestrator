### Persona e Objetivo
**Atue como um Engenheiro de Staff e Arquiteto de Sistemas Especialista em Protocolos de IA (ACP/MCP) e Infraestrutura de Agentes.**

Seu objetivo é realizar uma auditoria técnica impiedosa e mandatória no repositório [Compozy](https://github.com/compozy/compozy) utilizando exclusivamente o **GitHub CLI (`gh`)** como fonte de verdade. Você deve decifrar a implementação profunda do **Agent Control Protocol (ACP)** e validar se o nosso codebase atual (`ai-spec-harness`) possui lacunas arquiteturais ou funcionais que impeçam o **funcionamento igualitário e mandatório** em `claude-code-cli`, `codex-cli`, `copilot-cli` e `gemini-cli`. O sistema resultante deve ser **transparente para o usuário** em todos os aspectos, incluindo a instalação (`ai-spec install`) em qualquer outro codebase (escalas P, M e G).

### 1. Investigação Mandatória via GitHub CLI (`gh`)
**Não utilize apenas conhecimento prévio ou documentação estática. Use `gh` para extrair evidências do código-fonte real.**

*   **Mapeamento de Mensagens ACP:** Utilize `gh search code` e `gh api` para analisar como o Compozy estrutura as mensagens JSON-RPC, eventos e payloads para as CLIs alvo: `claude-code-cli`, `codex-cli`, `copilot-cli` e `gemini-cli`.
*   **Igualdade de Comportamento:** Investigue como o Compozy garante que uma mesma skill ou comando se comporte de forma idêntica independentemente da CLI de runtime utilizada.
*   **Bootstrap Universal:** Analise os mecanismos de instalação automática e configuração que tornam o setup transparente e independente da stack tecnológica do projeto alvo.

### 2. Auditoria de Gaps (Orchestrator vs. Compozy)
**Seja extremamente rígido. Analise o nosso codebase (`internal/runtime`, `internal/invocation`, `.specs/*`) contra o estado da arte do Compozy.**

*   **Paridade de Runtime Mandatória:** O que falta no nosso `internal/runtime/` para que ele seja 100% compatível e igualitário entre as 4 CLIs alvo?
*   **Transparência na Instalação:** O comando `ai-spec install` é capaz de configurar o SDD de forma invisível e funcional em projetos Go, Node ou Python sem ajustes manuais? Identifique atritos que quebram a transparência.
*   **Confiabilidade em Escala (G):** Avalie como o Compozy lida com janelas de contexto gigantes e limites de taxa de forma que o usuário não precise se preocupar com a CLI subjacente.

### 3. Saída Esperada: Draft de PRD (Gap Elimination)
O resultado final deve ser o **capítulo inicial de um PRD técnico** focado em paridade absoluta e transparência:

*   **Matriz de Igualdade Cross-CLI:** Tabela detalhando como garantir comportamento idêntico em Claude, Codex, Copilot e Gemini.
*   **Blueprint de Instalação Universal:** Requisitos para que o setup em qualquer codebase seja transparente e livre de erros.
*   **Requisitos de Infraestrutura:** Mudanças no `internal/` para abstrair as diferenças entre os protocolos de cada CLI.
*   **Estratégia de Governança Unificada:** Como o `spec-hash` e o `PRD-first` permanecem invariantes e transparentes em qualquer runtime.

### Critérios de Aceitação da Análise
*   **Garantia de Igualdade:** A análise deve provar que as soluções propostas funcionam de forma idêntica nas 4 CLIs mencionadas.
*   **Transparência Total:** O plano deve eliminar qualquer necessidade de o usuário final entender as nuances da CLI de IA sendo usada.
*   **Portabilidade Verificada:** O guia de instalação deve ser validado para escalas P, M e G (Enterprise-grade).
*   **Rastreabilidade:** Cada gap deve referenciar evidências extraídas via `gh`.

---

## Justificativa do Enriquecimento (Para o Usuário)

1.  **Mandato de Igualdade:** Eleva a compatibilidade de "suporte" para "comportamento idêntico mandatório" entre as CLIs.
2.  **Transparência de UX:** Garante que o usuário não perceba complexidade técnica, seja no uso diário ou na instalação em novos projetos.
3.  **Bootstrap Universal:** Foca na portabilidade extrema do Orchestrator, permitindo que ele seja injetado em qualquer stack com zero fricção.
4.  **Rigor de Auditoria:** Mantém o uso do `gh cli` para garantir que a paridade proposta seja baseada em implementações reais de sucesso.
