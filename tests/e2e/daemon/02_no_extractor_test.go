// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Extractor-backed HTTP endpoints on a gateway with no OASF extractor.
//
// The "default" testenv points the daemon's extractor at an asset directory that
// does not exist and configures no OASF-SDK server, so resolution finds no
// backend. What that must produce is a node that still starts and serves its
// other routes, with only the extractor-backed endpoints degraded — the reason
// this lives in e2e rather than beside the handler unit tests, which construct a
// controller with a nil extractor directly and cannot catch a node that fails to
// come up at all.
var _ = ginkgo.Describe("Extractor-backed HTTP API without an extractor", func() {
	ginkgo.BeforeEach(func() {
		if testEnv.Config.GatewayAddress == "" {
			ginkgo.Skip("HTTP gateway address not configured for this environment")
		}

		if !testEnv.Config.ExpectNoExtractor {
			ginkgo.Skip("this environment configures an extractor; 503 assertions do not apply")
		}
	})

	ginkgo.It("serves a non-extractor endpoint, proving the gateway is up", func(ctx ginkgo.SpecContext) {
		// Without this the 503s below are ambiguous: a gateway that never
		// started would produce a connection error, not a 503, but a gateway
		// that started and is broken for every route would look the same as one
		// that is healthy apart from the extractor.
		gomega.Eventually(func(g gomega.Gomega) {
			status, _ := getGateway(ctx, "/v1/agents")
			g.Expect(status).To(gomega.Equal(http.StatusOK))
		}).WithContext(ctx).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(gomega.Succeed())
	})

	ginkgo.It("answers POST /v1/extract with 503", func(ctx ginkgo.SpecContext) {
		status, body := postGateway(ctx, "/v1/extract", `{"text":"an agent that reviews source code"}`)

		gomega.Expect(status).To(gomega.Equal(http.StatusServiceUnavailable))
		gomega.Expect(body).To(gomega.ContainSubstring("no OASF extractor is configured"))
	})

	ginkgo.It("answers POST /v1/search with 503", func(ctx ginkgo.SpecContext) {
		status, body := postGateway(ctx, "/v1/search", `{"query":"an agent that reviews source code"}`)

		gomega.Expect(status).To(gomega.Equal(http.StatusServiceUnavailable))
		gomega.Expect(body).To(gomega.ContainSubstring("no OASF extractor is configured"))
	})

	ginkgo.It("rejects an invalid request before reaching the extractor", func(ctx ginkgo.SpecContext) {
		// Argument validation runs ahead of the extractor check, so an empty
		// query is a 400 even here. Asserting it pins the order: a handler that
		// checked the extractor first would report an outage for a request that
		// was never going to be served anyway.
		status, _ := postGateway(ctx, "/v1/search", `{"query":""}`)

		gomega.Expect(status).To(gomega.Equal(http.StatusBadRequest))
	})
})

// getGateway issues a GET against the deployed HTTP gateway and returns the
// status code and response body.
func getGateway(ctx context.Context, path string) (int, string) {
	ginkgo.GinkgoHelper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testEnv.Config.GatewayAddress+path, nil)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return doGateway(req)
}

// postGateway issues a JSON POST against the deployed HTTP gateway and returns
// the status code and response body.
func postGateway(ctx context.Context, path, body string) (int, string) {
	ginkgo.GinkgoHelper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		testEnv.Config.GatewayAddress+path, bytes.NewReader([]byte(body)))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	req.Header.Set("Content-Type", "application/json")

	return doGateway(req)
}

func doGateway(req *http.Request) (int, string) {
	ginkgo.GinkgoHelper()

	resp, err := http.DefaultClient.Do(req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return resp.StatusCode, string(body)
}
