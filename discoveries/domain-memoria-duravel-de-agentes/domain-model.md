# DOMAIN MODEL

## Titulo
Memoria Duravel de Agentes

## Resumo Executivo
Contexto:
O harness orquestra sessoes de agentes de codigo em quatro CLIs e persiste evidencia rica de cada sessao, mas nao possui conhecimento que sobreviva a sessao. O subsistema atual sobrescreve um arquivo unico por sessao, sem nocao de fato, origem ou durabilidade, e o custo aparece como retrabalho silencioso: convencoes redescobertas, diagnosticos refeitos e erros ja corrigidos repetidos.

Decisao central:
O modelo organiza o ciclo de vida do conhecimento produzido em sessao. A unidade e o Fato, com durabilidade declarada na origem; a durabilidade governa em que camada o Fato vive, se pode ser promovido e quando e arquivado. Nada e removido: arquivamento substitui delete.

Status de prontidao:
done

## Problema e Objetivo
Problema atual:
A memoria e sobrescrita a cada sessao, o tier de task nunca e escrito, a compactacao e uma diretiva textual sem garantia de execucao, e nada do que se aprende atravessa a fronteira de um PRD. O resultado e que o conhecimento de uma sessao e estruturalmente descartavel.

Objetivo de negocio:
Eliminar o custo de re-explicar arquitetura e refazer diagnostico a cada sessao de agente, em uso real de producao, sem alterar o comportamento atual de quem nao ativar a funcionalidade.

Objetivo de modelagem:
Tornar inequivocos a unidade de conhecimento, sua identidade, sua durabilidade, as invariantes que o protegem, a ordem de precedencia entre elas em caso de conflito, e o ownership transacional. Sem isso, a implementacao repetiria o colapso atual entre memoria e arquivo.

## Materiais e Evidencias
Materiais usados:
- PRD `.specs/prd-memoria-duravel-agentes/prd.md` com 35 requisitos funcionais e 14 decisoes fechadas.
- `AGENTS.md` como fonte canonica de convencoes, stack e invariantes de governanca.
- ADR-001 (binario unico via go:embed), ADR-006 (telemetria opt-in), ADR-008 (paridade multi-tool), ADR-016 (config hierarquico), ADR-023 (politica de janela).
- Desenho publico do ai-memory 2.0 como referencia conceitual de captura, consolidacao, recuperacao e handoff.

Confronto com codebase:
- Escopo analisado: repositorio local `ai-spec-harness` em Go 1.27.1, com 7 dependencias diretas e 182 arquivos de teste. Pacotes centrais examinados: `internal/runtime/memory`, `internal/runtime/hooks`, `internal/runtime`, `internal/config`, `internal/taskloop`.
- Status do confronto: executado sobre codigo de producao. Nove achados classificados como confirmado, quatro como ausente. Nenhum achado foi elevado a confirmado com base em teste, mock, fixture ou documentacao.
- Evidencias:
  - Sobrescrita destrutiva a cada sessao: internal/runtime/hooks/memory_persist.go:65 grava com `memory.WriteModeReplace`.
  - Tier de task sem chamador de producao: contrato em internal/runtime/memory/store.go:68, implementacao em internal/runtime/memory/store.go:188, leitura em internal/runtime/runner.go:482, nenhuma escrita.
  - Compactacao como diretiva textual: internal/runtime/runner.go:516.
  - Ponto de ciclo de vida disponivel para captura: internal/runtime/hooks/dispatcher.go:25.
  - Politica de limites sensivel a janela ja existente: internal/runtime/memory/window_policy.go:37.
  - Caminho da memoria derivado de tasksDir: internal/runtime/memory/store.go:88.
  - Marcador de raiz de projeto para a camada nova: internal/config/resolver.go:21.
  - Lock por plataforma reutilizavel: internal/taskloop/orchestrator_lock_unix.go:12.
  - Ausentes: nocao de Fato com identidade e durabilidade, nocao de bastao de continuidade, camada de conhecimento acima do PRD, e ligacao tipada entre conteudos.
- Riscos de compatibilidade: a assinatura de `memory.Store` e consumida por `internal/runtime/runner.go` e por mocks gerados; estender o contrato sem preservar o comportamento de zero-value quebraria a paridade exigida. O caminho de internal/runtime/memory/store.go:88 e composto a partir de `tasksDir`, portanto a camada de projeto exige uma raiz independente em vez de reaproveitar a composicao atual.

## Escopo e Fora de Escopo
Inclui:
- Fato como unidade de conhecimento, com identidade, origem, durabilidade e ligacoes tipadas.
- Tres camadas de memoria: projeto, PRD e task, com resolucao determinista por durabilidade.
- Captura ao fim da sessao com reconciliacao de sessao orfa na abertura seguinte.
- Consolidacao nao destrutiva, idempotencia por conteudo e arquivamento explicito.
- Recuperacao sob orcamento com teto global e cotas por camada.
- Bastao de continuidade com dono unico, lease com prazo e verificacao de processo vivo.
- Deteccao de contradicao sem uso de modelo de linguagem.
- Sanitizacao bloqueante com redacao de trecho.
- Compactacao determinista executada pelo proprio harness.

Exclui:
- Busca semantica, embeddings e indice persistido.
- Servidor, interface web e exposicao de memoria como ferramenta MCP.
- Multiusuario, autenticacao e sincronizacao entre maquinas.
- Camada global do usuario fora do repositorio.
- Promocao automatica para a camada de projeto.
- Enriquecimento de memoria por modelo de linguagem.
- Ativacao por padrao: a funcionalidade nasce opt-in.

## Linguagem Ubiqua
| Termo | Definicao | Sinonimos proibidos/ambiguous | Observacoes |
| --- | --- | --- | --- |
| Fato | Unidade atomica de conhecimento, com chave semantica, conteudo, origem e durabilidade declarada | Nota, Registro, Entrada, Item | Unica unidade que se promove, arquiva e contradiz |
| Pagina | Documento Markdown que agrupa Fatos e e a forma de persistencia da camada | Arquivo de memoria, Blob | Nao e entidade de dominio: e a serializacao do agregado |
| Camada de Memoria | Conjunto consistente de Fatos ativos de um escopo: projeto, PRD ou task | Tier, Nivel, Bucket | Raiz de agregado e fronteira de lock |
| Durabilidade | Classificacao do Fato em Efemero, DePRD ou Duravel | Prioridade, Peso, Confianca | Governa camada, promocao e arquivamento |
| Chave Semantica | Identificador declarado do assunto sobre o qual o Fato afirma algo | Titulo, Slug, Nome | Base da deteccao de contradicao sem modelo de linguagem |
| Promocao | Passagem explicita de um Fato para a camada de projeto | Merge, Sync, Consolidacao | Somente por marcacao explicita; nunca automatica |
| Arquivamento | Retirada de um Fato do conjunto ativo, preservando-o no repositorio | Exclusao, Delete, Poda, Purge | Substitui delete em todo o modelo |
| Bastao de Continuidade | Direito exclusivo de continuar uma linha de trabalho entre sessoes | Lock de sessao, Semaforo, Trava | Lease com prazo e dono unico |
| Contradicao | Divergencia de conteudo entre Fatos de mesma chave semantica | Conflito, Duplicidade, Merge conflict | Distinta de duplicidade, que e conteudo identico |
| Redacao | Substituicao de trecho sensivel por marcador antes de persistir | Mascaramento, Descarte, Bloqueio | O Fato sobrevive; apenas o trecho sai |
| Orcamento de Contexto | Teto de conteudo de memoria injetavel, com cotas por camada | Limite, Budget de token, Quota bruta | Derivado da classe de janela da CLI ativa |
| Evidencia | Prova imutavel e selavel da execucao de uma sessao | Memoria, Historico, Log de memoria | Natureza distinta de memoria: nao se consolida nem se revisa |

## Bounded Contexts e Fronteiras
Contextos:
- Contexto: Memoria
  Objetivo: capturar, consolidar, recuperar e transferir conhecimento produzido em sessoes de agente, com durabilidade explicita e sem perda silenciosa.
  Ownership: proprietario exclusivo de Fato, Camada de Memoria, Bastao de Continuidade, Durabilidade, Chave Semantica, Ligacao e Orcamento de Contexto.
  Fronteiras: consome pontos de ciclo de vida do Runtime de Sessao e o filesystem do repositorio. Nao possui, nao le e nao altera artefato de Evidencia. Nao expoe contrato de rede.

Mapa de contexto:
- Runtime de Sessao e upstream de Memoria: publica os pontos de ciclo de vida que disparam captura e recuperacao. Memoria e conformist em relacao a esses pontos, pois nao pode alterar o ciclo de vida existente sem violar a restricao de zero regressao.
- Evidencia e contexto vizinho deliberadamente separado: guarda prova imutavel e selavel de execucao, enquanto Memoria guarda conhecimento mutavel e consolidado. A relacao e de traducao unidirecional: Memoria emite um resumo da propria operacao para Evidencia registrar, e nunca o contrario.
- Configuracao e upstream de Memoria: fornece parametros de politica pela cascata de precedencia existente. Memoria nao redefine a cascata.
- Curadoria humana atua por interface de linha de comando como ator externo, nao como contexto: e a unica origem valida de promocao para a camada de projeto, junto da marcacao declarada em sessao.

## Workflow Principal
Gatilho:
O workflow de captura, consolidacao, recuperacao e handoff do conhecimento de sessao inicia em dois pontos. No caminho normal, ao encerramento da sessao, em qualquer condicao de saida, inclusive timeout, cancelamento e permissao negada. No caminho de excecao, na abertura da sessao seguinte, quando a sessao anterior morreu sem encerrar e deixou trabalho orfao.

Passos:
1. Na abertura da sessao, reconciliar sessao orfa anterior quando existir, derivando Fatos do registro de eventos que ela deixou.
2. Reivindicar o Bastao de Continuidade da linha de trabalho, ou reconhecer explicitamente que outra sessao viva o detem.
3. Resolver o Orcamento de Contexto a partir da classe de janela da CLI ativa e distribuir cotas por camada.
4. Selecionar Fatos ativos relevantes em cada camada pela politica de relevancia, respeitando a cota e cedendo a sobra as demais camadas.
5. Entregar o contexto selecionado ao Runtime de Sessao, sinalizando contradicoes conhecidas e omissoes por orcamento.
6. Ao encerrar, derivar Fatos do sinal estruturado da sessao e da secao declarada, quando houver.
7. Sanitizar cada Fato candidato, redigindo trechos sensiveis e registrando cada redacao.
8. Consolidar na camada resolvida pela durabilidade: descartar regravacao identica, marcar contradicao quando a chave semantica colidir com conteudo divergente, e nunca remover Fato ativo.
9. Compactar a camada de forma determinista quando exceder limites, arquivando o excedente em vez de apaga-lo.
10. Liberar o Bastao e emitir para Evidencia o resumo da operacao de memoria da sessao.

Ponto de decisao:
- Resolucao de camada por durabilidade declarada: Efemero permanece na camada de task, DePRD na camada de PRD, Duravel torna-se elegivel a promocao.
- Colisao de chave semantica: conteudo identico e idempotencia, conteudo divergente e contradicao.
- Estado do Bastao: livre, detido por processo vivo dentro do prazo, ou reivindicavel por prazo vencido ou dono inexistente.
- Excedente de camada apos consolidacao: arquivar por criterio determinista, nunca remover.
- Conteudo de autoria humana que nao corresponde a Fato valido: preservar, contar no orcamento e sinalizar para compactacao humana.

## Comandos
- Comando: RegistrarFato
  Intencao: afirmar um Fato na camada resolvida pela sua durabilidade.
  Pre-condicoes: chave semantica presente, durabilidade declarada, conteudo sanitizado, lock da camada adquirido.
  Resultado esperado: Fato em estado Ativo, ou nenhuma alteracao quando ja existir Fato identico.
  Falhas de negocio: chave semantica ausente, durabilidade ausente, segredo nao redigivel, serializacao que nao preserva round-trip.

- Comando: PromoverFato
  Intencao: passar um Fato marcado como Duravel para a camada de projeto.
  Pre-condicoes: Fato em estado Ativo, durabilidade Duravel, marcacao explicita de origem declarada ou humana, lock das duas camadas.
  Resultado esperado: Fato presente e ativo na camada de projeto e marcado como Promovido na camada de origem.
  Falhas de negocio: promocao sem marcacao explicita, durabilidade Efemero, Fato ja arquivado.

- Comando: ArquivarFato
  Intencao: retirar um Fato do conjunto ativo preservando-o no repositorio.
  Pre-condicoes: Fato existente, decisao explicita de origem declarada, humana ou de compactacao determinista.
  Resultado esperado: Fato em estado Arquivado, fora do contexto injetavel e presente no repositorio.
  Falhas de negocio: tentativa de remocao fisica, arquivamento de Fato inexistente.

- Comando: RecuperarContexto
  Intencao: montar o contexto de memoria da sessao dentro do orcamento.
  Pre-condicoes: orcamento resolvido pela classe de janela e cotas distribuidas.
  Resultado esperado: conjunto selecionado de Fatos ativos por camada, com contradicoes sinalizadas e omissoes por orcamento declaradas.
  Falhas de negocio: orcamento excedido apos selecao, pagina ilegivel isolada, ausencia total de memoria.

- Comando: CompactarCamada
  Intencao: trazer uma camada de volta aos seus limites sem depender de colaboracao do agente.
  Pre-condicoes: camada excedendo limite de linhas ou bytes, lock adquirido.
  Resultado esperado: camada dentro dos limites, com excedente arquivado e conteudo de autoria humana preservado intacto.
  Falhas de negocio: limite inalcancavel por conteudo humano preservado, que deve ser reportado em vez de silenciado.

- Comando: ReivindicarBastao
  Intencao: obter o direito exclusivo de continuar uma linha de trabalho.
  Pre-condicoes: bastao livre, prazo vencido, ou processo dono inexistente.
  Resultado esperado: bastao detido pela sessao solicitante, com takeover registrado quando aplicavel.
  Falhas de negocio: bastao detido por processo vivo dentro do prazo.

- Comando: LiberarBastao
  Intencao: devolver o direito de continuidade ao encerrar a linha de trabalho.
  Pre-condicoes: sessao solicitante e a detentora registrada.
  Resultado esperado: bastao livre para a proxima sessao.
  Falhas de negocio: liberacao por sessao que nao detem o bastao.

- Comando: ReconciliarSessaoOrfa
  Intencao: recuperar o conhecimento de uma sessao que morreu sem encerrar.
  Pre-condicoes: registro de eventos de sessao anterior sem captura correspondente.
  Resultado esperado: Fatos derivados e consolidados, e sessao marcada como reconciliada.
  Falhas de negocio: registro de eventos ilegivel, reconciliacao ja aplicada.

- Comando: MigrarMemoria
  Intencao: converter memoria em formato antigo para o formato de Fatos.
  Pre-condicoes: comando explicito do operador, backup verificavel gravado antes da conversao.
  Resultado esperado: conteudo integralmente preservado em Fatos, com backup verificavel disponivel.
  Falhas de negocio: migracao ja aplicada, backup nao verificavel.

## Eventos de Dominio
- Evento: FatoRegistrado
  Quando ocorre: apos consolidacao bem-sucedida de um Fato novo em uma camada.
  Quem observa: Evidencia, telemetria opt-in e o verificador de contradicao.
  Impacto: o conjunto ativo da camada muda e passa a influenciar o contexto das sessoes seguintes.

- Evento: FatoArquivado
  Quando ocorre: quando um Fato sai do conjunto ativo por decisao explicita ou por compactacao determinista.
  Quem observa: Evidencia e telemetria opt-in.
  Impacto: reducao do conjunto ativo sem perda de conteudo no repositorio.

- Evento: FatoPromovido
  Quando ocorre: quando um Fato Duravel passa para a camada de projeto por marcacao explicita.
  Quem observa: Evidencia, telemetria opt-in e curadoria humana.
  Impacto: o conhecimento passa a atravessar PRDs e sessoes, e a camada de projeto muda em conteudo versionado.

- Evento: ContradicaoDetectada
  Quando ocorre: quando um Fato novo colide em chave semantica com um Fato ativo de conteudo divergente.
  Quem observa: Evidencia, curadoria humana e a montagem de contexto.
  Impacto: os dois Fatos permanecem, sinalizados, e a decisao de qual prevalece fica com quem cura.

- Evento: SegredoRedigido
  Quando ocorre: quando a sanitizacao substitui um trecho sensivel antes de persistir.
  Quem observa: Evidencia e revisor humano.
  Impacto: o Fato e preservado sem o segredo, e a auditoria registra que houve redacao.

- Evento: CompactacaoExecutada
  Quando ocorre: quando uma camada e trazida de volta aos seus limites pelo harness.
  Quem observa: Evidencia e telemetria opt-in.
  Impacto: o conjunto ativo diminui de forma determinista, com o excedente arquivado.

- Evento: BastaoTransferido
  Quando ocorre: quando o bastao muda de dono, seja por liberacao ordenada, seja por takeover apos prazo vencido ou dono inexistente.
  Quem observa: Evidencia e as sessoes concorrentes.
  Impacto: define qual sessao e a continuacao legitima da linha de trabalho.

## Regras, Politicas e Invariantes
Regras de negocio:
- Regra: a durabilidade declarada de um Fato governa a camada em que ele vive, sua elegibilidade a promocao e o momento do seu arquivamento.
  Motivo: sem uma regra unica de organizacao, a decisao de camada se fragmenta em heuristicas concorrentes e reintroduz a ambiguidade atual.

- Regra: a identidade de um Fato e formada pela chave semantica declarada em conjunto com o hash do conteudo.
  Motivo: a chave estabelece sobre o que o Fato fala, permitindo detectar contradicao sem modelo de linguagem; o hash estabelece se o conteudo mudou, permitindo idempotencia exata.

- Regra: promocao para a camada de projeto ocorre exclusivamente por marcacao explicita.
  Motivo: a camada de projeto e versionada e compartilhada; promocao automatica transformaria ruido de sessao em conhecimento permanente.

- Regra: remocao fisica de Fato nao existe no modelo; arquivamento a substitui integralmente.
  Motivo: a perda silenciosa de conhecimento e o defeito que o modelo existe para corrigir, e nenhum caminho pode reintroduzi-la.

- Regra: em conflito entre invariantes, a ordem de precedencia e seguranca de segredo, depois nao perda de Fato, depois dono unico do bastao, depois orcamento de contexto.
  Motivo: seguranca vem primeiro porque e irreversivel apos publicacao no remoto; nao perda vem em seguida porque e irrecuperavel por inspecao; as duas ultimas produzem dano corrigivel.

Politicas:
- Politica: PoliticaDeRelevancia
  Entradas: camada, durabilidade, chave semantica, task ativa, marcacao de contradicao e parametros da cascata de configuracao.
  Saida/decisao: ordenacao deterministica dos Fatos ativos candidatos ao contexto, reproduzivel para a mesma entrada.

- Politica: PoliticaDeOrcamento
  Entradas: classe de janela da CLI ativa, cotas configuradas por camada e tamanho dos Fatos candidatos.
  Saida/decisao: teto global, cota por camada, cessao de sobra entre camadas e lista de omissoes declaradas.

- Politica: PoliticaDeLease
  Entradas: identidade da linha de trabalho, dono registrado, prazo configurado e existencia do processo dono.
  Saida/decisao: conceder, recusar ou transferir o Bastao de Continuidade, sempre com registro do resultado.

- Politica: PoliticaDeSanitizacao
  Entradas: conteudo candidato e catalogo minimo obrigatorio de padroes sensiveis, extensivel por configuracao.
  Saida/decisao: conteudo com trechos redigidos e lista de redacoes aplicadas, ou recusa de escrita quando o trecho nao for isolavel.

- Politica: PoliticaDeCompactacao
  Entradas: camada excedente, limites vigentes, durabilidade e idade dos Fatos ativos e blocos de autoria humana.
  Saida/decisao: conjunto de Fatos a arquivar de forma deterministica, preservando conteudo de autoria humana e reportando limite inalcancavel.

Invariantes:
- Invariante: nenhum Fato persistido contem segredo.
  Como impedir estado ilegal: sanitizacao e pre-condicao de escrita, nao pos-processamento; trecho sensivel nao isolavel resulta em recusa da escrita com erro tipado.

- Invariante: um Fato ativo so deixa o conjunto ativo por contradicao registrada, promocao marcada ou arquivamento explicito.
  Como impedir estado ilegal: o modelo nao expoe transicao de remocao, e o comando de arquivamento exige decisao explicita registrada.

- Invariante: existe no maximo um dono de Bastao de Continuidade por linha de trabalho.
  Como impedir estado ilegal: concessao condicionada a bastao livre, prazo vencido ou dono inexistente; segunda reivindicacao concorrente e recusada com erro tipado.

- Invariante: o contexto injetado nunca excede o teto global de orcamento.
  Como impedir estado ilegal: selecao interrompida ao atingir o teto, com omissoes declaradas em vez de truncamento silencioso.

- Invariante: a serializacao e a leitura de uma Pagina preservam integralmente o conteudo que nao foi gerado pelo dominio.
  Como impedir estado ilegal: escrita recusada quando o round-trip de leitura e serializacao nao reproduzir o conteudo original de autoria humana.

- Invariante: um Fato Efemero nunca alcanca a camada de projeto.
  Como impedir estado ilegal: promocao valida apenas para durabilidade Duravel, verificada antes da aquisicao de lock da camada de projeto.

## Estados e Transicoes
Estado inicial:
Proposto, atribuido ao Fato derivado da sessao antes de sanitizacao e consolidacao.

Estados validos:
- Proposto
- Ativo
- Contradito
- Promovido
- Arquivado

Transicoes permitidas:
- Proposto -> Ativo: sanitizacao concluida, identidade formada e nenhum Fato identico presente na camada.
- Ativo -> Contradito: registro de Fato de mesma chave semantica com conteudo divergente.
- Ativo -> Promovido: durabilidade Duravel e marcacao explicita de promocao.
- Ativo -> Arquivado: decisao explicita de arquivamento ou compactacao determinista.
- Contradito -> Ativo: contradicao resolvida pelo arquivamento do Fato conflitante.
- Contradito -> Arquivado: decisao explicita de arquivamento.
- Promovido -> Arquivado: decisao explicita de arquivamento na camada de origem.
- Arquivado -> Ativo: reversao explicita de arquivamento por decisao humana.

Transicoes proibidas:
- Ativo -> inexistente: remocao fisica de Fato nao existe no modelo, porque perda silenciosa e o defeito a corrigir.
- Proposto -> Ativo com segredo nao redigido: violaria a invariante de maior precedencia.
- Ativo -> Promovido sem marcacao explicita: promocao automatica e proibida.
- Ativo -> Promovido com durabilidade Efemero: contraria a regra de que durabilidade governa camada.
- Proposto -> Arquivado: um Fato que nunca foi ativo nao tem o que arquivar.

## Tipos Conceituais
Entidades:
- Fato: identidade formada por chave semantica e hash de conteudo; responsavel por carregar uma afirmacao, sua origem, sua durabilidade, suas ligacoes e seu estado.
- CamadaDeMemoria: identidade formada pelo escopo, sendo projeto, PRD ou task; responsavel por manter o conjunto consistente de Fatos ativos do escopo e por aplicar limites e compactacao.
- BastaoDeContinuidade: identidade formada pela linha de trabalho; responsavel por registrar dono, prazo e historico de transferencia.

Value Objects:
- Durabilidade: restringe a classificacao a Efemero, DePRD ou Duravel, e nenhum outro valor e admitido.
- ChaveSemantica: identifica o assunto da afirmacao e e obrigatoria na criacao do Fato.
- HashConteudo: representa o conteudo de forma comparavel para idempotencia exata.
- OrigemDeFato: registra sessao, CLI, task e data, e sustenta a rastreabilidade de cada Fato ate a sessao que o produziu.
- Ligacao: expressa relacao tipada entre Fatos, restrita a substitui, causa, corrige e contradiz.
- OrcamentoDeContexto: expressa teto global e cotas por camada derivados da classe de janela ativa.
- LeaseDeBastao: expressa dono, prazo e referencia de processo, e determina se o bastao e reivindicavel.
- TrechoRedigido: registra que um trecho sensivel foi substituido, sem preservar o valor original.

Agregados:
- CamadaDeMemoria: raiz de consistencia do conjunto de Fatos ativos de um escopo; o limite e uma unica camada, e toda escrita ocorre sob lock dessa camada com serializacao atomica. Justificativa: e a menor fronteira que permite aplicar idempotencia, contradicao e compactacao sem coordenacao distribuida, e coincide com a granularidade do lock por plataforma que o repositorio ja possui.
- BastaoDeContinuidade: raiz de consistencia do direito de continuidade de uma linha de trabalho; o limite e o proprio bastao. Justificativa: um segundo agregado se paga porque o ciclo de vida do bastao e o das camadas sao independentes, e uni-los faria a reivindicacao de continuidade bloquear escrita de conhecimento sem necessidade.

## Erros de Dominio
- Erro: ChaveSemanticaAusente
  Quando ocorre: tentativa de registrar Fato sem chave semantica.
  Impacto no fluxo: escrita recusada e nenhuma alteracao de estado.
  Acao esperada: falhar fechado com erro tipado, sem gravar Fato parcial.

- Erro: DurabilidadeAusente
  Quando ocorre: tentativa de registrar Fato sem durabilidade declarada.
  Impacto no fluxo: escrita recusada, pois a camada de destino seria indeterminada.
  Acao esperada: falhar fechado com erro tipado.

- Erro: SegredoNaoRedigivel
  Quando ocorre: padrao sensivel detectado sem que o trecho possa ser isolado com seguranca.
  Impacto no fluxo: escrita recusada para preservar a invariante de maior precedencia.
  Acao esperada: falhar fechado, registrar a deteccao e nao persistir conteudo algum do Fato.

- Erro: PromocaoSemMarcacao
  Quando ocorre: tentativa de promover Fato sem marcacao explicita, ou com durabilidade Efemero.
  Impacto no fluxo: promocao recusada e camada de projeto inalterada.
  Acao esperada: falhar fechado com erro tipado.

- Erro: BastaoJaReivindicado
  Quando ocorre: reivindicacao de bastao detido por processo vivo dentro do prazo.
  Impacto no fluxo: a sessao solicitante nao assume a continuidade e e informada de quem detem.
  Acao esperada: falhar fechado, informando dono e prazo restante, sem sobrescrever.

- Erro: RoundTripNaoPreservado
  Quando ocorre: a serializacao proposta nao reproduz o conteudo de autoria humana existente na Pagina.
  Impacto no fluxo: escrita recusada para nao reescrever conteudo humano.
  Acao esperada: falhar fechado e reportar a divergencia.

- Erro: PaginaIlegivel
  Quando ocorre: leitura de Pagina corrompida ou invalida.
  Impacto no fluxo: a Pagina e isolada e a sessao prossegue com o restante da memoria.
  Acao esperada: degradar de forma explicita, reportando a Pagina isolada sem abortar a sessao.

- Erro: OrcamentoExcedido
  Quando ocorre: os Fatos selecionados nao caberiam no teto global.
  Impacto no fluxo: selecao interrompida e omissoes declaradas.
  Acao esperada: degradar de forma explicita, declarando o que ficou fora e por que.

- Erro: LimiteInalcancavelPorConteudoHumano
  Quando ocorre: a compactacao determinista nao consegue trazer a camada aos limites sem reescrever conteudo de autoria humana.
  Impacto no fluxo: a camada permanece acima do limite.
  Acao esperada: degradar de forma explicita, reportando a violacao e sinalizando para compactacao humana, sem reescrever o conteudo.

- Erro: MigracaoJaAplicada
  Quando ocorre: comando de migracao executado sobre conteudo ja migrado.
  Impacto no fluxo: nenhuma conversao ocorre.
  Acao esperada: falhar fechado, preservando o estado atual e o backup existente.

## Fronteiras Externas e Traducao
Entradas externas:
- Pontos de ciclo de vida do Runtime de Sessao, que disparam recuperacao na abertura e captura no encerramento.
- Sinal estruturado da sessao encerrada, incluindo task, condicao de saida, resultado de gates, arquivos tocados e comandos de validacao executados.
- Secao declarada pela sessao, quando o agente colabora, como fonte adicional e nao substitutiva.
- Registro de eventos de sessao anterior, usado apenas na reconciliacao de sessao orfa.
- Parametros de politica vindos da cascata de configuracao existente.
- Decisoes de curadoria humana pela interface de linha de comando.

Saidas externas:
- Contexto de memoria entregue ao Runtime de Sessao para composicao de prompt, com contradicoes e omissoes sinalizadas.
- Paginas Markdown gravadas no filesystem do repositorio, versionaveis e editaveis por ferramentas externas.
- Resumo da operacao de memoria emitido para Evidencia registrar no relatorio da sessao.
- Metricas de memoria disponibilizadas a telemetria opt-in.
- Artefato de exportacao autocontido, quando solicitado.

Traducao entre dominio e contrato externo:
- O modelo interno nunca e dirigido pelo formato externo. Fato, Durabilidade, Ligacao e Estado sao traduzidos para secoes e metadados de Pagina na borda de persistencia, e a traducao inversa reconstroi Fatos na leitura.
- A traducao de leitura e obrigada a preservar round-trip do conteudo de autoria humana: o que o dominio nao gerou, o dominio nao reescreve.
- Para o Runtime de Sessao, o dominio expoe um bloco de contexto textual derivado dos Fatos selecionados. O formato desse bloco e contrato de saida e nao pode influenciar a modelagem interna.
- Para Evidencia, a traducao e unidirecional e resumida: Memoria informa o que fez, e nao le nem altera artefato de Evidencia, preservando a natureza selavel e imutavel daquele contexto.
- Nao existe camada de traducao de rede, porque nao existe fronteira de rede no escopo.

## Persistencia, Consistencia e Auditoria
Persistencia necessaria:
- Fatos ativos e arquivados de cada camada, como Markdown versionavel no repositorio, porque a portabilidade e a ausencia de lock-in sao requisitos de produto.
- Metadados de identidade, origem, durabilidade, ligacoes e estado de cada Fato, sem os quais idempotencia, contradicao, promocao e rastreabilidade nao existem.
- Estado do Bastao de Continuidade, incluindo dono, prazo e historico de transferencia.
- Marca de reconciliacao por sessao, para impedir reconciliacao repetida da mesma sessao orfa.
- Backup verificavel gerado antes de qualquer migracao de formato.

Consistencia requerida:
- Hibrida por decisao: forte dentro de uma Camada de Memoria, com escrita atomica sob lock; eventual entre camadas, porque promocao e operacao explicita desacoplada do encerramento da sessao. A consequencia aceita e que promocao pendente existe como estado observavel e precisa aparecer na inspecao.

Auditoria/rastreabilidade:
- Cada Fato deve ser rastreavel ate a sessao, a CLI e a data que o produziram.
- Cada redacao de segredo, cada compactacao, cada promocao, cada arquivamento, cada contradicao e cada transferencia de bastao devem ser recuperaveis a partir dos eventos de dominio emitidos.
- O relatorio de execucao da sessao deve conter o resumo da operacao de memoria: o que foi lido, o que foi gravado, orcamento consumido por camada, compactacao, redacao e takeover.

## Observabilidade e Operacao
Sinais minimos:
- Metricas: numero de Fatos ativos e arquivados por camada, orcamento consumido por camada, contradicoes abertas, redacoes aplicadas, compactacoes executadas, takeovers de bastao, latencia de recuperacao.
- Logs: ativacao e desativacao da funcionalidade, degradacao por pagina isolada, degradacao por orcamento, limite inalcancavel por conteudo humano, recusa de reivindicacao de bastao, recusa de escrita por segredo nao redigivel.
- Alertas: nao se aplica alerta automatizado, porque o dominio executa em processo local de linha de comando e nao possui operador de plantao. A contrapartida e que toda degradacao deve ser visivel no relatorio da sessao.

Falhas operacionais relevantes:
- Sessao morta sem encerramento, deixando trabalho orfao e bastao retido.
- Duas CLIs escrevendo na mesma camada simultaneamente.
- Pagina editada a mao de forma invalida.
- Camada crescendo acima do limite por conteudo de autoria humana.
- Contradicoes acumuladas sem curadoria, degradando a confianca no contexto.

Rollback/contingencia:
- Desativar a funcionalidade por sinalizador de execucao restaura o comportamento anterior sem downgrade de versao, e a desativacao e registrada de forma explicita.
- Migracao e reversivel pelo backup verificavel gravado antes da conversao.
- Arquivamento e reversivel por decisao humana explicita.
- Como as Paginas sao Markdown versionado, o historico do repositorio funciona como ultima linha de recuperacao.

## Economia, Eficiencia e Custos
Decisoes para reduzir custo:
- Nenhum indice persistido: leitura e filtragem determinista a cada consulta elimina staleness, invalidacao e versionamento de formato de indice.
- Reuso do lock por plataforma existente em vez de novo mecanismo de exclusao.
- Reuso dos pontos de ciclo de vida do runtime em vez de novo mecanismo de interceptacao.
- Captura derivada de artefatos que a sessao ja produz, sem chamada de modelo de linguagem no caminho padrao.
- Nenhuma dependencia direta nova: o frontmatter de Pagina e servido por biblioteca de serializacao ja presente.
- Um unico contexto novo e dois agregados, ambos justificados por fronteira de consistencia distinta.

Custo cognitivo:
- Evitou-se banco de dados, servidor, autenticacao, sincronizacao entre maquinas e camada global de usuario.
- Evitou-se enumeracao numerica de durabilidade, que exigiria justificar limiares arbitrarios.
- Evitou-se um agregado por Fato, que espalharia as invariantes por coordenacao entre agregados.
- Evitou-se um agregado unico das tres camadas, que faria escrita de task bloquear leitura de projeto.

Drivers de custo residual:
- Leitura sem indice cresce linearmente com o volume de Fatos, com garantia declarada apenas no volume alvo de centenas de paginas.
- Round-trip de leitura e serializacao exige teste dedicado permanente, porque e a invariante que protege conteudo humano.
- Contradicoes acumuladas dependem de curadoria humana, que e trabalho recorrente e nao automatizavel por decisao.
- Duas fontes de captura significam dois caminhos de teste: o determinista e o dependente de colaboracao do agente.
- O caminho de reconciliacao de sessao orfa e codigo que so executa em falha, portanto exige teste explicito para nao apodrecer.

## Trade-offs e Decisoes
Alternativas rejeitadas:
- Adotar o ai-memory como servidor externo: quebraria o binario unico e tornaria a evidencia dependente de estado externo mutavel.
- Pagina como agregado com Fato sem identidade: perderia a granularidade necessaria para promocao e contradicao.
- Um arquivo por Fato: produziria explosao de arquivos pequenos e historico de repositorio ruidoso.
- Pagina como projecao de um arquivo de dados: destruiria Markdown como fonte de verdade.
- Indice persistido: adicionaria staleness, invalidacao e versionamento sem necessidade medida.
- Promocao automatica por recorrencia ou por categoria: promoveria ruido com a mesma facilidade que conhecimento.
- Poda por idade ou por tamanho: apagaria conhecimento por tempo ou por raridade, nao por validade.
- Falha fechada em toda leitura: uma pagina corrompida abortaria a sessao de trabalho.
- Resumo de sessao gerado por modelo de linguagem no caminho padrao: custo por sessao e resultado nao determinista.
- Camada global do usuario: risco de vazamento de contexto entre repositorios e clientes.

Trade-offs aceitos:
- Memoria capturada sem modelo de linguagem e mais pobre em prosa, e a aposta e que fato derivado de sinal real vale mais em producao e custa zero.
- Consolidacao determinista pode manter Fato obsoleto, e a assimetria e deliberada: manter obsoleto sinalizado e preferivel a perder valido em silencio.
- Leitura sem indice limita a garantia de latencia ao volume alvo, e essa limitacao e declarada em vez de escondida.
- Consistencia eventual entre camadas cria estado de promocao pendente, que precisa ser visivel na inspecao.
- Preservar conteudo de autoria humana significa aceitar que a compactacao pode nao alcancar o limite, reportando a violacao em vez de reescrever texto humano.
- Curadoria explicita para promocao transfere trabalho ao humano, em troca de uma camada de projeto que permanece confiavel.

Decisoes consolidadas:
- Evolucao nativa em Go adotando os conceitos de captura, consolidacao, recuperacao e handoff, sem daemon, sem rede e sem chamada de modelo de linguagem no caminho padrao.
- Fato e o termo canonico e a unidade de conhecimento; Pagina e a forma de persistencia.
- Contexto Memoria proprio, separado de Evidencia, porque conhecimento consolidado e prova selada tem naturezas distintas.
- Tres camadas de memoria: projeto, PRD e task.
- Durabilidade em tres niveis, Efemero, DePRD e Duravel, governando camada e ciclo de vida.
- Identidade do Fato formada por chave semantica declarada e hash de conteudo.
- Captura ao encerramento da sessao, com reconciliacao de sessao orfa na abertura seguinte.
- Sete eventos de dominio, alinhados ao que o relatorio de execucao precisa registrar.
- Ordem de precedencia entre invariantes: segredo, depois nao perda, depois dono unico, depois orcamento.
- Erros tipados com falha fechada na escrita e degradacao explicita na leitura.
- Agregado CamadaDeMemoria como fronteira transacional, com Fato como entidade interna, e BastaoDeContinuidade como agregado proprio.
- Consistencia forte por camada e eventual entre camadas.
- Menor desenho que preserva as invariantes, sem indice persistido e reutilizando lock e pontos de ciclo de vida existentes.
- Promocao para a camada de projeto exclusivamente por marcacao explicita.
- Conteudo de autoria humana preservado, contado no orcamento e sinalizado para compactacao humana, nunca reescrito.

## Itens em Aberto
- Nenhum item aberto bloqueante.

## Proximo Passo Recomendado
create-technical-specification para traduzir este modelo em arquitetura, interfaces e ADRs, seguido de implementacao guiada com as skills go-implementation e design-patterns-mandatory como carga obrigatoria.
