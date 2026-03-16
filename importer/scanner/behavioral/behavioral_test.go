// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"testing"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestExtractSourceInfo_NilRecord(t *testing.T) {
	url, sub := extractSourceInfo(nil)
	assert.Empty(t, url)
	assert.Empty(t, sub)
}

func TestExtractSourceInfo_NoLocators(t *testing.T) {
	record := corev1.New(&typesv1.Record{
		SchemaVersion: "1.0.0",
	})

	url, sub := extractSourceInfo(record)
	assert.Empty(t, url)
	assert.Empty(t, sub)
}

func TestExtractSourceInfo_SourceCodeLocator(t *testing.T) {
	record := corev1.New(&typesv1.Record{
		SchemaVersion: "1.0.0",
		Locators: []*typesv1.Locator{
			{Type: "source_code", Urls: []string{"https://github.com/example/repo"}},
		},
	})

	url, sub := extractSourceInfo(record)
	assert.Equal(t, "https://github.com/example/repo", url)
	assert.Empty(t, sub)
}

func TestExtractSourceInfo_SkipsNonSourceCodeLocator(t *testing.T) {
	record := corev1.New(&typesv1.Record{
		SchemaVersion: "1.0.0",
		Locators: []*typesv1.Locator{
			{Type: "container_image", Urls: []string{"https://ghcr.io/example/img"}},
		},
	})

	url, sub := extractSourceInfo(record)
	assert.Empty(t, url)
	assert.Empty(t, sub)
}

func TestExtractSubfolder(t *testing.T) {
	moduleData, _ := structpb.NewStruct(map[string]any{
		"mcp_data": map[string]any{
			"repository": map[string]any{
				"subfolder": "packages/mcp-server",
			},
		},
	})

	record := corev1.New(&typesv1.Record{
		SchemaVersion: "1.0.0",
		Locators: []*typesv1.Locator{
			{Type: "source_code", Urls: []string{"https://github.com/example/mono"}},
		},
		Modules: []*typesv1.Module{
			{Data: moduleData},
		},
	})

	url, sub := extractSourceInfo(record)
	assert.Equal(t, "https://github.com/example/mono", url)
	assert.Equal(t, "packages/mcp-server", sub)
}
