// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/builder/extensions/crewai"
	"github.com/agntcy/dir/cli/builder/extensions/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/extensions/runtime"
	"github.com/agntcy/dir/cli/builder/manager"
)

func Build(ctx context.Context, cfg *config.Config) ([]*apicore.Extension, error) {
	extManager := manager.NewExtensionManager()

	// Register extensions
	extManager.Register(runtime.ExtensionName, cfg.Source)

	if cfg.CrewAI {
		extManager.Register(crewai.ExtensionName, cfg)
	}

	if cfg.LLMAnalyzer {
		extManager.Register(llmanalyzer.ExtensionName, cfg)
	}

	// Build and append extensions to agent
	extensions, err := extManager.Build(ctx)
	if err != nil {
		return nil, err
	}

	return extensions, nil
}
