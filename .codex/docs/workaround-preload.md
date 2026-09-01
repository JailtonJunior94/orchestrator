# Workaround de Preload de Governanca — Codex

> Carregar este arquivo apenas se o hook `.codex/hooks/validate-preload.sh` falhar ou nao estiver presente.

## Contexto

Este adaptador não declara capacidades estáticas. Antes de uma operação que dependa de isolamento
ou escrita concorrente, consulte `ai-spec runtime-capabilities <raiz-do-worktree>`; o CLI é a fonte
de verdade local e bloqueia escrita concorrente sem isolamento comprovado.

## Capacidades do adaptador

Consulte o runtime instalado e seus adaptadores em vez de inferir suporte por esta documentação.

## Workaround recomendado

Quando os hooks configurados não fornecerem o gate necessário, use as seguintes alternativas para
manter compliance:

1. **Variavel de ambiente pre-sessao:** Configurar `GOVERNANCE_PRELOAD_CONFIRMED=1` antes de iniciar uma sessao Codex confirma explicitamente que o contrato de carga base sera seguido.

2. **Hook pre-commit no git:** Adicionar ao `.git/hooks/pre-commit` uma chamada a `bash .codex/hooks/validate-preload.sh` para capturar edicoes indevidas em arquivos de governanca antes do commit.

   ```bash
   # .git/hooks/pre-commit
   git diff --cached --name-only | while read file; do
     bash .codex/hooks/validate-preload.sh "$file" || exit 1
   done
   ```

3. **Instrucao explicita no AGENTS.md:** O Codex le `AGENTS.md` como system prompt — o contrato de carga base esta documentado nele e o modelo e instruido a segui-lo antes de editar codigo.

## Gap registrado

Instruções e scripts não substituem gates do CLI. Sem prova local de capacidade, bloqueie a
operação ou reduza-a ao modo read-only/sequencial aceito pelo CLI.
