---
name: semantic-commit
description: Gera, revisa e refina mensagens de commit semântico em pt-BR a partir de mudanças staged, unstaged ou descritas pelo usuário, usando Conventional Commits com fluxo de decisão explícito, definição de scope e detecção de divisão lógica entre commits. Use quando precisar sugerir mensagem de commit, validar se um commit segue padrão semântico, escolher tipo e scope, ou resumir um conjunto de mudanças para commit ou PR. Não use para executar correções de código, reescrever histórico Git, criar commits automaticamente ou substituir revisão técnica detalhada.
---

# Commit Semântico

<critical>Inferir o tipo de commit a partir da evidência mais forte disponível, não por preferência textual do usuário quando ela contrariar o diff.</critical>
<critical>Usar pt-BR no conteúdo gerado e formato Conventional Commits em inglês no cabeçalho.</critical>
<critical>Sugerir divisão em múltiplos commits quando houver mudanças independentes sem objetivo dominante.</critical>

## Procedimentos

**Etapa 1: Coletar a Evidência**
1. Priorize a fonte de verdade nesta ordem:
   - `git diff --staged`
   - `git diff`
   - descrição textual fornecida pelo usuário
2. Execute `git diff --staged` para capturar as mudanças staged.
3. Se o resultado estiver vazio, execute `git diff` para capturar mudanças unstaged.
4. Se ambos estiverem vazios e não houver descrição suficiente do usuário, retorne `needs_input` informando que faltam mudanças observáveis para gerar a mensagem.
5. Registre, antes de classificar, quais arquivos, módulos ou áreas foram alterados e qual parece ser o objetivo principal da mudança.

**Etapa 2: Determinar a Intenção Dominante**
1. Agrupe as mudanças por intenção real, não por extensão de arquivo.
2. Considere como sinais fortes:
   - adição de comportamento novo: `feat`
   - correção de comportamento incorreto: `fix`
   - reorganização sem alterar comportamento esperado: `refactor`
   - melhoria de desempenho mensurável ou claramente intencional: `perf`
   - atualização de documentação: `docs`
   - inclusão ou ajuste de testes: `test`
   - manutenção operacional, tooling ou housekeeping: `chore`
   - build, dependências de build ou empacotamento: `build`
   - pipeline, automação ou CI/CD: `ci`
   - formatação sem efeito funcional: `style`
3. Leia `references/type-selection.md` quando houver ambiguidade entre tipos próximos, especialmente `feat` versus `fix`, `refactor` versus `perf`, e `chore` versus `build` ou `ci`.
4. Se houver mais de uma intenção, escolha a dominante apenas quando as demais existirem para habilitar o mesmo objetivo principal.
5. Se não houver intenção dominante clara, prepare divisão em múltiplos commits e avance para a Etapa 5.

**Etapa 3: Definir Scope e Sinalizar Breaking Change**
1. Defina `scope` apenas quando houver um componente, domínio, pacote ou área claramente identificável.
2. Use nomes curtos e estáveis para `scope`, preferindo o termo do domínio ou do módulo em vez de nomes de arquivo isolados.
3. Omita `scope` quando a mudança atravessar muitas áreas sem um centro claro.
4. Marque breaking change com `!` apenas quando a evidência indicar quebra de compatibilidade, por exemplo:
   - remoção ou renomeação incompatível de API pública
   - mudança obrigatória de contrato, payload, assinatura ou comportamento consumido externamente
   - alteração de migração obrigatória para quem usa o artefato
5. Se a quebra não puder ser confirmada, não use `!`; descreva a dúvida no campo de observações.

**Etapa 4: Gerar a Mensagem Principal**
1. Leia `assets/output-template.md` para seguir o formato de saída esperado.
2. Monte o cabeçalho no formato `<type>(<scope-opcional>): <descrição concisa>` ou `<type>: <descrição concisa>`.
3. Escreva a descrição em pt-BR, com verbo no infinitivo ou forma verbal objetiva, e com foco no resultado principal da mudança.
4. Evite:
   - títulos vagos como `ajustes gerais`, `melhorias`, `update`
   - múltiplos objetivos na mesma linha
   - nomes de arquivo como descrição principal
5. Se o usuário pedir alternativas, gere de 2 a 4 variações mantendo o mesmo tipo inferido, salvo quando a própria classificação estiver em dúvida.

**Etapa 5: Avaliar Divisão em Múltiplos Commits**
1. Sugira divisão obrigatória quando o conjunto incluir mudanças independentes, como por exemplo:
   - feature e refactor sem acoplamento direto
   - correção funcional e formatação massiva
   - alteração de CI junto com mudanças de produto
   - documentação extensa sem depender da implementação alterada
2. Para cada grupo lógico, gere:
   - tipo inferido
   - scope opcional
   - mensagem final
   - justificativa curta da separação
3. Se a divisão for sugerida, trate a mensagem única apenas como fallback e sinalize que ela não é a opção preferida.

**Etapa 6: Validar Antes de Entregar**
1. Verifique se a mensagem final:
   - tem exatamente um objetivo principal
   - usa um tipo permitido
   - usa `scope` apenas quando agrega contexto real
   - não contradiz a evidência observada
   - está em pt-BR no conteúdo textual
2. Execute `python3 scripts/validate-commit-header.py "<cabecalho-final>"` para validar o formato do cabeçalho.
3. Se o usuário pedir revisão de uma mensagem já existente, valide a mensagem contra a evidência e aponte:
   - tipo inadequado
   - scope fraco ou excessivo
   - descrição vaga
   - necessidade de split
4. Se solicitado, gere também um resumo curto de PR alinhado aos commits sugeridos.

## Formato de Saída
Use o esqueleto definido em `assets/output-template.md`.

## Tratamento de Erros
* Se `git diff --staged` ou `git diff` falhar, verifique se o diretório atual é um repositório Git válido e informe a falha sem inventar contexto ausente.
* Se o diff estiver vazio e o usuário só fornecer um pedido genérico, retorne `needs_input` pedindo um diff, arquivos afetados ou descrição objetiva das mudanças.
* Se o diff for muito grande para leitura completa, priorize arquivos centrais, mensagens existentes, nomes de módulos e padrões repetidos antes de propor a classificação.
* Se a mudança misturar muitos objetivos e a evidência estiver fragmentada, priorize recomendar divisão em commits menores em vez de forçar uma mensagem única ruim.
* Se houver dúvida material entre dois tipos válidos, apresente o tipo recomendado, a alternativa e a razão da escolha.
* Se `scripts/validate-commit-header.py` falhar, corrija o cabeçalho antes de entregar a mensagem final.
