# Prompt Enriquecido: Otimização de Workflow (PRD -> Execução)

## Prompt Original
"analise criteriosamente create-prd, create-task, create-tech spec, execute task e execute all task, e crie uma sessão exclusiva no readme.md para obter o melhor, eficiencia, confiança, economia para desenvolvimento de uma task nesse projeto e em outros também e analisando, qual nota e confiabilidade desse SDD?"

## Prompt Enriquecido

### Objetivo
Sintetizar a inteligência operacional das skills de governança (`create-prd`, `create-technical-specification`, `create-tasks`, `execute-task`, `execute-all-tasks`) em um guia prático de alta performance para o `README.md`, focando no equilíbrio entre custo de tokens e taxa de sucesso da implementação.

### Contexto do Sistema
- **Projeto:** `ai-spec-harness` (Orchestrator).
- **Mecanismo de Confiança:** `spec-hash` (SHA-256) ligando PRD -> TechSpec -> Tasks.
- **Mecanismo de Economia:** Isolamento de contexto via subagentes e carregamento sob demanda de skills de linguagem.
- **Mecanismo de Rigor:** Stage Gates mandatórios, validação de evidências e loop de review/bugfix.
- **Portabilidade:** O harness é agnóstico e **deve** ser instalado em outros projetos (Go, Node, Python) para replicar a mesma governança e eficiência.

### Instruções de Execução

1. **Análise de Eficiência (Custo vs. Valor):**
   - Comparar o overhead de criar documentos (PRD/TechSpec) vs. o custo de retrabalho por alucinação em prompts diretos.
   - Definir o "Caminho Crítico" para tasks isoladas vs. fatias completas de funcionalidade.
   - **Instalação em Outros Projetos:** Analisar como o comando `ai-spec install` permite levar esta eficiência para qualquer repositório, tornando o SDD mandatório e idêntico em diferentes contextos.

2. **Redação da Seção para o README.md:**
   - **Título:** Estratégia de Desenvolvimento de Alta Performance.
   - **Seção "O Caminho do Sucesso":** Fluxo `PRD -> TechSpec -> Tasks -> Execute`.
   - **Seção "Instalação e Portabilidade":** Instruções claras de como instalar o SDD em outros projetos usando `ai-spec install`, garantindo que a governança funcione de forma igual em qualquer codebase.
   - **Seção "Maximizando a Confiança":** Explicação curta sobre o `spec-hash` e por que ele impede regressões de requisito.
   - **Seção "Economia de Contexto":** Quando usar `execute-task` (manual/focado) vs `execute-all-tasks` (automático/batch).
   - **Cheat Sheet:** Tabela de decisão rápida "Qual skill usar agora?".

3. **Avaliação do SDD (Software Development Design):**
   - Atribuir uma nota de 0 a 10 baseada na robustez das invariantes (hashes, gates, schemas).
   - Justificar a confiabilidade técnica com base na detecção de drift e isolamento de falhas.

### Formato de Saída Esperado
- Um bloco de texto em Markdown pronto para ser inserido no `README.md`.
- Uma análise técnica separada com a nota e justificativa de confiabilidade.

---

## Justificativa das Adições
- **Instalação Cross-Project:** Adicionado para garantir que o SDD seja portátil e mandatório em qualquer projeto (Go, Node, Python), mantendo a mesma governança e eficiência em diferentes codebases.
- **Contexto de spec-hash:** Adicionado para garantir que o guia mencione a principal barreira contra drift de requisitos.
- **Diferenciação de execução:** Foco explícito em economia de tokens (contexto) para orientar o usuário a não "queimar" orçamento de IA sem necessidade.
- **Formato Cheat Sheet:** Facilita a adoção rápida por desenvolvedores que não querem ler manuais extensos.
- **Avaliação Estruturada:** Transforma a "nota" subjetiva em uma análise de engenharia sobre invariantes e gates.
