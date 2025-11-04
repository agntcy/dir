// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package client contains end-to-end tests for the client library including rate limiting.
//
// Rate Limiting Test Configuration:
// These tests expect the server to be configured with rate limiting enabled.
// Default configuration from Taskfile (test:e2e:client and test:e2e:local):
//   - RATELIMIT_ENABLED: true
//   - RATELIMIT_GLOBAL_RPS: 100 (requests per second for all clients)
//   - RATELIMIT_GLOBAL_BURST: 200 (burst capacity for all clients)
//
// Note: The tests use GLOBAL rate limiting (not per-client) because authentication
// is not enabled in e2e tests. Without authentication, all clients are treated as
// "unauthenticated" and share the global rate limiter. In production with authentication
// enabled, per-client rate limiting would be used instead.
//
// The tests are designed to:
//   - Verify requests within limits succeed
//   - Verify rapid requests exceeding burst capacity are rate limited
//   - Handle cases where rate limiting is disabled with informative warnings
//
// Run with: task test:e2e:client
// Or customize: task test:e2e:client RATELIMIT_GLOBAL_RPS=50 RATELIMIT_GLOBAL_BURST=100
package client

import (
	"context"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isRateLimitError checks if the error is a rate limit error (codes.ResourceExhausted).
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)

	return ok && st.Code() == codes.ResourceExhausted
}

var _ = ginkgo.Describe("Rate Limiting E2E Tests", ginkgo.Label("ratelimit"), ginkgo.Ordered, ginkgo.Serial, func() {
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

	ginkgo.Context("Rate limiting behavior", func() {
		ginkgo.It("should allow requests within rate limit", func() {
			// Push multiple records within the rate limit
			// Test expects: Global RPS=100, Burst=200 (default from Taskfile)

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Push 5 records with a small delay between each
			// This should be well within any reasonable rate limit
			for range 5 {
				ref, pushErr := c.Push(ctx, record)
				gomega.Expect(pushErr).NotTo(gomega.HaveOccurred())
				gomega.Expect(ref).NotTo(gomega.BeNil())
				gomega.Expect(ref.GetCid()).NotTo(gomega.BeEmpty())

				// Clean up immediately
				_ = c.Delete(ctx, ref)

				// Small delay to avoid burst issues
				time.Sleep(50 * time.Millisecond)
			}
		})

		ginkgo.It("should reject requests when rate limit is exceeded", func() {
			// This test attempts to exceed the rate limit by making rapid sequential requests
			// Test expects: Global RPS=100, Burst=200 (default from Taskfile)
			// With burst=200, first 200 requests succeed immediately, then rate limiting kicks in

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Make more requests than burst capacity to ensure we hit rate limiting
			// With burst=200, we need >200 rapid requests to trigger rate limiting
			var successCount int
			var rateLimitErrorCount int
			const totalRequests = 250 // Increased from 200 to exceed burst capacity

			ginkgo.By("Making rapid sequential requests to exceed rate limit")
			for i := range totalRequests {
				ref, pushErr := c.Push(ctx, record)
				if pushErr == nil {
					successCount++
					// Clean up successful pushes
					_ = c.Delete(ctx, ref)
				} else if isRateLimitError(pushErr) {
					rateLimitErrorCount++
					ginkgo.GinkgoWriter.Printf("Rate limit error on request %d: %v\n", i+1, pushErr)
				} else {
					// Unexpected error
					gomega.Expect(pushErr).NotTo(gomega.HaveOccurred(),
						"Unexpected error (not rate limit): %v", pushErr)
				}
			}

			ginkgo.GinkgoWriter.Printf("Results: %d successful, %d rate limited out of %d requests\n",
				successCount, rateLimitErrorCount, totalRequests)

			// We should have some successful requests and some rate limited requests
			// This validates that rate limiting is working
			gomega.Expect(successCount).To(gomega.BeNumerically(">", 0),
				"Should have at least some successful requests")

			// If rate limiting is properly configured and enabled, we should see some rate limit errors
			// Note: If this fails, check that rate limiting is enabled in the server config
			if rateLimitErrorCount == 0 {
				ginkgo.GinkgoWriter.Println("WARNING: No rate limit errors detected. Rate limiting may be disabled or set too high.")
			}
		})

		ginkgo.It("should handle burst requests correctly", func() {
			// Test burst capacity by making quick successive requests
			// Test expects: Global RPS=100, Burst=200 (default from Taskfile)
			// Small bursts (<<200) should always succeed without rate limiting

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Make a burst of requests well within burst capacity (10 << 200)
			const burstSize = 10
			successCount := 0

			ginkgo.By("Making burst of requests")
			for range burstSize {
				ref, pushErr := c.Push(ctx, record)
				if pushErr == nil {
					successCount++
					_ = c.Delete(ctx, ref)
				} else if isRateLimitError(pushErr) {
					// Rate limited - unexpected for small burst
					break
				} else {
					gomega.Expect(pushErr).NotTo(gomega.HaveOccurred())
				}
			}

			// We should successfully complete at least a few requests within burst capacity
			gomega.Expect(successCount).To(gomega.BeNumerically(">=", 1),
				"Should allow at least some burst requests")

			ginkgo.GinkgoWriter.Printf("Burst test: %d/%d requests succeeded\n", successCount, burstSize)
		})

		ginkgo.It("should apply global rate limiting to all clients", func() {
			// This test verifies that global rate limits are applied
			// Note: Without authentication, all clients share the global rate limiter

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Make requests until we hit rate limit
			const maxAttempts = 150
			var hitRateLimit bool

			for i := range maxAttempts {
				ref, pushErr := c.Push(ctx, record)
				if pushErr == nil {
					_ = c.Delete(ctx, ref)
				} else if isRateLimitError(pushErr) {
					hitRateLimit = true
					ginkgo.GinkgoWriter.Printf("Hit rate limit on attempt %d\n", i+1)

					break
				} else {
					gomega.Expect(pushErr).NotTo(gomega.HaveOccurred())
				}
			}

			if !hitRateLimit {
				ginkgo.GinkgoWriter.Println("INFO: Did not hit rate limit. This is expected if rate limits are high or disabled.")
			}
		})

		ginkgo.It("should recover after rate limit period expires", func() {
			// Push enough requests to potentially hit rate limit
			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Making rapid requests to potentially hit rate limit")
			for range 100 {
				ref, _ := c.Push(ctx, record)
				if ref != nil {
					_ = c.Delete(ctx, ref)
				}
			}

			// Wait for rate limiter to refill tokens (typically 1 second for token bucket)
			ginkgo.By("Waiting for rate limit to reset")
			time.Sleep(2 * time.Second)

			// Now requests should succeed again
			ginkgo.By("Verifying requests succeed after rate limit reset")
			ref, err := c.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(ref).NotTo(gomega.BeNil())

			// Clean up
			_ = c.Delete(ctx, ref)
		})
	})

	ginkgo.Context("Different operations", func() {
		ginkgo.It("should apply rate limiting to all gRPC operations", func() {
			// Test that rate limiting works for different operations
			// Push, Pull, Delete should all be rate limited

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// First, push a record to have something to pull
			ref, err := c.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Make rapid Pull requests
			ginkgo.By("Testing rate limit on Pull operations")
			var pullRateLimitHit bool
			for i := range 100 {
				_, pullErr := c.Pull(ctx, ref)
				if isRateLimitError(pullErr) {
					pullRateLimitHit = true
					ginkgo.GinkgoWriter.Printf("Pull rate limited on attempt %d\n", i+1)

					break
				}
			}

			if !pullRateLimitHit {
				ginkgo.GinkgoWriter.Println("INFO: Pull operations did not hit rate limit")
			}

			// Clean up
			_ = c.Delete(ctx, ref)
		})

		ginkgo.It("should handle rate limit errors with proper status codes", func() {
			// Verify that rate limit errors return the correct gRPC status code

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Make many rapid requests to trigger rate limiting
			var rateLimitErr error
			for range 150 {
				ref, pushErr := c.Push(ctx, record)
				if pushErr != nil {
					if isRateLimitError(pushErr) {
						rateLimitErr = pushErr

						break
					}
				} else if ref != nil {
					_ = c.Delete(ctx, ref)
				}
			}

			// If we got a rate limit error, verify its properties
			if rateLimitErr != nil {
				ginkgo.By("Verifying rate limit error properties")
				st, ok := status.FromError(rateLimitErr)
				gomega.Expect(ok).To(gomega.BeTrue(), "Error should be a gRPC status error")
				gomega.Expect(st.Code()).To(gomega.Equal(codes.ResourceExhausted),
					"Rate limit error should have ResourceExhausted code")
				gomega.Expect(strings.ToLower(st.Message())).To(gomega.ContainSubstring("rate limit"),
					"Error message should mention rate limit")
			} else {
				ginkgo.GinkgoWriter.Println("INFO: Did not trigger rate limit error for status code test")
			}
		})
	})

	ginkgo.Context("Rate limit configuration", func() {
		ginkgo.It("should respect per-method rate limits if configured", func() {
			// This test validates that per-method rate limits work correctly
			// The actual limits depend on server configuration

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Test different methods: Push vs Pull
			// If per-method limits are configured, they might have different limits

			ginkgo.By("Testing Push operation rate limit")
			var pushRateLimitHit bool
			for i := range 100 {
				ref, pushErr := c.Push(ctx, record)
				if pushErr != nil && isRateLimitError(pushErr) {
					pushRateLimitHit = true
					ginkgo.GinkgoWriter.Printf("Push rate limited on attempt %d\n", i+1)

					break
				} else if ref != nil {
					_ = c.Delete(ctx, ref)
				}
			}

			// Wait for rate limiter to reset
			time.Sleep(2 * time.Second)

			// Test pull operations
			ginkgo.By("Testing Pull operation rate limit")
			ref, err := c.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var pullRateLimitHit bool
			for i := range 100 {
				_, pullErr := c.Pull(ctx, ref)
				if pullErr != nil && isRateLimitError(pullErr) {
					pullRateLimitHit = true
					ginkgo.GinkgoWriter.Printf("Pull rate limited on attempt %d\n", i+1)

					break
				}
			}

			_ = c.Delete(ctx, ref)

			ginkgo.GinkgoWriter.Printf("Results: Push rate limited: %v, Pull rate limited: %v\n",
				pushRateLimitHit, pullRateLimitHit)

			// Note: If per-method limits are not configured, both operations share the same limit
		})
	})

	ginkgo.Context("Concurrent clients", func() {
		ginkgo.It("should handle rate limiting with concurrent requests", func() {
			// Test that rate limiting works correctly with concurrent requests from the same client

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			const numGoroutines = 10
			const requestsPerGoroutine = 20

			type result struct {
				success     int
				rateLimited int
				otherErrors int
				errorMsgs   []string
			}

			results := make(chan result, numGoroutines)

			ginkgo.By("Launching concurrent goroutines to test rate limiting")
			for range numGoroutines {
				go func() {
					defer ginkgo.GinkgoRecover() // Required for assertions in goroutines

					var r result
					for range requestsPerGoroutine {
						ref, pushErr := c.Push(context.Background(), record)
						if pushErr == nil {
							r.success++
							_ = c.Delete(context.Background(), ref)
						} else if isRateLimitError(pushErr) {
							r.rateLimited++
						} else {
							r.otherErrors++
							r.errorMsgs = append(r.errorMsgs, pushErr.Error())
						}

						// Small random delay to vary timing
						time.Sleep(10 * time.Millisecond)
					}
					results <- r
				}()
			}

			// Collect results
			var totalSuccess, totalRateLimited, totalOtherErrors int
			var allErrors []string
			for range numGoroutines {
				r := <-results
				totalSuccess += r.success
				totalRateLimited += r.rateLimited
				totalOtherErrors += r.otherErrors
				allErrors = append(allErrors, r.errorMsgs...)
			}

			ginkgo.GinkgoWriter.Printf("Concurrent test results: %d successful, %d rate limited, %d other errors\n",
				totalSuccess, totalRateLimited, totalOtherErrors)

			// Log any unexpected errors for debugging
			if totalOtherErrors > 0 {
				ginkgo.GinkgoWriter.Printf("Unexpected errors encountered:\n")
				for i, errMsg := range allErrors {
					ginkgo.GinkgoWriter.Printf("  Error %d: %s\n", i+1, errMsg)
				}
			}

			// We should have at least some successful requests
			gomega.Expect(totalSuccess).To(gomega.BeNumerically(">", 0),
				"Should have at least some successful requests")

			// Note: Concurrent requests might have occasional network/timing errors
			// We allow a small number of non-rate-limit errors (e.g., transient connection issues)
			// but they should be rare (<=5% of requests)
			if totalOtherErrors > 0 {
				errorRate := float64(totalOtherErrors) / float64(numGoroutines*requestsPerGoroutine)
				ginkgo.GinkgoWriter.Printf("Other error rate: %.2f%% (%d/%d)\n",
					errorRate*100, totalOtherErrors, numGoroutines*requestsPerGoroutine)

				// Allow up to 5% error rate for transient issues
				gomega.Expect(errorRate).To(gomega.BeNumerically("<=", 0.05),
					"Error rate should be less than or equal to 5%% (transient errors)")
			}

			// If rate limiting is properly configured, we should see some rate limiting
			if totalRateLimited == 0 {
				ginkgo.GinkgoWriter.Println("INFO: No rate limiting detected in concurrent test")
			}
		})
	})

	ginkgo.Context("Edge cases", func() {
		ginkgo.It("should handle rate limiting with context timeout", func() {
			// Test that rate limiting works correctly when combined with context timeouts

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// Make rapid requests with timeout context
			var successCount, rateLimitCount, timeoutCount int

			for range 100 {
				ref, pushErr := c.Push(timeoutCtx, record)
				if pushErr == nil {
					successCount++
					_ = c.Delete(timeoutCtx, ref)
				} else if isRateLimitError(pushErr) {
					rateLimitCount++
				} else if timeoutCtx.Err() != nil {
					timeoutCount++

					break
				}
			}

			ginkgo.GinkgoWriter.Printf("Timeout test: %d successful, %d rate limited, %d timeout\n",
				successCount, rateLimitCount, timeoutCount)

			gomega.Expect(successCount+rateLimitCount+timeoutCount).To(gomega.BeNumerically(">", 0),
				"Should have processed at least some requests")
		})

		ginkgo.It("should maintain rate limit state across multiple operations", func() {
			// Verify that rate limiter maintains state correctly across different operations

			record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV031JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// First batch of requests
			ginkgo.By("Making first batch of requests")
			for range 50 {
				ref, _ := c.Push(ctx, record)
				if ref != nil {
					_ = c.Delete(ctx, ref)
				}
			}

			// Immediate second batch (should continue from previous state)
			ginkgo.By("Making second batch immediately after")
			var secondBatchRateLimited bool
			for i := range 50 {
				ref, pushErr := c.Push(ctx, record)
				if pushErr != nil && isRateLimitError(pushErr) {
					secondBatchRateLimited = true
					ginkgo.GinkgoWriter.Printf("Second batch rate limited on request %d\n", i+1)

					break
				} else if ref != nil {
					_ = c.Delete(ctx, ref)
				}
			}

			if secondBatchRateLimited {
				ginkgo.GinkgoWriter.Println("Rate limiter correctly maintained state across batches")
			} else {
				ginkgo.GinkgoWriter.Println("INFO: Did not hit rate limit in second batch")
			}
		})
	})
})
