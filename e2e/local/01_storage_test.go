// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a local single node deployment", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		utils.ResetCLIState()
		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	// Setup temp directory for all tests
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// Test cases for each OASF version
	testVersions := []struct {
		name               string
		fileName           string
		jsonData           []byte
		expectedAgentName  string
		expectedSkillIDs   []string
		expectedSkillNames []string
		expectedLocator    string
		expectedModule     string
	}{
		{
			name:              "OASF_0.3.1_Record",
			fileName:          "oasf_0.3.1_record_test.json",
			jsonData:          testdata.ExpectedRecordV031JSON,
			expectedAgentName: "directory.agntcy.org/cisco/marketing-strategy-v1",
			expectedSkillIDs:  []string{"10201", "10702"},
			expectedSkillNames: []string{
				"Natural Language Processing/Text Completion",
				"Natural Language Processing/Problem Solving",
			},
			expectedLocator: "docker-image:https://ghcr.io/agntcy/marketing-strategy",
			expectedModule:  "schema.oasf.agntcy.org/features/runtime/framework",
		},
		{
			name:              "OASF_0.7.0_Record",
			fileName:          "oasf_0.7.0_record_test.json",
			jsonData:          testdata.ExpectedRecordV070JSON,
			expectedAgentName: "directory.agntcy.org/cisco/marketing-strategy-v3",
			expectedSkillIDs:  []string{"10201", "10702"},
			expectedSkillNames: []string{
				"natural_language_processing/natural_language_generation/text_completion",
				"natural_language_processing/analytical_reasoning/problem_solving",
			},
			expectedLocator: "docker_image:https://ghcr.io/agntcy/marketing-strategy",
			expectedModule:  "runtime/framework",
		},
		{
			name:              "OASF_0.8.0_Record",
			fileName:          "oasf_0.8.0_record_test.json",
			jsonData:          testdata.ExpectedRecordV080JSON,
			expectedAgentName: "directory.agntcy.org/cisco/marketing-strategy-v4",
			expectedSkillIDs:  []string{"10201", "10702"},
			expectedSkillNames: []string{
				"natural_language_processing/natural_language_generation/text_completion",
				"natural_language_processing/analytical_reasoning/problem_solving",
			},
			expectedLocator: "docker_image:https://ghcr.io/agntcy/marketing-strategy",
			expectedModule:  "runtime/framework",
		},
	}

	// Test each OASF version (V1, V2, V3) to identify JSON marshal/unmarshal issues
	for _, v := range testVersions {
		version := v // Capture loop variable by value to avoid closure issues
		ginkgo.Context(version.name, ginkgo.Ordered, ginkgo.Serial, func() {
			var cid string

			// Setup file path and create file
			tempPath := filepath.Join(tempDir, version.fileName)

			// Create directory and write record data once per version
			_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
			_ = os.WriteFile(tempPath, version.jsonData, 0o600)

			// Step 1: Push
			ginkgo.It("should successfully push an record", func() {
				cid = cli.Push(tempPath).WithArgs("--output", "raw").ShouldSucceed()

				// Validate that the returned CID correctly represents the pushed data
				utils.LoadAndValidateCID(cid, tempPath)
			})

			// Step 2: Pull (depends on push)
			ginkgo.It("should successfully pull an existing record", func() {
				cli.Pull(cid).ShouldSucceed()
			})

			// Step 3: Verify push/pull consistency (depends on pull)
			ginkgo.It("should return identical record when pulled after push", func() {
				// Pull the record and get the output JSON
				pulledJSON := cli.Pull(cid).WithArgs("--output", "json").ShouldSucceed()

				// Compare original embedded JSON with pulled JSON using version-aware comparison
				equal, err := utils.CompareOASFRecords(version.jsonData, []byte(pulledJSON))
				gomega.Expect(err).NotTo(gomega.HaveOccurred(),
					"JSON comparison should not error for %s", version.name)
				gomega.Expect(equal).To(gomega.BeTrue(),
					"PUSH/PULL MISMATCH for %s: Original and pulled record should be identical. "+
						"This indicates data loss during push/pull cycle - possibly the skills issue!", version.name)
			})

			// Step 4: Verify duplicate push returns same CID (depends on push)
			ginkgo.It("should push the same record again and return the same cid", func() {
				cli.Push(tempPath).WithArgs("--output", "raw").ShouldReturn(cid)
			})

			// Step 5: Search by first skill (depends on push)
			ginkgo.It("should search for records with first skill and return their CID", func() {
				// This test will FAIL if skills are lost during JSON marshal/unmarshal
				// or during the push/pull process, helping identify the root cause
				search := cli.Search().
					WithLimit(10).
					WithOffset(0).
					WithArgs("--output", "raw").
					WithName(version.expectedAgentName). // Use version-specific record name to prevent conflicts between V1/V2/V3 tests
					WithSkillID(version.expectedSkillIDs[0]).
					WithSkillName(version.expectedSkillNames[0])

				// Add locator and module queries only if they exist (not empty for minimal test)
				if version.expectedLocator != "" {
					search = search.WithLocator(version.expectedLocator)
				}
				if version.expectedModule != "" {
					search = search.WithModule(version.expectedModule)
				}

				search.ShouldContain(cid)
			})

			// Step 6: Search by second skill (depends on push)
			ginkgo.It("should search for records with second skill and return their CID", func() {
				// This test specifically checks the second skill to ensure ALL skills are preserved
				// Skip if there's only one skill (like in minimal test)
				if len(version.expectedSkillIDs) < 2 {
					ginkgo.Skip("Skipping second skill test - only one skill in test data")
				}

				search := cli.Search().
					WithLimit(10).
					WithOffset(0).
					WithArgs("--output", "raw").
					WithName(version.expectedAgentName). // Use version-specific record name to prevent conflicts between V1/V2/V3 tests
					WithSkillID(version.expectedSkillIDs[1]).
					WithSkillName(version.expectedSkillNames[1])

				// Add locator and module queries only if they exist (not empty for minimal test)
				if version.expectedLocator != "" {
					search = search.WithLocator(version.expectedLocator)
				}
				if version.expectedModule != "" {
					search = search.WithModule(version.expectedModule)
				}

				search.ShouldContain(cid)
			})

			// Step 7: Test non-existent pull (independent test)
			ginkgo.It("should pull a non-existent record and return an error", func() {
				_ = cli.Pull("non-existent-CID").ShouldFail()
			})

			// Step 8: Delete (depends on previous steps)
			ginkgo.It("should successfully delete an record", func() {
				cli.Delete(cid).ShouldSucceed()
			})

			// Step 9: Verify deletion (depends on delete)
			ginkgo.It("should fail to pull a deleted record", func() {
				// Add a small delay to ensure delete operation is fully processed
				time.Sleep(100 * time.Millisecond)

				_ = cli.Pull(cid).ShouldFail()
			})
		})
	}
})
