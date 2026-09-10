# Pattern Implementation Bundle

## Objetivo
Resultado esperado:
Um agregado de dominio em Go que conduz o Ciclo de Aprovacao com transicoes declaradas em tabela
imutavel, tornando irrepresentavel o encerramento como aprovado sem veredito APPROVED e sem mapa 1:1
completo, e substituindo duas implementacoes divergentes por uma so.

## Contratos preservados
- API publica: os consumidores atuais em internal/runtime e internal/taskloop continuam expondo o mesmo
  comportamento observavel quando a capacidade nova esta desligada; a flag existente permanece opt-in e
  com o mesmo default.
- Invariantes: recursao de revisao permanece hard-bloqueada; o campo de status de revisao permanece
  vazio quando a revisao automatica esta desligada, conforme travado por teste existente.
- Comportamentos que nao podem mudar: os quatro vereditos canonicos e o mapeamento severidade para
  veredito; o formato canonico de bugs consumido pela correcao; a resolucao em cascata dos validadores.

## Participantes concretos
- Papel do pattern: nenhum pattern formal aplicado. A explicitude que motivaria o State e obtida por
  tabela de transicao declarativa mais Value Objects de conjunto fechado.
- Tipo ou modulo real: novo pacote de dominio isolado, contendo o agregado do ciclo, a entidade de
  rodada, os Value Objects de conjunto fechado, a tabela de transicoes e a politica imutavel. As portas
  de revisao, correcao e captura de mudanca sao declaradas no consumidor e implementadas por
  internal/runtime e internal/taskloop.

## Adaptacao para a linguagem
Paradigma:
Go 1.27, imperativo, com tipos concretos, sem heranca e com metodos de struct obrigatorios pela regra
R1 de go-implementation.

Escolha estrutural:
Value Objects com campo interno nao exportado e construtor validante, espelhando o padrao ja existente
no repositorio para identificadores de conjunto fechado. Tabela de transicoes como valor de pacote nao
exportado com prefixo underscore, sem funcao de inicializacao implicita, atendendo as regras R0 e R5.26.
Politica imutavel injetada no construtor. Interfaces declaradas no consumidor e mockadas pela
configuracao de mocks ja existente no projeto.

## Pseudocodigo adaptado
```text
tipo Estado           := VO fechado {EmRevisao, EmCorrecao, Aprovado, Bloqueado}
tipo Veredito         := VO fechado {APPROVED, APPROVED_WITH_REMARKS, REJECTED, BLOCKED}
tipo MotivoDeParada   := VO fechado {aprovado, limite_de_rodadas, nao_convergiu, sem_mudanca, entrada_bloqueada}
_transicoes           := conjunto imutavel de pares (origem, destino)

metodo (c *Ciclo) transicionar(destino Estado, motivo MotivoDeParada) erro:
    se par (c.estado, destino) ausente de _transicoes -> erro de transicao ilegal
    se destino == Aprovado e (c.veredito != APPROVED ou c.mapa incompleto) -> erro de invariante
    c.estado := destino; se destino terminal entao c.motivo := motivo

metodo (c *Ciclo) Submeter(ctx, portas) (Resultado, erro):
    repetir:
        texto := portas.Revisar(ctx, c.alvoDaRodada())
        veredito, achados := c.traduzir(texto)        # fail-closed: sem veredito canonico -> BLOCKED
        mapa := c.confrontarCriterios(achados)
        se veredito aprovado e mapa completo -> transicionar(Aprovado, aprovado); retornar
        fp := c.fingerprint(achados)
        se fp igual a c.fingerprintAnterior -> transicionar(Bloqueado, nao_convergiu); retornar
        se c.rodada >= c.politica.Teto() -> transicionar(Bloqueado, limite_de_rodadas); retornar
        transicionar(EmCorrecao, "")
        mudou := portas.Corrigir(ctx, achados)
        se nao mudou -> transicionar(Bloqueado, sem_mudanca); retornar
        c.fingerprintAnterior := fp; c.rodada++
        transicionar(EmRevisao, "")
```

## Plano de mudanca
1. criar o pacote de dominio com os Value Objects de conjunto fechado e seus construtores
validantes, acompanhados de testes que rejeitam valor fora do conjunto e zero-value.
2. declarar a tabela de transicoes e o metodo de transicao, com um teste por transicao proibida
listada no modelo de dominio.
3. implementar o agregado e o comando unico de submissao, com a politica injetada e o teto
validado na construcao.
4. declarar as portas minimas no consumidor e gerar os mocks pela configuracao existente.
5. migrar o loop atual para consumir o agregado, preservando o comportamento observavel dos
consumidores e mantendo verdes os testes que travam invariantes de nao-recursao.
6. ligar o runtime de sessao ao mesmo agregado, eliminando a segunda implementacao.

## Testes obrigatorios
Teste positivo:
Aprovacao na primeira rodada; aprovacao na terceira rodada apos duas correcoes; motivo de parada
registrado como aprovado; evidencia de cada rodada gravada em endereco proprio e nao sobrescrita.

Teste negativo:
Cada transicao proibida rejeitada; tentativa de aprovar com veredito de ressalvas; tentativa de aprovar
com mapa incompleto; texto de revisor sem veredito canonico resultando em bloqueio; politica com teto
menor que um recusada na construcao.

Teste de regressao:
Fingerprint repetida encerra por nao-convergencia sem consumir o teto; correcao sem mudanca encerra por
ausencia de mudanca; teto esgotado encerra bloqueado e nunca como concluido; cada rodada abre com a
profundidade de invocacao resetada; com a capacidade desligada, o comportamento observavel dos
consumidores permanece identico.

Teste de falha:
Falha de infraestrutura na porta de revisao propaga como erro comparavel por errors.Is e nao como
resultado terminal de negocio, garantindo que o caminho de retry nao reprocesse uma reprovacao legitima.

## Rollback mental
Sinal de que a implementacao ficou cara demais:
Se a tabela de transicoes precisar de condicoes dinamicas por entrada, ou se o metodo de transicao
acumular ramificacoes por estado, a tabela deixou de descrever a maquina e passou a esconde-la.

Acao corretiva:
Reavaliar o State formal com evidencia nova, ou extrair o comportamento por estado para metodos
nomeados do proprio agregado antes de introduzir hierarquia.
