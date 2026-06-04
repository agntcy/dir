// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Manager-scoped search e2e tests.
//
// Org chart (configured via tests/e2e/local/testenv/local/org-chart.yaml):
//
//	alice@example.com
//	├── bob@example.com
//	│   └── dave@example.com
//	└── carol@example.com
//
// Records pushed in BeforeAll:
//   - owner alias "alice@example.com"
//   - owner alias "bob@example.com"
//   - owner alias "carol@example.com"
//   - owner alias "dave@example.com"
//   - owner alias "eve@example.com"  (outside alice's subtree)

var _ = ginkgo.Describe("Manager-scoped search", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir  string
		cidAlice string
		cidBob   string
		cidCarol string
		cidDave  string
		cidEve   string
	)

	ginkgo.Context("searching by manager alias", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "manager-search-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			push := func(ownerAlias string) string {
				record := buildOwnerRecord(ownerAlias)
				data, marshalErr := json.Marshal(record)
				gomega.Expect(marshalErr).NotTo(gomega.HaveOccurred())

				path := filepath.Join(tempDir, fmt.Sprintf("record-%s.json", ownerAlias))
				gomega.Expect(os.WriteFile(path, data, 0o600)).To(gomega.Succeed())

				cid := testEnv.CLI.Push(path).WithArgs("--output", "raw").ShouldSucceed()
				gomega.Expect(cid).NotTo(gomega.BeEmpty())

				return cid
			}

			cidAlice = push("alice@example.com")
			cidBob = push("bob@example.com")
			cidCarol = push("carol@example.com")
			cidDave = push("dave@example.com")
			cidEve = push("eve@example.com")
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("returns all subtree members for top-level manager", func() {
			output := testEnv.CLI.Search().
				WithManager("alice@example.com").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(cidAlice))
			gomega.Expect(output).To(gomega.ContainSubstring(cidBob))
			gomega.Expect(output).To(gomega.ContainSubstring(cidCarol))
			gomega.Expect(output).To(gomega.ContainSubstring(cidDave))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidEve))
		})

		ginkgo.It("returns only direct and indirect reports for mid-level manager", func() {
			output := testEnv.CLI.Search().
				WithManager("bob@example.com").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(cidBob))
			gomega.Expect(output).To(gomega.ContainSubstring(cidDave))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidAlice))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidCarol))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidEve))
		})

		ginkgo.It("returns only the leaf member when they have no reports", func() {
			output := testEnv.CLI.Search().
				WithManager("carol@example.com").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(cidCarol))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidAlice))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidBob))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidDave))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidEve))
		})

		ginkgo.It("returns only the alias itself when not in the org chart", func() {
			output := testEnv.CLI.Search().
				WithManager("eve@example.com").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(cidEve))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidAlice))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidBob))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidCarol))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidDave))
		})

		ginkgo.It("returns nothing for an alias with no owned agents", func() {
			output := testEnv.CLI.Search().
				WithManager("unknown@example.com").
				ShouldSucceed()

			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidAlice))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidBob))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidCarol))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidDave))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidEve))
		})

		ginkgo.It("combines manager filter with annotation filter (AND logic)", func() {
			output := testEnv.CLI.Search().
				WithManager("alice@example.com").
				WithAnnotation("owner.id:bob@example.com").
				ShouldSucceed()

			// Only bob's record matches both manager(alice subtree) AND owner.id=bob
			gomega.Expect(output).To(gomega.ContainSubstring(cidBob))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidAlice))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidCarol))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidDave))
			gomega.Expect(output).NotTo(gomega.ContainSubstring(cidEve))
		})
	})
})

// buildOwnerRecord returns a minimal valid agent record JSON object with
// the given ownerAlias set as the owner.id annotation.
func buildOwnerRecord(ownerAlias string) map[string]any {
	return map[string]any{
		"name":           fmt.Sprintf("http://example.com/manager-test/%s", ownerAlias),
		"version":        "v1.0.0",
		"schema_version": "0.8.0",
		"description":    fmt.Sprintf("Test agent owned by %s", ownerAlias),
		"authors":        []string{"AGNTCY Contributors"},
		"created_at":     "2025-01-01T00:00:00Z",
		"annotations": map[string]string{
			"owner.id": ownerAlias,
		},
		"skills": []map[string]any{
			{"name": "natural_language_processing/natural_language_generation/text_completion", "id": 10201},
		},
		"locators": []map[string]any{
			{"type": "docker_image", "url": "https://ghcr.io/agntcy/test-agent"},
		},
		"domains": []map[string]any{
			{"id": 301, "name": "life_science/biotechnology"},
		},
	}
}
