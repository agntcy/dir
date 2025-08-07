// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running client end-to-end tests using a local single node deployment", func() {
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}
	})

	ctx := context.Background()

	// Create a new client
	c, err := client.New(client.WithEnvConfig())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Test cases for each OASF version (reusing same structure as dirctl_test.go)
	testVersions := []struct {
		name                 string
		jsonData             []byte
		expectedSkillLabels  []string
		expectedDomainLabel  string
		expectedFeatureLabel string
	}{
		{
			name:     "V1_Agent_OASF_v0.3.1",
			jsonData: expectedAgentV1JSON,
			expectedSkillLabels: []string{
				"/skills/Natural Language Processing/Text Completion",
				"/skills/Natural Language Processing/Problem Solving",
			},
			expectedDomainLabel:  "/domains/research",
			expectedFeatureLabel: "/features/runtime/framework",
		},
		{
			name:     "V2_AgentRecord_OASF_v0.4.0",
			jsonData: expectedAgentV2JSON,
			expectedSkillLabels: []string{
				"/skills/Natural Language Processing/Text Completion",
				"/skills/Natural Language Processing/Problem Solving",
			},
			expectedDomainLabel:  "/domains/research",
			expectedFeatureLabel: "/features/runtime/framework",
		},
		{
			name:     "V3_Record_OASF_v0.5.0",
			jsonData: expectedAgentV3JSON,
			expectedSkillLabels: []string{
				"/skills/Natural Language Processing/Text Completion",
				"/skills/Natural Language Processing/Problem Solving",
			},
			expectedDomainLabel:  "/domains/research",
			expectedFeatureLabel: "/features/runtime/framework",
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
			record, err = corev1.LoadOASFFromReader(bytes.NewReader(version.jsonData))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Use canonical marshaling for CID validation
			canonicalData, err = record.MarshalOASF()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Step 1: Push
			ginkgo.It("should push an agent to store", func() {
				var err error
				recordRef, err = c.Push(ctx, record)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Validate that the returned CID correctly represents the pushed data using canonical marshaling
				utils.ValidateCIDAgainstData(recordRef.GetCid(), canonicalData)
			})

			// Step 2: Pull (depends on push)
			ginkgo.It("should pull an agent from store", func() {
				// Pull the record object (using recordRef from push)
				pulledRecord, err := c.Pull(ctx, recordRef)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Get canonical data from pulled record for comparison
				pulledCanonicalData, err := pulledRecord.MarshalOASF()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Compare pushed and pulled records using canonical data
				equal, err := utils.CompareOASFRecords(canonicalData, pulledCanonicalData)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(equal).To(gomega.BeTrue(), "Pushed and pulled records should be identical")
			})

			// Step 3: Publish (depends on push)
			ginkgo.It("should publish an agent", func() {
				err := c.Publish(ctx, recordRef)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})

			// Step 4: List by one label (depends on publish)
			ginkgo.It("should list published agent by one label", func() {
				// Use the first skill label from this version's data
				itemsChan, err := c.List(ctx, &routingv1.ListRequest{
					LegacyListRequest: &routingv1.LegacyListRequest{
						Labels: []string{version.expectedSkillLabels[0]},
					},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect items from the channel using utility.
				items := utils.CollectChannelItems(itemsChan)

				// Validate the response.
				gomega.Expect(items).To(gomega.HaveLen(1))
				for _, item := range items {
					gomega.Expect(item).NotTo(gomega.BeNil())
					gomega.Expect(item.GetRef().GetCid()).To(gomega.Equal(recordRef.GetCid()))
				}
			})

			// Step 5: List by multiple labels (depends on publish)
			ginkgo.It("should list published agent by multiple labels", func() {
				// Use all skill labels from this version's data
				itemsChan, err := c.List(ctx, &routingv1.ListRequest{
					LegacyListRequest: &routingv1.LegacyListRequest{
						Labels: version.expectedSkillLabels,
					},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect items from the channel using utility.
				items := utils.CollectChannelItems(itemsChan)

				// Validate the response.
				gomega.Expect(items).To(gomega.HaveLen(1))
				for _, item := range items {
					gomega.Expect(item).NotTo(gomega.BeNil())
					gomega.Expect(item.GetRef().GetCid()).To(gomega.Equal(recordRef.GetCid()))
				}
			})

			// Step 6: List by feature and domain labels (depends on publish)
			ginkgo.It("should list published agent by feature and domain labels", func() {
				// Use extension labels from this version's data
				labels := []string{version.expectedDomainLabel, version.expectedFeatureLabel}

				for _, label := range labels {
					itemsChan, err := c.List(ctx, &routingv1.ListRequest{
						LegacyListRequest: &routingv1.LegacyListRequest{
							Labels: []string{label},
						},
					})
					gomega.Expect(err).NotTo(gomega.HaveOccurred())

					// Collect items from the channel using utility.
					items := utils.CollectChannelItems(itemsChan)

					// Validate the response.
					gomega.Expect(items).To(gomega.HaveLen(1))
					for _, item := range items {
						gomega.Expect(item).NotTo(gomega.BeNil())
						gomega.Expect(item.GetRef().GetCid()).To(gomega.Equal(recordRef.GetCid()))
					}
				}
			})

			// Step 7: Unpublish (depends on publish)
			ginkgo.It("should unpublish an agent", func() {
				err := c.Unpublish(ctx, recordRef)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})

			// Step 8: Verify unpublished agent is not found (depends on unpublish)
			ginkgo.It("should not find unpublished agent", func() {
				// Try to find the agent using the same skill label as before
				itemsChan, err := c.List(ctx, &routingv1.ListRequest{
					LegacyListRequest: &routingv1.LegacyListRequest{
						Labels: []string{version.expectedSkillLabels[0]},
					},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect items from the channel using utility.
				items := utils.CollectChannelItems(itemsChan)

				// Validate the response.
				gomega.Expect(items).To(gomega.BeEmpty())
			})

			// Step 9: Delete (depends on previous steps)
			ginkgo.It("should delete an agent from store", func() {
				err := c.Delete(ctx, recordRef)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})

			// Step 10: Verify deleted agent is not found (depends on delete)
			ginkgo.It("should not find deleted agent in store", func() {
				// Add a small delay to ensure delete operation is fully processed
				time.Sleep(100 * time.Millisecond)

				pulledRecord, err := c.Pull(ctx, recordRef)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(pulledRecord).To(gomega.BeNil())
			})
		})
	}
})
