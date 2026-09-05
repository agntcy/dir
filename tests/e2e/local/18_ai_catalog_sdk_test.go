// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agntcy/ai-catalog-go/catalog"
	"github.com/agntcy/ai-catalog-go/provider"
	"github.com/agntcy/ai-catalog-go/trust"
	"github.com/agntcy/ai-catalog-go/validate"
	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	catalogSDKSpecVersion = "1.0"
	a2aMediaType          = "application/a2a-agent-card+json"
	mcpMediaType          = "application/mcp-server-card+json"
	skillMediaType        = "application/agent-skills+md"
)

var _ = ginkgo.Describe("AI Catalog Go SDK conformance", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	var (
		tempDir      string
		recordCID    string
		skillCID     string
		directoryCID string
	)

	ginkgo.Context("HTTP Catalog endpoints", ginkgo.Ordered, ginkgo.Serial, func() {
		ginkgo.BeforeAll(func() {
			utils.ResetCLIState()

			if testEnv.Config.GatewayAddress == "" {
				ginkgo.Skip("HTTP gateway address not configured for this environment")
			}

			var err error
			tempDir, err = os.MkdirTemp("", "ai-catalog-sdk-e2e-*")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = pushCatalogFixture(filepath.Join(tempDir, "record-100.json"), testdata.ExpectedRecordV100JSON)
			skillCID = pushCatalogFixture(filepath.Join(tempDir, "skill-record.json"), testdata.CatalogSkillRecordJSON)

			directoryCID = pushCatalogFixture(filepath.Join(tempDir, "directory-record.json"), testdata.DirectoryRecordJSON)
		})

		ginkgo.AfterAll(func() {
			for _, cid := range []string{recordCID, skillCID, directoryCID} {
				if cid != "" {
					_ = testEnv.CLI.Delete(cid).ShouldSucceed()
				}
			}

			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("loads the well-known document and preserves its collections", func(ctx ginkgo.SpecContext) {
			endpoint := strings.TrimRight(testEnv.Config.GatewayAddress, "/") + catalog.WellKnownPath
			status, headers, body := getCatalogHTTP(ctx, http.MethodGet, endpoint, nil)

			gomega.Expect(status).To(gomega.Equal(http.StatusOK))
			gomega.Expect(headers.Get("Content-Type")).To(gomega.HavePrefix("application/json"))

			var wire catalogv1.WellKnownCatalog
			gomega.Expect(protojson.Unmarshal(body, &wire)).To(gomega.Succeed())
			gomega.Expect(wire.GetSpecVersion()).To(gomega.Equal(catalogSDKSpecVersion))
			gomega.Expect(wire.GetHost()).NotTo(gomega.BeNil())
			gomega.Expect(wire.GetHost().GetDisplayName()).NotTo(gomega.BeEmpty())
			gomega.Expect(wire.GetHost().GetIdentifier()).NotTo(gomega.BeEmpty())
			gomega.Expect(wire.GetEntries()).To(gomega.BeEmpty())
			gomega.Expect(wire.GetCollections()).To(gomega.HaveLen(4))

			expectedCollections := map[string]string{
				"A2A Agents":          a2aMediaType,
				"MCP Servers":         mcpMediaType,
				"Agent Skills":        skillMediaType,
				"Agent Skill Bundles": "application/agent-skills+gzip",
			}
			seenCollections := make(map[string]struct{}, len(wire.GetCollections()))
			for _, collection := range wire.GetCollections() {
				mediaType, ok := expectedCollections[collection.GetDisplayName()]
				gomega.Expect(ok).To(gomega.BeTrue(), "unexpected collection %q", collection.GetDisplayName())
				_, duplicate := seenCollections[collection.GetDisplayName()]
				gomega.Expect(duplicate).To(gomega.BeFalse(), "duplicate collection %q", collection.GetDisplayName())
				seenCollections[collection.GetDisplayName()] = struct{}{}
				gomega.Expect(collection.GetMediaType()).To(gomega.Equal(mediaType))

				collectionURL, err := url.Parse(collection.GetUrl())
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(collectionURL.Path).To(gomega.Equal("/v1/agents"))
				gomega.Expect(collectionURL.Query().Get("filter")).To(gomega.Equal("type=" + mediaType))
			}
			gomega.Expect(seenCollections).To(gomega.Equal(map[string]struct{}{
				"A2A Agents":          {},
				"MCP Servers":         {},
				"Agent Skills":        {},
				"Agent Skill Bundles": {},
			}))

			source, err := provider.Web(ctx, endpoint)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			doc, err := source.Load(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(doc.SpecVersion).To(gomega.Equal(catalogSDKSpecVersion))
			gomega.Expect(doc.Entries).To(gomega.BeEmpty())

			rawSource, ok := source.(catalog.RawSource)
			gomega.Expect(ok).To(gomega.BeTrue())
			raw, err := rawSource.Raw(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(json.Valid(raw)).To(gomega.BeTrue())

			trustManifest := wire.GetHost().GetTrustManifest()
			gomega.Expect(trustManifest).NotTo(gomega.BeNil())

			trustReport := trust.AnalyzeCatalog(doc)
			gomega.Expect(trustReport.Host).NotTo(gomega.BeNil())
			gomega.Expect(trustReport.Host.Identity).To(gomega.Equal(trustManifest.GetIdentity()))
		})

		ginkgo.It("validates MCP and A2A projections with the SDK", func(ctx ginkgo.SpecContext) {
			var response catalogv1.ListAgentsResponse
			gomega.Eventually(func(g gomega.Gomega) {
				status, _, body := getCatalogHTTP(ctx, http.MethodGet, catalogEndpoint("/v1/agents"), nil)
				g.Expect(status).To(gomega.Equal(http.StatusOK))
				g.Expect(protojson.Unmarshal(body, &response)).To(gomega.Succeed())
				g.Expect(findCatalogEntry(response.GetResults(), recordCID)).NotTo(gomega.BeNil())
			}).WithContext(ctx).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(gomega.Succeed())

			wireEntry := findCatalogEntry(response.GetResults(), recordCID)
			gomega.Expect(wireEntry.GetMediaType()).To(gomega.Equal(catalog.MediaTypeCatalog))
			gomega.Expect(wireEntry.GetData()).NotTo(gomega.BeNil())
			gomega.Expect(wireEntry.GetTags()).NotTo(gomega.BeEmpty())

			entry, err := adaptCatalogEntry(wireEntry, wireEntry.GetIdentifier())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			doc := &catalog.AICatalog{
				SpecVersion: catalogSDKSpecVersion,
				Entries:     []catalog.CatalogEntry{entry},
			}
			result := validate.Validate(doc)
			gomega.Expect(result.IsValid).To(gomega.BeTrue(), "SDK validation errors: %+v", result.Errors)
			gomega.Expect(doc.GetByType(catalog.MediaTypeCatalog)).To(gomega.HaveLen(1))
			gomega.Expect(doc.Search("burger_seller_agent")).To(gomega.HaveLen(1))

			nested, err := catalog.Parse(entry.Data)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(nested.Entries).To(gomega.HaveLen(2))
			gomega.Expect(nested.GetByType(a2aMediaType)).To(gomega.HaveLen(1))
			gomega.Expect(nested.GetByType(mcpMediaType)).To(gomega.HaveLen(1))
		})

		ginkgo.It("validates the Agent Skill projection with the SDK", func(ctx ginkgo.SpecContext) {
			query := url.Values{"filter": {"displayName=code-review"}}.Encode()
			var response catalogv1.ListAgentsResponse
			gomega.Eventually(func(g gomega.Gomega) {
				status, _, body := getCatalogHTTP(ctx, http.MethodGet, catalogEndpoint("/v1/agents?"+query), nil)
				g.Expect(status).To(gomega.Equal(http.StatusOK))
				g.Expect(protojson.Unmarshal(body, &response)).To(gomega.Succeed())
				g.Expect(findCatalogEntry(response.GetResults(), skillCID)).NotTo(gomega.BeNil())
			}).WithContext(ctx).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(gomega.Succeed())

			wireEntry := findCatalogEntry(response.GetResults(), skillCID)
			gomega.Expect(wireEntry.GetMediaType()).To(gomega.Equal(skillMediaType))
			gomega.Expect(wireEntry.GetData()).NotTo(gomega.BeNil())

			entry, err := adaptCatalogEntry(wireEntry, wireEntry.GetIdentifier())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			doc := &catalog.AICatalog{
				SpecVersion: catalogSDKSpecVersion,
				Entries:     []catalog.CatalogEntry{entry},
			}
			result := validate.Validate(doc)
			gomega.Expect(result.IsValid).To(gomega.BeTrue(), "SDK validation errors: %+v", result.Errors)
			gomega.Expect(doc.GetByType(skillMediaType)).To(gomega.HaveLen(1))
		})

		ginkgo.It("returns the same SDK-compatible entry from the detail endpoint", func(ctx ginkgo.SpecContext) {
			endpoint := catalogEndpoint("/v1/agents/" + url.PathEscape(recordCID))
			status, _, body := getCatalogHTTP(ctx, http.MethodGet, endpoint, nil)
			gomega.Expect(status).To(gomega.Equal(http.StatusOK))

			var wireEntry catalogv1.CatalogEntry
			gomega.Expect(protojson.Unmarshal(body, &wireEntry)).To(gomega.Succeed())
			gomega.Expect(wireEntry.GetIdentifier()).To(gomega.ContainSubstring(recordCID))

			entry, err := adaptCatalogEntry(&wireEntry, wireEntry.GetIdentifier())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			result := validate.Validate(&catalog.AICatalog{
				SpecVersion: catalogSDKSpecVersion,
				Entries:     []catalog.CatalogEntry{entry},
			})
			gomega.Expect(result.IsValid).To(gomega.BeTrue(), "SDK validation errors: %+v", result.Errors)
		})

		ginkgo.It("exposes identity and signature metadata for SDK trust analysis", func(ctx ginkgo.SpecContext) {
			if _, err := exec.LookPath("cosign"); err != nil {
				ginkgo.Skip("cosign is not installed — install cosign to enable signature E2E tests")
			}

			paths := setupSignTestPaths()
			defer os.RemoveAll(paths.tempDir)

			previousPassword, hadPassword := os.LookupEnv("COSIGN_PASSWORD")
			gomega.Expect(os.Setenv("COSIGN_PASSWORD", "testpassword")).To(gomega.Succeed())
			defer func() {
				if hadPassword {
					_ = os.Setenv("COSIGN_PASSWORD", previousPassword)
				} else {
					_ = os.Unsetenv("COSIGN_PASSWORD")
				}
			}()

			utils.GenerateCosignKeyPair(paths.tempDir, "testpassword")
			testEnv.CLI.Sign(recordCID, paths.privateKey).ShouldSucceed()

			var response catalogv1.ListAgentsResponse
			gomega.Eventually(func(g gomega.Gomega) {
				status, _, body := getCatalogHTTP(ctx, http.MethodGet, catalogEndpoint("/v1/agents"), nil)
				g.Expect(status).To(gomega.Equal(http.StatusOK))
				g.Expect(protojson.Unmarshal(body, &response)).To(gomega.Succeed())
				entry := findCatalogEntry(response.GetResults(), recordCID)
				g.Expect(entry).NotTo(gomega.BeNil())
				g.Expect(entry.GetTrustManifest()).NotTo(gomega.BeNil())
				g.Expect(entry.GetTrustManifest().GetIdentity()).NotTo(gomega.BeEmpty())
				g.Expect(entry.GetTrustManifest().GetSignature()).NotTo(gomega.BeEmpty())
			}).WithContext(ctx).WithTimeout(45 * time.Second).WithPolling(2 * time.Second).Should(gomega.Succeed())

			wireEntry := findCatalogEntry(response.GetResults(), recordCID)
			entry, err := adaptCatalogEntry(wireEntry, wireEntry.GetIdentifier())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			doc := &catalog.AICatalog{
				SpecVersion: catalogSDKSpecVersion,
				Entries:     []catalog.CatalogEntry{entry},
			}
			trustReport := trust.AnalyzeCatalog(doc)
			gomega.Expect(trustReport.Entries).NotTo(gomega.BeEmpty())
			gomega.Expect(trustReport.Entries[0].HasSignature).To(gomega.BeTrue())
			gomega.Expect(trustReport.Entries[0].Identity).To(gomega.Equal(wireEntry.GetTrustManifest().GetIdentity()))
		})

		ginkgo.It("parses natural-language search results with the SDK", func(ctx ginkgo.SpecContext) {
			if !extractorManifestExists() {
				ginkgo.Skip("OASF extractor not provisioned — run `dirctl init` to enable natural-language search tests")
			}

			requestBody, err := json.Marshal(map[string]any{
				"query":    "I need a MCP server to connect to a agntcy directory",
				"pageSize": 20,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var response catalogv1.SearchAgentsResponse
			gomega.Eventually(func(g gomega.Gomega) {
				status, _, body := getCatalogHTTP(ctx, http.MethodPost, catalogEndpoint("/v1/search"), requestBody)
				g.Expect(status).To(gomega.Equal(http.StatusOK))
				g.Expect(protojson.Unmarshal(body, &response)).To(gomega.Succeed())
				g.Expect(findCatalogEntry(response.GetResults(), directoryCID)).NotTo(gomega.BeNil())
			}).WithContext(ctx).WithTimeout(45 * time.Second).WithPolling(2 * time.Second).Should(gomega.Succeed())

			doc := &catalog.AICatalog{SpecVersion: catalogSDKSpecVersion}
			for _, wireEntry := range response.GetResults() {
				entry, err := adaptCatalogEntry(wireEntry, wireEntry.GetIdentifier())
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				doc.Entries = append(doc.Entries, entry)
			}
			directoryEntry := findCatalogEntry(response.GetResults(), directoryCID)
			adaptedEntry, found := doc.GetByID(directoryEntry.GetIdentifier())
			gomega.Expect(found).To(gomega.BeTrue())
			gomega.Expect(adaptedEntry).NotTo(gomega.BeNil())
			gomega.Expect(doc.Search("directory")).NotTo(gomega.BeEmpty())
		})
	})
})

func pushCatalogFixture(path string, data []byte) string {
	gomega.Expect(os.WriteFile(path, data, 0o600)).To(gomega.Succeed())
	cid := testEnv.CLI.Push(path).WithArgs("--output", "raw").ShouldSucceed()
	gomega.Expect(cid).NotTo(gomega.BeEmpty())

	return cid
}

func catalogEndpoint(path string) string {
	return strings.TrimRight(testEnv.Config.GatewayAddress, "/") + path
}

func getCatalogHTTP(ctx context.Context, method, endpoint string, body []byte) (int, http.Header, []byte) {
	var reader io.Reader
	if body != nil {
		reader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer func() { _ = resp.Body.Close() }()

	responseBody, err := io.ReadAll(resp.Body)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return resp.StatusCode, resp.Header, responseBody
}

func findCatalogEntry(entries []*catalogv1.CatalogEntry, cid string) *catalogv1.CatalogEntry {
	for _, entry := range entries {
		if strings.Contains(entry.GetIdentifier(), cid) {
			return entry
		}
	}

	return nil
}

func extractorManifestExists() bool {
	home, err := os.UserHomeDir()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	_, err = os.Stat(filepath.Join(home, ".agntcy", "oasf-sdk", "extractor", "manifest.json"))
	return err == nil
}

func adaptCatalog(catalogWire *catalogv1.AICatalog, parentID string) (catalog.AICatalog, error) {
	adapted := catalog.AICatalog{SpecVersion: catalogWire.GetSpecVersion()}
	for index, wireEntry := range catalogWire.GetEntries() {
		identifier := wireEntry.GetIdentifier()
		if identifier == "" {
			identifier = fmt.Sprintf("%s:entry-%d", parentID, index)
		}

		entry, err := adaptCatalogEntry(wireEntry, identifier)
		if err != nil {
			return catalog.AICatalog{}, err
		}
		adapted.Entries = append(adapted.Entries, entry)
	}

	return adapted, nil
}

func adaptCatalogEntry(wireEntry *catalogv1.CatalogEntry, fallbackID string) (catalog.CatalogEntry, error) {
	if wireEntry == nil {
		return catalog.CatalogEntry{}, fmt.Errorf("catalog entry is nil")
	}

	entry := catalog.CatalogEntry{
		Identifier:    wireEntry.GetIdentifier(),
		DisplayName:   wireEntry.GetDisplayName(),
		Type:          wireEntry.GetMediaType(),
		Version:       wireEntry.GetVersion(),
		Description:   wireEntry.GetDescription(),
		Tags:          append([]string(nil), wireEntry.GetTags()...),
		UpdatedAt:     wireEntry.GetUpdatedAt(),
		TrustManifest: adaptTrustManifest(wireEntry.GetTrustManifest()),
	}
	if entry.Identifier == "" {
		entry.Identifier = fallbackID
	}
	if publisher := wireEntry.GetPublisher(); publisher != nil {
		entry.Publisher = &catalog.Publisher{
			Identifier:   publisher.GetIdentifier(),
			DisplayName:  publisher.GetDisplayName(),
			IdentityType: publisher.GetIdentityType(),
		}
	}

	if wireEntry.GetUrl() != "" {
		entry.URL = wireEntry.GetUrl()
		return entry, nil
	}
	if wireEntry.GetData() == nil {
		return entry, nil
	}

	data, err := protojson.Marshal(wireEntry.GetData())
	if err != nil {
		return catalog.CatalogEntry{}, fmt.Errorf("marshal %q data: %w", entry.Identifier, err)
	}
	if entry.Type != catalog.MediaTypeCatalog {
		entry.Data = data
		return entry, nil
	}

	var nestedWire catalogv1.AICatalog
	if err := protojson.Unmarshal(data, &nestedWire); err != nil {
		return catalog.CatalogEntry{}, fmt.Errorf("parse nested catalog %q: %w", entry.Identifier, err)
	}
	nested, err := adaptCatalog(&nestedWire, entry.Identifier)
	if err != nil {
		return catalog.CatalogEntry{}, fmt.Errorf("adapt nested catalog %q: %w", entry.Identifier, err)
	}
	entry.Data, err = json.Marshal(nested)
	if err != nil {
		return catalog.CatalogEntry{}, fmt.Errorf("marshal nested catalog %q: %w", entry.Identifier, err)
	}

	return entry, nil
}

func adaptTrustManifest(manifest *catalogv1.TrustManifest) *catalog.TrustManifest {
	if manifest == nil {
		return nil
	}

	adapted := &catalog.TrustManifest{
		Identity:     manifest.GetIdentity(),
		IdentityType: manifest.GetIdentityType(),
		Signature:    manifest.GetSignature(),
	}
	if schema := manifest.GetTrustSchema(); schema != nil {
		adapted.TrustSchema = &catalog.TrustSchema{
			Identifier:          schema.GetIdentifier(),
			Version:             schema.GetVersion(),
			GovernanceURI:       schema.GetGovernanceUri(),
			VerificationMethods: append([]string(nil), schema.GetVerificationMethods()...),
		}
	}
	for _, attestation := range manifest.GetAttestations() {
		adapted.Attestations = append(adapted.Attestations, catalog.Attestation{
			Type:        attestation.GetType(),
			URI:         attestation.GetUri(),
			Digest:      attestation.GetDigest(),
			Description: attestation.GetDescription(),
		})
	}
	for _, provenance := range manifest.GetProvenance() {
		adapted.Provenance = append(adapted.Provenance, catalog.ProvenanceLink{
			Relation:     provenance.GetRelation(),
			SourceID:     provenance.GetSourceId(),
			SourceDigest: provenance.GetSourceDigest(),
			RegistryURI:  provenance.GetRegistryUri(),
			StatementURI: provenance.GetStatementUri(),
			SignatureRef: provenance.GetSignatureRef(),
		})
	}

	return adapted
}
