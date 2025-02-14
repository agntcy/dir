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
)

func Build(
	ctx context.Context,
	fsPath string,
	agent *apicore.Agent,
	authors []string,
	categories []string,
	LLMAnalyzer bool,
) error {
	// Runtime extension
	runtimeBuilder := runtime.New(fsPath)
	runtimeExtension, err := runtimeBuilder.Build(ctx)
	if err != nil {
		return err
	}
	runtimeAPIExtension, err := runtimeExtension.ToAPIExtension()
	if err != nil {
		return err
	}

	// Category extension
	categoryBuilder := category.New(categories)
	categoryExtension, err := categoryBuilder.Build(ctx)
	if err != nil {
		return err
	}
	categoryAPIExtension, err := categoryExtension.ToAPIExtension()
	if err != nil {
		return err
	}

	// CrewAI extension
	crewaiBuilder := crewai.New(fsPath)
	crewaiExtension, err := crewaiBuilder.Build(ctx)
	if err != nil {
		return err
	}
	crewaiAPIExtension, err := crewaiExtension.ToAPIExtension()
	if err != nil {
		return err
	}

	agent.Extensions = append(
		agent.Extensions,
		&runtimeAPIExtension,
		&categoryAPIExtension,
		&crewaiAPIExtension,
	)

	// LLManalyzer extension
	if LLMAnalyzer {
		llmanalyzerBuilder, err := llmanalyzer.New(fsPath)
		if err != nil {
			return err
		}
		llmanalyzerExtension, err := llmanalyzerBuilder.Build(ctx)
		if err != nil {
			return err
		}
		llmanalyzerAPIExtension, err := llmanalyzerExtension.ToAPIExtension()
		if err != nil {
			return err
		}

		agent.Extensions = append(agent.Extensions, &llmanalyzerAPIExtension)
	}

	// Security extension
	//securityBuilder := security.New(agent)
	//securityExtension, err := securityBuilder.Build(ctx)
	//if err != nil {
	//	return err
	//}
	//securityAPIExtension, err := securityExtension.ToAPIExtension()
	//if err != nil {
	//	return err
	//}

	// agent.Extensions = append(
	// 	agent.Extensions,
	// &securityAPIExtension,
	// )

	return nil
}
