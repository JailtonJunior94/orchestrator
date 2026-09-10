# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Enforcement do OpenCode por exceção, com sanitização de ambiente e handshake ativo
- **Data:** 2026-09-10
- **Status:** Aceita
- **Decisores:** Solicitante do PRD
- **Relacionados:** `prd.md` (RF-19 a RF-21), `techspec.md`, `adr-003-opencode-acp-subcomando.md`

## Contexto

O mecanismo de enforcement deste agente foi decidido **duas vezes**, e o registro dessa correção importa
mais que a decisão final.

A primeira decisão escolheu o hook de permissão, porque a documentação embarcada no binário descreve o
contrato dos hooks como "mutar a saída e retornar vazio", o que sugeria que bloquear por exceção não
seria possível. Verificação empírica derrubou as duas premissas:

- **Bloquear por exceção funciona.** Provado por execução no modo direto **e** no modo ACP, com cliente
  de protocolo escrito para o teste: a ferramenta nunca executa, a chamada é reportada com estado de
  falha, a sessão sobrevive e o modelo lê a mensagem do erro e se adapta. O interceptador cobre a
  superfície inteira, incluindo ferramentas de servidores externos e chamadas dentro de subagente.
- **O hook de permissão é código morto** nesta versão: enumerando todos os disparos de hook no binário,
  ele não aparece — a única ocorrência da string está no texto da documentação embarcada. Um gate
  baseado nele falharia em silêncio, que é o pior modo de falha possível para um controle de segurança.

Verificou-se também que **três interruptores desligam o gate por completo**: uma flag e duas variáveis
de ambiente. Um quarto interruptor, apesar do nome sugestivo, é inofensivo — desliga apenas plugins
embutidos.

## Decisão

O enforcement primário é o **hook de pré-ferramenta lançando exceção**, que delega a decisão aos
validadores canônicos. É paridade direta com o ponto pré-ferramenta dos demais agentes, inspeciona
argumentos e devolve texto de governança que o modelo efetivamente lê.

O hook de permissão é **proibido** no desenho.

Como defesa em profundidade, o bloco de permissões nega declarativamente o que é categoricamente
proibido, com padrões finos.

Contra os interruptores, **duas camadas**:

1. O harness nunca passa a flag de modo puro e **sanitiza o ambiente do processo filho que ele mesmo
   cria**, removendo os interruptores conhecidos. Isso é distinto de alterar o ambiente do usuário: o
   harness é dono do processo que cria. O agente cuja política de ambiente é vazia tem o ambiente
   herdado intacto, o que preserva os demais agentes sem regressão.
2. **Handshake ativo**: o plugin sinaliza sua carga em um arquivo único por sessão, e a sessão é abortada
   se o sinal não chegar antes do primeiro prompt. Esta camada valida o **efeito**, não a causa, e por
   isso cobre também vetores futuros desconhecidos.

O handshake encaixa no único ponto correto do ciclo de vida: depois da negociação do protocolo, quando o
subprocesso já carregou plugins, e antes do primeiro prompt. O caminho do sentinela é removido antes do
spawn, porque sentinela de execução anterior faria o handshake passar com o plugin desligado.

Três decisões de comportamento do plugin, todas derivadas da mesma postura:

- **Timeout do validador é negação**, não aprovação.
- **Validador ausente** bifurca por modo: em execução orquestrada é falha fechada; em uso interativo
  avisa uma vez e segue, porque ali o usuário está no comando.
- **Só o ponto de pré-ferramenta bloqueia** — é o único que impede algo ainda não feito.

## Alternativas Consideradas

**Hook de permissão como mecanismo primário.** Foi a decisão inicial, revertida por prova de que é código
morto. Registrada aqui porque a reversão é a informação relevante.

**Apenas bloco de permissões declarativo, sem plugin.** Vantagem: nenhum código a manter, nenhuma
dependência de runtime. Desvantagem: perde inspeção de argumentos e a delegação aos validadores
canônicos, criando governança estática que os demais agentes não têm. Rejeitada.

**Apenas detecção estática dos interruptores.** Vantagem: menos código. Desvantagem: cega para qualquer
mecanismo novo de desligamento, com modo de falha silencioso. Rejeitada.

**Neutralizar as variáveis também no ambiente do usuário.** Rejeitada explicitamente: sobrescrever
intenção do usuário no shell dele é desproporcional.

## Consequências

### Benefícios Esperados

- Paridade de efeito com o ponto pré-ferramenta dos demais agentes, comprovada por execução.
- O gate deixa de ser desligável pelos três vetores conhecidos e passa a ser verificado por efeito.
- A mensagem de bloqueio é instrução corretiva: o modelo lê e se adapta, em vez de tentar contornar às
  cegas.

### Trade-offs e Custos

- Um plugin em outra linguagem a manter.
- Custo por chamada de ferramenta, mitigado por três camadas: filtro por ferramentas que mutam o
  repositório, verificação única de existência por sessão, e memoização por ferramenta e arquivos.
- O bloqueio é efetivo **por implementação**, não por contrato documentado — o tipo declara retorno
  vazio. Uma versão futura poderia mudar isso.

### Riscos e Mitigações

**O comportamento de bloqueio pode mudar a montante.** Impacto: gate silenciosamente inerte. Mitigação: o
handshake prova carga, e um teste-sonda dedicado prova bloqueio; ambos falham de forma visível.

**Dois interruptores foram testados apenas no modo direto**, não no modo ACP. Impacto: teórico — o modo
direto e o ACP apresentaram comportamento idêntico em todos os demais testes. Mitigação: a defesa adotada
**não depende** desse resultado, porque o handshake valida efeito; ainda assim, replicar os dois testes
no modo ACP é obrigação declarada.

**Negar uma ferramenta não impede o objetivo.** Foi observado que, bloqueado num caminho, o modelo tenta
outro. Mitigação: bloquear por **intenção**, não por nome de ferramenta.

## Plano de Implementação

Política de ambiente e validação de argumentos no registro; waiter do sentinela como componente próprio;
ponto de extensão no cliente de protocolo seguindo precedente já existente, para não churnar os fakes de
teste; plugin com as três camadas de custo.

## Monitoramento e Validação

Métrica: contagem de sessões recusadas por pré-condição — crescimento indica ambiente de integração mal
configurado. Sinal de sucesso: o bloqueio é observável no próprio fluxo do protocolo, como atualização
de chamada de ferramenta com estado de falha e o texto da exceção, permitindo registrar evidência sem ler
o log do plugin.

## Impacto em Documentação e Operação

Guia de instalação (dependência de runtime do plugin), matriz de paridade de hooks e o guia de resolução
de problemas, com os três interruptores nomeados.

## Revisão Futura

Revisitar se o hook de permissão passar a ser disparado — conceitualmente ele é o ponto certo, por ter
canal de negação declarado. Antes de confiar nele, exigir prova pela mesma sonda.
