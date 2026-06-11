// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	oasftypesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRecordToCatalog(t *testing.T) {
	for _, tc := range []struct {
		name      string
		record    *corev1.Record
		expect    []byte
		expectErr bool
	}{
		// record with single module
		{
			name: "record with single module",
			record: corev1.New(&oasftypesv1.Record{
				Annotations: map[string]string{
					"example/annotation": "example value",
				},
				SchemaVersion: "1.0.0",
				Name:          "Test Record 1",
				Description:   "A test record with a single module.",
				Version:       "1.0.0",
				CreatedAt:     "2024-01-01T00:00:00Z",
				Modules: []*oasftypesv1.Module{
					{
						Name: "integration/mcp",
						Id:   123,
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key1": structpb.NewStringValue("value1"),
								"key2": structpb.NewNumberValue(42),
							},
						},
						Annotations: map[string]string{
							"example/module/annotation": "example module value",
						},
					},
				},
				Domains: []*oasftypesv1.Domain{
					{
						Name: "/example/domain",
					},
				},
				Skills: []*oasftypesv1.Skill{
					{
						Name: "/example/skill",
					},
				},
			}),
			expect: []byte(`{
				"identifier": "urn:ai:org.agntcy:cid:baeareihpexgaj4vr5tcukj7uxnn5yrxv4kqt2tqcdn2atkqfg3tq5pahw4",
				"displayName": "Test Record 1",
				"version": "1.0.0",
				"description": "A test record with a single module.",
				"updatedAt": "2024-01-01T00:00:00Z",
				"mediaType": "application/mcp-server-card+json",
				"data": {
					"key1": "value1",
					"key2": 42
				},
				"tags": [
					"oasf:1.0.0:domains:/example/domain",
					"oasf:1.0.0:skills:/example/skill",
					"example/annotation=example value",
					"example/module/annotation=example module value"
				]
			}`),
		},
		// record with multiple modules
		{
			name: "record with multiple modules",
			record: corev1.New(&oasftypesv1.Record{
				Annotations: map[string]string{
					"example/annotation": "example value",
				},
				Authors:       []string{"Test Publisher"},
				SchemaVersion: "1.0.0",
				Name:          "Test Record 2",
				Description:   "A test record with multiple modules.",
				Version:       "2.0.0",
				CreatedAt:     "2024-02-01T00:00:00Z",
				Modules: []*oasftypesv1.Module{
					{
						Annotations: map[string]string{
							"example/module/annotation": "mcp",
						},
						Name: "integration/mcp",
						Id:   123,
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key1": structpb.NewStringValue("value1"),
							},
						},
					},
					{
						Name: "integration/a2a", // should be ignored
						Id:   456,
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key2": structpb.NewStringValue("value2"),
							},
						},
					},
				},
				Domains: []*oasftypesv1.Domain{
					{
						Name: "/example/domain2",
					},
				},
				Skills: []*oasftypesv1.Skill{
					{
						Name: "/example/skill2",
					},
				},
			}),
			expect: []byte(`{
				"identifier": "urn:ai:org.agntcy:cid:baeareifgarfb5mezccohu3h7ubfsdkn37edsmqvfv2q37k4qs6nzwqk4ve",
				"displayName": "Test Record 2",
				"version": "2.0.0",
				"description": "A test record with multiple modules.",
				"updatedAt": "2024-02-01T00:00:00Z",
				"mediaType": "application/ai-catalog+json",
				"data": {
					"specVersion": "1.0",
					"entries": [
						{
							"displayName": "Test Record 2 - A2A",
							"mediaType": "application/a2a-agent-card+json",
							"data": {
								"key2": "value2"
							}
						},
						{
							"displayName": "Test Record 2 - MCP",
							"mediaType": "application/mcp-server-card+json",
							"data": {
								"key1": "value1"
							},
							"tags": [
								"example/module/annotation=mcp"
							]
						}
					]
				},
				"tags": [
					"oasf:1.0.0:domains:/example/domain2",
					"oasf:1.0.0:skills:/example/skill2",
					"example/annotation=example value"
				]
			}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter, err := tc.record.Decode()
			require.NoError(t, err)

			gotCatalog, err := RecordToCatalog(adapter)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			wantCatalog := &CatalogEntry{}
			if err := protojson.Unmarshal(tc.expect, wantCatalog); err != nil {
				t.Fatalf("invalid test expectation JSON: %v", err)
			}

			wantJSON, _ := protojson.Marshal(wantCatalog)
			gotJSON, _ := protojson.Marshal(gotCatalog)

			assert.JSONEq(t, string(wantJSON), string(gotJSON), "expected and got catalog JSON do not match")
		})
	}
}
