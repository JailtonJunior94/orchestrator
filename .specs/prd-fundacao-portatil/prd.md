# Documento de Requisitos do Produto (PRD)

<!-- spec-version: 1 -->

# Fundação Portátil do ai-spec-harness

## Visão Geral

O `ai-spec-harness` orquestra CLIs de agente de IA (Claude, Codex, Gemini, Copilot) via
**ACP — Agent Client Protocol** por subprocess stdio, sem chamar APIs de LLM diretamente. Uma
análise técnica do projeto [Compozy](https://github.com/compozy/compozy) (par arquitetural que usa
o mesmo modelo ACP) confirmou que o **núcleo de interação com modelos já está resolvido** no
harness. O que separa o harness do estado da arte é **maturidade de plataforma**: instalação
portátil, hierarquia de configuração e robustez/agnosticismo de runtime entre as CLIs.

Este PRD cobre a **fundação** dessa evolução (Fases 1–3 do roadmap aprovado), pré-requisito de
todos os incrementos posteriores. O resultado é um harness que instala em **qualquer codebase em
menos de 30 segundos**, configura-se em camadas (global + projeto) com precedência determinística,
e produz **comportamento idêntico nas quatro CLIs** — mantendo persistência baseada em arquivos
versionáveis (sem banco de dados, sem daemon).

A persistência permanece **file-first** e os runtimes alvo permanecem **quatro** (claude, codex,
gemini, copilot), por decisão de produto.

## Objetivos

- **Portabilidade:** instalar o harness em um codebase vazio em **< 30s**, sem flags
  obrigatórias, com **detecção automática** dos agentes presentes no ambiente.
- **Idempotência:** reexecutar a instalação converge para o mesmo estado — segunda execução
  reporta **100% `current`** na verificação (zero drift).
- **Paridade cross-CLI:** a mesma tarefa nas 4 CLIs satisfaz **invariantes idênticas** (matriz
  4×4), inclusive pelo caminho de **fallback launcher** quando o binário direto está ausente.
- **Configuração determinística:** precedência `flags > workspace > global > built-in` resolvida
  de forma previsível e testável.
- **Zero regressão (F1):** na ausência de configuração nova e de fallback, o comportamento é
  equivalente ao atual.
- **Plataformas:** macOS e Linux suportados (symlink default, copy opcional).

### Métricas de sucesso (critério combinado — todas obrigatórias)

1. Bootstrap em repositório vazio **< 30s** e segunda execução **idempotente** (verify `current`).
2. **Paridade 4×4** validada por invariantes + cobertura **100%** do caminho de fallback launcher.
3. Precedência de configuração **global → projeto → flags** comprovada por teste determinístico.
4. **Regressão F1 zero** na suíte existente do projeto.

## Histórias de Usuário

**Persona primária — Desenvolvedor adotante externo** (instala o harness em outro projeto)
- Como adotante, quero rodar **um único comando** num repositório qualquer e ter o harness
  instalado nos agentes que eu já uso, **sem informar manualmente quais CLIs tenho**, para começar
  em segundos.
- Como adotante, quero **reexecutar** a instalação após um `git pull` sem medo de duplicar ou
  corromper assets, para manter o ambiente atualizado com segurança.
- Como adotante, quero definir **defaults globais uma vez** (ex.: modelo, timeout) em
  `~/.aispec/` e sobrescrevê-los por projeto, para não repetir configuração em cada repo.

**Persona secundária — Mantenedor do harness** (opera múltiplos projetos)
- Como mantenedor, quero que a **mesma tarefa** produza o **mesmo resultado** nas 4 CLIs, para
  confiar na orquestração independentemente do runtime escolhido.
- Como mantenedor, quero que o harness use um **launcher de fallback** automaticamente quando o
  binário ACP direto não estiver no `PATH`, para que execuções não falhem por ambiente incompleto.
- Como mantenedor, quero **verificar o estado** dos assets instalados (current/missing/drifted)
  por agente, para auditar drift entre projetos.

**Casos de borda**
- Ambiente sem nenhum agente detectado → instalação reporta claramente e não falha silenciosamente.
- Binário direto e fallback ambos ausentes → erro tipado e acionável (não trava a sessão).
- Subdiretório profundo dentro do projeto → config de workspace ainda é encontrada por upward-walk.
- Config global ausente → comportamento idêntico ao atual.

## Funcionalidades Core

1. **Fallback launchers por runtime** — cada runtime declara uma cadeia ordenada de launchers
   alternativos (ex.: `npx @zed-industries/codex-acp`). Importante porque ambientes reais nem
   sempre têm o binário ACP direto instalado; garante execução resiliente sem intervenção manual.

2. **Configuração de runtime unificada** — um único conjunto de parâmetros operacionais
   (timeout, retries, backoff, concorrência, batch) aplicável de forma consistente às 4 CLIs.
   Importante para previsibilidade e paridade; hoje esses controles estão parciais/dispersos.

3. **Sessão ACP resiliente** — fluxo de eventos com backpressure (buffer limitado + timeout de
   publicação) e contadores observáveis. Importante para não perder/empacar eventos sob carga e
   para diagnosticar comportamento em produção.

4. **Instalador portátil** — detecção automática de agentes, escopo projeto/global, modos
   symlink/copy, idempotência e verificação de estado. Importante para adoção zero-fricção em
   qualquer codebase.

5. **Camada de configuração universal** — config global (`~/.aispec/`) + projeto, descoberta por
   upward-walk e precedência determinística. Importante para operar múltiplos projetos sem repetir
   configuração e sem ambiguidade de qual valor vence.

6. **Validação de paridade estendida** — matriz de invariantes 4×4 cross-CLI e cross-project,
   evoluindo o framework existente (ADR-008). Importante para transformar "paridade" em garantia
   verificável, não promessa.

## Requisitos Funcionais

**Fase 1 — Agnosticismo de Protocolo (robustez de runtime)**
- RF-01: Cada `Spec` de runtime deve declarar uma cadeia **ordenada de fallback launchers**; quando
  o comando direto não estiver disponível no `PATH`, o harness tenta os launchers na ordem definida.
- RF-02: Deve existir uma **configuração de runtime unificada** que consolide
  `Timeout`, `MaxRetries`, `RetryBackoffMultiplier`, `Concurrent` e `BatchSize`, aplicável às 4 CLIs.
- RF-03: A sessão ACP deve aplicar **backpressure** (canal bufferizado com capacidade limitada e
  timeout de publicação) e **expor contadores** de publicações lentas e atualizações descartadas.
- RF-04: Falhas transitórias de runtime devem ser **reexecutadas com backoff exponencial**
  configurável, respeitando o limite de tentativas.
- RF-05: Na ausência de configuração nova e sem necessidade de fallback, o comportamento deve ser
  **equivalente ao atual (F1)** — sem regressão observável.

**Fase 2 — Motor de Instalação Portátil**
- RF-06: A instalação deve **detectar automaticamente** os agentes/CLIs presentes no ambiente, sem
  exigir a flag de seleção de ferramentas.
- RF-07: A instalação deve suportar **escopo de projeto** (default) e **escopo global** (`--global`),
  resolvendo os caminhos apropriados a cada escopo.
- RF-08: A instalação deve suportar modos **symlink** (default em Unix) e **copy** (portátil),
  mantendo os assets embarcados (`go:embed`, ADR-001) como fonte de verdade.
- RF-09: A instalação deve ser **idempotente** — reexecutar não duplica nem corrompe; o estado
  converge.
- RF-10: Deve existir uma operação de **verificação** que reporte, por skill/agente, o estado
  `current`, `missing` ou `drifted`.
- RF-11: O bootstrap em um codebase vazio deve concluir em **menos de 30 segundos**.
- RF-12: A instalação deve oferecer **modo interativo** (seleção de agentes/skills) e **modo
  não-interativo** (instalar tudo aos agentes detectados).

**Fase 3 — Camada de Configuração Universal**
- RF-13: O harness deve carregar **configuração global** (`~/.aispec/`) além da **configuração de
  projeto**, mantendo compatibilidade com o arquivo de config de projeto existente.
- RF-14: A descoberta do workspace deve usar **upward-walk** a partir do diretório atual até o
  diretório de configuração de projeto mais próximo.
- RF-15: A resolução de configuração deve seguir **precedência determinística**
  `flags CLI > workspace > global > defaults built-in`, com merge **campo a campo**.
- RF-16: Na ausência de configuração global, o comportamento deve ser **idêntico ao atual**.
- RF-17: A persistência de estado/artefatos deve permanecer **baseada em arquivos versionáveis**
  (sem banco de dados).

**Transversal — Validação de Paridade**
- RF-18: O framework de invariantes (ADR-008) deve ser **estendido** para validar saída idêntica
  nas 4 CLIs (**4×4**) e **cross-project** (instalação e execução em repositório temporário).
- RF-19: A suíte de paridade deve **cobrir o caminho de fallback launcher** (binário direto ausente
  → launcher → resultado idêntico ao do binário direto).

## Experiência do Usuário

- **Instalação:** um comando único; sem agentes detectados → mensagem clara e código de saída
  apropriado (não falha silenciosa). Modo interativo lista agentes detectados e skills; modo
  não-interativo instala tudo.
- **Verificação:** saída legível por agente com estados `current | missing | drifted`.
- **Configuração:** TOML/YAML legível com seção de defaults; valores de projeto sobrescrevem
  globais campo a campo; flags sempre vencem.
- **Execução:** quando o binário direto falta, o fallback é transparente (mensagem informativa,
  não erro), preservando o resultado.

## Restrições Técnicas de Alto Nível

- **Protocolo não negociável:** ACP via subprocess stdio; sem chamadas diretas a APIs de LLM.
- **Runtimes alvo:** exatamente 4 — claude, codex, gemini, copilot.
- **Persistência file-first:** artefatos versionáveis em disco (events/evidence/memory); **sem
  SQLite** e **sem daemon central** neste PRD.
- **Plataformas:** macOS e Linux (symlink default, copy opcional). Windows fora de escopo.
- **Governança do repositório:** preservar ADR-001 (assets via `go:embed`) e ADR-008 (invariantes
  de paridade). Mudanças estruturais devem consultar ADRs existentes.
- **Compatibilidade retroativa:** defaults preservam o comportamento F1 (zero regressão).

## Fora de Escopo

- Catálogo/índice central e **workspace registry** multi-projeto (PRD futuro).
- Execução assíncrona estilo **daemon** e `runs attach/watch/detach` (PRD futuro).
- **Review providers** plugáveis (CodeRabbit/GitHub/etc.) (PRD futuro).
- **Extension SDK** (Go/TS) e injeção de assets por extensões (PRD futuro).
- **Reusable agents** (`exec --agent`) (PRD futuro).
- **SQLite** ou qualquer banco de dados.
- Suporte a **Windows**.
- Expansão para além das 4 CLIs (ex.: Cursor, Droid, OpenCode, pi).

## Suposições e Questões em Aberto

- **Suposição:** o nome do diretório de config global será `~/.aispec/` e o de projeto reutilizará
  o arquivo de config existente (`.claude/config.yaml` / `.agents/config.yaml`), com `.aispec/`
  como alias aceito. O nome/forma exatos do arquivo (YAML vs TOML) serão fixados na Especificação
  Técnica.
- **Suposição:** "agentes detectados" abrange as 4 CLIs alvo via presença de binário no `PATH`
  e/ou diretórios de configuração conhecidos; a heurística exata será detalhada na TechSpec.
- **Questão aberta:** a meta de "< 30s" assume assets embarcados (sem download de rede). Confirmar
  que nenhuma etapa de instalação dependerá de rede.
- **Questão aberta:** definir se a unificação da configuração de runtime substitui ou apenas
  encapsula as estruturas atuais (`Job`/flags do task-loop) — decisão de desenho para a TechSpec.
