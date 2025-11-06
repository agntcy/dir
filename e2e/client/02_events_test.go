// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	eventsv1 "github.com/agntcy/dir/api/events/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/client/streaming"
	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// receiveEvent is a helper that receives a single event from a StreamResult.
// It returns the event response or fails the test on error/timeout.
func receiveEvent(ctx context.Context, result streaming.StreamResult[eventsv1.ListenResponse]) *eventsv1.ListenResponse {
	select {
	case resp := <-result.ResCh():
		return resp
	case err := <-result.ErrCh():
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "stream error")

		return nil
	case <-result.DoneCh():
		ginkgo.Fail("stream ended unexpectedly")

		return nil
	case <-ctx.Done():
		gomega.Expect(ctx.Err()).NotTo(gomega.HaveOccurred(), "context timeout")

		return nil
	}
}

// tryReceiveEvent attempts to receive an event but doesn't fail on timeout.
// Returns (response, error) where error is non-nil on timeout/completion.
func tryReceiveEvent(ctx context.Context, result streaming.StreamResult[eventsv1.ListenResponse]) (*eventsv1.ListenResponse, error) {
	select {
	case resp := <-result.ResCh():
		return resp, nil
	case err := <-result.ErrCh():
		return nil, err
	case <-result.DoneCh():
		// Stream ended normally - not an error, just no more events
		//nolint:nilnil
		return nil, nil
	case <-ctx.Done():
		// Return unwrapped context error so callers can check for context.Canceled
		//nolint:wrapcheck
		return nil, ctx.Err()
	}
}

var _ = ginkgo.Describe("Event Streaming E2E Tests", ginkgo.Ordered, ginkgo.Serial, func() {
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

	// Clean up all testdata records after all event tests
	// This prevents interfering with existing 01_client_test.go tests
	ginkgo.AfterAll(func() {
		// Get CIDs for V031, V070, and V080 testdata
		v031Record, _ := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
		v070Record, _ := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
		v080Record, _ := corev1.UnmarshalRecord(testdata.ExpectedRecordV080JSON)

		// IMPORTANT: Unpublish first to remove routing labels, then delete from store
		if v031Record != nil {
			v031Ref := &corev1.RecordRef{Cid: v031Record.GetCid()}
			_ = c.Unpublish(context.Background(), &routingv1.UnpublishRequest{
				Request: &routingv1.UnpublishRequest_RecordRefs{
					RecordRefs: &routingv1.RecordRefs{Refs: []*corev1.RecordRef{v031Ref}},
				},
			})
			_ = c.Delete(context.Background(), v031Ref)
		}
		if v070Record != nil {
			v070Ref := &corev1.RecordRef{Cid: v070Record.GetCid()}
			_ = c.Unpublish(context.Background(), &routingv1.UnpublishRequest{
				Request: &routingv1.UnpublishRequest_RecordRefs{
					RecordRefs: &routingv1.RecordRefs{Refs: []*corev1.RecordRef{v070Ref}},
				},
			})
			_ = c.Delete(context.Background(), v070Ref)
		}
		if v080Record != nil {
			v080Ref := &corev1.RecordRef{Cid: v080Record.GetCid()}
			_ = c.Unpublish(context.Background(), &routingv1.UnpublishRequest{
				Request: &routingv1.UnpublishRequest_RecordRefs{
					RecordRefs: &routingv1.RecordRefs{Refs: []*corev1.RecordRef{v080Ref}},
				},
			})
			_ = c.Delete(context.Background(), v080Ref)
		}

		// Close the client
		if c != nil {
			c.Close()
		}
	})

	ginkgo.Context("RECORD_PUSHED events", func() {
		ginkgo.It("should receive RECORD_PUSHED event when pushing a record", func() {
			// Subscribe to RECORD_PUSHED events
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create and push a record in background
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines

				time.Sleep(200 * time.Millisecond)

				// Use valid test record from testdata
				record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, pushErr := c.Push(context.Background(), record)
				gomega.Expect(pushErr).NotTo(gomega.HaveOccurred())
			}()

			// Receive the event
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent()).NotTo(gomega.BeNil())

			event := resp.GetEvent()
			gomega.Expect(event.GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED))
			gomega.Expect(event.GetResourceId()).NotTo(gomega.BeEmpty())
			gomega.Expect(event.GetLabels()).NotTo(gomega.BeEmpty()) // V031 has skills labels
		})
	})

	ginkgo.Context("RECORD_PUBLISHED events", func() {
		ginkgo.It("should receive RECORD_PUBLISHED event when publishing a record", func() {
			// First push a record using valid testdata
			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ref, err := c.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Subscribe to RECORD_PUBLISHED events
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Publish the record in background
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				publishErr := c.Publish(context.Background(), &routingv1.PublishRequest{
					Request: &routingv1.PublishRequest_RecordRefs{
						RecordRefs: &routingv1.RecordRefs{
							Refs: []*corev1.RecordRef{ref},
						},
					},
				})
				gomega.Expect(publishErr).NotTo(gomega.HaveOccurred())

				// Wait for async publish to complete
				time.Sleep(2 * time.Second)
			}()

			// Receive the event
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent()).NotTo(gomega.BeNil())

			event := resp.GetEvent()
			gomega.Expect(event.GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUBLISHED))
			gomega.Expect(event.GetResourceId()).To(gomega.Equal(ref.GetCid()))
			gomega.Expect(event.GetLabels()).NotTo(gomega.BeEmpty())
		})
	})

	ginkgo.Context("RECORD_DELETED events", func() {
		ginkgo.It("should receive RECORD_DELETED event when deleting a record", func() {
			// First push a record using valid testdata
			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ref, err := c.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Subscribe to RECORD_DELETED events
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_DELETED},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Delete the record in background
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				deleteErr := c.Delete(context.Background(), ref)
				gomega.Expect(deleteErr).NotTo(gomega.HaveOccurred())
			}()

			// Receive the event
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent()).NotTo(gomega.BeNil())

			event := resp.GetEvent()
			gomega.Expect(event.GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_DELETED))
			gomega.Expect(event.GetResourceId()).To(gomega.Equal(ref.GetCid()))
		})
	})

	ginkgo.Context("Event filtering", func() {
		ginkgo.It("should filter events by label", func() {
			// Subscribe with label filter (natural_language_processing is in V070 record)
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				LabelFilters: []string{"/skills/natural_language_processing"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push a matching record
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				// Use V070 test record which has natural_language_processing skills
				record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, _ = c.Push(context.Background(), record)
			}()

			// Should receive the matching event
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent()).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetEvent().GetLabels()).To(gomega.ContainElement(gomega.ContainSubstring("/skills/natural_language_processing")))
		})

		ginkgo.It("should filter events by CID", func() {
			// First push a record using valid testdata
			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ref, err := c.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Subscribe with CID filter
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				CidFilters: []string{ref.GetCid()},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push another record and pull the filtered one
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				// Push another record (different CID, should be filtered out)
				otherRecord, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, _ = c.Push(context.Background(), otherRecord)

				time.Sleep(100 * time.Millisecond)

				// Pull the filtered record (triggers RECORD_PULLED for the target CID)
				_, _ = c.Pull(context.Background(), ref)
			}()

			// Should receive only the event for the filtered CID
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent()).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetEvent().GetResourceId()).To(gomega.Equal(ref.GetCid()))
		})

		ginkgo.It("should filter events by event type", func() {
			// Subscribe only to PUSHED events (not PULLED or DELETED)
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Perform multiple operations
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				// Use valid testdata
				record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Push (should trigger PUSHED event - we should receive this)
				ref, pushErr := c.Push(context.Background(), record)
				gomega.Expect(pushErr).NotTo(gomega.HaveOccurred())

				time.Sleep(100 * time.Millisecond)

				// Pull (triggers PULLED event - should be filtered out)
				_, _ = c.Pull(context.Background(), ref)
			}()

			// Should receive only the PUSHED event
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent()).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetEvent().GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED))

			// Try to receive another event (should timeout - pull event was filtered)
			resp2, err := tryReceiveEvent(streamCtx, result)
			if err == nil && resp2 != nil {
				// If we got another event, it should still be PUSHED (not PULLED)
				gomega.Expect(resp2.GetEvent().GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED))
			}
		})
	})

	ginkgo.Context("Multiple subscribers", func() {
		ginkgo.It("should deliver events to multiple subscribers", func() {
			// Create two subscriptions
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result1, err1 := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
			})
			gomega.Expect(err1).NotTo(gomega.HaveOccurred())

			result2, err2 := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
			})
			gomega.Expect(err2).NotTo(gomega.HaveOccurred())

			// Push a record
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				// Use valid testdata
				record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, pushErr := c.Push(context.Background(), record)
				gomega.Expect(pushErr).NotTo(gomega.HaveOccurred())
			}()

			// Both streams should receive the same event
			resp1 := receiveEvent(streamCtx, result1)
			gomega.Expect(resp1.GetEvent().GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED))

			resp2 := receiveEvent(streamCtx, result2)
			gomega.Expect(resp2.GetEvent().GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED))

			// Both should have same resource ID (non-empty)
			gomega.Expect(resp1.GetEvent().GetResourceId()).NotTo(gomega.BeEmpty())
			gomega.Expect(resp1.GetEvent().GetResourceId()).To(gomega.Equal(resp2.GetEvent().GetResourceId()))
		})
	})

	ginkgo.Context("Stream lifecycle", func() {
		ginkgo.It("should handle context cancellation gracefully", func() {
			cancelCtx, cancelFunc := context.WithCancel(ctx)

			result, err := c.ListenStream(cancelCtx, &eventsv1.ListenRequest{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Cancel context immediately
			cancelFunc()

			// Should receive error or completion
			_, err = tryReceiveEvent(cancelCtx, result)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should receive multiple events in sequence", func() {
			streamCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{
					eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED,
					eventsv1.EventType_EVENT_TYPE_RECORD_PULLED,
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Perform multiple operations
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				// Push first record
				record1, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				ref1, _ := c.Push(context.Background(), record1)

				time.Sleep(200 * time.Millisecond)

				// Push second record
				record2, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				_, _ = c.Push(context.Background(), record2)

				time.Sleep(200 * time.Millisecond)

				// Pull first record
				_, _ = c.Pull(context.Background(), ref1)
			}()

			// Receive first PUSHED event
			resp1 := receiveEvent(streamCtx, result)
			gomega.Expect(resp1.GetEvent().GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED))

			// Receive second PUSHED event
			resp2 := receiveEvent(streamCtx, result)
			gomega.Expect(resp2.GetEvent().GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED))

			// Receive PULLED event
			resp3 := receiveEvent(streamCtx, result)
			gomega.Expect(resp3.GetEvent().GetType()).To(gomega.Equal(eventsv1.EventType_EVENT_TYPE_RECORD_PULLED))
		})
	})

	ginkgo.Context("Event metadata", func() {
		ginkgo.It("should include labels in record events", func() {
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				EventTypes: []eventsv1.EventType{eventsv1.EventType_EVENT_TYPE_RECORD_PUSHED},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push a record with multiple skills
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				// Use valid testdata (V031 has multiple skills)
				record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, _ = c.Push(context.Background(), record)
			}()

			// Receive event and verify labels
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent().GetLabels()).NotTo(gomega.BeEmpty())
			// V031 has "Natural Language Processing" skills
			gomega.Expect(resp.GetEvent().GetLabels()).To(gomega.ContainElement(gomega.ContainSubstring("/skills/")))
		})

		ginkgo.It("should include timestamp in all events", func() {
			streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push a record
			go func() {
				defer ginkgo.GinkgoRecover() // Required for assertions in goroutines
				time.Sleep(200 * time.Millisecond)

				// Use valid testdata
				record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, _ = c.Push(context.Background(), record)
			}()

			// Receive event and verify timestamp
			resp := receiveEvent(streamCtx, result)
			gomega.Expect(resp.GetEvent().GetTimestamp()).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetEvent().GetTimestamp().AsTime()).To(gomega.BeTemporally("~", time.Now(), 5*time.Second))
		})
	})

	ginkgo.Context("No events scenario", func() {
		ginkgo.It("should timeout when no events occur", func() {
			// Subscribe with very specific filter that won't match
			streamCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			result, err := c.ListenStream(streamCtx, &eventsv1.ListenRequest{
				CidFilters: []string{"bafynonexistent123456789"},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Try to receive - should eventually timeout
			_, err = tryReceiveEvent(streamCtx, result)
			gomega.Expect(err).To(gomega.Or(
				gomega.Equal(context.DeadlineExceeded),
				gomega.MatchError(gomega.ContainSubstring("deadline")),
				gomega.MatchError(gomega.ContainSubstring("cancel")),
			))
		})
	})
})
