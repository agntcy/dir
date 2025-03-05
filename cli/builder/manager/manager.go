package manager

import (
	"context"
	"fmt"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	builderconfig "github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/builder/extensions/crewai"
	"github.com/agntcy/dir/cli/builder/extensions/framework"
	"github.com/agntcy/dir/cli/builder/extensions/language"
	"github.com/agntcy/dir/cli/builder/extensions/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/extensions/runtime"
	"github.com/agntcy/dir/cli/builder/extensions/skill"
	clitypes "github.com/agntcy/dir/cli/types"
)

type ExtensionManager struct {
	extensions map[string]interface{}
}

func NewExtensionManager() *ExtensionManager {
	return &ExtensionManager{extensions: make(map[string]interface{})}
}

func (em *ExtensionManager) Register(name string, config interface{}) {
	em.extensions[name] = config
}

func (em *ExtensionManager) Build(ctx context.Context) ([]*apicore.Extension, error) {
	var builtExtensions []*apicore.Extension

	for name, config := range em.extensions {
		var ext *clitypes.AgentExtension
		var err error

		switch name {
		case crewai.ExtensionName:
			cfg := config.(*builderconfig.Config)
			ext, err = crewai.New(cfg.Source, cfg.SourceIgnore).Build(ctx)

		case llmanalyzer.ExtensionName:
			var extBuilder clitypes.ExtensionBuilder
			cfg := config.(*builderconfig.Config)
			extBuilder, err = llmanalyzer.New(cfg.Source, cfg.SourceIgnore)
			if err != nil {
				return nil, err
			}
			ext, err = extBuilder.Build(ctx)

		case runtime.ExtensionName:
			ext, err = runtime.New(config.(string)).Build(ctx)

		case framework.ExtensionName:
			cfg := config.(builderconfig.Framework)
			ext, err = framework.New(&cfg).Build(ctx)

		case language.ExtensionName:
			cfg := config.(builderconfig.Language)
			ext, err = language.New(&cfg).Build(ctx)

		case skill.ExtensionName:
			cfg := config.([]string)
			ext, err = skill.New(cfg).Build(ctx)

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
