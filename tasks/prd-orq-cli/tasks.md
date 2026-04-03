# Resumo das Tarefas de Implementação para ORQ CLI

## Metadados
- **PRD:** `tasks/prd-orq-cli/prd.md`
- **Tech Spec:** `tasks/prd-orq-cli/techspec.md`
- **Total de tarefas:** 10
- **Tarefas paralelizáveis:** 3.0/4.0/6.0 (entre si); 5.0/9.0 (entre si)

## Tarefas

| # | Título | Status | Dependências | Paralelizável |
|---|--------|--------|-------------|---------------|
| 1.0 | Scaffolding e Tooling | done | — | — |
| 2.0 | Domínio (Run, Step, Value Objects) | done | 1.0 | Não |
| 3.0 | Workflow (Parser, Validator, Template) | done | 1.0 | Com 4.0, 6.0 |
| 4.0 | Platform (Subprocess, Editor, Clock, FS) | done | 1.0 | Com 3.0, 6.0 |
| 5.0 | Providers (Claude CLI, Copilot CLI) | done | 4.0 | Com 9.0 |
| 6.0 | Output (Extração, Correção, Validação) | done | 1.0 | Com 3.0, 4.0 |
| 7.0 | State (Persistência em .orq/) | done | 2.0 | Não |
| 8.0 | Engine de Execução | done | 2.0, 3.0, 5.0, 6.0, 7.0 | Não |
| 9.0 | HITL (Prompt Interativo) | done | 4.0 | Com 5.0 |
| 10.0 | CLI Cobra + Bootstrap | done | 8.0, 9.0 | Não |

## Dependências Críticas
- **8.0 (Engine)** é o gargalo central — depende de 2.0, 3.0, 5.0, 6.0 e 7.0
- **2.0 (Domínio)** deve ser estável antes de 7.0 e 8.0
- **4.0 (Platform)** é pré-requisito para 5.0 e 9.0

## Riscos de Integração
- Integração do Engine (8.0) com todos os componentes pode revelar necessidade de ajustes nas interfaces definidas em tarefas anteriores
- Discovery de capability e formato de invocação do Copilot CLI pode variar por versão/SO, impactando adapter em 5.0
- Diferenças cross-platform em subprocess e paths podem surgir na integração de 4.0 com 5.0

## Legenda de Status
- `pending`: aguardando execução
- `in_progress`: em execução
- `needs_input`: aguardando informação do usuário
- `blocked`: bloqueado por dependência ou falha externa
- `failed`: falhou após limite de remediação
- `done`: completado e aprovado
