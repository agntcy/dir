// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package oci

import (
	"context"
	"os"
	"testing"

	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha0"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: this should be configurable to unified Storage API test flow.
var (
	// test config.
	testConfig = ociconfig.Config{
		LocalDir:        os.TempDir(),                         // used for local test/bench
		RegistryAddress: "localhost:5000",                     // used for remote test/bench
		RepositoryName:  "test-store",                         // used for remote test/bench
		AuthConfig:      ociconfig.AuthConfig{Insecure: true}, // used for remote test/bench
	}
	runLocal = true
	// TODO: this may blow quickly when doing rapid benchmarking if not tested against fresh OCI instance.
	runRemote = false

	// common test.
	testCtx = context.Background()
)

func TestStorePushLookupPullDelete(t *testing.T) {
	store := loadLocalStore(t)

	agent := &typesv1alpha0.Record{
		Name:          "test-agent",
		SchemaVersion: "v0.3.1",
		Description:   "A test agent",
	}

	record := corev1.New(agent)

	// Calculate CID for the record
	recordCID := record.GetCid()
	assert.NotEmpty(t, recordCID, "failed to calculate CID")

	// Push operation
	recordRef, err := store.Push(testCtx, record)
	assert.NoErrorf(t, err, "push failed")
	assert.Equal(t, recordCID, recordRef.GetCid())

	// Lookup operation
	recordMeta, err := store.Lookup(testCtx, recordRef)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, recordCID, recordMeta.GetCid())

	// Pull operation
	pulledRecord, err := store.Pull(testCtx, recordRef)
	assert.NoErrorf(t, err, "pull failed")

	pulledCID := pulledRecord.GetCid()
	assert.NotEmpty(t, pulledCID, "failed to get pulled record CID")
	assert.Equal(t, recordCID, pulledCID)

	// Verify the pulled agent data
	decoded, _ := record.Decode()
	pulledAgent := decoded.GetV1Alpha0()
	assert.NotNil(t, pulledAgent, "pulled agent should not be nil")
	assert.Equal(t, agent.GetName(), pulledAgent.GetName())
	assert.Equal(t, agent.GetSchemaVersion(), pulledAgent.GetSchemaVersion())
	assert.Equal(t, agent.GetDescription(), pulledAgent.GetDescription())

	// Delete operation
	err = store.Delete(testCtx, recordRef)
	assert.NoErrorf(t, err, "delete failed")

	// Lookup should fail after delete
	_, err = store.Lookup(testCtx, recordRef)
	assert.Error(t, err, "lookup should fail after delete")
	assert.ErrorContains(t, err, "not found")

	// Pull should also fail after delete
	_, err = store.Pull(testCtx, recordRef)
	assert.Error(t, err, "pull should fail after delete")
	assert.ErrorContains(t, err, "not found")
}

func BenchmarkLocalStore(b *testing.B) {
	if !runLocal {
		b.Skip()
	}

	store := loadLocalStore(&testing.T{})
	for range b.N {
		benchmarkStep(store)
	}
}

func BenchmarkRemoteStore(b *testing.B) {
	if !runRemote {
		b.Skip()
	}

	store := loadRemoteStore(&testing.T{})
	for range b.N {
		benchmarkStep(store)
	}
}

func benchmarkStep(store types.StoreAPI) {
	// Create test record
	agent := &typesv1alpha0.Record{
		Name:          "bench-agent",
		SchemaVersion: "v0.3.1",
		Description:   "A benchmark agent",
	}

	record := corev1.New(agent)

	// Record is ready for push operation

	// Push operation
	pushedRef, err := store.Push(testCtx, record)
	if err != nil {
		panic(err)
	}

	// Lookup operation
	fetchedMeta, err := store.Lookup(testCtx, pushedRef)
	if err != nil {
		panic(err)
	}

	// Assert equal
	if pushedRef.GetCid() != fetchedMeta.GetCid() {
		panic("not equal lookup")
	}
}

func loadLocalStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create tmp storage for test artifacts
	tmpDir, err := os.MkdirTemp(testConfig.LocalDir, "test-oci-store-*") //nolint:usetesting
	assert.NoErrorf(t, err, "failed to create test dir")
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatalf("failed to cleanup: %v", err)
		}
	})

	// create local
	store, err := New(ociconfig.Config{LocalDir: tmpDir})
	assert.NoErrorf(t, err, "failed to create local store")

	return store
}

func loadRemoteStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create remote
	store, err := New(
		ociconfig.Config{
			RegistryAddress: testConfig.RegistryAddress,
			RepositoryName:  testConfig.RepositoryName,
			AuthConfig:      testConfig.AuthConfig,
		})
	assert.NoErrorf(t, err, "failed to create remote store")

	return store
}

// TestAllVersionsSkillsAndLocatorsPreservation comprehensively tests skills and locators
// preservation across all OASF versions (v1, v2, v3) through OCI push/pull cycles.
// This addresses the reported issue where v3 record skills become empty after push/pull.
func TestAllVersionsSkillsAndLocatorsPreservation(t *testing.T) {
	store := loadLocalStore(t)

	testCases := []struct {
		name                 string
		record               *corev1.Record
		expectedSkillCount   int
		expectedLocatorCount int
		skillVerifier        func(t *testing.T, record *corev1.Record)
		locatorVerifier      func(t *testing.T, record *corev1.Record)
	}{
		{
			name: "V1_Agent_CategoryClass_Skills",
			record: corev1.New(&typesv1alpha0.Record{
				Name:          "test-v1-agent",
				Version:       "1.0.0",
				SchemaVersion: "v0.3.1",
				Description:   "Test v1 agent with hierarchical skills",
				Skills: []*typesv1alpha0.Skill{
					{
						CategoryName: stringPtr("Natural Language Processing"),
						CategoryUid:  1,
						ClassName:    stringPtr("Text Completion"),
						ClassUid:     10201,
					},
					{
						CategoryName: stringPtr("Machine Learning"),
						CategoryUid:  2,
						ClassName:    stringPtr("Classification"),
						ClassUid:     20301,
					},
				},
				Locators: []*typesv1alpha0.Locator{
					{
						Type: "docker-image",
						Url:  "ghcr.io/agntcy/test-v1-agent",
					},
					{
						Type: "helm-chart",
						Url:  "oci://registry.example.com/charts/test-agent",
					},
				},
			}),
			expectedSkillCount:   2,
			expectedLocatorCount: 2,
			skillVerifier: func(t *testing.T, record *corev1.Record) {
				t.Helper()

				decoded, _ := record.Decode()
				v1Agent := decoded.GetV1Alpha0()
				require.NotNil(t, v1Agent, "should be v1 agent")
				skills := v1Agent.GetSkills()
				require.Len(t, skills, 2, "v1 should have 2 skills")

				// V1 uses category/class format
				assert.Equal(t, "Natural Language Processing", skills[0].GetCategoryName())
				assert.Equal(t, "Text Completion", skills[0].GetClassName())
				assert.Equal(t, uint64(10201), skills[0].GetClassUid())

				assert.Equal(t, "Machine Learning", skills[1].GetCategoryName())
				assert.Equal(t, "Classification", skills[1].GetClassName())
				assert.Equal(t, uint64(20301), skills[1].GetClassUid())
			},
			locatorVerifier: func(t *testing.T, record *corev1.Record) {
				t.Helper()

				decoded, _ := record.Decode()
				v1Agent := decoded.GetV1Alpha0()
				locators := v1Agent.GetLocators()
				require.Len(t, locators, 2, "v1 should have 2 locators")

				assert.Equal(t, "docker-image", locators[0].GetType())
				assert.Equal(t, "ghcr.io/agntcy/test-v1-agent", locators[0].GetUrl())

				assert.Equal(t, "helm-chart", locators[1].GetType())
				assert.Equal(t, "oci://registry.example.com/charts/test-agent", locators[1].GetUrl())
			},
		},
		{
			name: "V3_Record_Simple_Skills",
			record: corev1.New(&typesv1alpha1.Record{
				Name:          "test-v3-record",
				Version:       "3.0.0",
				SchemaVersion: "0.7.0",
				Description:   "Test v3 record with simple skills",
				Skills: []*typesv1alpha1.Skill{
					{
						Name: "Natural Language Processing",
						Id:   10201,
					},
					{
						Name: "Data Analysis",
						Id:   20301,
					},
				},
				Locators: []*typesv1alpha1.Locator{
					{
						Type: "docker-image",
						Url:  "ghcr.io/agntcy/test-v3-record",
					},
					{
						Type: "oci-artifact",
						Url:  "oci://registry.example.com/artifacts/test-record",
					},
				},
			}),
			expectedSkillCount:   2,
			expectedLocatorCount: 2,
			skillVerifier: func(t *testing.T, record *corev1.Record) {
				t.Helper()

				decoded, _ := record.Decode()
				v3Record := decoded.GetV1Alpha1()
				require.NotNil(t, v3Record, "should be v3 record")
				skills := v3Record.GetSkills()
				require.Len(t, skills, 2, "SKILLS ISSUE: v3 should have 2 skills but has %d", len(skills))

				// V3 uses simple name/id format (same as v2)
				assert.Equal(t, "Natural Language Processing", skills[0].GetName())
				assert.Equal(t, uint32(10201), skills[0].GetId())

				assert.Equal(t, "Data Analysis", skills[1].GetName())
				assert.Equal(t, uint32(20301), skills[1].GetId())
			},
			locatorVerifier: func(t *testing.T, record *corev1.Record) {
				t.Helper()

				decoded, _ := record.Decode()
				v3Record := decoded.GetV1Alpha1()
				locators := v3Record.GetLocators()
				require.Len(t, locators, 2, "v3 should have 2 locators")

				assert.Equal(t, "docker-image", locators[0].GetType())
				assert.Equal(t, "ghcr.io/agntcy/test-v3-record", locators[0].GetUrl())

				assert.Equal(t, "oci-artifact", locators[1].GetType())
				assert.Equal(t, "oci://registry.example.com/artifacts/test-record", locators[1].GetUrl())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate CID for the original record
			originalCID := tc.record.GetCid()
			require.NotEmpty(t, originalCID, "failed to calculate CID for %s", tc.name)

			// Log original state
			t.Logf("🔄 Testing %s:", tc.name)
			t.Logf("   Original CID: %s", originalCID)
			t.Logf("   Expected skills: %d, locators: %d", tc.expectedSkillCount, tc.expectedLocatorCount)

			// Verify original skills and locators using verifiers
			tc.skillVerifier(t, tc.record)
			tc.locatorVerifier(t, tc.record)

			// PUSH operation
			recordRef, err := store.Push(testCtx, tc.record)
			require.NoError(t, err, "push should succeed for %s", tc.name)
			assert.Equal(t, originalCID, recordRef.GetCid(), "pushed CID should match original")

			// PULL operation
			pulledRecord, err := store.Pull(testCtx, recordRef)
			require.NoError(t, err, "pull should succeed for %s", tc.name)

			// Verify pulled record CID matches
			pulledCID := pulledRecord.GetCid()
			require.NotEmpty(t, pulledCID, "pulled record should have CID")
			assert.Equal(t, originalCID, pulledCID, "pulled CID should match original")

			// CRITICAL TEST: Verify skills and locators are preserved after push/pull cycle
			t.Logf("   Verifying skills preservation...")
			tc.skillVerifier(t, pulledRecord)

			t.Logf("   Verifying locators preservation...")
			tc.locatorVerifier(t, pulledRecord)

			t.Logf("✅ %s: Skills and locators preserved successfully", tc.name)

			// Cleanup - delete the record
			err = store.Delete(testCtx, recordRef)
			require.NoError(t, err, "cleanup delete should succeed for %s", tc.name)
		})
	}
}
