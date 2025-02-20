// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/extensions/category"
	"github.com/agntcy/dir/cli/builder/extensions/crewai"
	"github.com/agntcy/dir/cli/builder/extensions/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/extensions/runtime"
	"github.com/agntcy/dir/cli/builder/manager"
	"github.com/agntcy/dir/cli/cmd/build/config"
	"github.com/agntcy/dir/cli/types"
)

func Build(ctx context.Context, cfg *config.Config) ([]*apicore.Extension, error) {
	extManager := manager.NewExtensionManager()

	// Register extensions
	extManager.Register(runtime.ExtensionName, cfg.Source)
	extManager.Register(category.ExtensionName, cfg.Categories)
	extManager.Register(crewai.ExtensionName, cfg.Source)

	if cfg.LLMAnalyzer {
		extManager.Register(llmanalyzer.ExtensionName, cfg.Source)
	}

	// Build and append extensions to agent
	extensions, err := extManager.Build(ctx)
	if err != nil {
		return nil, err
	}

	// Append config extensions
	for _, ext := range cfg.Extensions {
		extension := types.AgentExtension{
			Name:    ext.Name,
			Version: ext.Version,
			Specs:   ext.Specs,
		}

		apiExt, err := extension.ToAPIExtension()
		if err != nil {
			return nil, err
		}

		extensions = append(extensions, &apiExt)
	}

	return extensions, nil
}
