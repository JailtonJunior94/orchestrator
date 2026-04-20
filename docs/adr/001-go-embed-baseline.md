# ADR-001: Uso de go:embed para distribuicao do baseline de governanca

**Status:** Aceita  
**Data:** 2024-01-01  
**Autores:** JailtonJunior94

---

## Contexto

O harness precisa distribuir um conjunto de arquivos de baseline (skills, hooks, configuracoes de agentes) para que o comando `install` os copie para o repositorio alvo. Esses arquivos formam a governanca operacional padrao que o harness aplica.

A questao central e: como empacotar e distribuir esses arquivos junto com o binario compilado?

Restricoes relevantes:
- O binario deve funcionar offline e sem acesso a internet apos instalacao.
- A reproducibilidade entre instalacoes e critica para auditoria.
- O projeto usa GoReleaser para distribuicao multiplataforma.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| `go:embed` (escolhida) | Zero dependencia externa; reproducibilidade total; funciona offline; sem estado de rede | Binario maior; atualizacao do baseline exige nova release |
| Download em runtime | Baseline sempre atualizado sem nova release | Requer conectividade; falha silenciosa possivel; superficie de ataque aumentada |
| Git submodule | Separa ciclo de vida do baseline | Complexidade de clone/update; nao funciona em binario compilado |
| Pacote Go separado | Versioning independente | Adiciona dependencia; complexidade de modulos Go |

## Decisao

Decidimos usar `go:embed` no pacote `internal/embedded` para empacotar todos os assets de baseline diretamente no binario compilado.

O diretorio `internal/embedded/assets/` contem a arvore de arquivos embarcados. A diretiva `//go:embed all:assets` carrega toda a arvore como `fs.FS`, que e passada por injecao de dependencia ao servico de instalacao.

## Consequencias

### Positivas
- Instalacao funciona completamente offline apos download do binario.
- Toda instalacao com a mesma versao do binario produz exatamente os mesmos arquivos.
- Sem surface de ataque de rede durante `install`.
- Simples de testar: basta inspecionar o FS embutido.

### Negativas / Riscos
- Binario maior que o estritamente necessario para a logica de negocio.
- Atualizacoes no baseline exigem nova release e novo download do binario.
- Sincronizacao entre `internal/embedded/assets/` e `.agents/skills/` e manual e propensa a drift.

### Neutras / Observacoes
- O arquivo `skills-lock.json` rastreia versoes de skills externas; o baseline embarcado e o ponto de partida, nao o destino final.

## Referencias

- `internal/embedded/` — pacote com diretiva go:embed
- `internal/install/install.go` — servico que le o FS embarcado
- `internal/embedded/assets/.agents/skills/` — arvore embarcada
