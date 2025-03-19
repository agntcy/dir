// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/builder/plugins/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime"
	clitypes "github.com/agntcy/dir/cli/types"
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

func (b *Builder) BuildUserAgent() (*coretypes.Agent, error) {
	APIExtensions := make([]*coretypes.Extension, 0, len(b.cfg.Model.Extensions))

	for _, i := range b.cfg.Model.Extensions {
		extension := clitypes.AgentExtension{
			Name:    i.Name,
			Version: i.Version,
			Data:    i.Data,
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

	skills, err := b.cfg.GetSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills from config: %w", err)
	}

	return &coretypes.Agent{
		Name:        b.cfg.Model.Name,
		Version:     b.cfg.Model.Version,
		Authors:     b.cfg.Model.Authors,
		CreatedAt:   time.Now().Format(time.RFC3339),
		Annotations: b.cfg.Model.Annotations,
		Skills:      skills,
		Locators:    locators,
		Extensions:  APIExtensions,
	}, nil
}

func (b *Builder) BuildAgent(ctx context.Context) (*coretypes.Agent, error) {
	var APIExtensions []*coretypes.Extension

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

	return &coretypes.Agent{
		Extensions: APIExtensions,
	}, nil
}
