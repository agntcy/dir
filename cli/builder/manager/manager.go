package manager

import (
	"context"
	"fmt"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/extensions/crewai"
	"github.com/agntcy/dir/cli/builder/extensions/framework"
	"github.com/agntcy/dir/cli/builder/extensions/language"
	"github.com/agntcy/dir/cli/builder/extensions/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/extensions/runtime"
	"github.com/agntcy/dir/cli/builder/extensions/skills"
	"github.com/agntcy/dir/cli/cmd/build/config"
	clitypes "github.com/agntcy/dir/cli/types"
)

type ExtensionManager struct {
	extensions map[string]interface{}
}

func NewExtensionManager() *ExtensionManager {
	return &ExtensionManager{extensions: make(map[string]interface{})}
}

func (em *ExtensionManager) register(name string, config interface{}) {
	em.extensions[name] = config
}

func (em *ExtensionManager) RegisterExtensions(cfg *config.Config) {
	// Register extensions
	em.register(framework.ExtensionName, cfg.Framework)
	em.register(language.ExtensionName, cfg.Language)
	em.register(skills.ExtensionName, cfg.Skills)

	em.register(runtime.ExtensionName, cfg.Source)

	if cfg.CrewAI {
		em.register(crewai.ExtensionName, cfg)
	}

	if cfg.LLMAnalyzer {
		em.register(llmanalyzer.ExtensionName, cfg)
	}
}

func (em *ExtensionManager) Run(ctx context.Context) ([]*apicore.Extension, error) {
	var builtExtensions []*apicore.Extension

	for name, givenCfg := range em.extensions {
		var ext *clitypes.AgentExtension
		var err error

		switch name {
		case crewai.ExtensionName:
			cfg := givenCfg.(*config.Config)
			ext, err = crewai.New(cfg.Source, cfg.SourceIgnore).Build(ctx)

		case llmanalyzer.ExtensionName:
			var extBuilder clitypes.ExtensionBuilder
			cfg := givenCfg.(*config.Config)
			extBuilder, err = llmanalyzer.New(cfg.Source, cfg.SourceIgnore)
			if err != nil {
				return nil, err
			}
			ext, err = extBuilder.Build(ctx)

		case runtime.ExtensionName:
			ext, err = runtime.New(givenCfg.(string)).Build(ctx)

		case framework.ExtensionName:
			cfg := givenCfg.(framework.Config)
			ext, err = framework.New(&cfg).Build(ctx)

		case language.ExtensionName:
			cfg := givenCfg.(language.Config)
			ext, err = language.New(&cfg).Build(ctx)

		case skills.ExtensionName:
			cfg := givenCfg.(skills.Config)
			ext, err = skills.New(&cfg).Build(ctx)

		default:
			return nil, fmt.Errorf("unknown extension: %s", name)
		}

		if err != nil {
			return nil, err
		}

		apiExt, err := ext.ToAPIExtension()

		if err != nil {
			return nil, err
		}
		builtExtensions = append(builtExtensions, &apiExt)
	}

	return builtExtensions, nil
}
