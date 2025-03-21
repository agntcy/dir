// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package runtime

import (
	"testing"

	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/framework"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/language"
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
	expectedVersion := ">=3.11,<3.13"

	r := New(source)
	ret, err := r.Build(t.Context())
	assert.NoError(t, err)

	frameworkData, ok := ret[0].Data.(framework.ExtensionData)
	assert.True(t, ok)
	assert.Equal(t, expectedSBOM, frameworkData.SBOM)

	languageData, ok := ret[1].Data.(language.ExtensionData)
	assert.True(t, ok)
	assert.Equal(t, analyzer.Python, languageData.Type)
	assert.Equal(t, expectedVersion, languageData.Version)
}

func TestBuildRuntimeWithInvalidSource(t *testing.T) {
	r := New("invalid")
	_, err := r.Build(t.Context())
	assert.Error(t, err)
}

func TestBuildRuntimeWithUnsupportedSource(t *testing.T) {
	r := New("./analyzer/python/testdata/unsupported")
	_, err := r.Build(t.Context())
	assert.Error(t, err)
}

func TestBuildRuntimeWithNoVersion(t *testing.T) {
	r := New("./analyzer/python/testdata/no-version")
	_, err := r.Build(t.Context())
	assert.Error(t, err)
}
