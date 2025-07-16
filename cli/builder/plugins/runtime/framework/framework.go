// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"context"
	"fmt"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
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

func (fw *Framework) Build(_ context.Context) (*objectsv1.Extension, error) {
	sbom, err := fw.typeAnalyzers[analyzer.Python].SBOM(fw.source)
	if err != nil {
		return nil, fmt.Errorf("failed to get SBOM: %w", err)
	}

	strct, err := types.ToStruct(map[string]any{
		"sbom": sbom,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to struct: %w", err)
	}

	return &objectsv1.Extension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Data:    strct,
	}, nil
}
