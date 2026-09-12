package aispecharness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
	"github.com/spf13/cobra"
)

const migrationBackupSuffix = ".pre-migration.bak"

func newMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Inspeciona, busca, exporta, compacta, migra e gerencia o bastao da memoria duravel de agentes",
		Long: `ai-spec-harness memory opera diretamente sobre os colaboradores do subsistema de
memoria duravel (Layer, politicas e HandoffLease), sem passar pela fachada usada
pelo runtime — a operacao humana exige granularidade que a porta estreita
deliberadamente esconde (MD-001).`,
	}

	cmd.AddCommand(newMemoryShowCmd())
	cmd.AddCommand(newMemorySearchCmd())
	cmd.AddCommand(newMemoryExportCmd())
	cmd.AddCommand(newMemoryCompactCmd())
	cmd.AddCommand(newMemoryMigrateCmd())
	cmd.AddCommand(newMemoryHandoffCmd())
	return cmd
}

type memoryCommand struct {
	locker durable.LayerLocker
}

func (h *memoryCommand) leaseLocker() durable.LayerLocker {
	if h.locker != nil {
		return h.locker
	}
	return durable.DefaultLayerLocker
}

func memoryLayerFlags(cmd *cobra.Command, projectDir, tasksDir, taskFileName *string) {
	cmd.Flags().StringVar(projectDir, "project-dir", ".", "Diretorio raiz do repositorio (camada de projeto)")
	cmd.Flags().StringVar(tasksDir, "tasks-dir", "", "Diretorio do PRD, ex: .specs/prd-x (camadas prd e task)")
	cmd.Flags().StringVar(taskFileName, "task", "", "Nome do arquivo de task (camada task)")
}

func newMemoryShowCmd() *cobra.Command {
	var projectDir, tasksDir, taskFileName string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Exibe fatos ativos por camada, com origem rastreada ate a sessao e contradicoes sinalizadas",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			layer := durable.NewLayer(fs.NewOSFileSystem())
			handler := &memoryCommand{}
			return handler.runShow(cmd.Context(), printer, layer, projectDir, tasksDir, taskFileName)
		},
	}

	memoryLayerFlags(cmd, &projectDir, &tasksDir, &taskFileName)
	return cmd
}

func (h *memoryCommand) runShow(
	ctx context.Context,
	printer *output.Printer,
	layer durable.Layer,
	projectDir, tasksDir, taskFileName string,
) error {
	found := false
	for _, scope := range h.buildScopes(projectDir, tasksDir, taskFileName) {
		facts, _, err := layer.Read(ctx, scope)
		if err != nil {
			if h.isUnresolvedScope(err) {
				continue
			}
			return fmt.Errorf("memory show: ler camada %s: %w", scope.Layer, err)
		}
		if len(facts) == 0 {
			continue
		}
		found = true
		printer.Info("== Camada: %s ==", scope.Layer)
		for _, fct := range h.sortedFacts(facts) {
			flag := ""
			if fct.State == durable.FactStateContradicted {
				flag = " [CONTRADITORIO]"
			}
			printer.Info("- %s (estado: %s%s)", fct.Identity.Key, fct.State, flag)
			printer.Info("  origem: sessao=%s cli=%s task=%s data=%s", fct.Origin.Session, fct.Origin.CLI, fct.Origin.Task, fct.Origin.Date)
			printer.Info("  conteudo: %s", fct.Content)
		}
	}
	if !found {
		printer.Info("Nenhum fato de memoria encontrado nas camadas resolvidas.")
	}
	return nil
}

func newMemorySearchCmd() *cobra.Command {
	var projectDir, tasksDir, taskFileName string
	var byEntity bool

	cmd := &cobra.Command{
		Use:   "search <termo>",
		Short: "Busca fatos por texto ou por entidade via scan deterministico das camadas, sem rede e sem chave de API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			layer := durable.NewLayer(fs.NewOSFileSystem())
			handler := &memoryCommand{}
			return handler.runSearch(cmd.Context(), printer, layer, projectDir, tasksDir, taskFileName, args[0], byEntity)
		},
	}

	memoryLayerFlags(cmd, &projectDir, &tasksDir, &taskFileName)
	cmd.Flags().BoolVar(&byEntity, "entity", false, "Busca por entidade (chave, sessao, CLI, task) em vez de texto livre")
	return cmd
}

func (h *memoryCommand) runSearch(
	ctx context.Context,
	printer *output.Printer,
	layer durable.Layer,
	projectDir, tasksDir, taskFileName, query string,
	byEntity bool,
) error {
	var allFacts []durable.Fact
	layerOf := make(map[durable.Identity]durable.TargetLayer)

	for _, scope := range h.buildScopes(projectDir, tasksDir, taskFileName) {
		facts, _, err := layer.Read(ctx, scope)
		if err != nil {
			if h.isUnresolvedScope(err) {
				continue
			}
			return fmt.Errorf("memory search: ler camada %s: %w", scope.Layer, err)
		}
		for _, fct := range facts {
			layerOf[fct.Identity] = scope.Layer
		}
		allFacts = append(allFacts, facts...)
	}

	var matches []durable.Fact
	if byEntity {
		matches = durable.DefaultSearchPolicy.Entity(allFacts, query)
	} else {
		matches = durable.DefaultSearchPolicy.Text(allFacts, query)
	}

	if len(matches) == 0 {
		printer.Info("Nenhum resultado para %q.", query)
		return nil
	}
	for _, fct := range matches {
		printer.Info("[%s] %s: %s", layerOf[fct.Identity], fct.Identity.Key, fct.Content)
	}
	return nil
}

func newMemoryExportCmd() *cobra.Command {
	var projectDir, tasksDir, taskFileName, out string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Produz um artefato Markdown autocontido e portatil com a memoria ativa das camadas resolvidas",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			fsys := fs.NewOSFileSystem()
			layer := durable.NewLayer(fsys)
			handler := &memoryCommand{}
			return handler.runExport(cmd.Context(), printer, fsys, layer, projectDir, tasksDir, taskFileName, out)
		},
	}

	memoryLayerFlags(cmd, &projectDir, &tasksDir, &taskFileName)
	cmd.Flags().StringVar(&out, "out", "memory-export.md", "Caminho do artefato exportado")
	return cmd
}

func (h *memoryCommand) runExport(
	ctx context.Context,
	printer *output.Printer,
	fsys fs.FileSystem,
	layer durable.Layer,
	projectDir, tasksDir, taskFileName, out string,
) error {
	var b strings.Builder
	b.WriteString("# Exportacao de Memoria Duravel\n\n")
	fmt.Fprintf(&b, "Gerado em: %s\n\n", time.Now().UTC().Format(time.RFC3339))

	hasContent := false
	for _, scope := range h.buildScopes(projectDir, tasksDir, taskFileName) {
		facts, _, err := layer.Read(ctx, scope)
		if err != nil {
			if h.isUnresolvedScope(err) {
				continue
			}
			return fmt.Errorf("memory export: ler camada %s: %w", scope.Layer, err)
		}
		if len(facts) == 0 {
			continue
		}
		hasContent = true
		fmt.Fprintf(&b, "## Camada: %s\n\n", scope.Layer)
		for _, fct := range h.sortedFacts(facts) {
			fmt.Fprintf(&b, "### %s\n\n", fct.Identity.Key)
			fmt.Fprintf(&b, "- Estado: %s\n", fct.State)
			fmt.Fprintf(&b, "- Origem: sessao=%s cli=%s task=%s data=%s\n\n", fct.Origin.Session, fct.Origin.CLI, fct.Origin.Task, fct.Origin.Date)
			fmt.Fprintf(&b, "%s\n\n", fct.Content)
		}
	}

	if !hasContent {
		b.WriteString("Nenhum fato de memoria encontrado nas camadas resolvidas.\n")
	}

	if err := fsys.WriteFileAtomic(out, []byte(b.String())); err != nil {
		return fmt.Errorf("memory export: gravar %s: %w", out, err)
	}
	printer.Info("Exportacao gravada em %s.", out)
	return nil
}

func newMemoryCompactCmd() *cobra.Command {
	var projectDir, tasksDir, taskFileName string
	var lineLimit, byteLimit int

	cmd := &cobra.Command{
		Use:   "compact",
		Short: "Executa a compactacao deterministica sob demanda nas camadas resolvidas, preservando o bloco humano",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			layer := durable.NewLayer(fs.NewOSFileSystem())
			handler := &memoryCommand{}
			cfg := durable.CompactionConfig{LineLimit: lineLimit, ByteLimit: byteLimit}
			return handler.runCompact(cmd.Context(), printer, layer, projectDir, tasksDir, taskFileName, cfg)
		},
	}

	memoryLayerFlags(cmd, &projectDir, &tasksDir, &taskFileName)
	cmd.Flags().IntVar(&lineLimit, "line-limit", durable.DefaultCompactionLineLimit, "Limite de linhas por pagina")
	cmd.Flags().IntVar(&byteLimit, "byte-limit", durable.DefaultCompactionByteLimit, "Limite de bytes por pagina")
	return cmd
}

func (h *memoryCommand) runCompact(
	ctx context.Context,
	printer *output.Printer,
	layer durable.Layer,
	projectDir, tasksDir, taskFileName string,
	cfg durable.CompactionConfig,
) error {
	unreachable := false
	for _, scope := range h.buildScopes(projectDir, tasksDir, taskFileName) {
		facts, human, err := layer.Read(ctx, scope)
		if err != nil {
			if h.isUnresolvedScope(err) {
				continue
			}
			return fmt.Errorf("memory compact: ler camada %s: %w", scope.Layer, err)
		}
		if len(facts) == 0 {
			continue
		}

		result, compErr := durable.DefaultCompactionPolicy.Compact(facts, human, cfg, taskFileName)
		if compErr != nil {
			if errors.Is(compErr, durable.ErrLimitUnreachable) {
				printer.Warn("camada %s: limite inalcancavel por conteudo humano (RF-37) — reportado, nao reescrito", scope.Layer)
				unreachable = true
				continue
			}
			return fmt.Errorf("memory compact: camada %s: %w", scope.Layer, compErr)
		}

		if len(result.ToArchive) == 0 {
			printer.Info("camada %s: ja dentro dos limites, nada arquivado", scope.Layer)
			continue
		}
		if err := layer.Archive(ctx, scope, result.ToArchive); err != nil {
			return fmt.Errorf("memory compact: arquivar camada %s: %w", scope.Layer, err)
		}
		printer.Info("camada %s: %d fato(s) arquivado(s)", scope.Layer, len(result.ToArchive))
	}

	if unreachable {
		return newExitError(1)
	}
	return nil
}

func newMemoryMigrateCmd() *cobra.Command {
	var tasksDir string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Converte o MEMORY.md legado da camada PRD para o formato de pagina duravel, com backup verificavel",
		Long: `Grava um backup verificavel do arquivo legado antes de qualquer conversao,
converte para o formato de pagina duravel preservando 100% do conteudo existente
como bloco de autoria humana, e recusa reaplicar a migracao sobre um arquivo
ja convertido (RF-33, MD-004 passo 7).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			handler := &memoryCommand{}
			return handler.runMigrate(printer, fs.NewOSFileSystem(), tasksDir)
		},
	}

	cmd.Flags().StringVar(&tasksDir, "tasks-dir", "", "Diretorio do PRD contendo memory/MEMORY.md a migrar")
	_ = cmd.MarkFlagRequired("tasks-dir")
	return cmd
}

func (h *memoryCommand) runMigrate(printer *output.Printer, fsys fs.FileSystem, tasksDir string) error {
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}
	page := durable.NewMarkdownPage()

	activePath, err := scope.ActivePath()
	if err != nil {
		return fmt.Errorf("memory migrate: %w", err)
	}

	if !fsys.Exists(activePath) {
		printer.Info("Nenhum arquivo de memoria legado encontrado em %s. Nada a migrar.", activePath)
		return nil
	}

	original, err := fsys.ReadFile(activePath)
	if err != nil {
		return fmt.Errorf("memory migrate: ler %s: %w", activePath, err)
	}

	_, human, err := page.Parse(original)
	if err != nil {
		return fmt.Errorf("memory migrate: %w", err)
	}
	if human.Header.FormatVersion != 0 {
		return fmt.Errorf("memory migrate: %s: %w", activePath, durable.ErrMigrationAlreadyApplied)
	}

	backupPath := activePath + migrationBackupSuffix
	if err := fs.RefuseExternalSymlink(fsys, tasksDir, backupPath, false); err != nil {
		return fmt.Errorf("memory migrate: %w", err)
	}
	if err := fsys.WriteFileAtomic(backupPath, original); err != nil {
		return fmt.Errorf("memory migrate: gravar backup %s: %w", backupPath, err)
	}
	backupReadBack, err := fsys.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("memory migrate: verificar backup %s: %w", backupPath, err)
	}
	if string(backupReadBack) != string(original) {
		return fmt.Errorf("memory migrate: backup %s nao confere byte a byte com o original", backupPath)
	}

	newHeader := durable.PageHeader{
		Identity:      tasksDir,
		Layer:         durable.TargetLayerPRD,
		Date:          time.Now().UTC().Format(time.RFC3339),
		FormatVersion: durable.FormatVersionCurrent,
	}
	migrated, err := page.Serialize(nil, durable.HumanBlock{Header: newHeader, Content: human.Content})
	if err != nil {
		return fmt.Errorf("memory migrate: serializar pagina migrada: %w", err)
	}

	if err := fs.RefuseExternalSymlink(fsys, tasksDir, activePath, false); err != nil {
		return fmt.Errorf("memory migrate: %w", err)
	}
	if err := fsys.WriteFileAtomic(activePath, migrated); err != nil {
		return fmt.Errorf("memory migrate: gravar %s: %w", activePath, err)
	}

	_, migratedHuman, err := page.Parse(migrated)
	if err != nil {
		return fmt.Errorf("memory migrate: verificar migracao de %s: %w", activePath, err)
	}
	if migratedHuman.Content != human.Content {
		return fmt.Errorf("memory migrate: conteudo nao preservado apos migracao de %s", activePath)
	}

	printer.Info("Migracao concluida: %s (backup verificado em %s)", activePath, backupPath)
	return nil
}

func newMemoryHandoffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Consulta, reivindica e libera o bastao de continuidade da memoria duravel de agentes",
	}

	cmd.AddCommand(newMemoryHandoffStatusCmd())
	cmd.AddCommand(newMemoryHandoffClaimCmd())
	cmd.AddCommand(newMemoryHandoffReleaseCmd())
	return cmd
}

func newMemoryHandoffStatusCmd() *cobra.Command {
	var tasksDir string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Exibe o estado atual do bastao de continuidade do PRD informado",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			handler := &memoryCommand{}
			return handler.runHandoffStatus(printer, fs.NewOSFileSystem(), tasksDir)
		},
	}

	cmd.Flags().StringVar(&tasksDir, "tasks-dir", "", "Diretorio do PRD ao qual o bastao pertence")
	_ = cmd.MarkFlagRequired("tasks-dir")
	return cmd
}

func (h *memoryCommand) runHandoffStatus(printer *output.Printer, fsys fs.FileSystem, tasksDir string) error {
	scope := h.handoffScope(tasksDir)
	store := durable.NewHandoffLeaseStore(fsys, h.leaseLocker(), tasksDir)

	lease, err := store.Load(scope)
	if err != nil {
		return fmt.Errorf("memory handoff status: %w", err)
	}
	if lease == nil {
		printer.Info("Nenhum bastao ativo para %s.", tasksDir)
		return nil
	}

	remaining := time.Until(lease.Deadline).Round(time.Second)
	printer.Info("Bastao detido por %s ate %s (restante: %s)", lease.Owner, lease.Deadline.Format(time.RFC3339), remaining)
	return nil
}

func newMemoryHandoffClaimCmd() *cobra.Command {
	var tasksDir, owner string
	var ttl time.Duration

	cmd := &cobra.Command{
		Use:   "claim",
		Short: "Reivindica o bastao de continuidade, recusando reivindicacao concorrente de dono vivo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			handler := &memoryCommand{}
			return handler.runHandoffClaim(printer, fs.NewOSFileSystem(), tasksDir, owner, ttl)
		},
	}

	cmd.Flags().StringVar(&tasksDir, "tasks-dir", "", "Diretorio do PRD ao qual o bastao pertence")
	cmd.Flags().StringVar(&owner, "owner", "", "Identificador do reivindicante (default: pid:<pid> do processo atual)")
	cmd.Flags().DurationVar(&ttl, "ttl", durable.DefaultLeaseTTL, "Prazo do lease (RF-23)")
	_ = cmd.MarkFlagRequired("tasks-dir")
	return cmd
}

func (h *memoryCommand) runHandoffClaim(printer *output.Printer, fsys fs.FileSystem, tasksDir, owner string, ttl time.Duration) error {
	scope := h.handoffScope(tasksDir)
	store := durable.NewHandoffLeaseStore(fsys, h.leaseLocker(), tasksDir)
	claimant := h.resolveOwner(owner)

	record, claimErr := store.Claim(scope, durable.DefaultLeasePolicy, claimant, durable.CurrentProcessRef(), ttl, time.Now())
	if claimErr != nil {
		if errors.Is(claimErr, durable.ErrBatonAlreadyClaimed) {
			printer.Error("bastao recusado: detido por %s ate %s", record.PreviousOwner, record.Deadline.Format(time.RFC3339))
			return newExitError(1)
		}
		return fmt.Errorf("memory handoff claim: %w", claimErr)
	}

	if record.Outcome == durable.LeaseOutcomeTransferred {
		printer.Info("bastao tomado de %s (motivo: %s); novo dono: %s ate %s",
			record.PreviousOwner, record.TransferReason, record.Claimant, record.Deadline.Format(time.RFC3339))
		return nil
	}
	printer.Info("bastao reivindicado por %s ate %s", record.Claimant, record.Deadline.Format(time.RFC3339))
	return nil
}

func newMemoryHandoffReleaseCmd() *cobra.Command {
	var tasksDir, owner string

	cmd := &cobra.Command{
		Use:   "release",
		Short: "Libera o bastao de continuidade detido pelo dono informado",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			handler := &memoryCommand{}
			return handler.runHandoffRelease(printer, fs.NewOSFileSystem(), tasksDir, owner)
		},
	}

	cmd.Flags().StringVar(&tasksDir, "tasks-dir", "", "Diretorio do PRD ao qual o bastao pertence")
	cmd.Flags().StringVar(&owner, "owner", "", "Identificador do dono a liberar (default: pid:<pid> do processo atual)")
	_ = cmd.MarkFlagRequired("tasks-dir")
	return cmd
}

func (h *memoryCommand) runHandoffRelease(printer *output.Printer, fsys fs.FileSystem, tasksDir, owner string) error {
	scope := h.handoffScope(tasksDir)
	store := durable.NewHandoffLeaseStore(fsys, h.leaseLocker(), tasksDir)
	claimant := h.resolveOwner(owner)

	outcome, previousOwner, err := store.Release(scope, claimant)
	if err != nil {
		return fmt.Errorf("memory handoff release: %w", err)
	}

	switch outcome {
	case durable.ReleaseOutcomeNotFound:
		printer.Info("Nenhum bastao ativo para %s.", tasksDir)
		return nil
	case durable.ReleaseOutcomeOwnerMismatch:
		printer.Error("bastao detido por %s; apenas o dono pode libera-lo (informe --owner %s para confirmar a identidade correta)", previousOwner, previousOwner)
		return newExitError(1)
	default:
		printer.Info("bastao liberado por %s.", claimant)
		return nil
	}
}

func (h *memoryCommand) handoffScope(tasksDir string) durable.Scope {
	return durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}
}

func (h *memoryCommand) resolveOwner(owner string) durable.LeaseOwner {
	if owner != "" {
		return durable.LeaseOwner(owner)
	}
	return durable.LeaseOwner(fmt.Sprintf("pid:%d", os.Getpid()))
}

func (h *memoryCommand) buildScopes(projectDir, tasksDir, taskFileName string) []durable.Scope {
	scopes := []durable.Scope{
		{Layer: durable.TargetLayerProject, ProjectDir: projectDir},
	}
	if tasksDir == "" {
		return scopes
	}
	scopes = append(scopes, durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir})
	if taskFileName != "" {
		scopes = append(scopes, durable.Scope{Layer: durable.TargetLayerTask, TasksDir: tasksDir, TaskFileName: taskFileName})
	}
	return scopes
}

func (h *memoryCommand) isUnresolvedScope(err error) bool {
	return errors.Is(err, durable.ErrProjectDirMissing) ||
		errors.Is(err, durable.ErrTasksDirMissing) ||
		errors.Is(err, durable.ErrTaskFileNameMissing) ||
		errors.Is(err, durable.ErrLayerUndefined)
}

func (h *memoryCommand) sortedFacts(facts []durable.Fact) []durable.Fact {
	sorted := make([]durable.Fact, len(facts))
	copy(sorted, facts)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Identity.Key < sorted[j].Identity.Key
	})
	return sorted
}
