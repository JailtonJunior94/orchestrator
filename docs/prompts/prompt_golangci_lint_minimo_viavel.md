## Contexto do Repositório

- **Módulo Go:** `github.com/JailtonJunior94/ai-spec-harness` (Go 1.26.2)
- **CLI framework:** spf13/cobra
- **Convenções:** comentários, erros e mensagens em PT-BR; Conventional Commits
- **Testes:** table-driven, FakeFileSystem para unit, `t.TempDir()` para integration
- **Cobertura mínima:** 70% (enforced no CI)

### Estado atual relevante

| Artefato | Estado |
|---|---|
| `.golangci.yml` | **Existe** — 7 linters: `errcheck`, `staticcheck`, `gosec`, `govet`, `ineffassign`, `unused`, `misspell` |
| `Makefile` target `lint` | **Existe** — `golangci-lint run ./...` |
| `.github/workflows/test.yml` step Lint | **Existe** — usa `golangci/golangci-lint-action@v6` com `version: latest`, mas **não chama `make lint`** |

---

## Prompt Enriquecido

Você é um engenheiro Go sênior trabalhando no repositório `ai-spec-harness`.  
O objetivo é garantir que o `golangci-lint` esteja configurado com o **mínimo viável** para a qualidade do projeto, que o `Makefile` exponha o comando `lint` e que o workflow `.github/workflows/test.yml` use **exatamente** esse target do Makefile.

### Restrições obrigatórias

1. **Não altere** o comportamento dos demais targets do Makefile (`test`, `integration`, `coverage`, `fuzz`, `vet`, `build`, `clean`).
2. **Não remova** nenhum step existente no `test.yml` (unit tests, coverage, integration, vet, fuzz).
3. **Idioma:** mensagens de erro e comentários em PT-BR.
4. **Commits:** Conventional Commits (tipo em inglês, corpo em PT-BR).
5. O step de lint no CI **deve chamar `make lint`**, não chamar o golangci-lint diretamente.

### O que "mínimo viável" significa neste contexto

Mínimo viável = conjunto de linters que detecta as categorias de bug **mais frequentes em projetos Go** sem gerar ruído (falsos positivos que bloqueiam o CI injustificadamente). Use como critério:

| Categoria | Linter recomendado | Justificativa |
|---|---|---|
| Erros não tratados | `errcheck` | Erros silenciados causam bugs silenciosos |
| Análise estática avançada | `staticcheck` | Substitui `go vet` com cobertura maior |
| Variáveis não usadas / código morto | `unused` | Reduz dívida técnica |
| Atribuições ineficientes | `ineffassign` | Detecta lógica incorreta antes de runtime |
| Segurança básica | `gosec` | Detecta padrões inseguros comuns (sem habilitar regras de baixo sinal) |
| Erros de ortografia | `misspell` | Mantém comentários e strings legíveis |

> `govet` pode ser **removido** do `.golangci.yml` se `staticcheck` estiver habilitado, pois `staticcheck` inclui todas as verificações de `go vet`. Alternativamente, mantê-lo não causa dano — decida com base no tempo de execução observado.

### Tarefas

#### 1. Revisar `.golangci.yml`

- Confirmar ou ajustar os linters da tabela acima.
- Fixar um `timeout` adequado (sugestão: `3m`).
- Excluir `dist/` e `testdata/` da análise (já presente — manter).
- Avaliar se `gosec` deve ter regras excluídas para reduzir falso-positivos (ex.: `G304` para leitura de arquivo por path de variável — comum em CLIs).
- **Não habilitar** linters de formatação (`gofmt`, `goimports`) — formatação é responsabilidade do desenvolvedor local, não do CI de qualidade.

#### 2. Verificar/ajustar `Makefile`

O target `lint` já existe (`golangci-lint run ./...`). Verifique:
- Se a flag `--config .golangci.yml` precisa ser explícita (golangci-lint a detecta automaticamente na raiz — geralmente não é necessário).
- Se deve passar `--out-format=colored-line-number` para melhor legibilidade local.

```makefile
lint:
	golangci-lint run ./...
```

Se o target já está correto, **não o altere**.

#### 3. Atualizar `.github/workflows/test.yml`

Substitua o step atual de lint:

```yaml
# ANTES (não usar)
- name: Lint
  uses: golangci/golangci-lint-action@v6
  with:
    version: latest
```

Por uma sequência que instala o golangci-lint e chama `make lint`:

```yaml
# DEPOIS
- name: Instalar golangci-lint
  uses: golangci/golangci-lint-action@v6
  with:
    version: v1.64.8        # fixar versão — não usar "latest"
    install-mode: binary    # apenas instala, não executa
    skip-cache: false

- name: Lint
  run: make lint
```

> **Por que fixar a versão?** `version: latest` pode quebrar o CI quando uma nova versão do golangci-lint muda o comportamento padrão ou habilita novos linters. Fixar permite reproductibilidade.

> **Por que `install-mode: binary`?** Evita que a action execute o lint por conta própria, delegando ao `make lint` — mantendo consistência entre ambiente local e CI.

### Critérios de aceitação

- [ ] `.golangci.yml` tem somente linters da tabela "mínimo viável" (ou subconjunto justificado).
- [ ] `make lint` executa localmente sem erros no repositório atual.
- [ ] O step "Lint" no `test.yml` chama `make lint` (não golangci-lint diretamente).
- [ ] A versão do golangci-lint no workflow está fixada (não `latest`).
- [ ] Nenhum outro step do `test.yml` foi alterado.
- [ ] O CI passa no branch `main` após as mudanças.

### Formato de entrega

1. Diff dos arquivos alterados (`.golangci.yml`, `Makefile` se necessário, `.github/workflows/test.yml`).
2. Saída do `make lint` local mostrando zero erros.
3. Justificativa em uma linha para cada linter mantido ou removido.

---

## Justificativas das adições ao prompt original

| Adição | Motivo |
|---|---|
| Estado atual dos artefatos | Evita recriar o que já existe (Makefile e .golangci.yml já presentes) |
| Definição de "mínimo viável" com tabela | O termo é ambíguo — a tabela torna o critério mensurável |
| Restrição de não alterar outros steps do CI | Protege cobertura mínima de 70% e outros gates já configurados |
| Fixar versão do golangci-lint | `version: latest` é fonte de instabilidade em CI |
| `install-mode: binary` | Garante que apenas `make lint` execute o linter, mantendo paridade local/CI |
| Critérios de aceitação com checklist | Torna a validação objetiva e verificável |
| Exclusão de linters de formatação | Evita ruído no CI por preferências de estilo não padronizadas |
