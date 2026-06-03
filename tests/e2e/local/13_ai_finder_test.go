// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// AI Finder ListAgents over the HTTP gateway (GET /v1/agents).
//
// Uses ExpectedRecordV100JSON ("burger_seller_agent"), which carries
// integration/mcp and integration/a2a modules and therefore projects to an
// AI Catalog entry. The gateway is only deployed in the host "local"
// environment, so these specs skip elsewhere.
var _ = ginkgo.Describe("AI Finder ListAgents HTTP API", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir    string
		recordPath string
		recordCID  string
	)

	ginkgo.Context("with a published record", ginkgo.Ordered, func() {
		ginkgo.BeforeAll(func() {
			if testEnv.Config.GatewayAddress == "" {
				ginkgo.Skip("HTTP gateway address not configured for this environment")
			}

			var err error

			tempDir, err = os.MkdirTemp("", "ai-finder-test")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath = filepath.Join(tempDir, "record_100.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV100JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("lists the published agent", func(ctx ginkgo.SpecContext) {
			gomega.Eventually(func(g gomega.Gomega) {
				status, body := getAgents(ctx, "")
				g.Expect(status).To(gomega.Equal(http.StatusOK))
				g.Expect(body).To(gomega.ContainSubstring(recordCID))
				g.Expect(body).To(gomega.ContainSubstring("burger_seller_agent"))
			}).WithContext(ctx).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(gomega.Succeed())
		})

		ginkgo.It("filters by displayName", func(ctx ginkgo.SpecContext) {
			query := url.Values{"filter": {"displayName=burger_seller_agent"}}.Encode()

			status, body := getAgents(ctx, query)
			gomega.Expect(status).To(gomega.Equal(http.StatusOK))
			gomega.Expect(body).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("returns no results for a non-matching filter", func(ctx ginkgo.SpecContext) {
			query := url.Values{"filter": {"displayName=does-not-exist"}}.Encode()

			status, body := getAgents(ctx, query)
			gomega.Expect(status).To(gomega.Equal(http.StatusOK))
			gomega.Expect(body).NotTo(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("rejects an invalid filter with HTTP 400", func(ctx ginkgo.SpecContext) {
			query := url.Values{"filter": {"displayName"}}.Encode()

			status, _ := getAgents(ctx, query)
			gomega.Expect(status).To(gomega.Equal(http.StatusBadRequest))
		})
	})
})

// getAgents issues GET /v1/agents against the deployed HTTP gateway and
// returns the status code and response body.
func getAgents(ctx context.Context, rawQuery string) (int, string) {
	target := testEnv.Config.GatewayAddress + "/v1/agents"
	if rawQuery != "" {
		target += "?" + rawQuery
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	resp, err := http.DefaultClient.Do(req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return resp.StatusCode, string(body)
}
