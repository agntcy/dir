// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// HTTP helpers shared by the AI Finder gateway specs (13, 18 and 19).
//
// The gateway marshals with UseProtoNames disabled, so response fields arrive
// camelCased ("nextPageToken", "totalCount", "displayName") rather than under
// their proto names. The structs below follow the wire, not the proto.

// searchResponse is the JSON shape of catalogv1.SearchAgentsResponse.
type searchResponse struct {
	Results       []catalogEntry `json:"results"`
	NextPageToken string         `json:"nextPageToken"`
	TotalCount    uint32         `json:"totalCount"`
}

// catalogEntry is the subset of catalogv1.CatalogEntry the search specs assert on.
type catalogEntry struct {
	Identifier  string `json:"identifier"`
	DisplayName string `json:"displayName"`
}

// scoredClass is the JSON shape of catalogv1.ScoredClass.
type scoredClass struct {
	ID    uint32  `json:"id"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Tier  uint32  `json:"tier"`
}

// extractResponse is the JSON shape of catalogv1.ExtractTaxonomyResponse.
type extractResponse struct {
	Skills   []scoredClass `json:"skills"`
	Domains  []scoredClass `json:"domains"`
	Keywords []string      `json:"keywords"`
}

// cids returns the record CIDs of the results, in the order the endpoint
// returned them. Rank order is the contract for POST /v1/search, so the specs
// compare these slices rather than sets.
func (r searchResponse) cids() []string {
	out := make([]string, 0, len(r.Results))
	for _, e := range r.Results {
		out = append(out, entryCID(e.Identifier))
	}

	return out
}

// entryCID recovers the record CID from a catalog entry identifier URN
// ("urn:ai:<host>:cid:<cid>"), returning the input unchanged when it is not a URN.
func entryCID(identifier string) string {
	if _, after, ok := strings.Cut(identifier, ":cid:"); ok {
		return after
	}

	return identifier
}

// gatewayGet issues a GET against the deployed HTTP gateway and returns the
// status code and raw response body. rawQuery is appended as the query string
// when non-empty.
func gatewayGet(ctx context.Context, path, rawQuery string) (int, string) {
	ginkgo.GinkgoHelper()

	target := testEnv.Config.GatewayAddress + path
	if rawQuery != "" {
		target += "?" + rawQuery
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return doGateway(req)
}

// gatewayPostJSON issues a JSON POST against the deployed HTTP gateway and
// returns the status code and raw response body.
func gatewayPostJSON(ctx context.Context, path string, payload any) (int, string) {
	ginkgo.GinkgoHelper()

	body, err := json.Marshal(payload)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		testEnv.Config.GatewayAddress+path, bytes.NewReader(body))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	req.Header.Set("Content-Type", "application/json")

	return doGateway(req)
}

func doGateway(req *http.Request) (int, string) {
	ginkgo.GinkgoHelper()

	status, body, err := tryGateway(req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return status, body
}

// tryGateway performs the request and returns the transport error instead of
// failing, for callers polling an endpoint that may not be serving yet. The
// asserting helpers above would abort a gomega.Eventually on the first refused
// connection rather than retry.
func tryGateway(req *http.Request) (int, string, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("%s %s: %w", req.Method, req.URL, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("read %s %s response: %w", req.Method, req.URL, err)
	}

	return resp.StatusCode, string(body), nil
}

// tryGatewayPostJSON is gatewayPostJSON for pollers: it reports a transport
// failure as an error rather than failing the spec outright.
func tryGatewayPostJSON(ctx context.Context, path string, payload any) (int, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		testEnv.Config.GatewayAddress+path, bytes.NewReader(body))
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return tryGateway(req)
}

// postSearch calls POST /v1/search and requires HTTP 200, returning the decoded
// response. Specs asserting on failures use gatewayPostJSON directly.
func postSearch(ctx context.Context, req searchRequest) searchResponse {
	ginkgo.GinkgoHelper()

	status, body := gatewayPostJSON(ctx, "/v1/search", req)
	gomega.Expect(status).To(gomega.Equal(http.StatusOK), "POST /v1/search failed: %s", body)

	var out searchResponse
	gomega.Expect(json.Unmarshal([]byte(body), &out)).To(gomega.Succeed())

	return out
}

// searchRequest is the JSON shape of catalogv1.SearchAgentsRequest. pageSize and
// pageToken are omitted when unset so the server applies its own defaults.
type searchRequest struct {
	Query     string `json:"query"`
	PageSize  uint32 `json:"pageSize,omitempty"`
	PageToken string `json:"pageToken,omitempty"`
}

// postExtract calls POST /v1/extract and requires HTTP 200, returning the
// decoded response.
func postExtract(ctx context.Context, text string) extractResponse {
	ginkgo.GinkgoHelper()

	status, body := gatewayPostJSON(ctx, "/v1/extract", map[string]string{"text": text})
	gomega.Expect(status).To(gomega.Equal(http.StatusOK), "POST /v1/extract failed: %s", body)

	var out extractResponse
	gomega.Expect(json.Unmarshal([]byte(body), &out)).To(gomega.Succeed())

	return out
}

// skipUnlessExtractorGateway skips a spec unless this environment deploys the
// HTTP gateway with an OASF extractor behind it.
func skipUnlessExtractorGateway() {
	ginkgo.GinkgoHelper()

	if testEnv.Config.GatewayAddress == "" {
		ginkgo.Skip("HTTP gateway address not configured for this environment")
	}

	if testEnv.Config.ExtractorMode == "" {
		ginkgo.Skip("no OASF extractor configured for this environment's gateway")
	}
}
