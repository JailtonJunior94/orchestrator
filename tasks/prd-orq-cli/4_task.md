# Tarefa 4.0: Platform (Subprocess, Editor, Clock, FS)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar abstrações de plataforma para subprocess, editor externo, clock e filesystem, isolando detalhes de OS do restante do código.

<requirements>
- Interface CommandRunner para execução de subprocessos com captura de stdout, stderr, exit code e timeout
- Interface Editor para abertura de editor externo ($EDITOR, fallback vi/notepad)
- Interface Clock para abstração de tempo (testabilidade)
- Interface FileSystem para operações de filesystem (testabilidade)
- Implementações reais e fakes para teste
- Cross-platform: paths com `filepath`, sem dependência implícita de shell
</requirements>

## Subtarefas

- [ ] 4.1 Definir e implementar `CommandRunner` em `internal/platform/subprocess.go` com captura de stdout, stderr, exit code, timeout via context
- [ ] 4.2 Definir e implementar `Editor` em `internal/platform/editor.go` com $EDITOR, fallback para vi (Unix) / notepad (Windows)
- [ ] 4.3 Definir e implementar `Clock` em `internal/platform/clock.go` (real e fake)
- [ ] 4.4 Definir e implementar `FileSystem` em `internal/platform/filesystem.go` para operações de leitura/escrita/criação de diretórios
- [ ] 4.5 Criar fakes/doubles de cada interface para uso em testes
- [ ] 4.6 Testes de CommandRunner usando técnica TestHelperProcess
- [ ] 4.7 Testes de Editor com fake
- [ ] 4.8 Testes de FileSystem com t.TempDir()

## Detalhes de Implementação

Referir seção "Interfaces Chave" (CommandRunner) e "Editor Externo" em `techspec.md`.

- CommandRunner deve construir comandos com argumentos explícitos (sem shell concatenation — R-SEC-001)
- Timeout deve ser controlado via `context.WithTimeout`
- Editor: gravar output em arquivo temporário → abrir editor → ler conteúdo editado → limpar temp
- Paths normalizados com `filepath.Join` e `filepath.Clean` (cross-platform)

## Critérios de Sucesso

- CommandRunner captura stdout, stderr e exit code corretamente
- Timeout cancela subprocess conforme esperado
- Editor abre e retorna conteúdo editado (testado via fake)
- Clock fake permite controlar tempo em testes
- FileSystem opera com t.TempDir() nos testes
- `go test ./internal/platform/...` passa

## Testes da Tarefa

- [ ] Testes de CommandRunner com TestHelperProcess (sucesso, erro, timeout)
- [ ] Testes de Editor com fake
- [ ] Testes de Clock (real retorna Now, fake retorna valor fixo)
- [ ] Testes de FileSystem com t.TempDir()

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/platform/subprocess.go`
- `internal/platform/editor.go`
- `internal/platform/clock.go`
- `internal/platform/filesystem.go`
- `internal/platform/*_test.go`
