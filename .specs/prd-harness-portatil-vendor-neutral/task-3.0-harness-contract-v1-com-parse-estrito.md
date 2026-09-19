# Tarefa 3.0: Harness Contract v1 com parse estrito

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Declarar o contrato de invariantes em arquivo proprio, validado por schema com rejeicao de campo desconhecido.

<requirements>
- RF-02
- RF-03
- RF-04
- RF-05
- RF-06
- RF-07
- RF-08
</requirements>

## Subtarefas

- [ ] 3.1 Criar o pacote de contrato com schema JSON embarcado via `//go:embed`, compilacao por `sync.OnceValues` e validador stateless
- [ ] 3.2 Implementar a ponte YAML->JSON PRESERVADORA (`map[string]any` + normalizacao recursiva de chaves para string); projecao tipada apenas APOS a validacao
- [ ] 3.3 Erros tipados distintos para campo desconhecido, tipo invalido e versao incompativel, nomeando encontrada, suportada e acao corretiva
- [ ] 3.4 Contrato v1 default embarcado quando o arquivo nao existe, reportado como `default` (RF-07)
- [ ] 3.5 Gate de nao-duplicacao de nomes de chave entre o schema do contrato e as 13 chaves operacionais de `internal/config/runtime.go` (RF-05)
- [ ] 3.6 Registrar o alvo de fuzz no `Makefile` E no step `Fuzz` do workflow
- [ ] 3.7 Declarar as interfaces novas em `mockery.yml` no MESMO commit que as introduz (V-39: nenhum gate pega isso)

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Contrato com chave desconhecida produz erro tipado e exit != 0
- O teste guardiao FALHA quando a ponte preservadora e removida — teste que passa nos dois modos esta validando o vacuo
- `config.yaml` com campo desconhecido continua sendo aceito (guardiao de RF-06)
- Projeto sem o arquivo exercita o fluxo completo com o default embarcado
- Validacao e estatica pura: sem rede, sem execucao de binario de provedor, sem LLM

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Ver `techspec.md`, seção *Arquivos Relevantes e Dependentes*.
