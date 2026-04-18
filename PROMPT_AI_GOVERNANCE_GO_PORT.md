# Prompt Enriquecido

## Input Bruto

```text
utilize gh (github-cli), para ler importar traduzindo para golang tudo que foi implementado em: https://github.com/JailtonJunior94/ai-governance/tree/release/1.1.0-harness-hardening, não deixe nada para trás a ideia é ter o mesmo conteúdo, porém em golang cli, repo: https://github.com/JailtonJunior94/ai-governance, branch: release/1.1.0-harness-hardening
```

## Classificacao

- Modo principal: `execution`
- Pressao principal de otimizacao: `robustness`

## Prompt Enriquecido

```text
[PAPEL OU POSTURA]
Atue como um engenheiro sênior de migração e paridade funcional, com foco em robustez e execução completa.

[OBJETIVO]
Usando `gh` (GitHub CLI) como fonte principal de leitura do repositório, leia integralmente o que foi implementado na branch `release/1.1.0-harness-hardening` do repositório `JailtonJunior94/ai-governance` e porte tudo para uma implementação equivalente em Go CLI, sem omitir funcionalidades, fluxos, validações, contratos, mensagens operacionais, testes relevantes, scripts necessários e comportamento observável.

[ENTRADAS]
- Repositório alvo: `https://github.com/JailtonJunior94/ai-governance`
- Branch de referência: `release/1.1.0-harness-hardening`
- Fonte principal de inspeção: `gh`
- Resultado esperado: mesma capacidade funcional da branch de referência, porém implementada como CLI em Go

[RESTRICOES]
- Preserve: cobertura funcional, regras de negócio, fluxos de execução, entradas e saídas esperadas, semântica de flags/comandos quando aplicável, comportamento de erro, logs relevantes, contratos de arquivos, convenções operacionais e qualquer hardening implementado no harness.
- Preserve: equivalência comportamental antes de simplificações estruturais. Refatorações são permitidas apenas se não reduzirem escopo, cobertura ou fidelidade.
- Evite: reimplementação parcial, resumo em vez de port completo, remoção silenciosa de edge cases, placeholders sem implementação, TODOs sem necessidade, mudança arbitrária de UX CLI, e invenção de comportamento não presente na branch de origem.
- Evite: depender de memória. Leia o material fonte com `gh` e derive a implementação a partir do conteúdo real do repositório.
- Limite: quando faltarem detalhes, investigue primeiro com `gh`; só declare premissas explícitas se a informação não estiver acessível. Não encerre cedo enquanto ainda houver arquivos, fluxos ou testes relevantes não avaliados.

[PROCESSO]
1. Use `gh repo clone`, `gh api`, `gh browse`, `gh pr view`, `gh release view`, `gh repo view`, `gh api repos/{owner}/{repo}/git/trees/{branch}?recursive=1`, `gh api repos/{owner}/{repo}/contents/{path}?ref={branch}` ou comandos equivalentes para enumerar e ler o conteúdo real da branch `release/1.1.0-harness-hardening`.
2. Identifique toda a superfície implementada na branch: comandos CLI, módulos, scripts, configs, templates, documentação operacional, fixtures, testes, CI relacionada, harnesses, políticas, validadores, schemas e qualquer arquivo que altere comportamento.
3. Monte um mapa de paridade contendo: arquivo fonte, responsabilidade, comportamento observável, dependências, entradas, saídas, erros e destino equivalente em Go CLI.
4. Implemente a versão Go CLI cobrindo integralmente o escopo identificado. Se houver estrutura CLI existente, evolua-a; se não houver, crie arquitetura coerente para comandos, subcomandos, flags, config, IO, validação e testes.
5. Converta scripts utilitários, validações e automações relevantes para Go sempre que fizerem parte do comportamento do produto. Mantenha scripts auxiliares fora de Go apenas se forem claramente externos ao escopo da CLI e explique essa exceção.
6. Reproduza o hardening do harness de forma equivalente em Go: validações defensivas, tratamento de erro, timeouts, retries, sanitização, isolamento de execução, checagens de pré-condição, mensagens operacionais e fluxos de fallback quando existirem na origem.
7. Compare o resultado implementado contra o mapa de paridade e contra a branch fonte. Marque explicitamente qualquer item não portado e só aceite ausência quando houver justificativa técnica objetiva e inevitável.
8. Execute testes, lint e verificações aplicáveis no repositório. Se faltar infraestrutura para alguma validação, registre exatamente o bloqueio e o impacto.

[CONTRATO DE SAIDA]
- Formato: markdown objetivo
- Inclua: 
  - resumo curto do escopo portado
  - inventário de paridade com itens implementados
  - lista de arquivos principais criados ou alterados
  - validações executadas e respectivos resultados
  - gaps restantes, se existirem, com justificativa objetiva
- Exclua: discurso genérico, disclaimers longos, sugestões especulativas e respostas sem evidência do conteúdo lido
- Tamanho: conciso, mas completo o suficiente para auditoria de paridade

[TRATAMENTO DE FALHAS]
- Se `gh` não estiver autenticado ou não tiver acesso suficiente, informe o comando exato que falhou e qual permissão ou autenticação falta.
- Se houver conflito entre manter paridade e adaptar para Go CLI, priorize paridade funcional e preserve a diferença estrutural apenas no nível de implementação.
- Se algum arquivo da branch parecer irrelevante, confirme pelo comportamento antes de excluir do porte.
- Se encontrar ambiguidade, investigue mais arquivos, histórico, testes e docs antes de assumir.

[NAO FACA]
- Não entregue apenas plano, pseudocódigo ou scaffold.
- Não reduza escopo por conveniência.
- Não substitua leitura do repositório por inferência.
- Não considere a tarefa concluída enquanto houver funcionalidades da branch sem equivalente verificável em Go CLI.
```

## Justificativa Curta

- O prompt foi endurecido para `execution` com foco em `robustness`, priorizando leitura real da branch com `gh` em vez de inferencia.
- As restricoes foram convertidas em criterios verificaveis de paridade funcional, cobrindo codigo, harness hardening, testes e comportamento observavel.
- O contrato de saida exige evidencia auditavel do que foi portado, reduzindo o risco de migracao parcial.

## Premissas

- O repositório de destino e a branch de origem sao o mesmo repositório informado pelo usuario.
- "Mesmo conteudo" foi interpretado como paridade funcional completa, nao equivalencia literal de estrutura ou linguagem.
- A implementacao final pode reorganizar arquivos para uma arquitetura idiomatica em Go, desde que preserve comportamento e escopo.
