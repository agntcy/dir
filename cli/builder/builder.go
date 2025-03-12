// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"fmt"
	"time"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/builder/plugins/crewai"
	"github.com/agntcy/dir/cli/builder/plugins/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime"
	clitypes "github.com/agntcy/dir/cli/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Builder struct {
	plugins []clitypes.Builder
	cfg     *config.Config
}

func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{
		plugins: make([]clitypes.Builder, 0),
		cfg:     cfg,
	}
}

func (b *Builder) RegisterPlugins() error {
	if b.cfg.Builder.CrewAI {
		b.plugins = append(b.plugins, crewai.New(b.cfg.Builder.Source, b.cfg.Builder.SourceIgnore))
	}

	if b.cfg.Builder.LLMAnalyzer {
		LLMAnalyzer, err := llmanalyzer.New(b.cfg.Builder.Source, b.cfg.Builder.SourceIgnore)
		if err != nil {
			return fmt.Errorf("failed to register LLMAnalyzer plugin: %w", err)
		}

		b.plugins = append(b.plugins, LLMAnalyzer)
	}

	if b.cfg.Builder.Runtime {
		b.plugins = append(b.plugins, runtime.New(b.cfg))
	}

	return nil
}

func (b *Builder) BuildUserAgent() (*apicore.Agent, error) {
	APIExtensions := make([]*apicore.Extension, 0, len(b.cfg.Model.Extensions))

	for _, i := range b.cfg.Model.Extensions {
		extension := clitypes.AgentExtension{
			Name:    i.Name,
			Version: i.Version,
			Specs:   i.Specs,
		}

		APIExtension, err := extension.ToAPIExtension()
		if err != nil {
			return nil, fmt.Errorf("failed to convert extension to API extension: %w", err)
		}

		APIExtensions = append(APIExtensions, &APIExtension)
	}

	locators, err := b.cfg.GetAPILocators()
	if err != nil {
		return nil, fmt.Errorf("failed to get locators from config: %w", err)
	}

	return &apicore.Agent{
		Name:       b.cfg.Model.Name,
		Version:    b.cfg.Model.Version,
		Authors:    b.cfg.Model.Authors,
		CreatedAt:  timestamppb.New(time.Now()),
		Skills:     b.cfg.Model.Skills,
		Locators:   locators,
		Extensions: APIExtensions,
	}, nil
}

func (b *Builder) BuildAgent(ctx context.Context) (*apicore.Agent, error) {
	var APIExtensions []*apicore.Extension

	for _, plugin := range b.plugins {
		extensions, err := plugin.Build(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to build extension: %w", err)
		}

		for _, extension := range extensions {
			APIExtension, err := extension.ToAPIExtension()
			if err != nil {
				return nil, fmt.Errorf("failed to convert extension to API extension: %w", err)
			}

			APIExtensions = append(APIExtensions, &APIExtension)
		}
	}

	return &apicore.Agent{
		Extensions: APIExtensions,
	}, nil
}
