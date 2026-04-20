# ADR-002: FakeFileSystem como estrategia de teste unitario

**Status:** Aceita  
**Data:** 2024-01-01  
**Autores:** JailtonJunior94

---

## Contexto

Os servicos do harness (`install`, `upgrade`, `skills`, etc.) realizam operacoes intensivas de filesystem: leitura de templates, escrita de arquivos, verificacao de existencia, copia de arvores de diretorios.

Para testar esses servicos de forma rapida, deterministica e sem efeitos colaterais, precisamos de uma estrategia de isolamento do filesystem real.

Restricoes relevantes:
- Testes unitarios nao devem tocar o OS filesystem.
- A abstracao de filesystem deve ser injetavel via construtor (principio DI do projeto).
- Nao adicionar dependencias externas desnecessarias.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| FakeFileSystem custom (escolhida) | Controle total; sem dependencia; comportamento previsivel; facil de instrumentar | Manutencao manual; edge cases de FS real nao cobertos |
| `github.com/spf13/afero` | Biblioteca madura com MemMapFs | Adiciona dependencia externa; API diferente da stdlib |
| Mocks gerados (mockery/gomock) | Geracao automatica | Acoplamento a interface; testes frageis a mudancas de assinatura |
| `t.TempDir()` em todos os testes | Testa comportamento real | Testes mais lentos; dependentes de permissoes do OS; nao sao unitarios |

## Decisao

Decidimos implementar `FakeFileSystem` em `internal/fs/fake.go` — um in-memory filesystem customizado que implementa a interface `FileSystem` definida no mesmo pacote.

Todos os servicos recebem `FileSystem` via construtor. Testes unitarios injetam `FakeFileSystem`; o binario de producao usa `OSFileSystem`. Testes de integracao com build tag `integration` usam `t.TempDir()` sobre o OS real.

## Consequencias

### Positivas
- Testes unitarios sao rapidos, deterministicos e paralelos sem conflito.
- Sem permissoes de OS, sem limpeza manual, sem side-effects.
- Facil de pre-popular com fixtures usando `FakeFileSystem.WriteFile(...)`.
- Nenhuma dependencia externa adicionada.

### Negativas / Riscos
- `FakeFileSystem` deve ser mantido manualmente quando novos metodos sao adicionados a interface.
- Comportamentos especificos do OS (symlinks, permissoes, limites de path) nao sao cobertos pelos testes unitarios.
- Esses gaps sao mitigados pelos testes de integracao (`//go:build integration`).

### Neutras / Observacoes
- A separacao unit/integration e explicita no Makefile: `make test` vs `make integration`.
- `io.Discard` e usado como `Printer` em testes para suprimir output sem mock.

## Referencias

- `internal/fs/fake.go` — implementacao do FakeFileSystem
- `internal/fs/fs.go` — interface FileSystem
- `internal/install/install_test.go` — exemplo de uso em testes unitarios
- `internal/integration/` — testes com filesystem real
