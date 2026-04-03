# Documento de Requisitos do Produto (PRD)

## Visão Geral

O **Orchestrator** é uma CLI em Go exposta ao usuário como comando `orq`, responsável por orquestrar workflows de desenvolvimento assistido por IA, integrando múltiplos agentes em um fluxo único, controlado e auditável.

O problema central é que ferramentas de IA para desenvolvimento operam de forma isolada, cada uma com seu próprio contexto, formato de output e fluxo de interação. Não existe uma camada de orquestração que conecte etapas como geração de PRD, especificação técnica, decomposição de tasks e execução de código em um pipeline contínuo com controle humano.

O Orchestrator resolve isso introduzindo um engine de workflow declarativo (YAML) com execução sequencial, validação estruturada, persistência de estado e human-in-the-loop (HITL) em cada etapa.

**Público inicial**: desenvolvedor individual que já utiliza CLIs de agentes de IA.
**Visão futura**: evolução para ferramenta de comunidade/equipes.

## Objetivos

- **OBJ-1**: Permitir execução de um pipeline completo (PRD -> TechSpec -> Tasks -> Execução) via um único comando CLI.
- **OBJ-2**: Integrar Claude CLI e Copilot CLI como providers na V1.
- **OBJ-3**: Garantir controle humano (HITL) em cada step do workflow.
- **OBJ-4**: Produzir outputs estruturados (Markdown + JSON válido com schema validation quando aplicável) e persistidos em disco.
- **OBJ-5**: Funcionar em macOS, Linux e Windows.

### Critérios de Sucesso Mensuráveis

| Critério | Meta |
|---|---|
| Pipeline `dev-workflow` executa end-to-end sem erro | 100% dos steps completam ou pausam para HITL |
| Outputs JSON válidos nos steps estruturados | Parse sem erro + JSON Schema validation quando houver schema do step |
| Providers funcionais | Claude CLI e Copilot CLI integrados e respondendo |
| Cross-platform | Build e execução em macOS, Linux e Windows |
| Tempo de feedback HITL | < 2s entre output do provider e prompt de aprovação |

## Histórias de Usuário

### Persona Primária: Desenvolvedor Individual

**US-1**: Como desenvolvedor, quero executar `orq run dev-workflow --input "criar API de login"` para que um pipeline completo de PRD, TechSpec, Tasks e Execução seja orquestrado automaticamente com aprovação humana em cada etapa.

**US-2**: Como desenvolvedor, quero aprovar, editar, refazer ou cancelar o output de cada step para que eu mantenha controle total sobre o que é gerado e executado.

**US-3**: Como desenvolvedor, quero que os outputs de cada step sejam persistidos em disco em `.orq/` para que eu possa retomar, revisar ou reutilizar artefatos gerados.

**US-4**: Como desenvolvedor, quero continuar um workflow pausado via `orq continue` para que eu não perca progresso caso interrompa a execução.

**US-5**: Como desenvolvedor, quero listar workflows disponíveis via `orq list` para que eu saiba quais pipelines posso executar.

**US-6**: Como desenvolvedor, quero fornecer input via texto direto (`--input`) ou arquivo (`-f input.md`) para que eu tenha flexibilidade na forma de alimentar o pipeline.

**US-7**: Como desenvolvedor, quero que a execução de tasks crie e edite arquivos no filesystem, mas nunca execute operações git (commit, push, PR) para que eu mantenha controle sobre versionamento.

## Funcionalidades Core

### F1. CLI (Go + Cobra)

Interface de linha de comando como ponto de entrada do sistema.

**Por que é importante**: Ponto único de interação do desenvolvedor com todo o sistema de orquestração.

**Requisitos Funcionais**:

- **F1.1**: O CLI deve aceitar o comando `orq run <workflow>` para iniciar execução de um workflow.
- **F1.2**: O CLI deve aceitar flag `--input "texto"` para input inline.
- **F1.3**: O CLI deve aceitar flag `-f <arquivo>` para input via arquivo (Markdown ou texto).
- **F1.4**: O CLI deve aceitar o comando `orq continue` para retomar workflow pausado.
- **F1.5**: O CLI deve aceitar o comando `orq list` para listar workflows disponíveis.
- **F1.6**: O CLI deve exibir progresso de execução com indicação clara do step atual.
- **F1.7**: O CLI deve funcionar em macOS, Linux e Windows.

### F2. Workflow Declarativo (YAML)

Definição de pipelines como arquivos YAML com steps sequenciais e resolução de variáveis.

**Por que é importante**: Permite definir fluxos de forma legível, versionável e extensível sem alterar código.

**Requisitos Funcionais**:

- **F2.1**: O sistema deve carregar workflows a partir de arquivos YAML.
- **F2.2**: Cada workflow deve conter `name` e uma lista ordenada de `steps`.
- **F2.3**: Cada step deve declarar: `name`, `provider` e `input`.
- **F2.4**: O sistema deve resolver variáveis de template: `{{input}}` para input do workflow e `{{steps.<name>.output}}` para output Markdown aprovado de steps anteriores.
- **F2.5**: O sistema deve validar o YAML antes de iniciar execução (steps referenciando steps inexistentes, providers inválidos, etc.).
- **F2.6**: Na V1, o workflow `dev-workflow` deve ser fornecido como built-in.
- **F2.7**: A estrutura YAML deve ser extensível para suportar novos campos no futuro (ex: `parallel`, `condition`) sem breaking changes.

### F3. Engine de Execução

Motor que processa workflows step a step com isolamento de contexto e determinismo.

**Por que é importante**: Garante que a execução é previsível, rastreável e pode ser retomada.

**Requisitos Funcionais**:

- **F3.1**: O engine deve executar steps sequencialmente na ordem definida no YAML.
- **F3.2**: Cada step deve ter contexto isolado, sem compartilhamento implícito de estado.
- **F3.3**: O engine deve resolver variáveis de template antes de enviar input ao provider.
- **F3.4**: O engine deve persistir estado após cada transição relevante do step para permitir `continue`.
- **F3.5**: O engine deve suportar retomada de workflow a partir do último step pendente ou em espera de aprovação.
- **F3.6**: O engine deve tratar falhas de provider com mensagem clara e opção de retry ou abort.
- **F3.7**: O runtime controla a execução, e o provider não tem autonomia para avançar steps.

### F4. Providers (Multi-Agent)

Abstração sobre CLIs de agentes de IA, permitindo invocação padronizada.

**Por que é importante**: Desacopla o engine dos detalhes de cada agente, permitindo trocar ou adicionar providers sem alterar o core.

**Requisitos Funcionais**:

- **F4.1**: O sistema deve definir uma interface de provider com método de execução padronizado.
- **F4.2**: Na V1, o sistema deve implementar provider para **Claude CLI** via subprocess.
- **F4.3**: Na V1, o sistema deve implementar provider para **Copilot CLI** via subprocess.
- **F4.4**: Cada provider deve capturar stdout e stderr do subprocess.
- **F4.5**: Cada provider deve respeitar timeout configurável por step.
- **F4.6**: O sistema deve validar que o CLI do provider está instalado e acessível no PATH antes de executar.
- **F4.7**: A interface de provider deve ser extensível para adicionar Gemini e Codex no futuro sem alteração no engine.
- **F4.8**: A forma exata de invocação de cada provider deve ser encapsulada no adapter e configurável por versão do CLI, evitando acoplamento do runtime a flags específicas ainda não validadas.

### F5. Output Híbrido e Validação

Cada step produz output legível para humanos e, quando aplicável, output estruturado para automação.

**Por que é importante**: Garante que outputs são consumíveis tanto por humanos quanto pelo próximo step do pipeline.

**Requisitos Funcionais**:

- **F5.1**: Cada step deve persistir um artefato Markdown aprovado pelo usuário.
- **F5.2**: Steps estruturados devem produzir também um artefato JSON separado ou extraível do output do provider.
- **F5.3**: O sistema deve extrair JSON do output do provider quando ele vier embutido em Markdown.
- **F5.4**: O sistema deve validar JSON via parse (sintaxe válida).
- **F5.5**: O sistema deve validar JSON via JSON Schema quando schema estiver definido para o step.
- **F5.6**: Em caso de JSON inválido, o sistema deve tentar correção automática limitada a normalizações seguras (ex: trailing commas, aspas).
- **F5.7**: Se correção automática falhar, o sistema deve fazer retry com o provider (máximo 2 retries automáticos).
- **F5.8**: Se retry falhar, o sistema deve pausar para intervenção humana via HITL.
- **F5.9**: O estado persistido deve distinguir output bruto do provider, output aprovado, artefato Markdown, artefato JSON e resultado de validação de schema.

### F6. Human-in-the-Loop (HITL)

Aprovação humana obrigatória entre steps.

**Por que é importante**: Garante que o desenvolvedor mantém controle total e pode corrigir rumo antes de avançar.

**Requisitos Funcionais**:

- **F6.1**: Após cada step, o sistema deve exibir o output e solicitar ação do usuário.
- **F6.2**: Ações disponíveis: `[A] Aprovar`, `[E] Editar`, `[R] Refazer`, `[S] Sair`.
- **F6.3**: `Aprovar` deve avançar para o próximo step.
- **F6.4**: `Editar` deve permitir que o usuário modifique o output antes de avançar.
- **F6.5**: `Refazer` deve re-executar o step atual com o mesmo input resolvido.
- **F6.6**: `Sair` deve persistir estado e encerrar, permitindo `orq continue` depois.
- **F6.7**: O prompt HITL deve aparecer em menos de 2 segundos após o output do provider.

### F7. Persistência de Estado

Armazenamento local de outputs e estado do workflow.

**Por que é importante**: Permite retomada, auditoria e reutilização de artefatos gerados.

**Requisitos Funcionais**:

- **F7.1**: O sistema deve persistir artefatos no diretório `.orq/` na raiz do projeto.
- **F7.2**: Cada execução deve ter diretório próprio em `.orq/runs/<run-id>/`.
- **F7.3**: Estrutura mínima por run: `state.json`, artefatos Markdown por step, artefatos JSON por step quando aplicável e `logs/run.log`.
- **F7.4**: `state.json` deve conter: workflow name, run ID, step atual, status de cada step, timestamps, referências para artefatos e versão de schema do estado.
- **F7.5**: O sistema deve localizar a run compatível mais recente em `.orq/runs/` para permitir `continue`.
- **F7.6**: O sistema não deve sobrescrever artefatos de runs anteriores.
- **F7.7**: Outputs de cada step devem ser persistidos individualmente, separando Markdown aprovado, JSON estruturado e metadados de validação.

### F8. Execução de Tasks no Filesystem

O step de execução pode criar e editar arquivos no projeto.

**Por que é importante**: Permite que o agente de IA execute código real, fechando o loop do pipeline.

**Requisitos Funcionais**:

- **F8.1**: O step de execução deve poder criar novos arquivos.
- **F8.2**: O step de execução deve poder editar arquivos existentes.
- **F8.3**: O step de execução deve poder executar comandos shell aprovados pelo runtime (ex: `go mod tidy`, `npm install`).
- **F8.4**: O step de execução **não deve** executar `git commit`, `git push` ou `gh pr create`.
- **F8.5**: O sistema deve auditar as operações mediadas pelo Orchestrator no step `execute`, incluindo comandos disparados, arquivos de artefato produzidos e ações explícitas aprovadas em HITL.
- **F8.6**: O HITL deve ser aplicado antes da execução do plano de tasks.
- **F8.7**: Se o provider suportar execução autônoma no filesystem, a V1 deve tratá-la como capability opcional e sujeita a adapter específico, sem prometer auditoria completa de mutações fora do controle do runtime.

## Restrições Técnicas de Alto Nível

- **Linguagem**: Go.
- **Invocação de providers**: Via subprocess. A forma exata de chamada fica isolada no adapter de cada provider.
- **Cross-platform**: macOS, Linux e Windows, com atenção especial a paths, shell commands e subprocess handling por SO.
- **Sem dependências de rede para o core**: engine, parser e persistência funcionam offline. Apenas providers requerem conectividade.
- **Segurança**: nenhum segredo hardcoded. Credenciais de providers via variáveis de ambiente, config do SO ou login prévio do próprio CLI.
- **Determinismo**: o runtime controla a execução. Providers são stateless do ponto de vista do engine.

## Experiência do Usuário

### Fluxo Principal

```text
$ orq run dev-workflow --input "criar API de login"

[1/4] PRD (claude)
  Gerando PRD...
  PRD gerado (1.2s)

  --- Output ---
  [conteúdo do PRD]
  ---------------

  [A] Aprovar  [E] Editar  [R] Refazer  [S] Sair
  > A

[2/4] TechSpec (claude)
  Gerando TechSpec...
  TechSpec gerado (2.1s)

  --- Output ---
  [conteúdo da TechSpec]
  ---------------

  [A] Aprovar  [E] Editar  [R] Refazer  [S] Sair
  > A

[3/4] Tasks (claude)
  Gerando Tasks...
  ...

[4/4] Execute (copilot)
  Preparando plano de execução...
  ...
```

### Tratamento de Erros

- Provider não encontrado no PATH: mensagem clara com instrução de instalação.
- Timeout de provider: mensagem com opção de retry ou abort.
- JSON inválido: tentativa automática de correção, retry com provider, fallback para HITL.
- Workflow não encontrado: listar workflows disponíveis.

## Fora de Escopo

### Explicitamente excluído da V1

- Execução paralela de steps.
- Benchmark automático entre modelos.
- Registry de skills/plugins.
- Cache de respostas de providers.
- Sandbox de execução completa.
- Planejamento automático via IA fora do workflow definido.
- Providers Gemini CLI e Codex CLI.
- Operações git: `git commit`, `git push`, abertura de PR.
- Interface web ou GUI.
- Autenticação/autorização multi-usuário.
- Workflows customizados pelo usuário. Na V1 apenas o `dev-workflow` built-in é suportado, embora a arquitetura YAML permaneça extensível.

## Suposições e Questões em Aberto

### Suposições

1. Claude CLI e Copilot CLI estão instalados e autenticados na máquina do desenvolvedor.
2. Os CLIs dos providers suportam invocação não-interativa por subprocess com input via stdin, args ou arquivo temporário.
3. O desenvolvedor possui conectividade com os serviços de IA dos providers durante a execução.
4. O diretório de trabalho contém permissões adequadas para escrita em `.orq/`.

### Questões em Aberto

1. **Contrato de schemas por step**: definir schemas específicos para PRD, TechSpec e Tasks ou validar apenas sintaxe e campos básicos na V1.
2. **Edição inline vs editor externo**: no HITL com opção `[E] Editar`, abrir `$EDITOR` ou permitir edição inline no terminal.
3. **Capability de execução autônoma do Copilot CLI**: validar em quais versões e SOs a execução direta no filesystem é compatível com as garantias mínimas de controle do runtime.
