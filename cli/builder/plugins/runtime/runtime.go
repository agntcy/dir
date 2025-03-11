// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer/python"
	"github.com/agntcy/dir/cli/types"
)

const (
	// ExtensionName is the name of the extension.
	ExtensionName = "runtime"
	// ExtensionVersion is the version of the extension.
	ExtensionVersion = "v0.0.0"
)

type ExtensionSpecs struct {
	Language string `json:"language,omitempty"`
	Version  string `json:"version,omitempty"`
	SBOM     any    `json:"sbom,omitempty"`
}

type runtime struct {
	// The source. Right not this must be the path to the directory containing the source code
	// of the application. In the future, this could be a git repository, a docker image, a tgz etc.
	source string

	// The analyzer to use to analyze the source code
	analyzer analyzer.Analyzer
}

func New(source string) types.ExtensionBuilder {
	// NOTE(msardara): while we are returning a python analyzer, here we
	// should be smart enough to determine the language of the source code,
	// or we should get it as input.
	return &runtime{
		source:   source,
		analyzer: python.New(),
	}
}

func (c *runtime) Build(_ context.Context) (*types.AgentExtension, error) {
	// get the runtime version and relevant dependencies
	runtimeInfo, err := c.analyzer.RuntimeVersion(c.source)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime version: %w", err)
	}

	sbom, err := c.analyzer.SBOM(c.source)
	if err != nil {
		return nil, fmt.Errorf("failed to get SBOM: %w", err)
	}

	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: ExtensionSpecs{
			Language: runtimeInfo.Language,
			Version:  runtimeInfo.Version,
			SBOM:     sbom,
		},
	}, nil
}
