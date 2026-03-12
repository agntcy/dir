// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"testing"

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
	record := &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{},
		},
	}

	url, sub := extractSourceInfo(record)
	assert.Empty(t, url)
	assert.Empty(t, sub)
}

func TestExtractSourceInfo_SourceCodeLocator(t *testing.T) {
	record := &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"locators": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"type": structpb.NewStringValue("source_code"),
								"urls": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStringValue("https://github.com/example/repo"),
									},
								}),
							},
						}),
					},
				}),
			},
		},
	}

	url, sub := extractSourceInfo(record)
	assert.Equal(t, "https://github.com/example/repo", url)
	assert.Empty(t, sub)
}

func TestExtractSourceInfo_SkipsNonSourceCodeLocator(t *testing.T) {
	record := &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"locators": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"type": structpb.NewStringValue("container_image"),
								"urls": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStringValue("https://ghcr.io/example/img"),
									},
								}),
							},
						}),
					},
				}),
			},
		},
	}

	url, sub := extractSourceInfo(record)
	assert.Empty(t, url)
	assert.Empty(t, sub)
}

func TestExtractSubfolder(t *testing.T) {
	record := &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"locators": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"type": structpb.NewStringValue("source_code"),
								"urls": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStringValue("https://github.com/example/mono"),
									},
								}),
							},
						}),
					},
				}),
				"modules": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"data": structpb.NewStructValue(&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"mcp_data": structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"repository": structpb.NewStructValue(&structpb.Struct{
													Fields: map[string]*structpb.Value{
														"subfolder": structpb.NewStringValue("packages/mcp-server"),
													},
												}),
											},
										}),
									},
								}),
							},
						}),
					},
				}),
			},
		},
	}

	url, sub := extractSourceInfo(record)
	assert.Equal(t, "https://github.com/example/mono", url)
	assert.Equal(t, "packages/mcp-server", sub)
}

func TestIsPlaceholderURL(t *testing.T) {
	tests := []struct {
		url         string
		placeholder bool
	}{
		{"https://example.com/mcp-server.git", true},
		{"https://www.example.com/mcp-server.git", true},
		{"https://example.org/foo", true},
		{"https://example.net/bar", true},
		{"https://github.com/org/repo", false},
		{"https://gitlab.com/org/repo", false},
		{"not-a-url", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.placeholder, isPlaceholderURL(tt.url))
		})
	}
}
