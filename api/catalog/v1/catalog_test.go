// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"reflect"
	"testing"

	oasftypesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRecordToCatalog(t *testing.T) {
	for _, tc := range []struct {
		name      string
		record    *corev1.Record
		expect    []byte
		expectErr bool
	}{
		// empty record
		{
			name:      "empty record",
			record:    &corev1.Record{},
			expectErr: true,
		},
		// record with single module
		{
			name: "record with single module",
			record: corev1.New(&oasftypesv1.Record{
				Annotations:   map[string]string{},
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
				"identifier": "urn:ai:org.agntcy:cid:baeareidwftjjh5rk4hrigmwdxeizxw7p4elhtow5kwhyxfnio2lakhbp64",
				"display_name": "Test Record 1",
				"version": "1.0.0",
				"description": "A test record with a single module.",
				"updated_at": "2024-01-01T00:00:00Z",
				"media_type": "application/mcp-server-card+json",
				"Artifact": {
					"Data": {
						"key1": "value1",
						"key2": 42
					}
				},
				"tags": [
					"oasf:1.0.0:domains:/example/domain",
					"oasf:1.0.0:skills:/example/skill"
				]
			}`),
		},
		// record with multiple modules
		{
			name: "record with multiple modules",
			record: corev1.New(&oasftypesv1.Record{
				Annotations:   map[string]string{},
				SchemaVersion: "1.0.0",
				Name:          "Test Record 2",
				Description:   "A test record with multiple modules.",
				Version:       "2.0.0",
				CreatedAt:     "2024-02-01T00:00:00Z",
				Modules: []*oasftypesv1.Module{
					{
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
				"identifier": "urn:ai:org.agntcy:cid:baeareievubig624mxq7xcf25xunpsr3tzkngirxnglbeolalenwqupxj34",
				"display_name": "Test Record 2",
				"version": "2.0.0",
				"description": "A test record with multiple modules.",
				"updated_at": "2024-02-01T00:00:00Z",
				"media_type": "application/ai-catalog+json",
				"Artifact": {
					"Data": {
						"spec_version": "1.0",
						"entries": [
							{
								"identifier": "urn:ai:org.agntcy:cid:baeareievubig624mxq7xcf25xunpsr3tzkngirxnglbeolalenwqupxj34:a2a",
								"display_name": "Test Record 2 - A2A",
								"media_type": "application/a2a-agent-card+json",
								"Artifact": {
									"Data": {
										"key2": "value2"
									}
								}
							},
							{
								"identifier": "urn:ai:org.agntcy:cid:baeareievubig624mxq7xcf25xunpsr3tzkngirxnglbeolalenwqupxj34:mcp",
								"display_name": "Test Record 2 - MCP",
								"media_type": "application/mcp-server-card+json",
								"Artifact": {
									"Data": {
										"key1": "value1"
									}
								}
							}
						]
					}
				},
				"tags": [
					"oasf:1.0.0:domains:/example/domain2",
					"oasf:1.0.0:skills:/example/skill2"
				]
			}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog, err := RecordToCatalog(tc.record)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// compare as any structs for better readability in test failures
			var got any

			//nolint:musttag
			catalogBytes, err := json.Marshal(catalog)
			if err != nil {
				t.Fatalf("failed to marshal catalog: %v", err)
			}

			if err := json.Unmarshal(catalogBytes, &got); err != nil {
				t.Fatalf("invalid test expectation JSON: %v", err)
			}

			var want any
			if err := json.Unmarshal(tc.expect, &want); err != nil {
				t.Fatalf("invalid test expectation JSON: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("unexpected catalog result:\nGot:  %v\nWant: %v", got, want)
			}
		})
	}
}
