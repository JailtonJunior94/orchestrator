---
name: bugfix
version: 1.0.1
description: Corrige bugs pela causa raiz com testes de regressao obrigatorios e evidencia de validacao. Use quando o usuario pedir para corrigir bugs ou referenciar bugs.md, especialmente a partir de achados emitidos pela skill review. Nao use para review ou auditoria sem alteracao, nem para refatoracao sem defeito confirmado.
---

# Corrigir Bugs

## Procedimentos

**Etapa 1: Validar entrada e escopo**
1. Verificar profundidade de invocação: resolver a raiz com `repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"` e localizar `check-invocation-depth.sh` em cascata `.agents/lib/` → `scripts/lib/` (B1, suporta bootstrap em projetos que copiam apenas `.agents/`):
   ```bash
   _depth_lib=""
   for d in "$repo_root/.agents/lib" "$repo_root/scripts/lib"; do
     [[ -r "$d/check-invocation-depth.sh" ]] && { _depth_lib="$d/check-invocation-depth.sh"; break; }
   done
   [[ -n "$_depth_lib" ]] || { echo "failed: check-invocation-depth.sh ausente em .agents/lib/ e scripts/lib/"; exit 1; }
   source "$_depth_lib" || { echo "failed: depth limit exceeded"; exit 1; }
   ```
2. Confirmar que a lista de bugs foi recebida no formato canonico `{ id, severity, file, line, reproduction, expected, actual }`.
3. Ler `references/canonical-bug-format.md` quando houver duvida sobre campos obrigatorios, severidades ou estados canonicos.
4. Se a lista vier em arquivo JSON, validar contra o schema canonico com `python3 "$repo_root/.agents/skills/bugfix/scripts/validate-bug-input.py" --input <caminho>` antes de prosseguir. O script tenta JSON Schema (`jsonschema`) e cai para validacao manual equivalente quando a lib nao esta disponivel.
5. Se a lista estiver ausente, incompleta ou fora do formato canonico, retornar `needs_input` com os campos faltantes.
6. Confirmar o escopo de bugs a corrigir antes de editar qualquer arquivo.

**Etapa 2: Carregar o contexto tecnico**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Se a correcao tocar codigo Go, ler tambem `.agents/skills/go-implementation/SKILL.md` e apenas as referencias exigidas pela mudanca. Para outras linguagens, carregar a skill `.agents/skills/<lang>-implementation/SKILL.md` quando existir; caso contrario, seguir apenas com governanca transversal.
3. Ler `bugs.md`, `prd.md`, `techspec.md`, arquivos de tarefa ou contexto de issue quando estiverem disponiveis e forem relevantes para o bug.
4. Mapear contratos publicos, comportamento esperado, pontos de integracao e risco de regressao antes de propor a correcao.

**Etapa 3: Priorizar e diagnosticar**
1. Ordenar os bugs por severidade: `critical`, `major`, `minor`.
2. Identificar a causa raiz de cada bug antes de editar.
3. Marcar como `blocked` qualquer bug que dependa de contexto externo indisponivel e seguir com os demais bugs do escopo.
4. Evitar patches superficiais quando a causa raiz ainda nao estiver clara.

**Etapa 4: Corrigir e testar**
1. Aplicar a menor mudanca segura focada na causa raiz.
2. Criar um teste de regressao para cada bug corrigido que reproduza `reproduction` e valide `expected`.
3. Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md`.
4. Se a validacao falhar, analisar o log e tentar apenas uma remediacao limitada adicional para o mesmo bug.
5. Se o limite de duas tentativas por bug for excedido, marcar o bug como `failed`, registrar o diagnostico e seguir para o proximo bug elegivel.

**Etapa 5: Revisar e registrar evidencias**
1. Registrar para cada bug o arquivo alterado, o teste de regressao adicionado e o resultado da validacao.
2. Atualizar o estado de cada bug usando apenas `fixed`, `blocked`, `skipped` ou `failed`.
3. Ler `assets/bugfix-report-template.md`.
4. Salvar o relatorio em `tasks/prd-<feature-slug>/bugfix_report.md` quando estiver em contexto de tarefa; caso contrario, em `./bugfix_report.md`.
5. Validar o relatorio com `bash "$repo_root/.claude/scripts/validate-bugfix-evidence.sh" <caminho-do-relatorio>`; corrigir secoes faltantes antes de encerrar.

**Etapa 6: Encerrar o fluxo**
1. Informar total de bugs no escopo, quantos foram corrigidos, quantos testes de regressao foram adicionados e quais itens ficaram pendentes com motivo.
2. Retornar apenas um destes estados canonicos:
   - `done`: escopo acordado corrigido e validado
   - `blocked`: existe bug critico bloqueado por contexto externo
   - `needs_input`: faltam dados obrigatorios de reproducao ou escopo
   - `failed`: limite de remediacao excedido ou validacao nao comprovou a correcao

## Tratamento de Erros

* Se a entrada nao corresponder ao formato canonico, solicitar a conversao antes de prosseguir com a correcao.
* Se uma correcao alterar comportamento publico, parar e explicitar a mudanca a menos que ela tenha sido solicitada.
* Se `go test ./...` ou o equivalente do projeto falhar apos a correcao, analisar o log de falha antes de reexecutar.
* Se a baseline do repositorio ja estiver quebrada, separar claramente a falha preexistente das falhas introduzidas pela correcao.
* Respeitar o limite de profundidade de invocacao definido em `.agents/skills/agent-governance/SKILL.md`. Bugfix nao deve re-invocar review se ja estiver sendo executado dentro de um ciclo review -> bugfix.

## Resolução de paths

Todo caminho `tasks/prd-<slug>/` referenciado neste documento resolve para `${AI_TASKS_ROOT:-tasks}/${AI_PRD_PREFIX:-prd-}<slug>/`. Defaults preservam o layout histórico. Customização via `.claude/config.yaml` ou `.agents/config.yaml`:

```yaml
tasks_root: tasks
prd_prefix: prd-
evidence_dir: ""
```

`check-invocation-depth.sh` (Etapa 1, resolvido em cascata `.agents/lib/` → `scripts/lib/`) exporta `AI_TASKS_ROOT`, `AI_PRD_PREFIX`, `AI_EVIDENCE_DIR`, `AI_TOOL` para skills, validators e runtime, garantindo paridade exata entre Claude Code, Codex, Gemini e Copilot.
