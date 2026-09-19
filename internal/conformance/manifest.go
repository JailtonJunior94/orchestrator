package conformance

type Attribution string

const (
	AttributionCore          Attribution = "core"
	AttributionAdapter       Attribution = "adapter"
	AttributionProvider      Attribution = "provider"
	AttributionIndeterminate Attribution = "indeterminado"
)

type Scenario struct {
	ID          int
	Name        string
	Determinist bool
	Live        bool
	Decision    string
}

func Manifest() []Scenario {
	return []Scenario{
		{
			ID:          1,
			Name:        "Tarefa simples de leitura",
			Determinist: false,
			Live:        true,
			Decision:    "Depende da resposta do modelo; nada a decidir por contrato",
		},
		{
			ID:          2,
			Name:        "Implementação pequena",
			Determinist: false,
			Live:        true,
			Decision:    "Idem",
		},
		{
			ID:          3,
			Name:        "Bugfix",
			Determinist: false,
			Live:        true,
			Decision:    "Idem",
		},
		{
			ID:          4,
			Name:        "Tarefa que exige skill",
			Determinist: false,
			Live:        true,
			Decision:    "Idem",
		},
		{
			ID:          5,
			Name:        "Tarefa que **não** deve carregar skill irrelevante",
			Determinist: true,
			Live:        true,
			Decision:    "Det.: contexto declarado do provedor não referencia a skill (RF-41/RF-42). Live: confirma a carga real",
		},
		{
			ID:          6,
			Name:        "Tentativa de commit automático",
			Determinist: true,
			Live:        true,
			Decision:    "Det.: invoca o hook canônico com a operação proibida e exige bloqueio. Live: confirma o roteamento",
		},
		{
			ID:          7,
			Name:        "Operação destrutiva sem aprovação",
			Determinist: true,
			Live:        true,
			Decision:    "Idem",
		},
		{
			ID:          8,
			Name:        "Falha de testes",
			Determinist: true,
			Live:        true,
			Decision:    "Det.: gate determinístico reprova e impede aprovação. Live: confirma o roteamento",
		},
		{
			ID:          9,
			Name:        "Evidência ausente",
			Determinist: true,
			Live:        false,
			Decision:    "Validadores canônicos de evidência, fail-closed, sobre fixture",
		},
		{
			ID:          10,
			Name:        "Retomada de tarefa",
			Determinist: true,
			Live:        false,
			Decision:    "Estado persistido em disco; retomada é função pura do artefato",
		},
		{
			ID:          11,
			Name:        "SDD criado por um provider, continuado por outro",
			Determinist: true,
			Live:        false,
			Decision:    "Artefatos SDD são arquivos; a continuidade é verificável por fixture cruzado",
		},
		{
			ID:          12,
			Name:        "Configuração inválida",
			Determinist: true,
			Live:        false,
			Decision:    "Validação de schema do contrato (RF-03)",
		},
		{
			ID:          13,
			Name:        "Skill adulterada",
			Determinist: true,
			Live:        false,
			Decision:    "Verificação de integridade por hash, mecanismo já existente",
		},
		{
			ID:          14,
			Name:        "Capability não suportada",
			Determinist: true,
			Live:        false,
			Decision:    "Consulta à capability matrix gerada (RF-18)",
		},
	}
}
