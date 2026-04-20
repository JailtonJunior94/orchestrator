<!-- spec-version: 1 -->

# Documento de Requisitos do Produto (PRD)

## Telemetria — Feedback Loop Acionável

**Slug:** `telemetry-feedback`
**Data:** 2026-04-20
**Status:** Aprovado

---

## Visão Geral

O `ai-spec-harness` coleta dados de uso de skills e referências via `internal/telemetry` (arquivo `.agents/telemetry.log`), mas esses dados não informam decisões de produto ou priorização. O ciclo está aberto: a coleta existe, o consumo não. Esta feature fecha o ciclo transformando dados brutos em relatório local com métricas acionáveis — quais skills geram mais valor, quais falham mais, onde há desperdício de tokens — e documenta como usar essas métricas para guiar evoluções do harness.

**Ator principal:** Desenvolvedor que mantém ou usa o harness em um repositório de software (self-hosted).

---

## Objetivos

- Tornar os dados de telemetria consumíveis sem análise manual do log bruto.
- Identificar 3–5 métricas que informem priorização de produto com base em uso real.
- Fechar o ciclo coleta → agregação → relatório → decisão como fluxo documentado.
- Não regredir cobertura de testes (threshold ≥ 70%).

**Critérios de sucesso mensuráveis:**
1. `ai-spec-harness telemetry report` gera saída legível com pelo menos 3 métricas em < 500ms para logs com até 10.000 linhas.
2. O ciclo de feedback está documentado em `docs/` e referenciado no `CLAUDE.md`.
3. `make test` e `make integration` verdes após a implementação.

---

## Histórias de Usuário

- Como **mantenedor do harness**, quero rodar um comando e ver quais skills são mais usadas, para priorizar melhorias nas que têm maior adoção.
- Como **mantenedor do harness**, quero identificar quais skills têm maior taxa de referências carregadas por invocação, para detectar candidatas a otimização de tokens.
- Como **desenvolvedor do projeto**, quero entender o custo estimado de tokens por período, para avaliar se estou dentro do budget esperado.
- Como **contribuidor**, quero que o relatório seja reproduzível e testável, para não introduzir regressões ao evoluir o código de telemetria.

---

## Funcionalidades Core

### RF-01 — Subcomando `report` em `telemetry`

Adicionar `ai-spec-harness telemetry report` (ou `telemetry-report`) que lê `.agents/telemetry.log` e exibe relatório estruturado em Markdown ou texto formatado via `internal/output.Printer`.

**Por que:** o subcomando `summary` existente fornece contagens brutas sem contexto acionável. `report` entrega interpretação além de agregação.

**Requisitos funcionais:**
- RF-01.1: Aceitar flag `--since <duração>` (ex: `24h`, `7d`) para filtrar por período.
- RF-01.2: Aceitar flag `--format [text|json]` para saída em texto formatado ou JSON estruturado.
- RF-01.3: Exibir seções: Resumo Geral, Skills Mais Usadas (top 5), Referências Mais Carregadas (top 5), Estimativa de Custo por Período, Alertas.
- RF-01.4: Retornar exit code 0 quando não houver dados (exibir mensagem amigável), não erro.

### RF-02 — Métricas Acionáveis

O relatório deve incluir pelo menos 3 métricas com interpretação, não apenas contagens:

**Requisitos funcionais:**
- RF-02.1: **Skill mais usada** — nome + contagem + % do total de invocações.
- RF-02.2: **Taxa de referências por invocação** — média de refs carregadas por chamada de skill (detecta skills "gordas").
- RF-02.3: **Estimativa de tokens por período** — total de refs × `tokensPerRefLoad` para janela filtrada.
- RF-02.4: **Alerta de skill sem referência** — skills invocadas sem nenhuma ref carregada (possível skip de governança).
- RF-02.5: (Opcional) **Tendência** — comparar janela atual com janela anterior equivalente (ex: últimas 24h vs. 24h anteriores). Implementar apenas se os dados do log permitirem sem overhead de parse adicional.

### RF-03 — Exportação Opcional (JSON)

Quando `--format json` for especificado, emitir JSON estruturado compatível com ferramentas externas (ex: `jq`, dashboards).

**Requisitos funcionais:**
- RF-03.1: Estrutura JSON deve incluir: `period`, `total_invocations`, `skills[]`, `refs[]`, `estimated_tokens`, `alerts[]`.
- RF-03.2: JSON deve ser válido (testável via `encoding/json.Unmarshal`).

### RF-04 — Documentação do Ciclo de Feedback

**Requisitos funcionais:**
- RF-04.1: Criar `docs/telemetry-feedback-cycle.md` descrevendo: ativação (`GOVERNANCE_TELEMETRY=1`), coleta, agregação, relatório, interpretação e loop de decisão.
- RF-04.2: Adicionar referência a `docs/telemetry-feedback-cycle.md` no `CLAUDE.md` para que agentes saibam que telemetria está disponível.

---

## Experiência do Usuário

_(Feature CLI — sem UI gráfica.)_

Fluxo principal:
1. Usuário ativa coleta: `export GOVERNANCE_TELEMETRY=1`.
2. Skills são usadas normalmente; log é populado em `.agents/telemetry.log`.
3. Usuário executa: `ai-spec-harness telemetry report --since 24h`.
4. Terminal exibe relatório formatado com métricas e alertas.
5. Usuário usa métricas para priorizar evolução de skills ou ajustar carregamento de referências.

---

## Restrições Técnicas de Alto Nível

- Implementar em Go seguindo convenções do projeto (`internal/`, DI via construtor, `internal/output.Printer`).
- Não introduzir dependências externas além das já declaradas no `go.mod`.
- Log de telemetria é plain text; parsing deve ser resiliente a linhas malformadas (não falhar, ignorar).
- Cobertura de testes ≥ 70% (threshold atual enforced no CI).
- Dados de telemetria são locais ao repositório; nenhum dado deve ser enviado a serviços externos.
- Compatível com Go 1.26+.

---

## Fora de Escopo

- Dashboard gráfico ou UI web.
- Envio de telemetria a serviços externos (Datadog, Grafana Cloud, etc.).
- Persistência de relatórios gerados (relatório é sempre gerado sob demanda a partir do log).
- Modificação do formato atual do log (retrocompatibilidade garantida).
- Suporte a múltiplos repositórios agregados em um único relatório.
- Integração com `internal/metrics` para tokens on-disk (mencionado no código como "use `ai-spec metrics`").

---

## Suposições e Questões em Aberto

| # | Suposição / Questão | Decisão |
|---|---|---|
| 1 | O subcomando `summary` existente será mantido por retrocompatibilidade | Sim — `report` é adicionado, `summary` não é removido |
| 2 | `7d` como duração em `--since` precisa de conversão (Go não suporta `168h` equivalente legível) | Aceitar apenas formato de `time.ParseDuration` (`168h` para 7 dias) na v1 |
| 3 | RF-02.5 (Tendência) é opcional — incluir apenas se não exigir segunda passagem de parse | A cargo da tech spec definir viabilidade |
| 4 | JSON export (RF-03) é obrigatório ou opcional? | Obrigatório na v1 para fechar o ciclo de integração com ferramentas externas |
