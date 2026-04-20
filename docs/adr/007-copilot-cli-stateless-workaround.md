# ADR-007: Copilot CLI stateless workaround

**Status:** Aceita  
**Data:** 2026-04-20  
**Autores:** -

---

## Contexto

O `gh copilot` CLI nao carrega contexto automaticamente — nem `.github/copilot-instructions.md` nem outros artefatos de governanca. Cada invocacao e completamente stateless, ao contrario do Copilot Chat integrado ao editor que respeita `copilot-instructions.md`. Isso impossibilita enforcement automatico de governanca via CLI.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Abandonar suporte ao Copilot CLI | Simplifica escopo, sem manutencao | Perde cobertura de um agente relevante, inconsistencia |
| Criar plugin custom para gh | Enforcement automatico, experiencia integrada | Complexidade alta, manutencao continua, dependencia de API instavel |
| Documentar workaround com injecao manual | Pragmatico, zero manutencao de codigo | Depende de disciplina do usuario, sem enforcement |

## Decisao

Decidimos documentar a limitacao em `COPILOT.md` e sugerir injecao manual de contexto via `#file:` no prompt do usuario. Mantemos `copilot-instructions.md` funcional para Copilot Chat no editor. O parity check registra a cobertura CLI como BestEffort, reconhecendo a limitacao tecnica.

## Consequencias

### Positivas
- Pragmatismo: solucao funciona hoje sem dependencia de API ou plugin
- Zero manutencao de codigo adicional
- Copilot Chat no editor funciona com enforcement via `copilot-instructions.md`

### Negativas / Riscos
- Enforcement impossivel via Copilot CLI
- Governanca depende de disciplina do usuario ao usar CLI
- Se GitHub atualizar o CLI para suportar contexto automatico, este ADR se torna obsoleto

### Neutras / Observacoes
- Parity check marca invariantes de Copilot CLI como BestEffort
- Decisao pode ser revisitada quando `gh copilot` suportar contexto nativo

## Referencias

- `COPILOT.md` — documentacao do workaround
- `.github/copilot-instructions.md` — instrucoes para Copilot Chat
