# Canonical Skill -- Modelo de Custo Contextual

## Objetivo

Esta skill define como medir e controlar o custo contextual de referencias
carregadas por sessao de IA. O custo e medido em tokens estimados (chars/3.5),
nao em tokens reais do provedor.

## Quando Usar

- Auditar crescimento de custo entre releases
- Verificar se custo da sessao esta dentro do budget
- Gerar relatorio de custo separado em tres eixos

## Tres Eixos de Custo

### on-disk

Tokens para todos os arquivos em disco da skill (SKILL.md e todas as
referencias). Representa o custo maximo potencial se todos os arquivos
fossem carregados simultaneamente em uma unica sessao.

Calculo: sum(chars / 3.5) para todo arquivo em disco da skill.

### loaded

Tokens dos arquivos efetivamente carregados na sessao atual. Sempre um
subconjunto de on-disk. Normalmente apenas SKILL.md e referencias
selecionadas sao carregadas, nao o conjunto completo.

Calculo: sum(chars / 3.5) para arquivos no contexto ativo da sessao.

### incremental-ref

Custo marginal por referencia adicional carregada individualmente.
Cada nova referencia adicionada ao contexto incrementa o custo em
aproximadamente este valor.

Calculo: media dos tokens por arquivo de referencia no conjunto em disco.

## Restricoes

- Nao inferir consumo real do provedor sem observabilidade adequada.
- Todos os valores sao estimativas operacionais (chars/3.5).
- Atualizar o baseline versionado a cada release publicada.
- Thresholds conservadores: tolerar variacao normal, detectar crescimento silencioso.

## Gate de Regressao

O gate de regressao falha quando o custo medido ultrapassa o baseline por
uma tolerancia definida (padrao: 25%). Esta tolerancia e conservadora para
absorver variacao irrelevante, mas sensivel para detectar regressao material
entre releases do projeto.

## Notas de Implementacao

Os valores de baseline devem ser atualizados a cada release com medicao
real sobre os artefatos canonicos do projeto. Nunca inferir crescimento
a partir de dados estimados sem confirmacao empirica.

Ao atualizar o baseline, registrar a versao do release e a data da medicao
no proprio arquivo JSON de baseline para rastreabilidade futura.

## Glossario

chars: quantidade de bytes no arquivo medida por len(text) em Go.
tokens_est: estimativa de tokens calculada como chars/3.5, arredondada.
on-disk: soma dos tokens_est de todos os arquivos em disco da skill.
loaded: tokens_est dos arquivos carregados no contexto da sessao base.
incremental-ref: media de tokens_est dos arquivos de referencia da skill.
baseline: valores de custo esperados para um conjunto de artefatos no release.
tolerancia: margem percentual acima do baseline que o gate permite sem falhar.
gate: verificacao automatizada que impede crescimento silencioso entre releases.
