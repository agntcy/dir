package manager

import (
	"context"
	"fmt"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/cmd/build/config"
	"github.com/agntcy/dir/cli/cmd/build/extensions/crewai"
	"github.com/agntcy/dir/cli/cmd/build/extensions/framework"
	"github.com/agntcy/dir/cli/cmd/build/extensions/language"
	"github.com/agntcy/dir/cli/cmd/build/extensions/llmanalyzer"
	"github.com/agntcy/dir/cli/cmd/build/extensions/runtime"
	"github.com/agntcy/dir/cli/types"
	clitypes "github.com/agntcy/dir/cli/types"
)

type ExtensionManager struct {
	extensions       []clitypes.ExtensionBuilder
	customExtensions []int
	cfg              *config.Config
}

func NewExtensionManager(cfg *config.Config) *ExtensionManager {
	return &ExtensionManager{
		extensions:       make([]clitypes.ExtensionBuilder, 0),
		customExtensions: make([]int, 0),
		cfg:              cfg,
	}
}

func (em *ExtensionManager) RegisterExtensions() error {
	for i, ext := range em.cfg.Extensions {
		switch ext.Name {
		case framework.ExtensionName:
			frameworkCfg := &framework.Config{}
			err := frameworkCfg.From(ext.Specs)
			if err != nil {
				return fmt.Errorf("failed to register framework extension: %w", err)
			}
			em.extensions = append(em.extensions, framework.New(frameworkCfg))

		case language.ExtensionName:
			languageCfg := &language.Config{}
			err := languageCfg.From(ext.Specs)
			if err != nil {
				return fmt.Errorf("failed to register language extension: %w", err)
			}
			em.extensions = append(em.extensions, language.New(languageCfg))

		default:
			em.customExtensions = append(em.customExtensions, i)

		}
	}

	if em.cfg.Builder.CrewAI {
		em.extensions = append(em.extensions, crewai.New(em.cfg.Builder.Source, em.cfg.Builder.SourceIgnore))
	}

	if em.cfg.Builder.LLMAnalyzer {
		LLMAnalyzer, err := llmanalyzer.New(em.cfg.Builder.Source, em.cfg.Builder.SourceIgnore)
		if err != nil {
			return fmt.Errorf("failed to register LLMAnalyzer extension: %w", err)
		}
		em.extensions = append(em.extensions, LLMAnalyzer)
	}

	if em.cfg.Builder.Runtime {
		em.extensions = append(em.extensions, runtime.New(em.cfg.Builder.Source))
	}

	return nil
}

func (em *ExtensionManager) Run(ctx context.Context) ([]*apicore.Extension, error) {
	var builtExtensions []*apicore.Extension

	for _, ext := range em.extensions {
		extension, err := ext.Build(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to build extension: %w", err)
		}

		apiExt, err := extension.ToAPIExtension()
		if err != nil {
			return nil, fmt.Errorf("failed to convert extension to API extension: %w", err)
		}

		builtExtensions = append(builtExtensions, &apiExt)
	}

	for _, i := range em.customExtensions {
		extension := types.AgentExtension{
			Name:    em.cfg.Extensions[i].Name,
			Version: em.cfg.Extensions[i].Version,
			Specs:   em.cfg.Extensions[i].Specs,
		}

		apiExt, err := extension.ToAPIExtension()
		if err != nil {
			return nil, fmt.Errorf("failed to convert extension to API extension: %w", err)
		}

		builtExtensions = append(builtExtensions, &apiExt)
	}

	return builtExtensions, nil
}
