// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package language

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer/python"
	"github.com/agntcy/dir/cli/types"
)

const (
	ExtensionName    = "oasf.agntcy.org/features/runtime/language"
	ExtensionVersion = "v0.0.0"
)

type ExtensionData struct {
	Type    analyzer.LanguageType `json:"type,omitempty"`
	Version string                `json:"version,omitempty"`
}

type Language struct {
	source        string
	typeAnalyzers map[analyzer.LanguageType]analyzer.Analyzer
}

func New(source string) *Language {
	return &Language{
		source: source,
		typeAnalyzers: map[analyzer.LanguageType]analyzer.Analyzer{
			analyzer.Python: python.New(),
		},
	}
}

func (l *Language) Build(_ context.Context) (*types.AgentExtension, error) {
	runtimeInfo, err := l.typeAnalyzers[analyzer.Python].RuntimeVersion(l.source)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime version: %w", err)
	}

	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Data: ExtensionData{
			Type:    runtimeInfo.Language,
			Version: runtimeInfo.Version,
		},
	}, nil
}
