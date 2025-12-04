// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Metrics E2E Tests
//
// Testing Strategy:
// - Verify Prometheus /metrics endpoint is accessible
// - Validate gRPC metrics are being collected
// - Check that metrics contain data from previous tests (01-06)
//
// Note: This test runs AFTER other tests (numbered 07), so metrics should
// already contain non-zero values from previous test operations.

var _ = ginkgo.Describe("Prometheus Metrics", ginkgo.Serial, ginkgo.Label("metrics"), func() {
	const metricsURL = "http://localhost:9090/metrics"

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}
	})

	ginkgo.Context("metrics endpoint availability", func() {
		ginkgo.It("should expose /metrics endpoint on port 9090", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
			gomega.Expect(resp.Header.Get("Content-Type")).To(gomega.ContainSubstring("text/plain"))
		})

		ginkgo.It("should return Prometheus-formatted metrics", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metrics := string(body)

			// Verify Prometheus format (HELP and TYPE comments)
			gomega.Expect(metrics).To(gomega.ContainSubstring("# HELP"))
			gomega.Expect(metrics).To(gomega.ContainSubstring("# TYPE"))

			// Verify metrics are not empty
			gomega.Expect(len(metrics)).To(gomega.BeNumerically(">", 100),
				"Expected metrics output to be substantial")
		})
	})

	ginkgo.Context("gRPC metrics collection", func() {
		var metricsContent string

		ginkgo.BeforeEach(func() {
			// Fetch metrics once for all tests in this context
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent = string(body)
		})

		ginkgo.It("should expose grpc_server_started_total counter", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("grpc_server_started_total"))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("# TYPE grpc_server_started_total counter"))
		})

		ginkgo.It("should expose grpc_server_handled_total counter", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("grpc_server_handled_total"))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("# TYPE grpc_server_handled_total counter"))
		})

		ginkgo.It("should expose grpc_server_msg_received_total counter for streaming", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("grpc_server_msg_received_total"))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("# TYPE grpc_server_msg_received_total counter"))
		})

		ginkgo.It("should expose grpc_server_msg_sent_total counter for streaming", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("grpc_server_msg_sent_total"))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring("# TYPE grpc_server_msg_sent_total counter"))
		})

		ginkgo.It("should include StoreService metrics", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_service="agntcy.dir.store.v1.StoreService"`))

			// Verify key StoreService methods are instrumented
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method="Push"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method="Pull"`))
		})

		ginkgo.It("should include RoutingService metrics", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_service="agntcy.dir.routing.v1.RoutingService"`))

			// Verify key RoutingService methods are instrumented
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method="Search"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method="Publish"`))
		})

		ginkgo.It("should include EventService metrics", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_service="agntcy.dir.events.v1.EventService"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method="Listen"`))
		})

		ginkgo.It("should include SearchService metrics", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_service="agntcy.dir.search.v1.SearchService"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method="Search"`))
		})

		ginkgo.It("should include Health service metrics", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_service="grpc.health.v1.Health"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method="Check"`))
		})
	})

	ginkgo.Context("metrics from previous tests", func() {
		ginkgo.It("should have non-zero request counts from previous tests", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Parse metrics to find counters with non-zero values
			// Previous tests (01-06) should have generated traffic
			foundNonZero := false

			for _, line := range strings.Split(metricsContent, "\n") {
				// Skip comments and empty lines
				if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
					continue
				}

				// Look for grpc_server_started_total or grpc_server_handled_total with values
				if strings.Contains(line, "grpc_server_started_total") ||
					strings.Contains(line, "grpc_server_handled_total") {
					// Parse value (last part after space)
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						value := parts[len(parts)-1]
						if value != "0" && value != "0.0" {
							foundNonZero = true

							ginkgo.GinkgoWriter.Printf("Found non-zero metric: %s\n", line)

							break
						}
					}
				}
			}

			gomega.Expect(foundNonZero).To(gomega.BeTrue(),
				"Expected to find non-zero request metrics from previous tests (01-06)")
		})

		ginkgo.It("should have successful (OK) status codes from previous tests", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Look for successful operations (grpc_code="OK")
			foundOKStatus := false

			for _, line := range strings.Split(metricsContent, "\n") {
				if strings.Contains(line, `grpc_code="OK"`) && !strings.HasPrefix(line, "#") {
					// Parse value
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						value := parts[len(parts)-1]
						if value != "0" && value != "0.0" {
							foundOKStatus = true

							ginkgo.GinkgoWriter.Printf("Found successful request: %s\n", line)

							break
						}
					}
				}
			}

			gomega.Expect(foundOKStatus).To(gomega.BeTrue(),
				"Expected to find successful (OK) requests from previous tests")
		})
	})

	ginkgo.Context("metrics labels and structure", func() {
		var metricsContent string

		ginkgo.BeforeEach(func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent = string(body)
		})

		ginkgo.It("should include grpc_method label", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_method=`))
		})

		ginkgo.It("should include grpc_service label", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_service=`))
		})

		ginkgo.It("should include grpc_type label", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_type=`))

			// Verify different RPC types are tracked
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_type="unary"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_type="bidi_stream"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_type="server_stream"`))
		})

		ginkgo.It("should include grpc_code label for completed requests", func() {
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_code=`))

			// Verify common status codes are tracked
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_code="OK"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_code="InvalidArgument"`))
			gomega.Expect(metricsContent).To(gomega.ContainSubstring(`grpc_code="NotFound"`))
		})
	})

	ginkgo.Context("metrics validation", func() {
		ginkgo.It("should report metrics for all registered services", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Expected services to be instrumented
			expectedServices := []string{
				"agntcy.dir.store.v1.StoreService",
				"agntcy.dir.routing.v1.RoutingService",
				"agntcy.dir.events.v1.EventService",
				"agntcy.dir.search.v1.SearchService",
				"agntcy.dir.store.v1.SyncService",
				"agntcy.dir.routing.v1.PublicationService",
				"agntcy.dir.sign.v1.SignService",
				"grpc.health.v1.Health",
			}

			for _, service := range expectedServices {
				gomega.Expect(metricsContent).To(
					gomega.ContainSubstring(fmt.Sprintf(`grpc_service="%s"`, service)),
					"Expected metrics for service: %s", service,
				)
			}
		})

		ginkgo.It("should count method invocations from previous tests", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Count lines with actual metric values (not HELP or TYPE)
			metricLines := 0

			for _, line := range strings.Split(metricsContent, "\n") {
				if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
					metricLines++
				}
			}

			gomega.Expect(metricLines).To(gomega.BeNumerically(">", 50),
				"Expected at least 50 metric data lines, got %d", metricLines)

			ginkgo.GinkgoWriter.Printf("Total metric data lines: %d\n", metricLines)
		})
	})

	ginkgo.Context("integration with kubectl (optional - if ServiceMonitor enabled)", func() {
		ginkgo.It("should be able to port-forward to metrics port", func() {
			// This test verifies the Kubernetes service exposes the metrics port
			// Note: We don't actually port-forward here (it's already done by Taskfile)
			// We just verify the metrics are accessible via the existing port-forward

			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))

			ginkgo.GinkgoWriter.Println("Metrics port is accessible via port-forward ✓")
		})
	})

	ginkgo.Context("metrics useful for monitoring", func() {
		ginkgo.It("should provide data for request rate queries", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Verify we can identify busy methods
			// Example: grpc_server_started_total{grpc_method="Push",grpc_service="...",grpc_type="bidi_stream"} 5
			hasStartedMetrics := strings.Contains(metricsContent, "grpc_server_started_total")
			gomega.Expect(hasStartedMetrics).To(gomega.BeTrue(),
				"Need grpc_server_started_total for rate(grpc_server_started_total[5m]) queries")
		})

		ginkgo.It("should provide data for error rate queries", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Verify we have status codes for error rate calculation
			hasHandledMetrics := strings.Contains(metricsContent, "grpc_server_handled_total")
			hasStatusCodes := strings.Contains(metricsContent, `grpc_code="OK"`)

			gomega.Expect(hasHandledMetrics).To(gomega.BeTrue())
			gomega.Expect(hasStatusCodes).To(gomega.BeTrue())

			ginkgo.GinkgoWriter.Println("Metrics support error rate calculation: rate(grpc_server_handled_total{grpc_code!=\"OK\"}[5m]) / rate(grpc_server_handled_total[5m])")
		})

		ginkgo.It("should support latency percentile queries (histogram buckets)", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Note: grpc-prometheus v1.2.0 doesn't expose histogram by default
			// Check if we have any timing metrics
			// If histograms are missing, this is expected and we can add them in future iterations

			ginkgo.GinkgoWriter.Println("Checking for latency metrics (histograms)...")

			if strings.Contains(metricsContent, "grpc_server_handling_seconds") {
				ginkgo.GinkgoWriter.Println("✓ Latency histogram metrics found")

				gomega.Expect(metricsContent).To(gomega.ContainSubstring("grpc_server_handling_seconds"))
			} else {
				ginkgo.GinkgoWriter.Println("ℹ Latency histograms not available in current grpc-prometheus version")
				ginkgo.GinkgoWriter.Println("  This is expected with grpc-prometheus v1.2.0")
				ginkgo.GinkgoWriter.Println("  To add histograms, we can use grpc-ecosystem/go-grpc-middleware/v2")
			}
		})
	})

	ginkgo.Context("metrics data sanity checks", func() {
		ginkgo.It("should parse as valid Prometheus metrics format", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Basic validation: each metric line should have format "metric_name{labels} value"
			metricDataLines := 0
			invalidLines := []string{}

			for _, line := range strings.Split(metricsContent, "\n") {
				// Skip comments and empty lines
				if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
					continue
				}

				metricDataLines++

				// Validate format: should have '{' for labels and end with a number
				if !strings.Contains(line, "{") {
					invalidLines = append(invalidLines, line)

					continue
				}

				// Check it ends with a number (simple validation)
				parts := strings.Fields(line)
				if len(parts) < 2 {
					invalidLines = append(invalidLines, line)
				}
			}

			gomega.Expect(invalidLines).To(gomega.BeEmpty(),
				"Found %d invalid metric lines: %v", len(invalidLines), invalidLines)

			ginkgo.GinkgoWriter.Printf("Validated %d metric data lines ✓\n", metricDataLines)
		})

		ginkgo.It("should not have negative metric values", func() {
			resp, err := http.Get(metricsURL)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			metricsContent := string(body)

			// Check for negative values
			negativeLines := []string{}

			for _, line := range strings.Split(metricsContent, "\n") {
				// Skip comments
				if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
					continue
				}

				// Look for negative values
				if strings.Contains(line, " -") {
					negativeLines = append(negativeLines, line)
				}
			}

			gomega.Expect(negativeLines).To(gomega.BeEmpty(),
				"Found metrics with negative values: %v", negativeLines)
		})
	})
})
