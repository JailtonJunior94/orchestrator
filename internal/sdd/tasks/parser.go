// Package tasks valida os vínculos estruturais entre PRD e tasks.md.
package tasks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	_requirementIDPattern = regexp.MustCompile(`^(?:RF|NFR)-[0-9]+$`)
	_taskIDPattern        = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
	_crossPRDPattern      = regexp.MustCompile(`^prd-[a-z0-9][a-z0-9-]*:[0-9]+\.[0-9]+$`)
)

// Task representa uma task e seus vínculos declarados no artefato humano.
type Task struct {
	ID           string
	Status       string
	Dependencies []string
	Ownership    []string
	Parallel     bool
}

// Document é a projeção estrutural validada de um PRD e seu tasks.md.
type Document struct {
	Requirements []string
	Tasks        []Task
	Coverage     map[string][]string
}

// Parser não possui estado e pode ser reutilizado concorrentemente.
type Parser struct{}

// NewParser cria um parser estrutural de artefatos SDD.
func NewParser() *Parser {
	return &Parser{}
}

// Parse extrai e valida requisitos, cobertura, dependências, ciclos e ownership.
func (p *Parser) Parse(prdContent, tasksContent []byte) (Document, error) {
	return p.ParseAt("", prdContent, tasksContent)
}

// ParseAt resolve também dependências cross-PRD a partir do diretório atual.
func (p *Parser) ParseAt(prdDir string, prdContent, tasksContent []byte) (Document, error) {
	requirements, err := p.parseRequirements(string(prdContent))
	if err != nil {
		return Document{}, err
	}
	tasks, err := p.parseTasks(string(tasksContent))
	if err != nil {
		return Document{}, err
	}
	coverage, err := p.parseCoverage(string(tasksContent))
	if err != nil {
		return Document{}, err
	}
	document := Document{Requirements: requirements, Tasks: tasks, Coverage: coverage}
	if err := p.validate(prdDir, document); err != nil {
		return Document{}, err
	}
	return document, nil
}

func (p *Parser) parseRequirements(content string) ([]string, error) {
	functional := p.section(content, "Requisitos Funcionais")
	nonFunctional := p.section(content, "Requisitos Não Funcionais")
	if len(functional) == 0 {
		return nil, fmt.Errorf("vinculos estruturais: seção Requisitos Funcionais ausente no PRD")
	}
	requirements := make([]string, 0)
	seen := make(map[string]struct{})
	for _, line := range append(functional, nonFunctional...) {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		id, _, found := strings.Cut(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), ":")
		id = strings.ToUpper(strings.TrimSpace(id))
		if !found || !_requirementIDPattern.MatchString(id) {
			return nil, fmt.Errorf("vinculos estruturais: requisito invalido %q", trimmed)
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("vinculos estruturais: requisito duplicado %s", id)
		}
		seen[id] = struct{}{}
		requirements = append(requirements, id)
	}
	if len(requirements) == 0 {
		return nil, fmt.Errorf("vinculos estruturais: nenhum requisito encontrado no PRD")
	}
	return requirements, nil
}

func (p *Parser) parseTasks(content string) ([]Task, error) {
	rows := p.table(content, "Tarefas")
	if len(rows) < 3 {
		rows = p.firstTable(content)
	}
	if len(rows) < 3 {
		return nil, fmt.Errorf("vinculos estruturais: tabela Tarefas ausente ou vazia")
	}
	headers := p.headers(rows[0])
	idIndex, ok := headers["#"]
	if !ok {
		return nil, fmt.Errorf("vinculos estruturais: coluna # ausente na tabela Tarefas")
	}
	depsIndex := p.column(headers, "dependências", "dependencias", "deps")
	statusIndex := p.column(headers, "status", "estado")
	parallelIndex := p.column(headers, "paralelizável", "paralelizavel", "parallel_group")
	ownershipIndex := p.column(headers, "ownership", "arquivos", "paths")

	tasks := make([]Task, 0, len(rows)-2)
	seen := make(map[string]struct{})
	for _, row := range rows[2:] {
		columns := p.columns(row)
		if idIndex >= len(columns) {
			return nil, fmt.Errorf("vinculos estruturais: linha de task sem ID")
		}
		id := strings.TrimSpace(columns[idIndex])
		if !_taskIDPattern.MatchString(id) {
			return nil, fmt.Errorf("vinculos estruturais: ID de task invalido %q", id)
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("vinculos estruturais: ID de task duplicado %s", id)
		}
		seen[id] = struct{}{}
		task := Task{ID: id}
		if statusIndex >= 0 && statusIndex < len(columns) {
			task.Status = strings.ToLower(strings.TrimSpace(columns[statusIndex]))
		}
		if depsIndex >= 0 && depsIndex < len(columns) {
			task.Dependencies = p.references(columns[depsIndex])
		}
		if ownershipIndex >= 0 && ownershipIndex < len(columns) {
			task.Ownership = p.references(columns[ownershipIndex])
		}
		if parallelIndex >= 0 && parallelIndex < len(columns) {
			task.Parallel = p.parallel(columns[parallelIndex])
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (p *Parser) firstTable(content string) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0)
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") {
			started = true
			result = append(result, trimmed)
			continue
		}
		if started {
			break
		}
	}
	return result
}

func (p *Parser) parseCoverage(content string) (map[string][]string, error) {
	rows := p.table(content, "Cobertura de Requisitos")
	if len(rows) < 3 {
		return nil, fmt.Errorf("vinculos estruturais: tabela Cobertura de Requisitos ausente ou vazia")
	}
	headers := p.headers(rows[0])
	taskIndex, ok := headers["tarefa"]
	if !ok {
		return nil, fmt.Errorf("vinculos estruturais: coluna Tarefa ausente na cobertura")
	}
	requirementsIndex := p.column(headers, "requisitos cobertos", "requisitos")
	if requirementsIndex < 0 {
		return nil, fmt.Errorf("vinculos estruturais: coluna Requisitos cobertos ausente na cobertura")
	}
	coverage := make(map[string][]string, len(rows)-2)
	for _, row := range rows[2:] {
		columns := p.columns(row)
		if taskIndex >= len(columns) || requirementsIndex >= len(columns) {
			return nil, fmt.Errorf("vinculos estruturais: linha de cobertura incompleta")
		}
		id := strings.TrimSpace(columns[taskIndex])
		if !_taskIDPattern.MatchString(id) {
			return nil, fmt.Errorf("vinculos estruturais: task de cobertura invalida %q", id)
		}
		if _, duplicate := coverage[id]; duplicate {
			return nil, fmt.Errorf("vinculos estruturais: cobertura duplicada para task %s", id)
		}
		coverage[id] = p.references(columns[requirementsIndex])
	}
	return coverage, nil
}

func (p *Parser) validate(prdDir string, document Document) error {
	tasksByID := make(map[string]Task, len(document.Tasks))
	for _, task := range document.Tasks {
		tasksByID[task.ID] = task
	}
	requirements := make(map[string]struct{}, len(document.Requirements))
	for _, requirement := range document.Requirements {
		requirements[requirement] = struct{}{}
	}
	covered := make(map[string]struct{}, len(requirements))
	for taskID, linkedRequirements := range document.Coverage {
		if _, exists := tasksByID[taskID]; !exists {
			return fmt.Errorf("vinculos estruturais: cobertura referencia task inexistente %s", taskID)
		}
		if len(linkedRequirements) == 0 {
			return fmt.Errorf("vinculos estruturais: task %s sem requisitos cobertos", taskID)
		}
		for _, requirement := range linkedRequirements {
			requirement = strings.ToUpper(requirement)
			if _, exists := requirements[requirement]; !exists {
				return fmt.Errorf("vinculos estruturais: requisito %s nao existe no PRD", requirement)
			}
			covered[requirement] = struct{}{}
		}
	}
	for _, requirement := range document.Requirements {
		if _, exists := covered[requirement]; !exists {
			return fmt.Errorf("vinculos estruturais: requisito %s sem cobertura", requirement)
		}
	}
	if err := p.validateDependencies(prdDir, tasksByID); err != nil {
		return err
	}
	return p.validateOwnership(document.Tasks)
}

func (p *Parser) validateDependencies(prdDir string, tasks map[string]Task) error {
	state := make(map[string]int, len(tasks))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("vinculos estruturais: ciclo de dependencias em %s", id)
		case 2:
			return nil
		}
		state[id] = 1
		for _, dependency := range tasks[id].Dependencies {
			if _crossPRDPattern.MatchString(dependency) {
				if prdDir == "" {
					return fmt.Errorf("vinculos estruturais: dependencia cross-PRD %s requer diretorio para resolucao", dependency)
				}
				root, err := filepath.Abs(prdDir)
				if err != nil {
					return err
				}
				if err := p.validateCrossDependency(root, dependency, map[string]bool{root + ":" + id: true}); err != nil {
					return err
				}
				continue
			}
			if !_taskIDPattern.MatchString(dependency) {
				return fmt.Errorf("vinculos estruturais: dependencia invalida %q da task %s", dependency, id)
			}
			if _, exists := tasks[dependency]; !exists {
				return fmt.Errorf("vinculos estruturais: dependencia %s da task %s nao existe", dependency, id)
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	ids := make([]string, 0, len(tasks))
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

type externalState struct {
	SchemaVersion int `json:"schema_version"`
	Artifacts     map[string]struct {
		SHA256   string `json:"sha256"`
		Approved bool   `json:"approved"`
	} `json:"artifacts"`
	Tasks map[string]struct {
		Status       string   `json:"status"`
		Dependencies []string `json:"dependencies"`
	} `json:"tasks"`
}

func (p *Parser) validateCrossDependency(currentDir, reference string, stack map[string]bool) error {
	slug, taskID, _ := strings.Cut(reference, ":")
	targetDir := filepath.Join(filepath.Dir(currentDir), slug)
	return p.validateExternalTask(targetDir, taskID, stack)
}

func (p *Parser) validateExternalTask(prdDir, taskID string, stack map[string]bool) error {
	absolute, err := filepath.Abs(prdDir)
	if err != nil {
		return fmt.Errorf("vinculos estruturais: resolver PRD externo: %w", err)
	}
	key := absolute + ":" + taskID
	if stack[key] {
		return fmt.Errorf("vinculos estruturais: ciclo cross-PRD em %s", key)
	}
	stack[key] = true
	defer delete(stack, key)
	content, err := os.ReadFile(filepath.Join(absolute, "sdd-state.json"))
	if err != nil {
		return fmt.Errorf("vinculos estruturais: estado do PRD externo %s ausente: %w", absolute, err)
	}
	var state externalState
	if err := json.Unmarshal(content, &state); err != nil || state.SchemaVersion != 2 {
		return fmt.Errorf("vinculos estruturais: estado do PRD externo %s invalido", absolute)
	}
	artifact, exists := state.Artifacts["tasks"]
	if !exists || !artifact.Approved || len(artifact.SHA256) != sha256.Size*2 {
		return fmt.Errorf("vinculos estruturais: tasks externo nao aprovado em %s", absolute)
	}
	tasksContent, err := os.ReadFile(filepath.Join(absolute, "tasks.md"))
	if err != nil {
		return fmt.Errorf("vinculos estruturais: ler tasks externo: %w", err)
	}
	digest := sha256.Sum256(tasksContent)
	if hex.EncodeToString(digest[:]) != artifact.SHA256 {
		return fmt.Errorf("vinculos estruturais: tasks externo stale em %s", absolute)
	}
	task, exists := state.Tasks[taskID]
	if !exists || task.Status != "done" {
		return fmt.Errorf("vinculos estruturais: task externa %s nao esta done", key)
	}
	for _, dependency := range task.Dependencies {
		if _crossPRDPattern.MatchString(dependency) {
			if err := p.validateCrossDependency(absolute, dependency, stack); err != nil {
				return err
			}
			continue
		}
		if local, ok := state.Tasks[dependency]; !ok || local.Status != "done" {
			return fmt.Errorf("vinculos estruturais: dependencia %s da task externa %s nao esta done", dependency, key)
		}
		if err := p.validateExternalTask(absolute, dependency, stack); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) validateOwnership(tasks []Task) error {
	owners := make(map[string]string)
	for _, task := range tasks {
		if task.Parallel && len(task.Ownership) == 0 {
			return fmt.Errorf("vinculos estruturais: task paralela %s sem ownership", task.ID)
		}
		for _, ownedPath := range task.Ownership {
			clean := filepath.Clean(ownedPath)
			if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				return fmt.Errorf("vinculos estruturais: ownership invalido %q da task %s", ownedPath, task.ID)
			}
			for path, owner := range owners {
				if owner != task.ID && p.overlaps(path, clean) {
					return fmt.Errorf("vinculos estruturais: ownership sobreposto em %s (%s e %s)", clean, owner, task.ID)
				}
			}
			owners[clean] = task.ID
		}
	}
	return nil
}

func (p *Parser) overlaps(left, right string) bool {
	return left == right || strings.HasPrefix(left, right+string(filepath.Separator)) || strings.HasPrefix(right, left+string(filepath.Separator))
}

func (p *Parser) section(content, title string) []string {
	lines := strings.Split(content, "\n")
	inSection := false
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSection {
				break
			}
			inSection = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), title)
			continue
		}
		if inSection {
			result = append(result, line)
		}
	}
	return result
}

func (p *Parser) table(content, section string) []string {
	lines := p.section(content, section)
	result := make([]string, 0)
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			result = append(result, strings.TrimSpace(line))
		}
	}
	return result
}

func (p *Parser) headers(row string) map[string]int {
	columns := p.columns(row)
	headers := make(map[string]int, len(columns))
	for index, column := range columns {
		headers[strings.ToLower(strings.TrimSpace(column))] = index
	}
	return headers
}

func (p *Parser) columns(row string) []string {
	parts := strings.Split(strings.TrimSpace(row), "|")
	if len(parts) < 3 {
		return nil
	}
	return parts[1 : len(parts)-1]
}

func (p *Parser) column(headers map[string]int, names ...string) int {
	for _, name := range names {
		if index, exists := headers[name]; exists {
			return index
		}
	}
	return -1
}

func (p *Parser) references(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "—" || raw == "-" || strings.EqualFold(raw, "nenhuma") {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func (p *Parser) parallel(raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	return value == "sim" || strings.HasPrefix(value, "sim,") || strings.HasPrefix(value, "com ") || strings.HasPrefix(value, "grupo ")
}
