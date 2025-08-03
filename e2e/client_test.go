// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
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

	var err error
	ctx := context.Background()

	// Create a new client
	c, err := client.New(client.WithEnvConfig())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create agent object using new Record structure.
	agent := &objectsv1.Agent{
		Name:    "test-agent",
		Version: "v1",
		Skills: []*objectsv1.Skill{
			{
				CategoryName: utils.Ptr("test-category-1"),
				ClassName:    utils.Ptr("test-class-1"),
			},
			{
				CategoryName: utils.Ptr("test-category-2"),
				ClassName:    utils.Ptr("test-class-2"),
			},
		},
		Extensions: []*objectsv1.Extension{
			{
				Name:    "schema.oasf.agntcy.org/domains/domain-1",
				Version: "v1",
				Data:    nil,
			},
			{
				Name:    "schema.oasf.agntcy.org/features/feature-1",
				Version: "v1",
				Data:    nil,
			},
		},
		Signature: &objectsv1.Signature{},
	}

	// Create Record with the agent.
	record := &corev1.Record{
		Data: &corev1.Record_V1{V1: agent},
	}

	// Marshal the agent for comparison (we'll still need this for testing).
	agentData, err := json.Marshal(agent)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Variable to hold the record reference (will be set by Push).
	var recordRef *corev1.RecordRef

	ginkgo.Context("agent push and pull", func() {
		ginkgo.It("should push an agent to store", func() {
			recordRef, err = c.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate that the returned CID correctly represents the pushed data
			utils.ValidateCIDAgainstData(recordRef.GetCid(), agentData)
		})

		ginkgo.It("should pull an agent from store", func() {
			// Pull the agent object.
			pulledRecord, err := c.Pull(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Extract the agent from the pulled record.
			pulledAgent := pulledRecord.GetV1()
			gomega.Expect(pulledAgent).NotTo(gomega.BeNil())

			// Compare pushed and pulled agent directly.
			equal, err := utils.CompareAgents(agent, pulledAgent)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})
	})

	ginkgo.Context("routing publish and list", func() {
		ginkgo.It("should publish an agent", func() {
			err = c.Publish(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should list published agent by one label", func() {
			itemsChan, err := c.List(ctx, &routingv1.ListRequest{
				LegacyListRequest: &routingv1.LegacyListRequest{
					Labels: []string{"/skills/test-category-1/test-class-1"},
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

		ginkgo.It("should list published agent by multiple labels", func() {
			itemsChan, err := c.List(ctx, &routingv1.ListRequest{
				LegacyListRequest: &routingv1.LegacyListRequest{
					Labels: []string{"/skills/test-category-1/test-class-1", "/skills/test-category-2/test-class-2"},
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

		ginkgo.It("should list published agent by feature and domain labels", func() {
			labels := []string{"/domains/domain-1", "/features/feature-1"}

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
	})

	ginkgo.Context("agent unpublish", func() {
		ginkgo.It("should unpublish an agent", func() {
			err = c.Unpublish(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should not find unpublish agent", func() {
			itemsChan, err := c.List(ctx, &routingv1.ListRequest{
				LegacyListRequest: &routingv1.LegacyListRequest{
					Labels: []string{"/skills/test-category-1/test-class-1"},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Collect items from the channel using utility.
			items := utils.CollectChannelItems(itemsChan)

			// Validate the response.
			gomega.Expect(items).To(gomega.BeEmpty())
		})
	})

	ginkgo.Context("agent delete", func() {
		ginkgo.It("should delete an agent from store", func() {
			err = c.Delete(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should not find deleted agent in store", func() {
			// Add a small delay to ensure delete operation is fully processed
			time.Sleep(100 * time.Millisecond)

			pulledRecord, err := c.Pull(ctx, recordRef)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(pulledRecord).To(gomega.BeNil())
		})
	})
})
