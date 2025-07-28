// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Integration test configuration.
	integrationRegistryAddress = "localhost:5555"
	integrationRepositoryName  = "integration-test"
	integrationTimeout         = 30 * time.Minute
)

// Integration test configuration.
var integrationConfig = ociconfig.Config{
	RegistryAddress: integrationRegistryAddress,
	RepositoryName:  integrationRepositoryName,
	AuthConfig: ociconfig.AuthConfig{
		Insecure: true, // Required for local zot registry
	},
}

// createTestRecord creates a comprehensive test record for integration testing.
func createTestRecord() *corev1.Record {
	return &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:          "integration-test-agent",
				Version:       "v1.0.0",
				Description:   "Integration test agent for OCI storage",
				SchemaVersion: "v0.3.1",
				CreatedAt:     "2023-01-01T00:00:00Z",
				Authors:       []string{"integration-test@example.com"},
				Skills: []*objectsv1.Skill{
					{CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
					{CategoryName: stringPtr("ml"), ClassName: stringPtr("inference")},
				},
				Locators: []*objectsv1.Locator{
					{Type: "docker"},
					{Type: "helm"},
				},
				Extensions: []*objectsv1.Extension{
					{Name: "security"},
					{Name: "monitoring"},
				},
				Annotations: map[string]string{
					"team":        "integration-test",
					"environment": "test",
					"project":     "oci-storage",
				},
			},
		},
	}
}

// setupIntegrationStore creates a store connected to the local zot registry.
func setupIntegrationStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// Check if zot registry is available
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+integrationRegistryAddress+"/v2/", nil)
	require.NoError(t, err, "Failed to create registry health check request")

	resp, err := client.Do(req)
	if err != nil {
		t.Skip("Zot registry not available at localhost:5555. Start with manual docker command or task server:store:start")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Zot registry health check failed with status: %d", resp.StatusCode)
	}

	// Create store
	store, err := New(integrationConfig)
	require.NoError(t, err, "Failed to create integration store")

	return store
}

// getRegistryTags fetches all tags for the repository from zot registry.
func getRegistryTags(ctx context.Context, t *testing.T) []string {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://%s/v2/%s/tags/list", integrationRegistryAddress, integrationRepositoryName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create tags request")

	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to fetch tags from registry")
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{} // Repository doesn't exist yet
	}

	require.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status when fetching tags")

	var response struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode tags response")

	return response.Tags
}

// getManifest fetches manifest for a specific tag from zot registry.
func getManifest(ctx context.Context, t *testing.T, tag string) map[string]interface{} {
	t.Helper()

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://%s/v2/%s/manifests/%s", integrationRegistryAddress, integrationRepositoryName, tag)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err, "Failed to create manifest request")
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")

	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to fetch manifest from registry")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status when fetching manifest")

	var manifest map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&manifest)
	require.NoError(t, err, "Failed to decode manifest response")

	return manifest
}

//nolint:maintidx // Function handles multiple test cases with justified complexity
func TestIntegrationOCIStoreWorkflow(t *testing.T) {
	// NOTE: This integration test uses V1 records where skills have "categoryName/className" format
	// This differs from V2/V3 which use simple skill names
	ctx, cancel := context.WithTimeout(t.Context(), integrationTimeout)
	defer cancel()

	store := setupIntegrationStore(t)
	record := createTestRecord()

	t.Run("Push Record", func(t *testing.T) {
		// Push the record
		recordRef, err := store.Push(ctx, record)
		require.NoError(t, err, "Failed to push record")
		require.NotNil(t, recordRef, "Record reference should not be nil")
		require.NotEmpty(t, recordRef.GetCid(), "Record CID should not be empty")

		t.Logf("Pushed record with CID: %s", recordRef.GetCid())
	})

	t.Run("Verify Tags Generated", func(t *testing.T) {
		// Give registry a moment to process
		time.Sleep(1 * time.Second)

		tags := getRegistryTags(ctx, t)
		require.NotEmpty(t, tags, "Registry should contain tags after push")

		t.Logf("Found %d tags in registry: %v", len(tags), tags)

		// Verify expected tag patterns
		var (
			hasContentAddressable = false
			hasNameTag            = false
			hasVersionTag         = false
			hasLatestTag          = false
			hasSkillTags          = false
			hasExtensionTags      = false
			hasDeployTags         = false
			hasTeamTag            = false
		)

		for _, tag := range tags {
			switch {
			case len(tag) > 50 && strings.HasPrefix(tag, "bae"): // CID pattern (JSON-based CIDs start with "bae")
				hasContentAddressable = true
			case tag == "integration-test-agent":
				hasNameTag = true
			case tag == "integration-test-agent_v1.0.0": // Underscores due to OCI normalization
				hasVersionTag = true
			case tag == "integration-test-agent_latest": // Underscores due to OCI normalization
				hasLatestTag = true
			case strings.HasPrefix(tag, "skill."):
				hasSkillTags = true
			case strings.HasPrefix(tag, "ext."):
				hasExtensionTags = true
			case strings.HasPrefix(tag, "deploy."):
				hasDeployTags = true
			case strings.HasPrefix(tag, "team."):
				hasTeamTag = true
			}
		}

		assert.True(t, hasContentAddressable, "Should have content-addressable tag")
		assert.True(t, hasNameTag, "Should have name tag")
		assert.True(t, hasVersionTag, "Should have version tag")
		assert.True(t, hasLatestTag, "Should have latest tag")
		assert.True(t, hasSkillTags, "Should have skill tags")
		assert.True(t, hasExtensionTags, "Should have extension tags")
		assert.True(t, hasDeployTags, "Should have deploy tags")
		assert.True(t, hasTeamTag, "Should have team tag")
	})

	t.Run("Verify Manifest Annotations", func(t *testing.T) {
		// Test with name tag
		manifest := getManifest(ctx, t, "integration-test-agent")

		// Check manifest structure
		require.Contains(t, manifest, "annotations", "Manifest should contain annotations")
		annotations, ok := manifest["annotations"].(map[string]interface{})
		require.True(t, ok, "Annotations should be a map")

		t.Logf("Found %d manifest annotations", len(annotations))

		// Verify core annotations
		expectedAnnotations := map[string]string{
			"org.agntcy.dir/type":           "record",
			"org.agntcy.dir/name":           "integration-test-agent",
			"org.agntcy.dir/version":        "v1.0.0",
			"org.agntcy.dir/description":    "Integration test agent for OCI storage",
			"org.agntcy.dir/oasf-version":   "v0.3.1",
			"org.agntcy.dir/schema-version": "v0.3.1",
			"org.agntcy.dir/created-at":     "2023-01-01T00:00:00Z",
			"org.agntcy.dir/authors":        "integration-test@example.com",
			// NOTE: V1 skills use "categoryName/className" hierarchical format
			"org.agntcy.dir/skills":          "nlp/processing,ml/inference",
			"org.agntcy.dir/locator-types":   "docker,helm",
			"org.agntcy.dir/extension-names": "security,monitoring",
			"org.agntcy.dir/signed":          "false",
		}

		for key, expectedValue := range expectedAnnotations {
			actualValue, exists := annotations[key]
			assert.True(t, exists, "Annotation %s should exist", key)
			assert.Equal(t, expectedValue, actualValue, "Annotation %s should have correct value", key)
		}

		// Verify custom annotations
		customAnnotations := map[string]string{
			"org.agntcy.dir/custom.team":        "integration-test",
			"org.agntcy.dir/custom.environment": "test",
			"org.agntcy.dir/custom.project":     "oci-storage",
		}

		for key, expectedValue := range customAnnotations {
			actualValue, exists := annotations[key]
			assert.True(t, exists, "Custom annotation %s should exist", key)
			assert.Equal(t, expectedValue, actualValue, "Custom annotation %s should have correct value", key)
		}
	})

	t.Run("Verify Descriptor Annotations", func(t *testing.T) {
		manifest := getManifest(ctx, t, "integration-test-agent")

		// Get layers to check descriptor annotations
		require.Contains(t, manifest, "layers", "Manifest should contain layers")
		layers, ok := manifest["layers"].([]interface{})
		require.True(t, ok, "Layers should be an array")
		require.NotEmpty(t, layers, "Should have at least one layer")

		// Check first layer descriptor annotations
		layer, ok := layers[0].(map[string]interface{})
		require.True(t, ok, "First layer should be a map")
		require.Contains(t, layer, "annotations", "Layer should contain annotations")
		annotations, ok := layer["annotations"].(map[string]interface{})
		require.True(t, ok, "Layer annotations should be a map")

		t.Logf("Found %d descriptor annotations", len(annotations))

		// Verify descriptor annotations
		expectedDescriptorAnnotations := map[string]string{
			"org.agntcy.dir/encoding":      "json",
			"org.agntcy.dir/blob-type":     "oasf-record",
			"org.agntcy.dir/schema":        "oasf.v0.3.1.Agent",
			"org.agntcy.dir/compression":   "none",
			"org.agntcy.dir/signed":        "false",
			"org.agntcy.dir/store-version": "v1",
		}

		for key, expectedValue := range expectedDescriptorAnnotations {
			actualValue, exists := annotations[key]
			assert.True(t, exists, "Descriptor annotation %s should exist", key)
			assert.Equal(t, expectedValue, actualValue, "Descriptor annotation %s should have correct value", key)
		}

		// Verify CID annotation exists and is not empty
		contentCid, exists := annotations["org.agntcy.dir/content-cid"]
		assert.True(t, exists, "Content CID annotation should exist")
		assert.NotEmpty(t, contentCid, "Content CID should not be empty")

		// Verify stored-at timestamp exists and is valid
		storedAt, exists := annotations["org.agntcy.dir/stored-at"]
		assert.True(t, exists, "Stored-at annotation should exist")
		assert.NotEmpty(t, storedAt, "Stored-at should not be empty")

		// Verify it's a valid RFC3339 timestamp
		storedAtStr, ok := storedAt.(string)
		require.True(t, ok, "Stored-at should be a string")

		_, err := time.Parse(time.RFC3339, storedAtStr)
		assert.NoError(t, err, "Stored-at should be valid RFC3339 timestamp")
	})

	t.Run("Lookup Record", func(t *testing.T) {
		recordRef := &corev1.RecordRef{Cid: record.GetCid()}

		meta, err := store.Lookup(ctx, recordRef)
		require.NoError(t, err, "Failed to lookup record")
		require.NotNil(t, meta, "Lookup should return metadata")

		t.Logf("Lookup returned metadata with %d annotations", len(meta.GetAnnotations()))

		// Verify metadata contains expected fields
		assert.Equal(t, "integration-test-agent", meta.GetAnnotations()["name"])
		assert.Equal(t, "v1.0.0", meta.GetAnnotations()["version"])
		// NOTE: V1 skills use "categoryName/className" hierarchical format
		assert.Equal(t, "nlp/processing,ml/inference", meta.GetAnnotations()["skills"])
		assert.Equal(t, "v0.3.1", meta.GetSchemaVersion())
		assert.Equal(t, "2023-01-01T00:00:00Z", meta.GetCreatedAt())
	})

	t.Run("Pull Record", func(t *testing.T) {
		recordRef := &corev1.RecordRef{Cid: record.GetCid()}

		pulledRecord, err := store.Pull(ctx, recordRef)
		require.NoError(t, err, "Failed to pull record")
		require.NotNil(t, pulledRecord, "Pull should return record")

		// Verify pulled record matches original
		originalAgent := record.GetV1()
		pulledAgent := pulledRecord.GetV1()

		assert.Equal(t, originalAgent.GetName(), pulledAgent.GetName())
		assert.Equal(t, originalAgent.GetVersion(), pulledAgent.GetVersion())
		assert.Equal(t, originalAgent.GetDescription(), pulledAgent.GetDescription())
		assert.Equal(t, len(originalAgent.GetSkills()), len(pulledAgent.GetSkills()))
		assert.Equal(t, len(originalAgent.GetLocators()), len(pulledAgent.GetLocators()))
		assert.Equal(t, len(originalAgent.GetExtensions()), len(pulledAgent.GetExtensions()))

		t.Logf("Successfully pulled and verified record integrity")
	})

	t.Run("Tag Reconstruction", func(t *testing.T) {
		// Get metadata from lookup
		recordRef := &corev1.RecordRef{Cid: record.GetCid()}
		meta, err := store.Lookup(ctx, recordRef)
		require.NoError(t, err, "Failed to lookup record for tag reconstruction")

		// Reconstruct tags from metadata
		reconstructedTags := reconstructTagsFromRecord(meta.GetAnnotations(), record.GetCid())
		require.NotEmpty(t, reconstructedTags, "Should reconstruct tags from metadata")

		t.Logf("Reconstructed %d tags from metadata: %v", len(reconstructedTags), reconstructedTags)

		// Verify key tags are reconstructed
		tagSet := make(map[string]bool)
		for _, tag := range reconstructedTags {
			tagSet[tag] = true
		}

		assert.True(t, tagSet["integration-test-agent"], "Should reconstruct name tag")
		assert.True(t, tagSet["integration-test-agent_v1.0.0"], "Should reconstruct version tag")
		// NOTE: V1 skills use "categoryName/className" format, so tags become "skill.nlp.processing"
		assert.True(t, tagSet["skill.nlp.processing"], "Should reconstruct skill tag")
		assert.True(t, tagSet["ext.security"], "Should reconstruct extension tag")
		assert.True(t, tagSet["deploy.docker"], "Should reconstruct deploy tag")
	})

	t.Run("Duplicate Push Handling", func(t *testing.T) {
		// Push the same record again
		recordRef, err := store.Push(ctx, record)
		require.NoError(t, err, "Duplicate push should not fail")
		require.NotNil(t, recordRef, "Duplicate push should return reference")
		assert.Equal(t, record.GetCid(), recordRef.GetCid(), "CID should remain the same")

		t.Logf("Duplicate push handled correctly for CID: %s", recordRef.GetCid())
	})
}

func TestIntegrationTagStrategy(t *testing.T) {
	// Create record with minimal data to test different tag strategies
	minimalRecord := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:    "minimal-agent",
				Version: "v1.0.0",
			},
		},
	}

	t.Run("Custom Tag Strategy", func(t *testing.T) {
		// Test with limited tag strategy
		strategy := TagStrategy{
			EnableNameTags:           true,
			EnableCapabilityTags:     false, // Disable capability tags
			EnableInfrastructureTags: false, // Disable infrastructure tags
			EnableTeamTags:           false, // Disable team tags
			EnableContentAddressable: true,
			MaxTagsPerRecord:         5,
		}

		// Get CID for the minimal record
		tags := generateDiscoveryTags(minimalRecord, strategy)
		require.NotEmpty(t, tags, "Should generate tags even with limited strategy")

		t.Logf("Generated %d tags with limited strategy: %v", len(tags), tags)

		// Should have CID and name-based tags only
		var hasContentAddressable, hasNameTag, hasVersionTag bool

		for _, tag := range tags {
			switch {
			case len(tag) > 50:
				hasContentAddressable = true
			case tag == "minimal-agent":
				hasNameTag = true
			case tag == "minimal-agent_v1.0.0": // Underscores due to OCI normalization
				hasVersionTag = true
			}
		}

		assert.True(t, hasContentAddressable, "Should have content-addressable tag")
		assert.True(t, hasNameTag, "Should have name tag")
		assert.True(t, hasVersionTag, "Should have version tag")
		assert.LessOrEqual(t, len(tags), 5, "Should respect max tags limit")
	})
}
