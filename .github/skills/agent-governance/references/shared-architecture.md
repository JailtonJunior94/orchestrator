# Principios de Arquitetura (Cross-Linguagem)

<!-- TL;DR
Princípios de arquitetura aplicáveis a qualquer linguagem: módulos coesos, DI explícita, regras de domínio fora de handlers e sinais de excesso de abstração.
Keywords: arquitetura, módulos, di, domínio, coesão, camadas, cross-linguagem
Load complete when: tarefa envolve revisão de arquitetura, organização de módulos ou injeção de dependências independente da linguagem.
-->

## Diretrizes
- Modulos/packages coesos com dependencias direcionadas.
- Regras de dominio fora de handlers, controllers e infraestrutura.
- Orquestracao em camadas de aplicacao ou servicos explicitos.
- Evitar cross-module helpers que misturem dominio, IO e formatacao.
- Nomear pelo papel de negocio ou infraestrutura real.

## Injecao de Dependencias
- DI manual via construtores ou factory functions por padrao.
- Container de DI apenas quando a arvore de dependencias justificar o custo.
- Construtor recebe dependencias como parametros explicitos, nao via global ou service locator.

## Projeto Existente
- Seguir layout ja adotado. Nao reorganizar para "alinhar com padrao" sem demanda.
- Novas adicoes respeitam convencao local de nomes, profundidade e agrupamento.

## Sinais de Excesso
- Modulo/package para uma unica funcao sem necessidade estrutural.
- Interface/ABC sem consumidor alternativo.
- Pattern introduzido apenas para "preparar o futuro".
- Container de DI para < 10 dependencias raiz.
