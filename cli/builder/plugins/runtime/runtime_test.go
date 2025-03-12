// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package runtime

import (
	"context"
	"testing"

	"github.com/agntcy/dir/cli/builder/config"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/framework"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/language"
	"github.com/stretchr/testify/assert"
)

const (
	source = "./analyzer/utils/syft/testdata"
)

var cfg = &config.Config{
	Builder: config.Builder{
		Source: source,
	},
}

func TestNewRuntime(t *testing.T) {
	r := New(cfg)
	assert.NotNil(t, r)
}

func TestBuildRuntime(t *testing.T) {
	expectedSBOM := analyzer.SBOM{
		Name: "testdata",
		Packages: []analyzer.Package{
			{Name: "crewai", Version: "0.83.0"},
			{Name: "langchain", Version: "0.3.14"},
			{Name: "langchain-openai", Version: "0.2.14"},
		},
	}
	expectedVersion := ">=3.11,<3.13"

	r := New(cfg)
	ret, err := r.Build(context.Background())
	assert.NoError(t, err)

	frameworkSpecs, ok := ret[0].Specs.(framework.ExtensionSpecs)
	assert.True(t, ok)
	assert.Equal(t, expectedSBOM, frameworkSpecs.SBOM)

	languageSpecs, ok := ret[1].Specs.(language.ExtensionSpecs)
	assert.True(t, ok)
	assert.Equal(t, analyzer.Python, languageSpecs.Type)
	assert.Equal(t, expectedVersion, languageSpecs.Version)
}

func TestBuildRuntimeWithInvalidSource(t *testing.T) {
	cfg.Builder.Source = "invalid"
	r := New(cfg)
	_, err := r.Build(context.Background())
	assert.Error(t, err)
}

func TestBuildRuntimeWithUnsupportedSource(t *testing.T) {
	cfg.Builder.Source = "./analyzer/python/testdata/unsupported"
	r := New(cfg)
	_, err := r.Build(context.Background())
	assert.Error(t, err)
}

func TestBuildRuntimeWithNoVersion(t *testing.T) {
	cfg.Builder.Source = "./analyzer/python/testdata/no-version"
	r := New(cfg)
	_, err := r.Build(context.Background())
	assert.Error(t, err)
}
