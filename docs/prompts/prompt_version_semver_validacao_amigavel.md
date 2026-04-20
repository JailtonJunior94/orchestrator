## Prompt enriquecido

```md
Atue como um engenheiro senior Go especializado em CLIs com `spf13/cobra`. Trabalhe com foco em causa raiz, mudanca localizada, compatibilidade com o comportamento existente, reuso de logica e validacao objetiva.

## Objetivo

Ajustar o projeto para que o comando `version` exponha ao usuario a versao instalada do `ai-spec-harness` de forma aderente ao SemVer do projeto, e melhorar a validacao de parametros obrigatorios para que todos os comandos que usam o mesmo comportamento retornem mensagens amigaveis com exemplo de uso quando a flag obrigatoria estiver ausente ou vazia.

## Contexto confirmado no repositorio

- Stack: Go 1.26+, Cobra, mensagens e comentarios em PT-BR.
- Existe comando atual em `cmd/ai_spec_harness/version.go`.
- Existe pacote `internal/version` com:
  - `Version = "dev"`, `Commit = "none"`, `Date = "unknown"`;
  - helper `ReadVersionFile(dir string)` que le `VERSION`.
- Existe arquivo `VERSION` na raiz do projeto e ele ja e usado em outras partes do codigo.
- Existe comando `update-version` em `cmd/ai_spec_harness/update_version.go`, com regex SemVer `MAJOR.MINOR.PATCH` sem prefixo `v`.
- O repositorio possui contrato de CLI em `docs/cli-schema.json` e teste de drift em `cmd/ai_spec_harness/cli_contract_test.go`.
- Hoje existem flags marcadas com `MarkFlagRequired(...)` nas seguintes superficies:
  - `install --tools`
  - `task-loop --tool`
  - `changelog --version`
  - `update-version --version`
- Hoje `install` valida `--tools` manualmente em `parseToolsFlag("")` e retorna `flag --tools e obrigatoria`.
- Convencoes do repo:
  - erros e mensagens em PT-BR;
  - testes table-driven;
  - `fmt.Errorf("contexto: %w", err)`;
  - zero estado global desnecessario;
  - usar `internal/fs/fake.go` quando fizer sentido em testes unitarios.

## Escopo

1. Revisar o comportamento atual do comando `version`.
2. Definir uma unica fonte de verdade para a versao exibida ao usuario, alinhada ao SemVer do projeto.
3. Implementar a menor mudanca segura para que `version` retorne a versao instalada de forma previsivel.
4. Melhorar a UX de validacao para flags obrigatorias em todos os comandos que compartilham esse mesmo padrao, nao apenas `--tools`.
5. Preferir uma abordagem reutilizavel e localizada, em vez de mensagens soltas copiadas em cada comando.
6. Atualizar testes e contrato da CLI apenas se a mudanca exigir.

## Requisitos funcionais

1. O comando `version` deve exibir uma versao coerente com o SemVer do projeto.
2. Se houver conflito entre metadados de build e arquivo `VERSION`, escolha uma unica estrategia, implemente-a e justifique no resultado.
3. Se o comando continuar exibindo commit e data, a versao principal ainda deve ser claramente identificavel.
4. Flags obrigatorias devem diferenciar:
   - flag ausente;
   - flag informada mas vazia;
   - flag com valor invalido.
5. Quando uma flag obrigatoria estiver ausente ou vazia, a mensagem deve ser amigavel, em PT-BR, e incluir exemplo real de uso do proprio comando.
6. A melhoria deve cobrir pelo menos todas as flags obrigatorias atualmente identificadas no repositorio com comportamento equivalente.
7. Quando houver validacao de dominio adicional, como ferramenta ou versao invalida, a mensagem deve continuar clara e listar opcoes/formato aceito.
8. Nao mudar outras regras de parsing sem necessidade comprovada.

## Exemplos esperados de UX

### Exemplo 1: versao

Comando:

```bash
ai-spec-harness version
```

Saida esperada:

```text
ai-spec-harness 1.2.3
```

ou, se mantiver metadados:

```text
ai-spec-harness 1.2.3 (commit: abc123, built: 2026-04-20T10:00:00Z)
```

### Exemplo 2: flag obrigatoria ausente em `install`

Comando:

```bash
ai-spec-harness install ./meu-projeto
```

Saida de erro esperada:

```text
flag --tools e obrigatoria.
Exemplo:
  ai-spec-harness install ./meu-projeto --tools claude,gemini --langs go,python
```

### Exemplo 3: flag obrigatoria vazia em `install`

Comando:

```bash
ai-spec-harness install ./meu-projeto --tools ""
```

Saida de erro esperada:

```text
flag --tools nao pode ficar vazia.
Exemplo:
  ai-spec-harness install ./meu-projeto --tools claude,gemini --langs go,python
```

### Exemplo 4: flag obrigatoria ausente em `task-loop`

Comando:

```bash
ai-spec-harness task-loop ./tasks
```

Saida de erro esperada:

```text
flag --tool e obrigatoria.
Exemplo:
  ai-spec-harness task-loop ./tasks --tool codex
```

### Exemplo 5: flag obrigatoria ausente em `changelog`

Comando:

```bash
ai-spec-harness changelog .
```

Saida de erro esperada:

```text
flag --version e obrigatoria.
Exemplo:
  ai-spec-harness changelog . --version 1.3.0
```

## Superficies candidatas a alterar

- `cmd/ai_spec_harness/version.go`
- `internal/version/version.go`
- `cmd/ai_spec_harness/install.go`
- `cmd/ai_spec_harness/task_loop.go`
- `cmd/ai_spec_harness/changelog.go`
- `cmd/ai_spec_harness/update_version.go`
- `cmd/ai_spec_harness/*_test.go`
- `internal/version/*_test.go`
- `docs/cli-schema.json` (somente se houver impacto real no contrato)

## Processo esperado

1. Ler os arquivos relevantes e confirmar como a versao e resolvida hoje.
2. Escolher a estrategia de fonte de verdade da versao com a menor mudanca segura.
3. Implementar a alteracao no comando `version`.
4. Identificar o ponto mais adequado para centralizar ou padronizar mensagens amigaveis de flags obrigatorias.
5. Aplicar a melhoria aos comandos que hoje dependem do mesmo comportamento de obrigatoriedade.
6. Adicionar ou ajustar testes table-driven cobrindo:
   - `version` com a fonte de verdade escolhida;
   - flag obrigatoria ausente;
   - flag obrigatoria vazia;
   - valor invalido;
   - exemplos corretos por comando.
7. Atualizar o schema da CLI se houver mudanca de contrato observavel.

## Criterios de aceite

- `version` retorna uma versao SemVer coerente com o projeto.
- A estrategia de versao tem uma unica fonte de verdade implementada no codigo.
- Todos os comandos com flags obrigatorias equivalentes entregam mensagem amigavel em PT-BR para flag ausente ou vazia.
- Cada mensagem amigavel inclui exemplo real do proprio comando.
- Validacoes de valor invalido continuam claras e especificas.
- Os testes relevantes foram atualizados para cobrir o novo comportamento.
- Nao ha drift entre implementacao Cobra e `docs/cli-schema.json`.

## Restricoes

- Nao invente nova arquitetura.
- Nao introduza dependencias novas.
- Nao mude o significado publico de outras flags sem evidencia no codigo.
- Nao use mensagens em ingles nas novas validacoes.
- Nao remova metadados existentes de `version` sem justificar pela compatibilidade.
- Nao implemente uma solucao especial apenas para `install --tools` se houver oportunidade clara de padronizacao segura.

## Tratamento de ambiguidade

Se "versao instalada no modelo" continuar ambigua apos a leitura do codigo, assuma "versao instalada do binario/CLI exposta ao usuario" e explicite essa premissa no resultado final.

## Output contract

Responda em Markdown e inclua, nesta ordem:

1. `Diagnostico atual`
2. `Decisao de fonte de verdade da versao`
3. `Estrategia para flags obrigatorias`
4. `Implementacao`
5. `Arquivos alterados`
6. `Criterios de aceite atendidos`
7. `Riscos ou premissas`
```

## Justificativa das adicoes

| Adicao | Justificativa curta |
|---|---|
| Escopo ampliado para todas as flags equivalentes | Atende ao pedido explicito de nao limitar a UX amigavel apenas a `--tools`. |
| Lista de comandos com `MarkFlagRequired` | Ancora o prompt nas superficies reais do repositorio. |
| Reuso/padronizacao da validacao | Incentiva corrigir a causa raiz em vez de replicar mensagens manualmente. |
| Exemplos por comando | Torna mensuravel o requisito de mensagem amigavel com input valido. |
| Estrategia de fonte de verdade da versao | O projeto ja tem `VERSION`, `internal/version` e metadados de build; o prompt precisa forcar uma decisao unica. |
| Output contract separado para flags obrigatorias | Facilita avaliar se a resposta cobriu tanto SemVer quanto a padronizacao de UX. |
