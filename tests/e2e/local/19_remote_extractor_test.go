// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var _ = ginkgo.Describe("Remote extractor HTTP API", ginkgo.Ordered, ginkgo.Label("remote-extractor"), func() {
	const (
		query      = "I need a MCP server to connect to an AGNTCY directory and integrate APIs."
		recordName = "remote_extractor_e2e_directory"
	)

	var recordCID string

	ginkgo.BeforeAll(func() {
		if testEnv.Config.ExtractorMode != "remote" {
			ginkgo.Skip("remote extractor not enabled for this environment")
		}

		gomega.Expect(testEnv.Config.GatewayAddress).NotTo(gomega.BeEmpty(),
			`extractor_mode: "remote" requires gateway_address`)
		utils.ResetCLIState()

		tempDir, err := os.MkdirTemp("", "remote-extractor-e2e-*")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		ginkgo.DeferCleanup(func() { _ = os.RemoveAll(tempDir) })

		// Publish a local fixture, without importing from a source registry or
		// provisioning extractor assets on the test runner.
		var record map[string]any
		gomega.Expect(json.Unmarshal(testdata.DirectoryRecordJSON, &record)).To(gomega.Succeed())
		record["name"] = recordName
		data, err := json.Marshal(record)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		recordPath := filepath.Join(tempDir, "record.json")
		gomega.Expect(os.WriteFile(recordPath, data, 0o600)).To(gomega.Succeed())
		recordCID = strings.TrimSpace(testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed())
		gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		ginkgo.DeferCleanup(func() {
			testEnv.CLI.Delete(recordCID).ShouldSucceed()
		})
	})

	ginkgo.It("extracts relevant taxonomy through the deployed gateway", func(ctx ginkgo.SpecContext) {
		var response catalogv1.ExtractTaxonomyResponse

		// Pod readiness can precede the gateway reconnecting to the extractor.
		// Keep retries bounded so an enabled but broken deployment still fails.
		gomega.Eventually(func(g gomega.Gomega) {
			postRemoteExtractor(ctx, g, "/v1/extract", &catalogv1.ExtractTaxonomyRequest{Text: query}, &response)
		}).WithContext(ctx).WithTimeout(90 * time.Second).WithPolling(2 * time.Second).Should(gomega.Succeed())

		// Match a relevant taxonomy family rather than model-specific scores or
		// ordering, which can change between extractor model versions.
		domainNames := make([]string, 0, len(response.GetDomains()))
		for _, domain := range response.GetDomains() {
			gomega.Expect(domain.GetId()).NotTo(gomega.BeZero())
			gomega.Expect(domain.GetScore()).To(gomega.BeNumerically(">", 0))
			domainNames = append(domainNames, domain.GetName())
		}

		gomega.Expect(domainNames).To(gomega.ContainElement(gomega.HavePrefix("technology/software_engineering")))
	}, ginkgo.SpecTimeout(2*time.Minute))

	ginkgo.It("finds the published directory record by natural language", func(ctx ginkgo.SpecContext) {
		gomega.Eventually(func(g gomega.Gomega) {
			var response catalogv1.SearchAgentsResponse
			postRemoteExtractor(ctx, g, "/v1/search", &catalogv1.SearchAgentsRequest{Query: query, PageSize: 100}, &response)

			var matches []string

			for _, entry := range response.GetResults() {
				if entry.GetDisplayName() == recordName {
					_, cid, found := strings.Cut(entry.GetIdentifier(), ":cid:")
					g.Expect(found).To(gomega.BeTrue())

					matches = append(matches, cid)
				}
			}

			g.Expect(matches).To(gomega.ContainElement(recordCID))
		}).WithContext(ctx).WithTimeout(time.Minute).WithPolling(time.Second).Should(gomega.Succeed())
	}, ginkgo.SpecTimeout(2*time.Minute))
})

// Requests have a deadline even if the gateway or its remote extractor hangs.
// An enabled but broken deployment fails rather than skipping these specs.
func postRemoteExtractor(ctx context.Context, g gomega.Gomega, path string, request, response proto.Message) {
	ginkgo.GinkgoHelper()

	data, err := protojson.Marshal(request)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(testEnv.Config.GatewayAddress, "/")+path, bytes.NewReader(data))
	g.Expect(err).NotTo(gomega.HaveOccurred())
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK), "%s: %s", path, body)
	g.Expect(protojson.Unmarshal(body, response)).To(gomega.Succeed())
}
