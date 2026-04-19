package metrics

// CostBreakdown separa o custo contextual em tres eixos de medicao.
// Todos os valores sao estimativas operacionais (chars/3.5), nao tokens reais do provedor.
//
// Os tres eixos:
//   - OnDisk: custo maximo potencial se todos os arquivos da skill fossem carregados.
//   - Loaded: custo dos arquivos na sessao base (apenas SKILL.md, sem referencias).
//   - IncrementalRef: custo marginal medio por referencia adicional carregada.
type CostBreakdown struct {
	// OnDisk: tokens estimados para todos os arquivos em disco (SKILL.md + todas as referencias).
	// ESTIMATIVA: chars/3.5 sobre o total de bytes em disco.
	OnDisk int `json:"on_disk"`
	// Loaded: tokens estimados para os arquivos na sessao base (apenas SKILL.md).
	// ESTIMATIVA: chars/3.5 sobre o SKILL.md. Subconjunto de OnDisk.
	Loaded int `json:"loaded"`
	// IncrementalRef: tokens estimados por referencia adicional carregada (media do conjunto em disco).
	// ESTIMATIVA: media(chars/3.5) sobre os arquivos de referencia. Zero quando nao ha referencias.
	IncrementalRef int `json:"incremental_ref"`
	// RefCount: numero de arquivos de referencia em disco para esta skill.
	RefCount int `json:"ref_count"`
}

// CostNote documenta os limites do modelo de custo.
// Deve ser incluido em qualquer relatorio ou baseline que exponha esses valores.
const CostNote = `Estimativa usa chars/3.5 por default (sem dependencia externa).
Para contagem precisa (~15% mais acurada), use: ai-spec-harness metrics --precise
  --precise usa tiktoken cl100k_base (cl100k). Requer download inicial do modelo BPE.
Os limiares de budget foram calibrados para a heuristica chars/3.5.`
