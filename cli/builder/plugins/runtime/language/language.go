// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package language

import (
	"context"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer/python"
	"github.com/agntcy/dir/cli/types"
)

const (
	ExtensionName    = "schema.oasf.agntcy.org/features/runtime/language"
	ExtensionVersion = "v0.0.0"
)

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

func (l *Language) Build(_ context.Context) (*coretypes.Extension, error) {
	runtimeInfo, err := l.typeAnalyzers[analyzer.Python].RuntimeVersion(l.source)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime version: %w", err)
	}

	strct, err := types.ToStruct(map[string]any{
		"type":    runtimeInfo.Language,
		"version": runtimeInfo.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to struct: %w", err)
	}

	return &coretypes.Extension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Data:    strct,
	}, nil
}
