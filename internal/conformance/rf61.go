package conformance

type RF61Scenario struct {
	Index int
	Name  string
}

func RF61Scenarios() []RF61Scenario {
	return []RF61Scenario{
		{Index: 1, Name: "comando seguro permitido"},
		{Index: 2, Name: "git commit nao autorizado"},
		{Index: 3, Name: "git push nao autorizado"},
		{Index: 4, Name: "comando destrutivo"},
		{Index: 5, Name: "teste obrigatorio falhando"},
		{Index: 6, Name: "evidencia ausente"},
		{Index: 7, Name: "evidencia invalida"},
		{Index: 8, Name: "checkpoint valido"},
		{Index: 9, Name: "checkpoint corrompido"},
		{Index: 10, Name: "telemetria sem contagem de tokens disponivel"},
		{Index: 11, Name: "evento desconhecido"},
		{Index: 12, Name: "capability nao suportada"},
		{Index: 13, Name: "adapter retornando erro"},
		{Index: 14, Name: "tentativa de bypass por variacao de comando"},
	}
}
