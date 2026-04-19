## Prompt final

```text
[PAPEL OU POSTURA]
Atue como um auditor tecnico senior de paridade funcional entre repositorios, com foco em robustez, rastreabilidade e cobertura completa. Execute a analise usando `gh` como interface principal para inspecionar o repositorio remoto e use leitura direta do workspace local para inspecionar a implementacao atual em Go.

[OBJETIVO]
Comparar exaustivamente o repositório remoto `JailtonJunior94/ai-governance`, branch `main`, contra o repositório local aberto no workspace atual, verificando se a fonte da verdade remota esta presente e devidamente traduzida para Go no repo local. Identifique o que ja foi portado, o que diverge e o que ainda falta implementar para atingir equivalencia funcional e estrutural.

[ENTRADAS]
- Repositorio remoto fonte da verdade: `https://github.com/JailtonJunior94/ai-governance`
- Branch remota obrigatoria: `main`
- Repositorio local alvo da comparacao: workspace atual
- Linguagem alvo esperada no repo local: Go
- Ferramenta principal para inspecao remota: `gh`
- Ferramentas locais permitidas: `rg`, `git`, `find`, `sed`, `ls`, `go` e leitura direta de arquivos

[RESTRICOES]
- Analise o conteudo do repositorio remoto de forma abrangente: codigo-fonte, estrutura de diretorios, comandos, fluxos, artefatos de configuracao, templates, contratos, validacoes, testes, documentacao operacional e automacoes relevantes.
- Nao limite a comparacao a nomes de arquivos. Compare tambem responsabilidade, comportamento observado, invariantes, interfaces, formatos de dados, checks, fluxos CLI, cobertura de validacao e objetivo funcional.
- Considere que a equivalencia pode ser implementada com nomes, organizacao de pacotes ou decomposicao interna diferentes. Compare por capacidade real, nao apenas por semelhanca textual.
- Se um item remoto existir no repo local com adaptacao legitima para Go, marque como `Presente com adaptacao` e explique a adaptacao.
- Se um item remoto nao tiver equivalente confiavel no local, marque como `Ausente`.
- Se um item existir parcialmente, marque como `Parcial`.
- Se um item local divergir do comportamento esperado da fonte da verdade, marque como `Divergente`.
- Nao invente equivalencias sem evidencias concretas em arquivos, simbolos, comandos, testes ou documentacao.
- Para cada conclusao relevante, cite evidencias objetivas dos dois lados: caminho do arquivo, comando, funcao, struct, teste, documento ou trecho operacional correspondente.
- Se `gh` nao estiver autenticado, nao estiver instalado ou nao conseguir listar o conteudo remoto, declare a falha explicitamente e pare antes de inferir cobertura.
- Nao modifique codigo algum. Esta tarefa e somente de analise.
- Nao use internet aberta, scraping generico ou buscas fora do `gh`, salvo se a propria execucao do `gh` redirecionar para URLs oficiais do GitHub ja relacionadas ao repositório alvo.
- Trate o repositorio remoto como fonte da verdade e o local como implementacao candidata em Go.
- Quando houver ambiguidade sobre o que constitui traducao para Go, prefira julgar por comportamento, CLI exposta, validacoes, fluxos e contratos, nao por semelhanca de sintaxe.

[PROCESSO]
1. Valide precondicoes:
   - confirme que `gh` esta disponivel;
   - confirme acesso ao repositório remoto e a branch `main`;
   - confirme o caminho raiz do repo local analisado.
2. Mapeie o repositorio remoto de forma sistematica usando `gh`:
   - enumere arvore principal, diretorios relevantes e arquivos de configuracao;
   - identifique executaveis, comandos, modulos, bibliotecas, scripts, workflows, schemas, templates e testes;
   - derive uma lista de capacidades remotas, agrupadas por dominio funcional.
3. Mapeie o repositorio local com a mesma granularidade:
   - enumere estrutura, modulos Go, comandos CLI, pacotes internos, testes, validadores, adapters, scaffolds, manifestos e artefatos operacionais;
   - derive uma lista de capacidades locais efetivamente implementadas.
4. Construa uma matriz de equivalencia entre remoto e local:
   - unidade minima de comparacao: capacidade funcional, fluxo operacional, contrato de interface ou artefato obrigatorio;
   - para cada item remoto, classifique em `Presente`, `Presente com adaptacao`, `Parcial`, `Divergente` ou `Ausente`;
- associe sempre a evidencia remota e a evidencia local correspondente;
   - quando nao houver correspondencia local, declare explicitamente a lacuna.
5. Verifique profundidade da traducao para Go:
   - identifique funcionalidades do remoto que no local viraram comandos CLI, pacotes, funcoes, structs, validadores, testes ou geradores;
   - aponte quando a traducao existe mas perdeu comportamento, cobertura, automacao, validacao, ergonomia de uso ou integracao.
6. Liste diferencas reais:
   - inclua somente diferencas materialmente relevantes para paridade funcional, operacional ou de manutencao;
   - elimine falso positivo causado por renomeacao, reorganizacao de pasta ou idiomatismo legitimo de Go.
7. Monte a lista de equalizacao:
   - gere uma backlog objetiva do que falta implementar no repo local para atingir paridade com a fonte da verdade;
   - cada item deve ser acionavel, verificavel e com escopo tecnico claro.
8. Valide consistencia da propria analise antes de responder:
   - nao deixe item remoto sem classificacao;
- nao classifique como equivalente algo sem evidencia dos dois lados;
   - nao misture divergencias com backlog de ausencias.

[CONTRATO DE SAIDA]
- Formato: Markdown
- Estrutura obrigatoria:
  - `## Escopo analisado`
  - `## Premissas e limitacoes`
  - `## Inventario remoto resumido`
  - `## Inventario local resumido`
  - `## Matriz de paridade`
  - `## Diferencas encontradas`
  - `## O que falta implementar para equalizar`
  - `## Riscos de analise ou pontos que exigem confirmacao`
- Em `## Escopo analisado`, informe exatamente qual repo remoto, branch, commit ou referencia resolvida e qual caminho local foram analisados.
- Em `## Premissas e limitacoes`, registre indisponibilidade de ferramenta, autenticacao, arquivos inacessiveis ou qualquer recorte imposto pela execucao.
- Em `## Inventario remoto resumido` e `## Inventario local resumido`, agrupe por dominios funcionais em vez de despejar a arvore inteira.
- Em `## Matriz de paridade`, use tabela com as colunas:
  `Item remoto | Dominio | Evidencia remota | Equivalente local | Evidencia local | Status | Observacao`
- Status permitidos na tabela: `Presente`, `Presente com adaptacao`, `Parcial`, `Divergente`, `Ausente`
- Em `## Diferencas encontradas`, gere lista numerada apenas com divergencias e lacunas materialmente relevantes. Cada item deve conter:
  `descricao`, `impacto`, `evidencia remota`, `evidencia local`
- Em `## O que falta implementar para equalizar`, gere lista numerada orientada a execucao. Cada item deve conter:
  `o que implementar`, `porque falta`, `onde encaixa no repo local`, `criterio objetivo de conclusao`
- Se nao houver diferencas, declare isso explicitamente e ainda entregue a matriz de paridade completa.
- Nao inclua texto promocional, disclaimers genericos, autoelogio ou opinioes vagas.

[NAO FACA]
- Nao responda com avaliacao superficial baseada apenas em README ou nomes de diretorio.
- Nao resuma a resposta sem antes construir a matriz de paridade.
- Nao trate codigo de teste, validacao, schemas, scaffolds e automacoes como detalhes secundarios; eles fazem parte da paridade.
- Nao omita lacunas so porque existe implementacao parcial.

[TRATAMENTO DE FALHAS]
- Se faltar acesso remoto via `gh`, responda somente com o bloqueio tecnico exato, o comando que falhou e o que precisa ser habilitado.
- Se a branch `main` nao existir, identifique a branch padrao real e declare que a analise solicitada nao pode ser concluida sem confirmacao.
- Se encontrar multiplas implementacoes locais candidatas para o mesmo item remoto, compare todas e explicite qual e a melhor equivalencia encontrada.
- Se um item remoto nao puder ser classificado com seguranca, marque como `Indeterminado`, explique o motivo em `## Riscos de analise ou pontos que exigem confirmacao` e nao o promova para `Presente`.
- Se houver conflito entre semelhanca estrutural e comportamento observado, priorize comportamento observado.
```

## Premissas explicitas

- `gh (github actions)` foi interpretado como uso do GitHub CLI `gh`, pois esta e a ferramenta operacional coerente para leitura e inspecao do repositorio remoto durante a analise.
- O repositorio local a ser auditado e o workspace atual deste projeto.
- A "traducao para Go" deve ser julgada por equivalencia funcional e operacional, nao por espelhamento literal da estrutura original.
