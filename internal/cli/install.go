package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	installapp "github.com/jailtonjunior/orchestrator/internal/install/application"
	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/spf13/cobra"
)

type installFlags struct {
	providers      []string
	assets         []string
	kinds          []string
	projectScope   bool
	globalScope    bool
	conflictPolicy string
	yes            bool
}

// NewInstallCommand creates the `orq install` command tree.
func NewInstallCommand(app *bootstrap.App) *cobra.Command {
	flags := &installFlags{}
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Instala e gerencia assets suportados pelo ORQ",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMutatingInstallCommand(cmd, app, install.OperationInstall, flags)
		},
	}

	bindMutatingInstallFlags(cmd, flags)
	cmd.AddCommand(
		newInstallUpdateCommand(app),
		newInstallRemoveCommand(app),
		newInstallListCommand(app),
		newInstallVerifyCommand(app),
	)
	return cmd
}

func newInstallUpdateCommand(app *bootstrap.App) *cobra.Command {
	flags := &installFlags{}
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Atualiza assets gerenciados pelo ORQ",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMutatingInstallCommand(cmd, app, install.OperationUpdate, flags)
		},
	}
	bindMutatingInstallFlags(cmd, flags)
	return cmd
}

func newInstallRemoveCommand(app *bootstrap.App) *cobra.Command {
	flags := &installFlags{}
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove assets gerenciados pelo ORQ",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMutatingInstallCommand(cmd, app, install.OperationRemove, flags)
		},
	}
	bindMutatingInstallFlags(cmd, flags)
	return cmd
}

func newInstallListCommand(app *bootstrap.App) *cobra.Command {
	flags := &installFlags{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lista assets instalados ou elegíveis no escopo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installServiceOrError(app); err != nil {
				return err
			}

			req, err := flags.listRequest()
			if err != nil {
				return err
			}

			view, err := app.Install.List(cmd.Context(), req)
			if err != nil {
				return translateError(err)
			}
			return renderInventoryView(cmd.OutOrStdout(), view)
		},
	}
	bindReadOnlyInstallFlags(cmd, flags)
	return cmd
}

func newInstallVerifyCommand(app *bootstrap.App) *cobra.Command {
	flags := &installFlags{}
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verifica assets gerenciados pelo ORQ",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := installServiceOrError(app); err != nil {
				return err
			}

			req, err := flags.verifyRequest()
			if err != nil {
				return err
			}

			report, err := app.Install.Verify(cmd.Context(), req)
			if err != nil {
				return translateError(err)
			}
			return renderVerificationReport(cmd.OutOrStdout(), report)
		},
	}
	bindReadOnlyInstallFlags(cmd, flags)
	return cmd
}

func bindMutatingInstallFlags(cmd *cobra.Command, flags *installFlags) {
	bindReadOnlyInstallFlags(cmd, flags)
	cmd.Flags().StringVar(&flags.conflictPolicy, "conflict", string(install.ConflictPolicyAbort), "Política de conflito: abort, skip ou overwrite")
	cmd.Flags().BoolVar(&flags.yes, "yes", false, "Executa sem pedir confirmação interativa")
}

func bindReadOnlyInstallFlags(cmd *cobra.Command, flags *installFlags) {
	cmd.Flags().StringSliceVar(&flags.providers, "provider", nil, "Provider alvo (repetível)")
	cmd.Flags().StringSliceVar(&flags.assets, "asset", nil, "Nome do asset (repetível)")
	cmd.Flags().StringSliceVar(&flags.kinds, "kind", nil, "Tipo do asset: skill, command, instruction")
	cmd.Flags().BoolVar(&flags.projectScope, "project", false, "Opera no escopo do projeto atual (default)")
	cmd.Flags().BoolVar(&flags.globalScope, "global", false, "Opera no escopo global do usuário")
	cmd.MarkFlagsMutuallyExclusive("project", "global")
}

func runMutatingInstallCommand(cmd *cobra.Command, app *bootstrap.App, operation install.Operation, flags *installFlags) error {
	if err := installServiceOrError(app); err != nil {
		return err
	}

	previewReq, err := flags.previewRequest(operation)
	if err != nil {
		return err
	}

	preview, err := app.Install.Preview(cmd.Context(), previewReq)
	if err != nil {
		return translateError(err)
	}
	if err := renderOperationPreview(cmd.OutOrStdout(), preview); err != nil {
		return err
	}

	if preview.Plan != nil && preview.Plan.Summary.ConflictCount > 0 && previewReq.ConflictPolicy == install.ConflictPolicyAbort {
		return fmt.Errorf("conflitos detectados; revise o preview acima e escolha `--conflict skip` ou `--conflict overwrite` para prosseguir")
	}

	if !flags.yes {
		confirmed, confirmErr := confirmInstallOperation(cmd)
		if confirmErr != nil {
			return confirmErr
		}
		if !confirmed {
			return fmt.Errorf("operação cancelada pelo usuário")
		}
	}

	req := installapp.OperationRequest{
		Scope:          previewReq.Scope,
		Providers:      previewReq.Providers,
		AssetNames:     previewReq.AssetNames,
		AssetKinds:     previewReq.AssetKinds,
		ConflictPolicy: previewReq.ConflictPolicy,
		Interactive:    previewReq.Interactive,
	}

	var result *installapp.OperationResult
	switch operation {
	case install.OperationInstall:
		result, err = app.Install.Install(cmd.Context(), installapp.InstallRequest{OperationRequest: req})
	case install.OperationUpdate:
		result, err = app.Install.Update(cmd.Context(), installapp.UpdateRequest{OperationRequest: req})
	case install.OperationRemove:
		result, err = app.Install.Remove(cmd.Context(), installapp.RemoveRequest{OperationRequest: req})
	default:
		return fmt.Errorf("unsupported mutating operation %q", operation)
	}
	if err != nil {
		return translateError(err)
	}

	return renderOperationResult(cmd.OutOrStdout(), result)
}

func confirmInstallOperation(cmd *cobra.Command) (bool, error) {
	if _, err := fmt.Fprint(cmd.OutOrStdout(), "Confirmar operação? [y/N]: "); err != nil {
		return false, err
	}

	reader := bufio.NewReader(cmd.InOrStdin())
	answer, err := reader.ReadString('\n')
	if err != nil && len(answer) == 0 {
		return false, err
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes" || answer == "s" || answer == "sim", nil
}

func (f *installFlags) previewRequest(operation install.Operation) (installapp.PreviewRequest, error) {
	req, err := f.operationRequest()
	if err != nil {
		return installapp.PreviewRequest{}, err
	}

	return installapp.PreviewRequest{
		Operation:        operation,
		OperationRequest: req,
	}, nil
}

func (f *installFlags) operationRequest() (installapp.OperationRequest, error) {
	scope := install.ScopeProject
	if f.globalScope {
		scope = install.ScopeGlobal
	}

	providers, err := parseProviders(f.providers)
	if err != nil {
		return installapp.OperationRequest{}, err
	}
	kinds, err := parseKinds(f.kinds)
	if err != nil {
		return installapp.OperationRequest{}, err
	}

	policy := install.ConflictPolicy(strings.ToLower(strings.TrimSpace(f.conflictPolicy)))
	if policy == "" {
		policy = install.ConflictPolicyAbort
	}
	if err := install.ValidateConflictPolicy(policy); err != nil {
		return installapp.OperationRequest{}, err
	}

	return installapp.OperationRequest{
		Scope:          scope,
		Providers:      providers,
		AssetNames:     append([]string(nil), f.assets...),
		AssetKinds:     kinds,
		ConflictPolicy: policy,
		Interactive:    !f.yes,
	}, nil
}

func (f *installFlags) listRequest() (installapp.ListRequest, error) {
	scope, providers, kinds, err := f.scopeProvidersKinds()
	if err != nil {
		return installapp.ListRequest{}, err
	}
	return installapp.ListRequest{
		Scope:      scope,
		Providers:  providers,
		AssetNames: append([]string(nil), f.assets...),
		AssetKinds: kinds,
	}, nil
}

func (f *installFlags) verifyRequest() (installapp.VerifyRequest, error) {
	scope, providers, kinds, err := f.scopeProvidersKinds()
	if err != nil {
		return installapp.VerifyRequest{}, err
	}
	return installapp.VerifyRequest{
		Scope:      scope,
		Providers:  providers,
		AssetNames: append([]string(nil), f.assets...),
		AssetKinds: kinds,
	}, nil
}

func (f *installFlags) scopeProvidersKinds() (install.Scope, []install.Provider, []install.AssetKind, error) {
	scope := install.ScopeProject
	if f.globalScope {
		scope = install.ScopeGlobal
	}

	providers, err := parseProviders(f.providers)
	if err != nil {
		return "", nil, nil, err
	}
	kinds, err := parseKinds(f.kinds)
	if err != nil {
		return "", nil, nil, err
	}
	return scope, providers, kinds, nil
}

func parseProviders(values []string) ([]install.Provider, error) {
	if len(values) == 0 {
		return nil, nil
	}

	providers := make([]install.Provider, 0, len(values))
	for _, value := range values {
		provider := install.Provider(strings.ToLower(strings.TrimSpace(value)))
		if err := install.ValidateProvider(provider); err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}
	return install.NormalizeProviders(providers)
}

func parseKinds(values []string) ([]install.AssetKind, error) {
	if len(values) == 0 {
		return nil, nil
	}

	kinds := make([]install.AssetKind, 0, len(values))
	for _, value := range values {
		kind := install.AssetKind(strings.ToLower(strings.TrimSpace(value)))
		if err := install.ValidateAssetKind(kind); err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}
