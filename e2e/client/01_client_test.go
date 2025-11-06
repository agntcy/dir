// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// convertLabelsToRecordQueries converts legacy label format to RecordQuery format for e2e tests.
func convertLabelsToClientRecordQueries(labels []string) []*routingv1.RecordQuery {
	var queries []*routingv1.RecordQuery

	for _, label := range labels {
		switch {
		case strings.HasPrefix(label, "/skills/"):
			skillName := strings.TrimPrefix(label, "/skills/")
			queries = append(queries, &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: skillName,
			})
		case strings.HasPrefix(label, "/domains/"):
			domainName := strings.TrimPrefix(label, "/domains/")
			_ = domainName
			// Note: domains might need to be mapped to locator or handled differently
			// For now, skip domains as they're not in the current RecordQueryType
		case strings.HasPrefix(label, "/modules/"):
			moduleName := strings.TrimPrefix(label, "/modules/")
			queries = append(queries, &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE,
				Value: moduleName,
			})
		case strings.HasPrefix(label, "/locators/"):
			locatorType := strings.TrimPrefix(label, "/locators/")
			queries = append(queries, &routingv1.RecordQuery{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR,
				Value: locatorType,
			})
		}
	}

	return queries
}

var _ = ginkgo.Describe("Running client end-to-end tests using a local single node deployment", ginkgo.Ordered, ginkgo.Serial, func() {
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}
	})

	var c *client.Client
	var ctx context.Context

	ginkgo.BeforeAll(func() {
		ctx = context.Background()

		// Create a new client
		var err error
		c, err = client.New(ctx, client.WithEnvConfig())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		if c != nil {
			c.Close()
		}
	})

	// Test cases for each OASF version (matches testdata files)
	testVersions := []struct {
		name                string
		jsonData            []byte
		expectedSkillLabels []string
		expectedDomainLabel string
		expectedModuleLabel string
	}{
		{
			name:     "Record_031_Agent",
			jsonData: testdata.ExpectedRecordV031JSON,
			expectedSkillLabels: []string{
				"/skills/Natural Language Processing/Text Completion",
				"/skills/Natural Language Processing/Problem Solving",
			},
			expectedModuleLabel: "/modules/runtime/language", // From record_v031.json extensions (schema prefix stripped)
		},
		{
			name:     "Record_070_Agent",
			jsonData: testdata.ExpectedRecordV070JSON,
			expectedSkillLabels: []string{
				"/skills/natural_language_processing/natural_language_generation/text_completion",
				"/skills/natural_language_processing/analytical_reasoning/problem_solving",
			},
			expectedDomainLabel: "/domains/life_science/biotechnology",
			expectedModuleLabel: "/modules/runtime/language", // From record_070.json modules
		},
		{
			name:     "Record_080_Agent",
			jsonData: testdata.ExpectedRecordV080JSON,
			expectedSkillLabels: []string{
				"/skills/natural_language_processing/natural_language_generation/text_completion",
				"/skills/natural_language_processing/analytical_reasoning/problem_solving",
			},
			expectedDomainLabel: "/domains/life_science/biotechnology",
			expectedModuleLabel: "/modules/runtime/language", // From record_080.json modules
		},
	}

	// Test each OASF version dynamically
	for _, v := range testVersions {
		version := v // Capture loop variable by value to avoid closure issues
		ginkgo.Context(version.name, ginkgo.Ordered, ginkgo.Serial, func() {
			var record *corev1.Record
			var canonicalData []byte
			var recordRef *corev1.RecordRef // Shared across the business flow

			// Load the record once per version context (inline initialization)
			var err error
			record, err = corev1.UnmarshalRecord(version.jsonData)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Use canonical marshaling for CID validation
			canonicalData, err = record.Marshal()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Step 1: Push
			ginkgo.It("should push a record to store", func() {
				var err error
				recordRef, err = c.Push(ctx, record)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Validate that the returned CID correctly represents the pushed data using canonical marshaling
				utils.ValidateCIDAgainstData(recordRef.GetCid(), canonicalData)
			})

			// Step 2: Pull (depends on push)
			ginkgo.It("should pull a record from store", func() {
				// Pull the record object (using recordRef from push)
				pulledRecord, err := c.Pull(ctx, recordRef)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Get canonical data from pulled record for comparison
				pulledCanonicalData, err := pulledRecord.Marshal()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Compare pushed and pulled records using canonical data
				equal, err := utils.CompareOASFRecords(canonicalData, pulledCanonicalData)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(equal).To(gomega.BeTrue(), "Pushed and pulled records should be identical")
			})

			// Step 3: Publish (depends on push)
			ginkgo.It("should publish a record", func() {
				err := c.Publish(ctx, &routingv1.PublishRequest{
					Request: &routingv1.PublishRequest_RecordRefs{
						RecordRefs: &routingv1.RecordRefs{
							Refs: []*corev1.RecordRef{recordRef},
						},
					},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Wait at least 10 seconds to ensure the record is published.
				time.Sleep(15 * time.Second)
			})

			// Step 4: List by one label (depends on publish)
			ginkgo.It("should list published record by one label", func() {
				// Convert skill label to RecordQuery
				queries := convertLabelsToClientRecordQueries([]string{version.expectedSkillLabels[0]})

				itemsChan, err := c.List(ctx, &routingv1.ListRequest{
					Queries: queries,
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect items from the channel using utility.
				items := utils.CollectListItems(itemsChan)

				// Validate the response.
				gomega.Expect(items).To(gomega.HaveLen(1))
				for _, item := range items {
					gomega.Expect(item).NotTo(gomega.BeNil())
					gomega.Expect(item.GetRecordRef().GetCid()).To(gomega.Equal(recordRef.GetCid()))
				}
			})

			// Step 5: List by multiple labels (depends on publish)
			ginkgo.It("should list published record by multiple labels", func() {
				// Convert all skill labels to RecordQueries
				queries := convertLabelsToClientRecordQueries(version.expectedSkillLabels)

				itemsChan, err := c.List(ctx, &routingv1.ListRequest{
					Queries: queries,
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect items from the channel using utility.
				items := utils.CollectListItems(itemsChan)

				// Validate the response.
				gomega.Expect(items).To(gomega.HaveLen(1))
				for _, item := range items {
					gomega.Expect(item).NotTo(gomega.BeNil())
					gomega.Expect(item.GetRecordRef().GetCid()).To(gomega.Equal(recordRef.GetCid()))
				}
			})

			// Step 6: List by module and domain labels (depends on publish)
			ginkgo.It("should list published record by module and domain labels", func() {
				// ✅ Domain and module queries are now supported in RecordQuery API!
				// Test domain query
				if version.expectedDomainLabel != "" {
					domainQuery := &routingv1.RecordQuery{
						Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
						Value: "life_science/biotechnology", // From record_070.json domains
					}
					domainItemsChan, err := c.List(ctx, &routingv1.ListRequest{
						Queries: []*routingv1.RecordQuery{domainQuery},
						Limit:   utils.Ptr[uint32](10),
					})
					gomega.Expect(err).NotTo(gomega.HaveOccurred())

					domainResults := utils.CollectListItems(domainItemsChan)
					gomega.Expect(domainResults).ToNot(gomega.BeEmpty(), "Should find record with domain query")
					gomega.Expect(domainResults[0].GetRecordRef().GetCid()).To(gomega.Equal(recordRef.GetCid()))
				}

				// Test module query using the expected module label from test data
				moduleQueries := convertLabelsToClientRecordQueries([]string{version.expectedModuleLabel})
				if len(moduleQueries) > 0 {
					moduleItemsChan, err := c.List(ctx, &routingv1.ListRequest{
						Queries: moduleQueries,
						Limit:   utils.Ptr[uint32](10),
					})
					gomega.Expect(err).NotTo(gomega.HaveOccurred())

					moduleResults := utils.CollectListItems(moduleItemsChan)
					gomega.Expect(moduleResults).ToNot(gomega.BeEmpty(), "Should find record with module query")
					gomega.Expect(moduleResults[0].GetRecordRef().GetCid()).To(gomega.Equal(recordRef.GetCid()))
				}

				ginkgo.GinkgoWriter.Printf("✅ SUCCESS: Queries working correctly")
			})

			// Step 7: Search routing for remote records (depends on publish)
			ginkgo.It("should search routing for remote records", func() {
				// Convert skill labels to RecordQuery format
				queries := convertLabelsToClientRecordQueries([]string{version.expectedSkillLabels[0]})

				searchChan, err := c.SearchRouting(ctx, &routingv1.SearchRequest{
					Queries:       queries,
					Limit:         utils.Ptr[uint32](10),
					MinMatchScore: utils.Ptr[uint32](1),
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect search results using utility
				results := utils.CollectSearchItems(searchChan)

				// For single-peer testing, we should get an empty slice (no remote records)
				// This test validates the SearchRouting method works without errors
				// In multi-peer e2e tests, we'll test actual remote discovery
				gomega.Expect(results).To(gomega.BeEmpty()) // Should be empty slice in local mode
			})

			// Step 8: Unpublish (depends on publish)
			ginkgo.It("should unpublish a record", func() {
				err := c.Unpublish(ctx, &routingv1.UnpublishRequest{
					Request: &routingv1.UnpublishRequest_RecordRefs{
						RecordRefs: &routingv1.RecordRefs{
							Refs: []*corev1.RecordRef{recordRef},
						},
					},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})

			// Step 8: Verify unpublished record is not found (depends on unpublish)
			ginkgo.It("should not find unpublished record", func() {
				// Convert skill label to RecordQuery
				queries := convertLabelsToClientRecordQueries([]string{version.expectedSkillLabels[0]})

				itemsChan, err := c.List(ctx, &routingv1.ListRequest{
					Queries: queries,
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect items from the channel using utility.
				items := utils.CollectListItems(itemsChan)

				// Validate the response.
				gomega.Expect(items).To(gomega.BeEmpty())
			})

			// Step 9: Delete (depends on previous steps)
			ginkgo.It("should delete a record from store", func() {
				err := c.Delete(ctx, recordRef)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})

			// Step 10: Verify deleted record is not found (depends on delete)
			ginkgo.It("should not find deleted record in store", func() {
				// Add a small delay to ensure delete operation is fully processed
				time.Sleep(100 * time.Millisecond)

				pulledRecord, err := c.Pull(ctx, recordRef)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(pulledRecord).To(gomega.BeNil())
			})
		})
	}
})
