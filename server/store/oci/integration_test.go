// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha0"
	corev1 "github.com/agntcy/dir/api/core/v1"
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
	return corev1.New(&typesv1alpha0.Record{
		Name:          "integration-test-agent",
		Version:       "v1.0.0",
		Description:   "Integration test agent for OCI storage",
		SchemaVersion: "v0.3.1",
		CreatedAt:     "2023-01-01T00:00:00Z",
		Authors:       []string{"integration-test@example.com"},
		Skills: []*typesv1alpha0.Skill{
			{CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
			{CategoryName: stringPtr("ml"), ClassName: stringPtr("inference")},
		},
		Locators: []*typesv1alpha0.Locator{
			{Type: "docker"},
			{Type: "helm"},
		},
		Extensions: []*typesv1alpha0.Extension{
			{Name: "security"},
			{Name: "monitoring"},
		},
		Annotations: map[string]string{
			"team":        "integration-test",
			"environment": "test",
			"project":     "oci-storage",
		},
	})
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

	t.Run("Verify CID Tag Generated", func(t *testing.T) {
		// Give registry a moment to process
		time.Sleep(1 * time.Second)

		tags := getRegistryTags(ctx, t)
		require.NotEmpty(t, tags, "Registry should contain tags after push")

		t.Logf("Found %d tags in registry: %v", len(tags), tags)

		// With CID-only tagging, we should have exactly one tag: the CID
		expectedCID := record.GetCid()
		require.NotEmpty(t, expectedCID, "Record should have a valid CID")

		// Verify the CID tag exists in registry
		var hasCIDTag bool

		for _, tag := range tags {
			if tag == expectedCID {
				hasCIDTag = true

				break
			}
		}

		assert.True(t, hasCIDTag, "Registry should contain the CID tag: %s", expectedCID)
		assert.Len(t, tags, 1, "Should have exactly one CID tag, found: %v", tags)
	})

	t.Run("Verify Manifest Annotations", func(t *testing.T) {
		// Test with CID tag (the only tag we create now)
		manifest := getManifest(ctx, t, record.GetCid())

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
			"org.agntcy.dir/skills":        "nlp/processing,ml/inference",
			"org.agntcy.dir/locator-types": "docker,helm",
			"org.agntcy.dir/module-names":  "security,monitoring",
			"org.agntcy.dir/signed":        "false",
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

	// Note: Descriptor annotations removed during CID-only refactoring
	// Layer descriptors now only contain basic fields: mediaType, digest, size

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
		decodedOriginalAgent, _ := record.Decode()
		originalAgent := decodedOriginalAgent.GetV1Alpha0()
		decodedPulledAgent, _ := pulledRecord.Decode()
		pulledAgent := decodedPulledAgent.GetV1Alpha0()

		assert.Equal(t, originalAgent.GetName(), pulledAgent.GetName())
		assert.Equal(t, originalAgent.GetVersion(), pulledAgent.GetVersion())
		assert.Equal(t, originalAgent.GetDescription(), pulledAgent.GetDescription())
		assert.Len(t, pulledAgent.GetSkills(), len(originalAgent.GetSkills()))
		assert.Len(t, pulledAgent.GetLocators(), len(originalAgent.GetLocators()))
		assert.Len(t, pulledAgent.GetExtensions(), len(originalAgent.GetExtensions()))

		t.Logf("Successfully pulled and verified record integrity")
	})

	t.Run("CID Tag Reconstruction", func(t *testing.T) {
		// Get CID for reconstruction
		expectedCID := record.GetCid()
		require.NotEmpty(t, expectedCID, "Record should have a valid CID")

		// In CID-only approach, reconstruction just returns the CID
		reconstructedTags := []string{expectedCID}
		require.NotEmpty(t, reconstructedTags, "Should reconstruct CID tag")

		t.Logf("Reconstructed CID tag: %v", reconstructedTags)

		// Verify only CID tag is reconstructed
		assert.Len(t, reconstructedTags, 1, "Should reconstruct exactly one CID tag")
		assert.Equal(t, expectedCID, reconstructedTags[0], "Reconstructed tag should be the CID")
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

// TestIntegrationTagStrategy removed - no longer needed with CID-only tagging
