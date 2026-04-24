package taskloop

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestParseTasksFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
		wantErr bool
	}{
		{
			name: "tabela valida com 3 tasks",
			content: `# Resumo

## Tarefas

| # | Titulo | Status | Dependencias | Paralelizavel |
|---|--------|--------|-------------|---------------|
| 1.0 | Setup domain | pending | — | — |
| 2.0 | Implement ports | pending | 1.0 | Nao |
| 3.0 | Add adapters | done | 1.0, 2.0 | Nao |
`,
			want:    3,
			wantErr: false,
		},
		{
			name: "tabela com status em portugues",
			content: `| # | Titulo | Status | Dependencias | Paralelizavel |
|---|--------|--------|-------------|---------------|
| 1.0 | Task A | Concluido | — | — |
| 2.0 | Task B | pendente | 1.0 | — |
`,
			want:    2,
			wantErr: false,
		},
		{
			name:    "sem tabela",
			content: "# Apenas um titulo\n\nSem tabela aqui.\n",
			want:    0,
			wantErr: true,
		},
		{
			name:    "conteudo vazio",
			content: "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := ParseTasksFile([]byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Fatal("esperava erro, mas nao recebeu")
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if len(entries) != tt.want {
				t.Fatalf("esperava %d entries, recebeu %d", tt.want, len(entries))
			}
		})
	}
}

func TestParseTasksFile_Fields(t *testing.T) {
	content := `| # | Titulo | Status | Dependencias | Paralelizavel |
|---|--------|--------|-------------|---------------|
| 1.0 | Setup domain | pending | — | — |
| 2.0 | Implement ports | in_progress | 1.0 | Nao |
| 3.0 | Add adapters | done | 1.0, 2.0 | Com 2.0 |
`
	entries, err := ParseTasksFile([]byte(content))
	if err != nil {
		t.Fatalf("erro: %v", err)
	}

	// Task 1
	if entries[0].ID != "1.0" {
		t.Errorf("task 1 ID = %q, want 1.0", entries[0].ID)
	}
	if entries[0].Status != "pending" {
		t.Errorf("task 1 Status = %q, want pending", entries[0].Status)
	}
	if len(entries[0].Dependencies) != 0 {
		t.Errorf("task 1 Deps = %v, want empty", entries[0].Dependencies)
	}

	// Task 2
	if entries[1].Status != "in_progress" {
		t.Errorf("task 2 Status = %q, want in_progress", entries[1].Status)
	}
	if len(entries[1].Dependencies) != 1 || entries[1].Dependencies[0] != "1.0" {
		t.Errorf("task 2 Deps = %v, want [1.0]", entries[1].Dependencies)
	}

	// Task 3
	if len(entries[2].Dependencies) != 2 {
		t.Errorf("task 3 Deps = %v, want [1.0, 2.0]", entries[2].Dependencies)
	}
}

func TestParseTasksFile_WithArquivoColumn(t *testing.T) {
	content := `# Resumo das Tarefas

## Tarefas

| # | Título | Arquivo | Status | Dependências | Paralelizável |
|---|--------|---------|--------|-------------|---------------|
| 1.0 | Representar Travel Rule | task-1.0-shared.md | pending | — | Sim, com 2.0 |
| 2.0 | Declarar contrato | task-2.0-contract.md | pending | — | Sim, com 1.0 |
| 3.0 | Preservar filtros | task-3.0-service.md | pending | 2.0 | Não |
| 4.0 | Registrar templates | task-4.0-tapi.md | done | — | Sim |
| 5.0 | Implementar adapter | task-5.0-adapter.md | pending | 1.0, 2.0, 4.0 | Não |
`
	entries, err := ParseTasksFile([]byte(content))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("esperava 5 entries, recebeu %d", len(entries))
	}

	// Verificar que Status e Dependencies foram mapeados corretamente
	if entries[0].Status != "pending" {
		t.Errorf("task 1.0 Status = %q, want pending", entries[0].Status)
	}
	if len(entries[0].Dependencies) != 0 {
		t.Errorf("task 1.0 Deps = %v, want empty", entries[0].Dependencies)
	}
	if entries[3].Status != "done" {
		t.Errorf("task 4.0 Status = %q, want done", entries[3].Status)
	}
	if len(entries[4].Dependencies) != 3 {
		t.Errorf("task 5.0 Deps = %v, want [1.0, 2.0, 4.0]", entries[4].Dependencies)
	}

	// FindEligible deve retornar 1.0 e 2.0 (sem deps) mas nao 5.0 (deps nao satisfeitas)
	eligible := FindEligible(entries, nil)
	ids := make(map[string]bool, len(eligible))
	for _, e := range eligible {
		ids[e.ID] = true
	}
	if !ids["1.0"] || !ids["2.0"] {
		t.Errorf("1.0 e 2.0 deveriam ser elegiveis, recebeu %v", eligible)
	}
	if ids["5.0"] {
		t.Error("5.0 nao deveria ser elegivel (deps nao satisfeitas)")
	}
}

func TestParseTasksFile_WithoutArquivoColumn(t *testing.T) {
	content := `# Resumo das Tarefas

## Tarefas

| # | Título | Status | Dependências | Paralelizável |
|---|--------|--------|-------------|---------------|
| 1.0 | Pacote de taxonomia | done | — | Com 2.0 |
| 2.0 | Modelos de domínio | done | — | Com 1.0 |
| 3.0 | Validação de domínio | pending | 1.0, 2.0 | Não |
`
	entries, err := ParseTasksFile([]byte(content))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("esperava 3 entries, recebeu %d", len(entries))
	}
	if entries[0].Status != "done" {
		t.Errorf("task 1.0 Status = %q, want done", entries[0].Status)
	}
	if entries[2].Status != "pending" {
		t.Errorf("task 3.0 Status = %q, want pending", entries[2].Status)
	}
	if len(entries[2].Dependencies) != 2 {
		t.Errorf("task 3.0 Deps = %v, want [1.0, 2.0]", entries[2].Dependencies)
	}
}

func TestReadTaskFileStatus(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "status pending",
			content: "# Task\n**Status:** pending\n**Prioridade:** Alta\n",
			want:    "pending",
		},
		{
			name:    "status done em portugues",
			content: "# Task\n**Status:** Concluído (done)\n",
			want:    "done",
		},
		{
			name:    "status simples",
			content: "**Status:** in_progress\n",
			want:    "in_progress",
		},
		{
			name:    "sem status",
			content: "# Apenas titulo\nSem campo de status.\n",
			want:    "",
		},
		{
			name:    "status com whitespace apos dois-pontos retorna vazio",
			content: "# Task\n**Status:**   \n",
			want:    "",
		},
		{
			name:    "status vazio apos dois-pontos retorna vazio",
			content: "# Task\n**Status:**\n",
			want:    "",
		},
		{
			name:    "status com CRLF aciona guard clause fields vazio",
			content: "# Task\n**Status:** \r\n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReadTaskFileStatus([]byte(tt.content))
			if got != tt.want {
				t.Errorf("ReadTaskFileStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindEligible(t *testing.T) {
	tasks := []TaskEntry{
		{ID: "1.0", Title: "A", Status: "done", Dependencies: nil},
		{ID: "2.0", Title: "B", Status: "pending", Dependencies: []string{"1.0"}},
		{ID: "3.0", Title: "C", Status: "pending", Dependencies: []string{"2.0"}},
		{ID: "4.0", Title: "D", Status: "pending", Dependencies: nil},
		{ID: "5.0", Title: "E", Status: "failed", Dependencies: nil},
		{ID: "6.0", Title: "F", Status: "in_progress", Dependencies: []string{"1.0"}},
	}

	t.Run("sem skipped", func(t *testing.T) {
		eligible := FindEligible(tasks, nil)
		// 6.0 retomavel em in_progress deve vir antes das pendentes elegiveis.
		if len(eligible) != 3 {
			t.Fatalf("esperava 3 elegiveis, recebeu %d", len(eligible))
		}
		if eligible[0].ID != "6.0" {
			t.Errorf("primeiro elegivel = %q, want 6.0", eligible[0].ID)
		}
		if eligible[1].ID != "2.0" {
			t.Errorf("segundo elegivel = %q, want 2.0", eligible[1].ID)
		}
		if eligible[2].ID != "4.0" {
			t.Errorf("terceiro elegivel = %q, want 4.0", eligible[2].ID)
		}
	})

	t.Run("com skipped", func(t *testing.T) {
		skipped := map[string]bool{"2.0": true}
		eligible := FindEligible(tasks, skipped)
		if len(eligible) != 2 {
			t.Fatalf("esperava 2 elegiveis, recebeu %d", len(eligible))
		}
		if eligible[0].ID != "6.0" {
			t.Errorf("primeiro elegivel = %q, want 6.0", eligible[0].ID)
		}
		if eligible[1].ID != "4.0" {
			t.Errorf("segundo elegivel = %q, want 4.0", eligible[1].ID)
		}
	})

	t.Run("deps nao satisfeitas", func(t *testing.T) {
		blocked := []TaskEntry{
			{ID: "1.0", Title: "A", Status: "pending", Dependencies: nil},
			{ID: "2.0", Title: "B", Status: "pending", Dependencies: []string{"1.0"}},
		}
		eligible := FindEligible(blocked, nil)
		// Apenas 1.0 eh elegivel (sem deps)
		if len(eligible) != 1 {
			t.Fatalf("esperava 1 elegivel, recebeu %d", len(eligible))
		}
		if eligible[0].ID != "1.0" {
			t.Errorf("elegivel = %q, want 1.0", eligible[0].ID)
		}
	})
}

func TestFindEligible_InProgressRetomavel(t *testing.T) {
	tasks := []TaskEntry{
		{ID: "1.0", Title: "A", Status: "done", Dependencies: nil},
		{ID: "2.0", Title: "B", Status: "in_progress", Dependencies: []string{"1.0"}},
		{ID: "3.0", Title: "C", Status: "in_progress", Dependencies: []string{"9.0"}},
	}

	eligible := FindEligible(tasks, nil)
	if len(eligible) != 1 {
		t.Fatalf("esperava 1 elegivel retomavel, recebeu %d", len(eligible))
	}
	if eligible[0].ID != "2.0" {
		t.Errorf("elegivel = %q, want 2.0", eligible[0].ID)
	}
}

func TestFindEligible_PrioritizesInProgressOverPending(t *testing.T) {
	tasks := []TaskEntry{
		{ID: "1.0", Title: "Pending first", Status: "pending", Dependencies: nil},
		{ID: "2.0", Title: "Retomada", Status: "in_progress", Dependencies: nil},
		{ID: "3.0", Title: "Pending second", Status: "pending", Dependencies: nil},
	}

	eligible := FindEligible(tasks, nil)
	if len(eligible) != 3 {
		t.Fatalf("esperava 3 elegiveis, recebeu %d", len(eligible))
	}
	if eligible[0].ID != "2.0" {
		t.Fatalf("primeiro elegivel = %q, want 2.0", eligible[0].ID)
	}
}

func TestReconcileTaskStatusesUsesTaskFileStatus(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	fsys.Files["/prd/tasks.md"] = []byte("| 1.0 | Task One | in_progress | — | Nao |\n")
	fsys.Files["/prd/prd.md"] = []byte("# PRD\n")
	fsys.Files["/prd/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files["/prd/task-1.0-task.md"] = []byte("**Status:** blocked\n")

	tasks := []TaskEntry{
		{ID: "1.0", Title: "Task One", Status: "in_progress"},
	}

	got := reconcileTaskStatuses(tasks, "/prd", fsys)
	if got[0].Status != "blocked" {
		t.Fatalf("status reconciliado = %q, want blocked", got[0].Status)
	}
	if tasks[0].Status != "in_progress" {
		t.Fatalf("slice original nao deveria ser mutado, obteve %q", tasks[0].Status)
	}
}

func TestAllTerminal(t *testing.T) {
	t.Run("todas terminal", func(t *testing.T) {
		tasks := []TaskEntry{
			{Status: "done"},
			{Status: "failed"},
			{Status: "blocked"},
		}
		if !AllTerminal(tasks) {
			t.Error("esperava true")
		}
	})

	t.Run("nem todas terminal", func(t *testing.T) {
		tasks := []TaskEntry{
			{Status: "done"},
			{Status: "pending"},
		}
		if AllTerminal(tasks) {
			t.Error("esperava false")
		}
	})
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Concluido", "done"},
		{"Concluído", "done"},
		{"pendente", "pending"},
		{"bloqueado", "blocked"},
		{"em execução", "in_progress"},
		{"falhou", "failed"},
		{"done", "done"},
		{"PENDING", "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeStatus(tt.input)
			if got != tt.want {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveTaskFile(t *testing.T) {
	tests := []struct {
		name     string
		taskID   string
		files    []string
		wantFile string
		wantErr  bool
	}{
		{
			name:     "convencao prefixo simples: 1-desc.md",
			taskID:   "1.0",
			files:    []string{"tasks.md", "prd.md", "techspec.md", "1-baseline.md"},
			wantFile: "/prd/1-baseline.md",
		},
		{
			name:     "convencao ID completo: 1.0-desc.md",
			taskID:   "1.0",
			files:    []string{"tasks.md", "prd.md", "techspec.md", "1.0-baseline.md"},
			wantFile: "/prd/1.0-baseline.md",
		},
		{
			name:     "convencao task-ID: task-1.0-desc.md",
			taskID:   "1.0",
			files:    []string{"tasks.md", "prd.md", "techspec.md", "task-1.0-baseline.md"},
			wantFile: "/prd/task-1.0-baseline.md",
		},
		{
			name:     "convencao task-ID com underscore: task-1.0_desc.md",
			taskID:   "1.0",
			files:    []string{"tasks.md", "prd.md", "techspec.md", "task-1.0_baseline.md"},
			wantFile: "/prd/task-1.0_baseline.md",
		},
		{
			name:    "nenhum arquivo corresponde",
			taskID:  "1.0",
			files:   []string{"tasks.md", "prd.md", "techspec.md", "task-2.0-outro.md"},
			wantErr: true,
		},
		{
			name:    "relatorio de execucao e ignorado",
			taskID:  "1.0",
			files:   []string{"tasks.md", "prd.md", "techspec.md", "1.0_execution_report.md"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fs.NewFakeFileSystem()
			for _, f := range tt.files {
				fsys.Files["/prd/"+f] = []byte("content")
			}
			entry := TaskEntry{ID: tt.taskID}
			got, err := ResolveTaskFile("/prd", entry, fsys)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("esperava erro, recebeu %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tt.wantFile {
				t.Errorf("ResolveTaskFile() = %q, want %q", got, tt.wantFile)
			}
		})
	}
}

func TestReadTaskStatus(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		files map[string][]byte
		want  string
	}{
		{
			name: "arquivo existente com status valido",
			path: "/prd/task.md",
			files: map[string][]byte{
				"/prd/task.md": []byte("# Task\n**Status:** pending\n"),
			},
			want: "pending",
		},
		{
			name: "arquivo existente com status done em portugues",
			path: "/prd/task.md",
			files: map[string][]byte{
				"/prd/task.md": []byte("# Task\n**Status:** Concluído (done)\n"),
			},
			want: "done",
		},
		{
			name:  "arquivo inexistente retorna string vazia",
			path:  "/prd/nao_existe.md",
			files: map[string][]byte{},
			want:  "",
		},
		{
			name: "arquivo sem campo Status retorna string vazia",
			path: "/prd/task.md",
			files: map[string][]byte{
				"/prd/task.md": []byte("# Task\nSem campo de status aqui.\n"),
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fs.NewFakeFileSystem()
			for path, content := range tt.files {
				fsys.Files[path] = content
			}
			got := readTaskStatus(tt.path, fsys)
			if got != tt.want {
				t.Errorf("readTaskStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMatchesTaskPrefix(t *testing.T) {
	tests := []struct {
		filename string
		prefix   string
		fullID   string
		want     bool
	}{
		// Convencao 1: prefixo simples
		{"1-setup.md", "1", "1.0", true},
		{"1_setup.md", "1", "1.0", true},
		{"2-impl.md", "1", "1.0", false},

		// Convencao 2: ID completo
		{"1.0-setup.md", "1", "1.0", true},
		{"1.0_setup.md", "1", "1.0", true},
		{"2.0-impl.md", "1", "1.0", false},

		// Convencao 3: task-N.N
		{"task-1.0-setup.md", "1", "1.0", true},
		{"task-1.0_setup.md", "1", "1.0", true},
		{"task-2.0-impl.md", "1", "1.0", false},

		// Convencao 4: TASK-NNN (zero-padded)
		{"TASK-001-quick-reference.md", "1", "1.0", true},
		{"TASK-013-dedup.md", "13", "13.0", true},
		{"TASK-020-propagacao.md", "20", "20.0", true},
		{"TASK-001-foo.md", "2", "2.0", false},
		{"TASK-010-bar.md", "1", "1.0", false},

		// Convencao 4: case-insensitive
		{"Task-003-test.md", "3", "3.0", true},
		{"task-005-schema.md", "5", "5.0", true},

		// Sem match
		{"README.md", "1", "1.0", false},
		{"prd.md", "1", "1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := matchesTaskPrefix(tt.filename, tt.prefix, tt.fullID)
			if got != tt.want {
				t.Errorf("matchesTaskPrefix(%q, %q, %q) = %v, want %v",
					tt.filename, tt.prefix, tt.fullID, got, tt.want)
			}
		})
	}
}

// TestParseTasksFile_DuplicateIDsFromAuxTable garante que tabelas auxiliares
// (ex: "Cobertura de Requisitos") com os mesmos IDs numericos nao sobrescrevam
// as entradas da tabela principal de tasks. Sem deduplicacao, o statusMap ficava
// corrompido e tasks elegiveis (pending com deps satisfeitas) nao eram encontradas.
func TestParseTasksFile_DuplicateIDsFromAuxTable(t *testing.T) {
	// Simula tasks.md com tabela principal + tabela de cobertura (mesmos IDs).
	content := `## Tarefas

| # | Titulo | Status | Dependencias | Paralelizavel |
|---|--------|--------|-------------|---------------|
| 8.0 | Estilos Lip Gloss | done | — | Nao |
| 9.0 | Dashboard superior | done | 8.0 | Nao |
| 10.0 | Painel task ativa | done | 8.0 | Nao |
| 11.0 | Fila resumo rodape | pending | 8.0 | Nao |

## Cobertura de Requisitos

| # | Responsabilidade unica | Entregavel verificavel | RF cobertos |
|---|------------------------|------------------------|-------------|
| 8.0 | Definir estilos | ` + "`" + `presenter_styles.go` + "`" + ` com tema | RNF-10 |
| 9.0 | Renderizar dashboard | ` + "`" + `renderDashboard()` + "`" + ` com testes | RF-13 |
| 10.0 | Renderizar task ativa | ` + "`" + `renderActiveTask()` + "`" + ` completo | RF-02 |
| 11.0 | Renderizar fila | ` + "`" + `renderQueueSummary()` + "`" + ` 6 contadores | RF-08 |
`
	entries, err := ParseTasksFile([]byte(content))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("esperava 4 entries (sem duplicatas), recebeu %d", len(entries))
	}

	byID := make(map[string]TaskEntry, len(entries))
	for _, e := range entries {
		byID[e.ID] = e
	}

	if byID["8.0"].Status != "done" {
		t.Errorf("8.0 status = %q, want done (tabela auxiliar nao deve sobrescrever)", byID["8.0"].Status)
	}
	if byID["11.0"].Status != "pending" {
		t.Errorf("11.0 status = %q, want pending", byID["11.0"].Status)
	}
	if len(byID["11.0"].Dependencies) != 1 || byID["11.0"].Dependencies[0] != "8.0" {
		t.Errorf("11.0 deps = %v, want [8.0]", byID["11.0"].Dependencies)
	}

	// Apos deduplicacao, FindEligible deve encontrar 11.0 (deps de 8.0 satisfeitas).
	eligible := FindEligible(entries, nil)
	found := false
	for _, e := range eligible {
		if e.ID == "11.0" {
			found = true
		}
	}
	if !found {
		t.Errorf("11.0 deveria ser elegivel apos deduplicacao, mas nao foi encontrada em %v", eligible)
	}
}
