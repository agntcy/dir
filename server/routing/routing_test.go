// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint
package routing

import (
	"encoding/json"
	"testing"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/ipfs/go-datastore"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func Ptr[T any](v T) *T {
	return &v
}

var testAgent = &coretypes.Agent{
	Skills: []*coretypes.Skill{
		{
			CategoryName: Ptr("category1"),
			ClassName:    Ptr("class1"),
		},
	},
	Locators: []*coretypes.Locator{
		{
			Type: "type1",
			Url:  "url1",
		},
	},
}

func getObjectRef(a *coretypes.Agent) *coretypes.ObjectRef {
	raw, _ := json.Marshal(a) //nolint:errchkjson

	return &coretypes.ObjectRef{
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest:      digest.FromBytes(raw).String(),
		Size:        uint64(len(raw)),
		Annotations: nil,
	}
}

func TestPublish_InvalidQuery(t *testing.T) {
	r := &routing{ds: nil}

	invalidRef := &coretypes.ObjectRef{
		Type: "invalid",
	}

	t.Run("Invalid ref: "+invalidRef.GetType(), func(t *testing.T) {
		err := r.Publish(t.Context(), invalidRef, testAgent)
		assert.Error(t, err)
		assert.Equal(t, "invalid object type: "+invalidRef.GetType(), err.Error())
	})
}

var invalidQueries = []string{
	"",
	"/",
	"/agents",
	"/agents/agentX",
	"/skills/",
	"/locators",
	"skills/",
	"locators/",
}

func TestList_InvalidQuery(t *testing.T) {
	r := &routing{ds: nil}

	for _, q := range invalidQueries {
		t.Run("Invalid query: "+q, func(t *testing.T) {
			_, err := r.List(t.Context(), q)
			assert.Error(t, err)
			assert.Equal(t, "invalid query: "+q, err.Error())
		})
	}
}

var (
	testRef    = getObjectRef(testAgent)
	testAgent2 = func() *coretypes.Agent {
		agent := *testAgent //nolint:govet
		agent.Skills = append(agent.Skills, &coretypes.Skill{
			CategoryName: Ptr("category2"),
			ClassName:    Ptr("class2"),
		})

		return &agent
	}()
)
var testRef2 = getObjectRef(testAgent2)

var validQueriesWithExpectedObjectRef = map[string][]*coretypes.ObjectRef{
	"/skills/category1/class1": {
		{
			Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
			Digest: testRef.GetDigest(),
		},
		{
			Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
			Digest: testRef2.GetDigest(),
		},
	},
	"/skills/category2/class2": {
		{
			Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
			Digest: testRef2.GetDigest(),
		},
	},
}

func TestPublishList_ValidQuery(t *testing.T) {
	// create in-memory datastore
	ds := datastore.NewMapDatastore()
	r := &routing{ds: ds}

	// Publish first agent
	err := r.Publish(t.Context(), testRef, testAgent)
	assert.NoError(t, err)

	// Publish second agent
	err = r.Publish(t.Context(), testRef2, testAgent2)
	assert.NoError(t, err)

	for k, v := range validQueriesWithExpectedObjectRef {
		t.Run("Valid query: "+k, func(t *testing.T) {
			// list
			refs, err := r.List(t.Context(), k)
			assert.NoError(t, err)

			// check if expected refs are present
			assert.Len(t, refs, len(v))

			for _, ref := range refs {
				for _, r := range v {
					if ref.GetDigest() == r.GetDigest() {
						break
					}
				}
			}
		})
	}
}
