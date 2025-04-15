// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package runtime

import (
	"testing"

	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
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
	expectedSBOM := map[string]interface{}{
		"name": "testdata",
		"packages": []interface{}{
			map[string]interface{}{"name": "crewai", "version": "0.83.0"},
			map[string]interface{}{"name": "langchain", "version": "0.3.14"},
			map[string]interface{}{"name": "langchain-openai", "version": "0.2.14"},
		},
	}
	expectedVersion := ">=3.11,<3.13"

	r := New(source)
	ret, err := r.Build(t.Context())
	assert.NoError(t, err)

	frameworkData := ret.GetExtensions()[0].GetData().AsMap()["sbom"]
	assert.Equal(t, expectedSBOM, frameworkData)

	languageData := ret.GetExtensions()[1].GetData().AsMap()
	assert.Equal(t, string(analyzer.Python), languageData["type"])
	assert.Equal(t, expectedVersion, languageData["version"])
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
