package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	installapp "github.com/jailtonjunior/orchestrator/internal/install/application"
	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
)

func renderOperationPreview(writer io.Writer, preview *installapp.OperationPreview) error {
	if preview == nil || preview.Plan == nil {
		return fmt.Errorf("preview must not be nil")
	}

	if _, err := fmt.Fprintf(writer, "Operação: %s\nEscopo: %s\nInventário: %s\n", preview.Operation, preview.Scope, preview.InventoryPath); err != nil {
		return err
	}

	providers := previewProviders(preview.Plan)
	if _, err := fmt.Fprintf(writer, "Providers: %s\n", strings.Join(providers, ", ")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Resumo: install=%d update=%d remove=%d skip=%d conflitos=%d\n", preview.Plan.Summary.InstallCount, preview.Plan.Summary.UpdateCount, preview.Plan.Summary.RemoveCount, preview.Plan.Summary.SkippedCount, preview.Plan.Summary.ConflictCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, "Plano:"); err != nil {
		return err
	}

	for _, change := range preview.Plan.Changes {
		if _, err := fmt.Fprintf(writer, "- [%s] %s %s -> %s\n", change.Provider(), change.Action(), change.AssetID(), change.TargetPath()); err != nil {
			return err
		}
	}

	conflicts := previewConflicts(preview.Plan)
	if len(conflicts) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(writer, "Conflitos:"); err != nil {
		return err
	}
	for _, conflict := range conflicts {
		if _, err := fmt.Fprintf(writer, "- [%s] %s em %s (%s)\n", conflict.Provider, conflict.AssetID, conflict.TargetPath, conflict.Reason); err != nil {
			return err
		}
	}

	return nil
}

func renderOperationResult(writer io.Writer, result *installapp.OperationResult) error {
	if result == nil {
		return fmt.Errorf("result must not be nil")
	}

	if _, err := fmt.Fprintf(writer, "Resultado: %s (%s)\n", result.Operation, result.Scope); err != nil {
		return err
	}
	for _, provider := range result.Providers {
		if _, err := fmt.Fprintf(writer, "- [%s] planned=%d applied=%d verify=%s inventory_saved=%t\n", provider.Provider, provider.PlannedChangeCount, provider.AppliedChangeCount, provider.Verification, provider.InventorySaved); err != nil {
			return err
		}
		for _, detail := range provider.Details {
			if _, err := fmt.Fprintf(writer, "  detalhe: %s\n", detail); err != nil {
				return err
			}
		}
	}

	return nil
}

func renderInventoryView(writer io.Writer, view *installapp.InventoryView) error {
	if view == nil {
		return fmt.Errorf("inventory view must not be nil")
	}

	if _, err := fmt.Fprintf(writer, "Escopo: %s\nInventário: %s\n", view.Scope, view.InventoryPath); err != nil {
		return err
	}
	for _, item := range view.Items {
		if _, err := fmt.Fprintf(writer, "- [%s] %s (%s) managed=%t verify=%s target=%s\n", item.Provider, item.Name, item.Kind, item.Managed, item.Verification, item.TargetPath); err != nil {
			return err
		}
	}

	return nil
}

func renderVerificationReport(writer io.Writer, report *installapp.VerificationReport) error {
	if report == nil {
		return fmt.Errorf("verification report must not be nil")
	}

	if _, err := fmt.Fprintf(writer, "Verificação: %s\nInventário: %s\n", report.Scope, report.InventoryPath); err != nil {
		return err
	}
	for _, provider := range report.Providers {
		if _, err := fmt.Fprintf(writer, "- [%s] status=%s inventory_saved=%t\n", provider.Provider, provider.Status, provider.InventorySaved); err != nil {
			return err
		}
		for _, detail := range provider.Details {
			if _, err := fmt.Fprintf(writer, "  detalhe: %s\n", detail); err != nil {
				return err
			}
		}
	}

	return nil
}

func previewProviders(plan *installapp.Plan) []string {
	if plan == nil {
		return nil
	}

	set := make(map[string]struct{})
	for _, change := range plan.Changes {
		set[string(change.Provider())] = struct{}{}
	}

	providers := make([]string, 0, len(set))
	for provider := range set {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

func previewConflicts(plan *installapp.Plan) []install.Conflict {
	if plan == nil {
		return nil
	}

	conflicts := make([]install.Conflict, 0)
	seen := make(map[string]struct{})
	for _, change := range plan.Changes {
		conflict := change.Conflict()
		if conflict == nil {
			continue
		}

		key := string(conflict.Provider) + "|" + conflict.AssetID + "|" + conflict.TargetPath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		conflicts = append(conflicts, *conflict)
	}

	sort.Slice(conflicts, func(i int, j int) bool {
		if conflicts[i].Provider == conflicts[j].Provider {
			return conflicts[i].AssetID < conflicts[j].AssetID
		}
		return conflicts[i].Provider < conflicts[j].Provider
	})
	return conflicts
}
