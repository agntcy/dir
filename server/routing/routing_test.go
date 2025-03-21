// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint
package routing

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestPublish_InvalidRef(t *testing.T) {
	r := &routing{}
	invalidRef := &coretypes.ObjectRef{Type: "invalid"}

	t.Run("Invalid ref: "+invalidRef.GetType(), func(t *testing.T) {
		err := r.Publish(t.Context(), invalidRef, nil)
		assert.Error(t, err)
		assert.Equal(t, "invalid object type: "+invalidRef.GetType(), err.Error())
	})
}

func TestList_InvalidQuery(t *testing.T) {
	r := &routing{}
	invalidQueries := []string{
		"",
		"/",
		"/agents",
		"/agents/agentX",
		"/skills/",
		"/locators",
		"skills/",
		"locators/",
	}

	for _, q := range invalidQueries {
		t.Run("Invalid query: "+q, func(t *testing.T) {
			_, err := r.List(t.Context(), q)
			assert.Error(t, err)
			assert.Equal(t, "invalid query: "+q, err.Error())
		})
	}
}

func TestPublishList_ValidQuery(t *testing.T) {
	// Test data
	var (
		testAgent = &coretypes.Agent{
			Skills: []*coretypes.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
			},
			Locators: []*coretypes.Locator{
				{Type: "type1", Url: "url1"},
			},
		}
		testAgent2 = &coretypes.Agent{
			Skills: []*coretypes.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
				{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
			},
		}

		testRef  = getObjectRef(testAgent)
		testRef2 = getObjectRef(testAgent2)

		validQueriesWithExpectedObjectRef = map[string][]*coretypes.ObjectRef{
			// tests exact lookup for skills
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
			// tests prefix based-lookup for skills
			"/skills/category2": {
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef2.GetDigest(),
				},
			},
			// tests exact lookup for locators
			"/locators/type1/url1": {
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef.GetDigest(),
				},
			},
			// tests prefix based-lookup for locators
			"/locators/type1": {
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef.GetDigest(),
				},
			},
		}
	)

	// create demo network
	mainNode, err := p2p.NewMockServer(t.Context(), protocol.ID(ProtocolPrefix))
	assert.NoError(t, err)

	// create routing
	r := &routing{
		dstore: datastore.NewMapDatastore(),
		server: mainNode,
	}
	assert.NoError(t, err)
	<-mainNode.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	// Publish first agent
	err = r.Publish(t.Context(), testRef, testAgent)
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

func getObjectRef(a *coretypes.Agent) *coretypes.ObjectRef {
	raw, _ := json.Marshal(a) //nolint:errchkjson

	return &coretypes.ObjectRef{
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest:      digest.FromBytes(raw).String(),
		Size:        uint64(len(raw)),
		Annotations: a.Annotations,
	}
}

func toPtr[T any](v T) *T {
	return &v
}

func TestSplitSkillPath(t *testing.T) {
	tests := []struct {
		skillPath string
		expected  []string
	}{
		{"/skills/", []string{}},
		{"/skills/X", []string{"/skills/X"}},
		{"/skills/X/Y", []string{"/skills/X", "/skills/X/Y"}},
		{"/skills/X/Y/Z", []string{"/skills/X", "/skills/X/Y", "/skills/X/Y/Z"}},
		{"/skills/X/Y/Z/K", []string{"/skills/X", "/skills/X/Y", "/skills/X/Y/Z", "/skills/X/Y/Z/K"}},
	}

	for _, test := range tests {
		t.Run(test.skillPath, func(t *testing.T) {
			result := splitSkillPath(test.skillPath)
			assert.Equal(t, test.expected, result, test.skillPath)
		})
	}
}

// Function to calculate Bloom filter size
func calculateBloomFilterSize(n uint, p float64) (uint, uint) {
	m := -float64(n) * math.Log(p) / (math.Pow(math.Log(2), 2))
	k := (m / float64(n)) * math.Log(2)

	return uint(m), uint(math.Ceil(k))
}

func TestNesto(t *testing.T) {
	// Number of elements and desired false positive rate
	n := uint(100000000)
	p := 0.01

	// Calculate Bloom filter size and optimal hash functions
	m, k := calculateBloomFilterSize(n, p)

	fmt.Printf("Bloom Filter Size: %d bytes\n", m/1024)
	fmt.Printf("Optimal Number of Hash Functions: %d\n", k)
}
