package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
)

// Catalog discovers installable assets from repository sources.
type Catalog interface {
	Discover(ctx context.Context, root string) ([]install.Asset, error)
}

// FileCatalog reads installable assets from the filesystem.
type FileCatalog struct{}

// New creates a filesystem-backed install catalog.
func New() *FileCatalog {
	return &FileCatalog{}
}

// Discover scans the supported V1 sources and returns normalized assets.
func (c *FileCatalog) Discover(ctx context.Context, root string) ([]install.Asset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if root == "" {
		return nil, errors.New("root must not be empty")
	}

	root = filepath.Clean(root)
	assets := make([]install.Asset, 0)

	if err := c.appendCommandAssets(ctx, &assets, root, ".claude", install.ProviderClaude, install.ProviderCopilot); err != nil {
		return nil, err
	}
	if err := c.appendSkillAssets(ctx, &assets, root, ".claude", install.ProviderClaude, install.ProviderCopilot); err != nil {
		return nil, err
	}
	if err := c.appendCommandAssets(ctx, &assets, root, ".gemini", install.ProviderGemini); err != nil {
		return nil, err
	}
	if err := c.appendSkillAssets(ctx, &assets, root, ".gemini", install.ProviderGemini); err != nil {
		return nil, err
	}
	if err := c.appendSkillAssets(ctx, &assets, root, ".codex", install.ProviderCodex); err != nil {
		return nil, err
	}
	if err := c.appendInstructionAsset(ctx, &assets, root, filepath.Join(".github", "copilot-instructions.md"), "copilot-instructions", install.ProviderCopilot); err != nil {
		return nil, err
	}
	if err := c.appendInstructionAsset(ctx, &assets, root, "AGENTS.md", "agents", install.ProviderCopilot); err != nil {
		return nil, err
	}

	sort.Slice(assets, func(i int, j int) bool {
		if assets[i].Kind() == assets[j].Kind() {
			return assets[i].ID() < assets[j].ID()
		}
		return assets[i].Kind() < assets[j].Kind()
	})

	return assets, nil
}

func (c *FileCatalog) appendCommandAssets(
	ctx context.Context,
	assets *[]install.Asset,
	root string,
	dirName string,
	providers ...install.Provider,
) error {
	commandRoot := filepath.Join(root, dirName, "commands")
	entries, err := os.ReadDir(commandRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read command catalog %q: %w", commandRoot, err)
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		assetPath := filepath.Join(commandRoot, entry.Name())
		relativePath, err := filepath.Rel(root, assetPath)
		if err != nil {
			return fmt.Errorf("resolve relative path for %q: %w", assetPath, err)
		}

		asset, err := install.NewAsset(
			buildAssetID(dirName, "command", strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))),
			strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			install.AssetKindCommand,
			assetPath,
			providers,
			install.AssetMetadata{
				EntryPath:     assetPath,
				SourceRoot:    commandRoot,
				RelativePath:  relativePath,
				ProviderHints: providers,
			},
		)
		if err != nil {
			return fmt.Errorf("build command asset %q: %w", assetPath, err)
		}

		*assets = append(*assets, asset)
	}

	return nil
}

func (c *FileCatalog) appendSkillAssets(
	ctx context.Context,
	assets *[]install.Asset,
	root string,
	dirName string,
	providers ...install.Provider,
) error {
	skillRoot := filepath.Join(root, dirName, "skills")
	entries, err := os.ReadDir(skillRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read skill catalog %q: %w", skillRoot, err)
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(skillRoot, entry.Name())
		entryPath := filepath.Join(skillDir, "SKILL.md")
		if _, err := os.Stat(entryPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("stat skill entrypoint %q: %w", entryPath, err)
		}

		relativePath, err := filepath.Rel(root, skillDir)
		if err != nil {
			return fmt.Errorf("resolve relative path for %q: %w", skillDir, err)
		}

		asset, err := install.NewAsset(
			buildAssetID(dirName, "skill", entry.Name()),
			entry.Name(),
			install.AssetKindSkill,
			skillDir,
			providers,
			install.AssetMetadata{
				EntryPath:     entryPath,
				SourceRoot:    skillRoot,
				RelativePath:  relativePath,
				ProviderHints: providers,
			},
		)
		if err != nil {
			return fmt.Errorf("build skill asset %q: %w", skillDir, err)
		}

		*assets = append(*assets, asset)
	}

	return nil
}

func (c *FileCatalog) appendInstructionAsset(
	ctx context.Context,
	assets *[]install.Asset,
	root string,
	relativePath string,
	name string,
	provider install.Provider,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	assetPath := filepath.Join(root, relativePath)
	if _, err := os.Stat(assetPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat instruction asset %q: %w", assetPath, err)
	}

	asset, err := install.NewAsset(
		buildAssetID(filepath.Dir(relativePath), "instruction", name),
		name,
		install.AssetKindInstruction,
		assetPath,
		[]install.Provider{provider},
		install.AssetMetadata{
			EntryPath:     assetPath,
			SourceRoot:    filepath.Join(root, filepath.Dir(relativePath)),
			RelativePath:  filepath.Clean(relativePath),
			ProviderHints: []install.Provider{provider},
		},
	)
	if err != nil {
		return fmt.Errorf("build instruction asset %q: %w", assetPath, err)
	}

	*assets = append(*assets, asset)
	return nil
}

func buildAssetID(namespace string, kind string, name string) string {
	namespace = strings.Trim(namespace, string(filepath.Separator))
	if namespace == "." || namespace == "" {
		namespace = "root"
	}
	namespace = strings.ReplaceAll(namespace, string(filepath.Separator), ":")
	return namespace + ":" + kind + ":" + name
}
