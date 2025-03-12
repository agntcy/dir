// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAgent_Merge(t *testing.T) {
	now := timestamppb.New(time.Now())
	digest := &Digest{
		Type:  DigestType_DIGEST_TYPE_SHA256,
		Value: "7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
	}

	specs1, _ := structpb.NewStruct(map[string]interface{}{
		"key1": "value1",
	})
	specs2, _ := structpb.NewStruct(map[string]interface{}{
		"key2": "value2",
	})

	tests := []struct {
		name     string
		receiver *Agent
		other    *Agent
		want     *Agent
	}{
		{
			name:     "nil receiver",
			receiver: nil,
			other:    &Agent{Name: "test"},
			want:     nil,
		},
		{
			name:     "nil other",
			receiver: &Agent{Name: "test"},
			other:    nil,
			want:     &Agent{Name: "test", Annotations: map[string]string{}},
		},
		{
			name: "merge scalar fields",
			receiver: &Agent{
				Name:    "",
				Version: "",
			},
			other: &Agent{
				Name:      "test-agent",
				Version:   "1.0.0",
				CreatedAt: now,
				Digest:    digest,
			},
			want: &Agent{
				Name:        "test-agent",
				Version:     "1.0.0",
				CreatedAt:   now,
				Digest:      digest,
				Annotations: map[string]string{},
			},
		},
		{
			name: "merge locators",
			receiver: &Agent{
				Locators: []*Locator{
					{
						Name: "loc1",
						Type: LocatorType_LOCATOR_TYPE_DOCKER_IMAGE,
						Source: &LocatorSource{
							Url:  "registry.example.com/agent:v1",
							Size: 1024,
							Digest: &Digest{
								Type:  DigestType_DIGEST_TYPE_SHA256,
								Value: "digest1",
							},
						},
						Annotations: map[string]string{
							"key1": "value1",
						},
						Digest: &Digest{
							Type:  DigestType_DIGEST_TYPE_SHA256,
							Value: "loc-digest1",
						},
					},
				},
				Annotations: map[string]string{},
			},
			other: &Agent{
				Locators: []*Locator{
					{
						Name: "loc1",
						Type: LocatorType_LOCATOR_TYPE_DOCKER_IMAGE,
						Source: &LocatorSource{
							Url:  "registry.example.com/agent:v2",
							Size: 2048,
							Digest: &Digest{
								Type:  DigestType_DIGEST_TYPE_SHA256,
								Value: "digest2",
							},
						},
					},
					{
						Name: "loc2",
						Type: LocatorType_LOCATOR_TYPE_HELM_CHART,
						Source: &LocatorSource{
							Url: "https://charts.example.com/agent",
						},
					},
				},
			},
			want: &Agent{
				Annotations: map[string]string{},
				Locators: []*Locator{
					{
						Name: "loc1",
						Type: LocatorType_LOCATOR_TYPE_DOCKER_IMAGE,
						Source: &LocatorSource{
							Url:  "registry.example.com/agent:v1",
							Size: 1024,
							Digest: &Digest{
								Type:  DigestType_DIGEST_TYPE_SHA256,
								Value: "digest1",
							},
						},
						Annotations: map[string]string{
							"key1": "value1",
						},
						Digest: &Digest{
							Type:  DigestType_DIGEST_TYPE_SHA256,
							Value: "loc-digest1",
						},
					},
					{
						Name: "loc2",
						Type: LocatorType_LOCATOR_TYPE_HELM_CHART,
						Source: &LocatorSource{
							Url: "https://charts.example.com/agent",
						},
					},
				},
			},
		},
		{
			name: "merge extensions",
			receiver: &Agent{
				Extensions: []*Extension{
					{
						Name:    "ext1",
						Version: "1.0",
						Annotations: map[string]string{
							"key1": "value1",
						},
						Specs: specs1,
						Digest: &Digest{
							Type:  DigestType_DIGEST_TYPE_SHA256,
							Value: "ext-digest1",
						},
					},
				},
				Annotations: map[string]string{},
			},
			other: &Agent{
				Extensions: []*Extension{
					{
						Name:    "ext1",
						Version: "2.0",
						Annotations: map[string]string{
							"key1": "other-value1",
						},
						Specs: specs2,
					},
					{
						Name:    "ext2",
						Version: "1.0",
						Specs:   specs2,
					},
				},
			},
			want: &Agent{
				Annotations: map[string]string{},
				Extensions: []*Extension{
					{
						Name:    "ext1",
						Version: "1.0",
						Annotations: map[string]string{
							"key1": "value1",
						},
						Specs: specs1,
						Digest: &Digest{
							Type:  DigestType_DIGEST_TYPE_SHA256,
							Value: "ext-digest1",
						},
					},
					{
						Name:    "ext2",
						Version: "1.0",
						Specs:   specs2,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.receiver.Merge(tt.other)
			assert.Equal(t, tt.want, tt.receiver)
		})
	}
}
