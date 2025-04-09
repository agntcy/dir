// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer/python"
	"github.com/agntcy/dir/cli/types"
)

const (
	ExtensionName    = "schema.oasf.agntcy.org/features/runtime/framework"
	ExtensionVersion = "v0.0.0"
)

type Type string

const (
	CrewAI     Type = "crewai"
	Autogen    Type = "autogen"
	Llamaindex Type = "llama-index"
	Langchain  Type = "langchain"
)

type ExtensionData struct {
	SBOM any `json:"sbom,omitempty"`
}

type Framework struct {
	source        string
	typeAnalyzers map[analyzer.LanguageType]analyzer.Analyzer
}

func New(source string) *Framework {
	return &Framework{
		source: source,
		typeAnalyzers: map[analyzer.LanguageType]analyzer.Analyzer{
			analyzer.Python: python.New(),
		},
	}
}

func (fw *Framework) Build(_ context.Context) (*types.AgentExtension, error) {
	sbom, err := fw.typeAnalyzers[analyzer.Python].SBOM(fw.source)
	if err != nil {
		return nil, fmt.Errorf("failed to get SBOM: %w", err)
	}

	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Data: ExtensionData{
			SBOM: sbom,
		},
	}, nil
}
