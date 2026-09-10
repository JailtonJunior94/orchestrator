# Pattern Implementation Bundle

## Objetivo
Resultado esperado:
Uma fachada concreta em Go que seja o unico ponto de contato do Runtime de Sessao com o subsistema de memoria duravel, satisfazendo uma interface pequena declarada no pacote consumidor, com todos os colaboradores injetados por construtor, sem heranca, sem estado global e sem funcao livre. Com a feature desativada, o caminho executado permanece byte-identico ao atual.

## Contratos preservados
- API publica: a interface atual de quatro metodos do pacote de memoria permanece existindo e continua satisfeita pela implementacao simples de arquivo, para nao quebrar internal/runtime/runner.go:470, internal/runtime/runner.go:481, internal/runtime/runner.go:482 e internal/runtime/hooks/memory_persist.go:65, nem o mock gerado declarado em mockery.yml:58. A superficie de linha de comando existente nao muda: o contrato bidirecional verificado em cmd/ai_spec_harness/cli_contract_test.go:81 e cmd/ai_spec_harness/cli_contract_test.go:189 continua valido, e apenas ganha entradas novas.
- Invariantes: precedencia segredo, depois nao perda de Fato, depois dono unico de bastao, depois orcamento de contexto. Remocao fisica de Fato nao existe. Round-trip de leitura e serializacao preserva conteudo de autoria humana. Fato efemero nunca alcanca a camada de projeto. Promocao somente por marcacao explicita.
- Comportamentos que nao podem mudar: com diretorio de tasks vazio, nenhuma injecao de contexto ocorre e nenhum hook de memoria e registrado, conforme internal/runtime/runner.go:463, internal/runtime/runner.go:476 e internal/runtime/runner.go:547. O bloco de contexto atual, montado em internal/runtime/runner.go:490, permanece reproduzivel byte a byte. Os shell hooks do modo interativo nao sao modificados. A secao de metricas permanece a ultima secao do relatorio de execucao, invariante mantido por internal/runtime/persistence/report.go:104.

## Participantes concretos
- Papel do pattern: Fachada.
- Tipo ou modulo real: struct concreta no pacote de memoria duravel, exposta ao consumidor por interface pequena declarada no pacote de runtime, no idioma de interface no consumidor que o repositorio adota.
- Papel do pattern: Subsistema, colaborador de serializacao.
- Tipo ou modulo real: leitor e serializador de Pagina, recebendo a abstracao de filesystem por construtor, com garantia de round-trip lossless.
- Papel do pattern: Subsistema, colaborador transacional.
- Tipo ou modulo real: agregado de camada, com lock por plataforma reutilizado de internal/taskloop/orchestrator_lock_unix.go:17 e escrita atomica portada de internal/sdd/state.go:299.
- Papel do pattern: Subsistema, colaboradores de politica.
- Tipo ou modulo real: cinco structs stateless com metodos, no idioma ja usado em internal/runtime/memory/window_policy.go:37, sem interface porque cada uma tem implementacao unica.
- Papel do pattern: Cliente que se beneficia da simplificacao.
- Tipo ou modulo real: o runner, que passa a depender de uma porta em vez de cinco colaboradores.
- Papel do pattern: Cliente que deliberadamente nao usa a fachada.
- Tipo ou modulo real: os subcomandos de operacao humana, que acessam os colaboradores em granularidade total.

## Adaptacao para a linguagem
Paradigma:
Go, orientacao a objetos leve com tipos estruturais. Composicao sobre heranca por imposicao da linguagem e das regras estritas da skill de implementacao Go. Interfaces pequenas declaradas no consumidor. Injecao por construtor. Zero estado global.

Escolha estrutural:
A fachada e uma struct com campos de dependencia e metodos, nunca uma classe base com hooks. As politicas nao viram interfaces porque nao existe segunda variante plausivel, o que evitaria abstracao vazia. Os erros de dominio sao valores sentinela no nivel do pacote, combinados com envelopamento por contexto, para que o chamador possa distinguir falha fechada de degradacao usando as funcoes de comparacao de erro da biblioteca padrao. Os metodos stateless auxiliares do pacote seguem o agrupador stateless existente, respeitando a regra que proibe funcao livre. A ordem de execucao fica explicita no corpo dos metodos, sem despacho implicito, para nao esconder ownership de recurso nem limite transacional.

## Pseudocodigo adaptado
```text
// pacote consumidor declara a porta estreita
interface PortaMemoria
    MontarContexto(ctx, escopo, classeJanela) -> (blocoContexto, sinalizacoes, erro)
    RegistrarSessao(ctx, sinalEstruturado, secaoDeclarada) -> (resumo, erro)

// pacote de memoria duravel implementa a fachada
struct Fachada
    fsys           AbstracaoFilesystem      // injetada, corrige o desvio de os.* direto
    paginas        LeitorSerializadorPagina // injetado
    camadas        AgregadoCamada           // injetado
    relevancia     PoliticaRelevancia       // struct stateless
    orcamento      PoliticaOrcamento        // struct stateless
    sanitizacao    PoliticaSanitizacao      // struct stateless
    compactacao    PoliticaCompactacao      // struct stateless
    lease          PoliticaLease            // struct stateless
    cfg            ConfiguracaoEfetiva      // resolvida pela cascata existente

func NovaFachada(fsys, paginas, camadas, cfg) *Fachada   // injecao por construtor

metodo (f *Fachada) MontarContexto(...)
    SE NAO f.cfg.Ativa ENTAO RETORNAR contexto vazio, sem sinalizacoes, sem erro
    ...  // ordem explicita: ler, orcar, ordenar, cortar, renderizar

metodo (f *Fachada) RegistrarSessao(...)
    ...  // ordem explicita: derivar, sanitizar, travar, consolidar, compactar, destravar

// consumidor: uma unica dependencia, resolvida por configuracao
SE cfg.MemoriaDuravelAtiva ENTAO porta <- NovaFachada(...) SENAO porta <- caminho anterior
```

## Plano de mudanca
1. Adicionar escrita atomica a abstracao de filesystem, portando o padrao de internal/sdd/state.go:299 ate internal/sdd/state.go:321, com implementacao real e implementacao falsa, e teste que prove que arquivo parcial nunca fica visivel.
2. Implementar o leitor e serializador de Pagina e travar o round-trip lossless com teste dedicado, antes de qualquer consumidor existir.
3. Implementar os tipos de dominio e as cinco politicas stateless, cada uma com teste proprio, sem tocar o consumidor.
4. Implementar o agregado de camada com lock por plataforma e escrita atomica, tratando explicitamente o lock orfao de internal/taskloop/orchestrator_lock_windows.go:15 por prazo vencido e por ausencia do processo dono.
5. Implementar a fachada, orquestrando os colaboradores na ordem definida, com traducao de erro na fronteira.
6. Declarar a porta estreita no pacote consumidor e ligar a fachada por resolucao de configuracao, mantendo o caminho anterior intacto quando a feature esta desativada.
7. Regenerar os mocks derivados de interface conforme mockery.yml:58 e confirmar o gate de mocks.
8. Expor os subcomandos de operacao humana acessando os colaboradores diretamente, e atualizar o contrato declarado de linha de comando exigido por cmd/ai_spec_harness/cli_contract_test.go:81.
9. Expor metricas de memoria pelo mapa de campos extra do conjunto de metricas existente, aproveitando que a renderizacao do relatorio itera os campos de forma generica em internal/runtime/persistence/report.go:71.

## Testes obrigatorios
Teste positivo:
Fachada com feature ativada monta contexto com Fatos das tres camadas, dentro do teto global e das cotas, e registra a sessao consolidando na camada resolvida pela durabilidade, devolvendo resumo com escritas, redacoes e compactacao. Executado com filesystem falso, sem tocar disco real.

Teste negativo:
Fato sem chave semantica, Fato sem durabilidade, trecho sensivel nao isolavel, promocao sem marcacao, promocao de Fato efemero e reivindicacao de bastao detido por processo vivo dentro do prazo, cada um falhando fechado com erro tipado distinguivel e sem escrever nada.

Teste de regressao:
Com a feature desativada, o bloco injetado e byte-identico ao produzido hoje em internal/runtime/runner.go:490, cobrindo os casos de ausencia total de memoria, somente workflow, workflow e task em ordem, e diretiva de compactacao anexada. A suite existente de memoria, hooks e runner permanece verde sem alteracao de expectativa. O relatorio de execucao mantem a secao de metricas como ultima secao.

Teste de falha:
Pagina corrompida isolada com a sessao prosseguindo. Compactacao que nao alcanca o limite por conteudo de autoria humana reportando a violacao sem reescrever. Dois processos reais competindo pela mesma camada, exercitados com processos separados e nao apenas com goroutines, sem corrupcao e sem perda de Fato. Lock orfao detectado e nunca sobrescrito em silencio. Falha do hook de persistencia deixando de ser silenciosa, dado que hoje o erro de despacho e descartado em internal/runtime/runner.go:428.

## Rollback mental
Sinal de que a implementacao ficou cara demais:
A fachada comeca a acumular decisao de dominio em vez de apenas ordem de chamada. A assinatura dos metodos cresce em parametros para servir dois clientes com necessidades diferentes. O consumidor passa a pedir operacoes internas uma a uma, provando que precisa de granularidade total. O numero de colaboradores cai para dois ou tres sem ordem obrigatoria entre eles. Os testes da fachada passam a exigir montagem de cinco dublês para exercitar um caminho simples.

Acao corretiva:
Dissolver a fachada e voltar a alternativa mais simples registrada, com metodos de alto nivel no agregado de camada chamados diretamente pelo consumidor, mantendo as politicas stateless como estao. A reversao e barata por construcao, porque os colaboradores nao conhecem a fachada e nao dependem dela; remover a fachada nao exige tocar nenhum colaborador, apenas religar o consumidor.
