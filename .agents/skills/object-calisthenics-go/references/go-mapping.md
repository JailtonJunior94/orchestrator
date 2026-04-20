# Mapeamento das Heuristicas para Elementos Go

## Packages

Usar packages para delimitar fronteiras de colaboracao e semantica. Evitar criar packages so para obedecer contagem de linhas ou simular camadas sem necessidade.

## Structs

Tratar structs como agregadores de estado e comportamento relacionado. Quando uma struct acumular dependencias, regras e formatos distintos, dividir por coesao antes de pensar em interfaces.

## Interfaces

Principio base definido em `go-implementation/SKILL.md` (Etapa 3, item 4) e expandido em `references/interfaces.md`. Enfase OC: nao usar interface para maquiar violacao de regra de design nem criar indirecao sem consumidor real.

## Funcoes e Metodos

Aplicar early return, extracao de funcao e reducao de branching primeiro. So mover comportamento para metodo quando o receptor realmente conhecer os dados ou invariantes envolvidos.

## Errors

Usar tratamento de erro explicito como parte do fluxo principal. A regra de evitar `else` costuma combinar com verificacao de erro cedo em Go. Ver `agent-governance/references/error-handling.md` para modelo completo.

## Slices e Maps

Promover para tipo dedicado apenas quando houver comportamento recorrente, invariantes ou consultas que merecam nome de dominio.

## Construtores e Fabrica

Pattern Factory Function definido em `go-implementation/SKILL.md` (secao "Patterns frequentes"). Enfase OC: evitar `NewX` vazios que nao validam invariantes nem montam dependencias obrigatorias.

## Testes

Antes de refatorar, proteger comportamento com teste quando a mudanca mexer em fluxo, erros, serializacao, consulta, agregacao ou concorrencia. Preferir teste pequeno e direcionado ao comportamento.

## Sinais de boa adaptacao

A adaptacao esta boa quando:
- o fluxo principal fica mais evidente
- a semantica dos tipos melhora
- a responsabilidade fica mais localizada
- os testes ficam mais simples ou mais precisos
- o codigo continua idiomatico para Go

## Sinais de ma adaptacao

A adaptacao esta ruim quando:
- surgem interfaces sem consumidor
- a navegacao entre tipos aumenta
- um problema local vira reorganizacao ampla
- a quantidade de arquivos cresce sem ganho de coesao
- os nomes ficam teoricos e menos aderentes ao dominio
