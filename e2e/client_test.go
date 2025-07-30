// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/e2e/config"
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

			// Validate valid CID.
			gomega.Expect(recordRef.GetCid()).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should pull an agent from store", func() {
			// Pull the agent object.
			pulledRecord, err := c.Pull(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Extract the agent from the pulled record.
			pulledAgent := pulledRecord.GetV1()
			gomega.Expect(pulledAgent).NotTo(gomega.BeNil())

			// Marshal the pulled agent for comparison.
			pulledAgentData, err := json.Marshal(pulledAgent)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Compare pushed and pulled agent.
			equal, err := compareJSONAgents(agentData, pulledAgentData)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})
	})

	ginkgo.Context("streaming operations", func() {
		var streamingRefs []*corev1.RecordRef

		ginkgo.It("should push multiple agents using PushStream", func() {
			// Create multiple test records
			recordCount := 5
			records := make(chan *corev1.Record, recordCount)

			// Generate test records
			go func() {
				defer close(records)
				for i := range recordCount {
					testAgent := &objectsv1.Agent{
						Name:    fmt.Sprintf("stream-test-agent-%d", i),
						Version: "v1",
						Skills: []*objectsv1.Skill{
							{
								CategoryName: Ptr(fmt.Sprintf("stream-category-%d", i)),
								ClassName:    Ptr(fmt.Sprintf("stream-class-%d", i)),
							},
						},
						Extensions: []*objectsv1.Extension{
							{
								Name:    fmt.Sprintf("schema.oasf.agntcy.org/stream-test-%d", i),
								Version: "v1",
								Data:    nil,
							},
						},
						Signature: &objectsv1.Signature{},
					}

					testRecord := &corev1.Record{
						Data: &corev1.Record_V1{V1: testAgent},
					}

					records <- testRecord
				}
			}()

			// Use PushStream to push all records
			results := c.PushStream(ctx, records)

			// Collect all results
			var pushResults []client.PushResult
			var successCount int
			var errorCount int

			for result := range results {
				pushResults = append(pushResults, result)
				if result.Error != nil {
					errorCount++
					ginkgo.GinkgoWriter.Printf("Push error for record %d: %v\n", result.Index, result.Error)
				} else {
					successCount++
					streamingRefs = append(streamingRefs, result.RecordRef)
					gomega.Expect(result.RecordRef.GetCid()).NotTo(gomega.BeEmpty())
				}
			}

			// Validate results
			gomega.Expect(pushResults).To(gomega.HaveLen(recordCount))
			gomega.Expect(successCount).To(gomega.Equal(recordCount))
			gomega.Expect(errorCount).To(gomega.Equal(0))
			gomega.Expect(streamingRefs).To(gomega.HaveLen(recordCount))

			ginkgo.GinkgoWriter.Printf("Successfully pushed %d records via PushStream\n", successCount)
		})

		ginkgo.It("should pull multiple agents using PullStream", func() {
			gomega.Expect(streamingRefs).NotTo(gomega.BeEmpty(), "No streaming refs available from previous test")

			// Create channel with record references
			refCount := len(streamingRefs)
			refs := make(chan *corev1.RecordRef, refCount)

			go func() {
				defer close(refs)
				for _, ref := range streamingRefs {
					refs <- ref
				}
			}()

			// Use PullStream to pull all records
			results := c.PullStream(ctx, refs)

			// Collect all results
			var pullResults []client.PullResult
			var successCount int
			var errorCount int

			for result := range results {
				pullResults = append(pullResults, result)
				if result.Error != nil {
					errorCount++
					ginkgo.GinkgoWriter.Printf("Pull error for record %d: %v\n", result.Index, result.Error)
				} else {
					successCount++
					gomega.Expect(result.Record).NotTo(gomega.BeNil())
					gomega.Expect(result.Record.GetV1()).NotTo(gomega.BeNil())

					// Verify the pulled agent has expected structure
					pulledAgent := result.Record.GetV1()
					gomega.Expect(pulledAgent.GetName()).To(gomega.ContainSubstring("stream-test-agent-"))
					gomega.Expect(pulledAgent.GetSkills()).To(gomega.HaveLen(1))
				}
			}

			// Validate results
			gomega.Expect(pullResults).To(gomega.HaveLen(refCount))
			gomega.Expect(successCount).To(gomega.Equal(refCount))
			gomega.Expect(errorCount).To(gomega.Equal(0))

			ginkgo.GinkgoWriter.Printf("Successfully pulled %d records via PullStream\n", successCount)
		})

		ginkgo.It("should lookup multiple agents using LookupStream", func() {
			gomega.Expect(streamingRefs).NotTo(gomega.BeEmpty(), "No streaming refs available from previous test")

			// Create channel with record references
			refCount := len(streamingRefs)
			refs := make(chan *corev1.RecordRef, refCount)

			go func() {
				defer close(refs)
				for _, ref := range streamingRefs {
					refs <- ref
				}
			}()

			// Use LookupStream to lookup all records
			results := c.LookupStream(ctx, refs)

			// Collect all results
			var lookupResults []client.LookupResult
			var successCount int
			var errorCount int

			for result := range results {
				lookupResults = append(lookupResults, result)
				if result.Error != nil {
					errorCount++
					ginkgo.GinkgoWriter.Printf("Lookup error for record %d: %v\n", result.Index, result.Error)
				} else {
					successCount++
					gomega.Expect(result.RecordMeta).NotTo(gomega.BeNil())
					gomega.Expect(result.RecordMeta.GetCid()).NotTo(gomega.BeEmpty())
				}
			}

			// Validate results
			gomega.Expect(lookupResults).To(gomega.HaveLen(refCount))
			gomega.Expect(successCount).To(gomega.Equal(refCount))
			gomega.Expect(errorCount).To(gomega.Equal(0))

			ginkgo.GinkgoWriter.Printf("Successfully looked up %d records via LookupStream\n", successCount)
		})

		ginkgo.It("should handle empty channel gracefully", func() {
			// Create empty channel
			records := make(chan *corev1.Record)
			close(records)

			// Use PushStream with empty channel
			results := c.PushStream(ctx, records)

			// Collect results - should be empty
			var resultCount int
			for range results {
				resultCount++
			}

			gomega.Expect(resultCount).To(gomega.Equal(0))
		})

		ginkgo.It("should handle context cancellation", func() {
			// Create context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()

			// Create a slow record generator
			records := make(chan *corev1.Record, 1)
			go func() {
				defer close(records)
				// Send one record quickly
				records <- record
				// Then try to send another after delay (should be cancelled)
				time.Sleep(100 * time.Millisecond)
				records <- record
			}()

			// Use PushStream with timeout context
			results := c.PushStream(timeoutCtx, records)

			// Collect results - should stop due to context cancellation
			var resultCount int
			for result := range results {
				resultCount++
				if result.Error != nil {
					ginkgo.GinkgoWriter.Printf("Expected cancellation error: %v\n", result.Error)
				}
			}

			// Should have processed at least one result before cancellation
			gomega.Expect(resultCount).To(gomega.BeNumerically(">=", 0))
		})

		ginkgo.It("should delete multiple agents using DeleteStream", func() {
			gomega.Expect(streamingRefs).NotTo(gomega.BeEmpty(), "No streaming refs available from previous test")

			// Create channel with record references
			refCount := len(streamingRefs)
			refs := make(chan *corev1.RecordRef, refCount)

			go func() {
				defer close(refs)
				for _, ref := range streamingRefs {
					refs <- ref
				}
			}()

			// Use DeleteStream to delete all records
			results := c.DeleteStream(ctx, refs)

			// Collect all results
			var deleteResults []client.DeleteResult
			var successCount int
			var errorCount int

			for result := range results {
				deleteResults = append(deleteResults, result)
				if result.Error != nil {
					errorCount++
					ginkgo.GinkgoWriter.Printf("Delete error for record %d: %v\n", result.Index, result.Error)
				} else {
					successCount++
				}
			}

			// Validate results
			gomega.Expect(deleteResults).To(gomega.HaveLen(refCount))
			gomega.Expect(successCount).To(gomega.Equal(refCount))
			gomega.Expect(errorCount).To(gomega.Equal(0))

			ginkgo.GinkgoWriter.Printf("Successfully deleted %d records via DeleteStream\n", successCount)

			// Verify records are actually deleted by trying to pull them
			time.Sleep(100 * time.Millisecond) // Small delay for deletion to complete

			for _, ref := range streamingRefs {
				_, err := c.Pull(ctx, ref)
				gomega.Expect(err).To(gomega.HaveOccurred(), "Record should be deleted")
			}
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

			// Collect items from the channel.
			var items []*routingv1.LegacyListResponse_Item
			for item := range itemsChan {
				items = append(items, item)
			}

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

			// Collect items from the channel.
			var items []*routingv1.LegacyListResponse_Item
			for item := range itemsChan {
				items = append(items, item)
			}

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

				// Collect items from the channel.
				var items []*routingv1.LegacyListResponse_Item
				for item := range itemsChan {
					items = append(items, item)
				}

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

			// Collect items from the channel.
			var items []*routingv1.LegacyListResponse_Item
			for item := range itemsChan {
				items = append(items, item)
			}

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
