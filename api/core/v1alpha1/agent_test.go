// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAgent_Merge(t *testing.T) {
	now := time.Now()
	timestamp := timestamppb.New(now)

	testDigest := &Digest{
		Type:  DigestType_DIGEST_TYPE_SHA256,
		Value: "7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
	}

	tests := []struct {
		name     string
		receiver *Agent
		other    *Agent
		want     *Agent
	}{
		{
			name:     "nil other",
			receiver: &Agent{Name: "test"},
			other:    nil,
			want:     &Agent{Name: "test"},
		},
		{
			name: "merge basic fields",
			receiver: &Agent{
				Name:      "original",
				Version:   "",
				CreatedAt: nil,
				Digest:    nil,
			},
			other: &Agent{
				Name:      "other",
				Version:   "1.0",
				CreatedAt: timestamp,
				Digest:    testDigest,
			},
			want: &Agent{
				Name:      "original",
				Version:   "1.0",
				CreatedAt: timestamp,
				Digest:    testDigest,
			},
		},
		{
			name: "merge lists and maps",
			receiver: &Agent{
				Authors: []string{"author1"},
				Skills:  []string{"skill1"},
				Annotations: map[string]string{
					"key1": "value1",
				},
			},
			other: &Agent{
				Authors: []string{"author2", "author1"},
				Skills:  []string{"skill2"},
				Annotations: map[string]string{
					"key1": "other-value1",
					"key2": "value2",
				},
			},
			want: &Agent{
				Authors: []string{"author2", "author1"},
				Skills:  []string{"skill2", "skill1"},
				Annotations: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.receiver.Merge(tt.other)

			// Use proto.Equal for comparing protobuf messages
			assert.True(t, proto.Equal(tt.want, tt.receiver))
		})
	}
}
