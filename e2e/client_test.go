// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingv1alpha1 "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/e2e/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
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

	// Create agent object
	agent := &coretypes.Agent{
		Agent: &objectsv1.Agent{
			Name:    "test-agent",
			Version: "v1",
			Skills: []*objectsv1.Skill{
				{
					CategoryName: Ptr("test-category-1"),
					ClassName:    Ptr("test-class-1"),
				},
				{
					CategoryName: Ptr("test-category-2"),
					ClassName:    Ptr("test-class-2"),
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
		},
	}

	// Marshal the Agent struct back to bytes.
	agentData, err := json.Marshal(agent)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create ref
	ref := &coretypes.ObjectRef{
		Digest:      digest.FromBytes(agentData).String(),
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Size:        uint64(len(agentData)),
		Annotations: agent.GetAnnotations(),
	}

	ginkgo.Context("agent push and pull", func() {
		ginkgo.It("should push an agent to store", func() {
			ref, err = c.Push(ctx, ref, bytes.NewReader(agentData))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate valid digest
			_, err = digest.Parse(ref.GetDigest())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should pull an agent from store", func() {
			// Pull the agent object
			reader, err := c.Pull(ctx, ref)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Reader to bytes
			pulledAgentData, err := io.ReadAll(reader)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Compare pushed and pulled agent
			equal, err := compareJSONAgents(agentData, pulledAgentData)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})
	})

	ginkgo.Context("routing publish and list", func() {
		ginkgo.It("should publish an agent", func() {
			err = c.Publish(ctx, &coretypes.ObjectRef{
				Digest: ref.GetDigest(),
				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should list published agent by one label", func() {
			itemsChan, err := c.List(ctx, &routingv1alpha1.ListRequest{
				Labels:  []string{"/skills/test-category-1/test-class-1"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Collect items from the channel
			var items []*routingv1alpha1.ListResponse_Item
			for item := range itemsChan {
				items = append(items, item)
			}

			// Validate the response
			gomega.Expect(items).To(gomega.HaveLen(1))
			for _, item := range items {
				gomega.Expect(item).NotTo(gomega.BeNil())
				gomega.Expect(item.GetRecord().GetDigest()).To(gomega.Equal(ref.GetDigest()))
			}
		})

		ginkgo.It("should list published agent by multiple labels", func() {
			itemsChan, err := c.List(ctx, &routingv1alpha1.ListRequest{
				Labels:  []string{"/skills/test-category-1/test-class-1", "/skills/test-category-2/test-class-2"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Collect items from the channel
			var items []*routingv1alpha1.ListResponse_Item
			for item := range itemsChan {
				items = append(items, item)
			}

			// Validate the response
			gomega.Expect(items).To(gomega.HaveLen(1))
			for _, item := range items {
				gomega.Expect(item).NotTo(gomega.BeNil())
				gomega.Expect(item.GetRecord().GetDigest()).To(gomega.Equal(ref.GetDigest()))
			}
		})

		ginkgo.It("should list published agent by feature and domain labels", func() {
			labels := []string{
				"/domains/domain-1",
				"/features/feature-1",
			}
			for _, label := range labels {
				itemsChan, err := c.List(ctx, &routingv1alpha1.ListRequest{
					Labels:  []string{label},
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Collect items from the channel
				var items []*routingv1alpha1.ListResponse_Item
				for item := range itemsChan {
					items = append(items, item)
				}

				// Validate the response
				gomega.Expect(items).To(gomega.HaveLen(1))
			}
		})
	})

	ginkgo.Context("agent unpublish", func() {
		ginkgo.It("should unpublish an agent", func() {
			err = c.Unpublish(ctx, &coretypes.ObjectRef{
				Digest: ref.GetDigest(),
				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should not find unpublish agent", func() {
			itemsChan, err := c.List(ctx, &routingv1alpha1.ListRequest{
				Labels:  []string{"/skills/test-category-1/test-class-1"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Collect items from the channel
			var items []*routingv1alpha1.ListResponse_Item
			for item := range itemsChan {
				items = append(items, item)
			}

			// Validate the response
			gomega.Expect(items).To(gomega.BeEmpty())
		})
	})

	ginkgo.Context("agent delete", func() {
		ginkgo.It("should delete an agent from store", func() {
			err = c.Delete(ctx, ref)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should not find deleted agent in store", func() {
			reader, err := c.Pull(ctx, ref)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(reader).To(gomega.BeNil())
		})
	})
})
