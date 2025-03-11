// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"fmt"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/builder/plugins/crewai"
	"github.com/agntcy/dir/cli/builder/plugins/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime"
	clitypes "github.com/agntcy/dir/cli/types"
)

type Builder struct {
	extensions []clitypes.ExtensionBuilder
	cfg        *config.Config
}

func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{
		extensions: make([]clitypes.ExtensionBuilder, 0),
		cfg:        cfg,
	}
}

func (em *Builder) RegisterExtensions() error {
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

func (em *Builder) Run(ctx context.Context) ([]*apicore.Extension, error) {
	builtExtensions := make([]*apicore.Extension, 0, len(em.extensions)+len(em.cfg.Model.Extensions))

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

	for _, i := range em.cfg.Model.Extensions {
		extension := clitypes.AgentExtension{
			Name:    i.Name,
			Version: i.Version,
			Specs:   i.Specs,
		}

		apiExt, err := extension.ToAPIExtension()
		if err != nil {
			return nil, fmt.Errorf("failed to convert extension to API extension: %w", err)
		}

		builtExtensions = append(builtExtensions, &apiExt)
	}

	return builtExtensions, nil
}
