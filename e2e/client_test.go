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
	defer c.Close()

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
			jsonData: expectedRecordV1JSON,
			expectedSkillLabels: []string{
				"/skills/Natural Language Processing/Text Completion",
				"/skills/Natural Language Processing/Problem Solving",
			},
			expectedDomainLabel:  "/domains/research",
			expectedFeatureLabel: "/features/runtime/framework",
		},
		{
			name:     "V2_AgentRecord_OASF_v0.4.0",
			jsonData: expectedRecordV2JSON,
			expectedSkillLabels: []string{
				"/skills/Natural Language Processing/Text Completion",
				"/skills/Natural Language Processing/Problem Solving",
			},
			expectedDomainLabel:  "/domains/research",
			expectedFeatureLabel: "/features/runtime/framework",
		},
		{
			name:     "V3_Record_OASF_v0.5.0",
			jsonData: expectedRecordV3JSON,
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
				pulledCanonicalData, err := pulledRecord.MarshalOASF()
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
			ginkgo.It("should list published record by multiple labels", func() {
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
			ginkgo.It("should list published record by feature and domain labels", func() {
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

			// TODO: Test if record is deleted from network after TTL

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
