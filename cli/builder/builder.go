// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/builder/plugins/llmanalyzer"
	"github.com/agntcy/dir/cli/builder/plugins/pyprojectparser"
	"github.com/agntcy/dir/cli/builder/plugins/runtime"
	clitypes "github.com/agntcy/dir/cli/types"
)

type Builder struct {
	plugins []clitypes.Builder
	source  string
	cfg     *config.Config
}

func NewBuilder(source string, cfg *config.Config) *Builder {
	return &Builder{
		plugins: make([]clitypes.Builder, 0),
		source:  source,
		cfg:     cfg,
	}
}

func (b *Builder) RegisterPlugins() error {
	if b.cfg.Builder.LLMAnalyzer {
		LLMAnalyzer, err := llmanalyzer.New(b.source, b.cfg.Builder.SourceIgnore)
		if err != nil {
			return fmt.Errorf("failed to register LLMAnalyzer plugin: %w", err)
		}

		b.plugins = append(b.plugins, LLMAnalyzer)
	}

	if b.cfg.Builder.Runtime {
		b.plugins = append(b.plugins, runtime.New(b.source))
	}

	if b.cfg.Builder.PyprojectParser {
		b.plugins = append(b.plugins, pyprojectparser.New(b.source))
	}

	return nil
}

func (b *Builder) BuildAgent(ctx context.Context) (*coretypes.Agent, error) {
	agent := &coretypes.Agent{
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	for _, plugin := range b.plugins {
		pluginAgent, err := plugin.Build(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to build plugin: %w", err)
		}

		agent.Merge(pluginAgent)
	}

	return agent, nil
}
