# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Pré-condições de enforcement como conceito de primeira classe
- **Data:** 2026-09-10
- **Status:** Aceita
- **Decisores:** Solicitante do PRD
- **Relacionados:** `prd.md` (RF-22 a RF-28), `techspec.md`, `adr-002-catalogo-de-agentes-registro-unico.md`

## Contexto

A matriz de paridade do harness afirma cobertura de enforcement que **não é verificada**. Quatro fatos
estabelecidos por teste e leitura:

- **Nenhum teste do repositório prova disparo de hook.** Todos verificam que o arquivo foi escrito. Teste
  de escrita não é teste de disparo, e a diferença permitiu que uma chave de evento inválida
  sobrevivesse: o gate de encerramento de um dos agentes nunca rodou.
- Os hooks de projeto de um agente **disparam**, mas apenas se a pasta estiver na lista de pastas
  confiáveis. Provado por teste de disparo real: a mesma configuração falhou em pasta não confiável e
  funcionou em pasta confiável.
- Os hooks de outro agente exigem **hash confiado**, concedido apenas por interface interativa. Sem isso
  ficam inertes — e na máquina de desenvolvimento onde a verificação foi feita, não havia nenhum hash
  confiado registrado.
- Um terceiro agente tem interruptores de ambiente que desligam o gate.

Ou seja: em dois dos quatro agentes, o gate pode estar inerte **e nada avisa**.

## Decisão

Pré-condição de enforcement passa a ser conceito de primeira classe do catálogo, com tipo próprio de
conjunto fechado e remédio acionável embutido.

A verificação ganha **dois estados novos**, ambos exigidos pela postura do modelo:

- **Inerte** — o artefato existe e está atualizado, mas a pré-condição não está satisfeita: o gate não
  dispara. Conta como **falha**, porque um gate inerte é indistinguível de um gate que aprovou.
- **Desconhecido** — a pré-condição só é verificável executando um binário, e a verificação não foi
  solicitada. **Não é sucesso**; é ausência de informação, e é impresso como tal.

A separação entre detecção e diagnóstico preserva a regra de segurança operacional: **a detecção
continua proibida de executar binários**. A verificação de pré-condição que exige comunicação com um CLI
pertence ao comando de diagnóstico, que é invocado explicitamente e anuncia o que vai executar. As
pré-condições verificáveis por leitura de arquivo ou de ambiente são avaliadas sempre.

O harness **nunca** concede confiança pelo usuário e **nunca** usa a flag de contorno de confiança:
confiar em hook por código é escalar privilégio em silêncio.

A matriz de paridade passa a ser verificada por gate de build que falha quando um agente deixa de cobrir
um ponto canônico, quando aponta para validador divergente, **ou quando a célula não tem teste de
disparo real associado**.

## Alternativas Consideradas

**Documentar as pré-condições no guia, sem verificar.** Vantagem: zero código. Desvantagem: reintroduz
exatamente o buraco que a entrega fecha — hoje um gate está inerte e nada avisa. Rejeitada.

**Usar a flag de contorno de confiança.** Vantagem: funciona sempre. Desvantagem: o próprio nome denuncia
o risco; o harness passaria a executar hooks sem o controle de procedência que o CLI exige por desenho.
Rejeitada.

**Ler o formato interno de configuração dos CLIs em vez de usar o canal oficial.** Vantagem: nenhum
binário executado em lugar nenhum. Desvantagem: acopla o harness a formato interno de terceiro, com
quebra silenciosa quando o layout mudar. Rejeitada para o caso que tem canal oficial de consulta;
adotada, sem alternativa, para o caso em que o estado vive apenas em arquivo.

## Consequências

### Benefícios Esperados

- Um gate que não pode disparar deixa de ser contabilizado como ativo.
- O usuário recebe o remédio exato, em vez de descobrir meses depois que a governança não rodava.
- A matriz de paridade deixa de poder mentir: cada célula exige prova de disparo.

### Trade-offs e Custos

- Mais estados na saída da verificação, com custo de leitura.
- Um comando de diagnóstico que executa binário de terceiro, com timeout e tratamento de falha próprios.
- A camada de disparo verdadeiro pelo CLI exige os quatro binários instalados e autenticados, e por isso
  **não é gate de merge**. Isso é dito explicitamente, e não vendido como cobertura de CI.

### Riscos e Mitigações

**A camada de prova real não roda no CI padrão.** Impacto: uma mudança de nome de evento a montante
passaria pelas camadas que testam o contrato próprio. Mitigação: job noturno seguindo o padrão de build
tag e workflow separado que o repositório **já tem** para caso análogo; o job falha se qualquer célula
for pulada, para não virar teste vazio.

**Arquivo de configuração de terceiro em formato não estrito.** Um deles traz comentários de linha e não
é interpretável diretamente. Mitigação: remoção de comentários fora de strings, sem expressão regular,
porque uma regex sobre o marcador de comentário engoliria endereços dentro de valores.

**Consolidação de estado.** Mitigação: aplica-se o pior estado — um único hook não confiado invalida o
conjunto, porque paridade parcial é indistinguível de ausência de paridade.

## Plano de Implementação

Tipos no catálogo; verificadores em pacote próprio, um por mecanismo; roteamento por tipo de pré-condição
na verificação; estados novos na saída; gate de paridade com exigência de teste de disparo; correção da
chave de evento inválida e da asserção de teste que hoje protege o comportamento errado.

## Monitoramento e Validação

Sucesso: nenhuma célula da matriz sem teste de disparo associado; a verificação reporta o escopo real em
que cada hook está ativo. Sinal de revisão: crescimento de estado desconhecido indica que o diagnóstico
não está sendo usado.

## Impacto em Documentação e Operação

Guia de instalação com os pré-requisitos por agente; matriz de degradação; guia de resolução de
problemas; e a nota de governança que hoje afirma, incorretamente, que um dos agentes não tem hooks
nativos.

## Revisão Futura

Revisitar quando algum dos CLIs oferecer concessão de confiança não interativa, ou quando a camada de
disparo real puder rodar no CI padrão.
