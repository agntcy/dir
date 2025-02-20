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
)

func Build(ctx context.Context, fsPath string, agent *apicore.Agent, categories []string, LLMAnalyzer bool) error {
	extManager := manager.NewExtensionManager()

	// Register extensions
	extManager.Register(runtime.ExtensionName, fsPath)
	extManager.Register(category.ExtensionName, categories)
	extManager.Register(crewai.ExtensionName, fsPath)

	if LLMAnalyzer {
		extManager.Register(llmanalyzer.ExtensionName, fsPath)
	}

	// Build and append extensions to agent
	extensions, err := extManager.Build(ctx)
	if err != nil {
		return err
	}

	agent.Extensions = append(agent.Extensions, extensions...)

	return nil
}
