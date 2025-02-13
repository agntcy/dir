// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"context"
	"github.com/agntcy/dir/cli/builder/extensions/runtime/analyzer"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	source = "./analyzer/utils/syft/testdata"
)

func TestNewRuntime(t *testing.T) {
	r := New(source)
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
	expectedLanguage := "python"
	expectedVersion := ">=3.11,<3.13"

	r := New(source)
	ret, err := r.Build(context.Background())
	assert.Nil(t, err)

	specs, ok := ret.Specs.(ExtensionSpecs)
	assert.True(t, ok)
	assert.Equal(t, expectedLanguage, specs.Language)
	assert.Equal(t, expectedVersion, specs.Version)
	assert.Equal(t, expectedSBOM, specs.SBOM)
}

func TestBuildRuntimeWithInvalidSource(t *testing.T) {
	r := New("invalid")
	_, err := r.Build(context.Background())
	assert.NotNil(t, err)
}

func TestBuildRuntimeWithUnsupportedSource(t *testing.T) {
	r := New("./analyzer/python/testdata/unsupported")
	_, err := r.Build(context.Background())
	assert.NotNil(t, err)
}

func TestBuildRuntimeWithNoVersion(t *testing.T) {
	r := New("./analyzer/python/testdata/no-version")
	_, err := r.Build(context.Background())
	assert.NotNil(t, err)
}
