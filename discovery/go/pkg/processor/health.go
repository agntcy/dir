// Package processor provides the health check processor.
package processor

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/discovery/pkg/config"
	"github.com/agntcy/dir/discovery/pkg/models"
)

// HealthProcessor probes workloads for health endpoints.
type HealthProcessor struct {
	enabled bool
	timeout time.Duration
	paths   []string
	client  *http.Client
}

// NewHealthProcessor creates a new health check processor.
func NewHealthProcessor(cfg *config.HealthProcessorConfig) *HealthProcessor {
	return &HealthProcessor{
		enabled: cfg.Enabled,
		timeout: cfg.Timeout,
		paths:   cfg.Paths,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Name returns the processor name.
func (p *HealthProcessor) Name() string {
	return "health"
}

// Enabled returns whether the processor is enabled.
func (p *HealthProcessor) Enabled() bool {
	return p.enabled
}

// ShouldProcess returns whether to process the workload.
func (p *HealthProcessor) ShouldProcess(workload *models.Workload) bool {
	return len(workload.Addresses) > 0 && len(workload.Ports) > 0
}

// Process probes health endpoints on the workload.
func (p *HealthProcessor) Process(ctx context.Context, workload *models.Workload) (interface{}, error) {
	if !p.ShouldProcess(workload) {
		result := models.NewHealthResult(false)
		result.Error = "No addresses or ports available"
		return result, nil
	}

	// Build list of URLs to try
	var urls []string
	for _, addr := range workload.Addresses {
		for _, port := range workload.Ports {
			for _, path := range p.paths {
				urls = append(urls, fmt.Sprintf("http://%s:%s%s", addr, port, path))
			}
		}
	}

	log.Printf("[health] Probing (%s) URLs for workload %s", strings.Join(urls, ","), workload.Name)

	// Try each URL
	for _, url := range urls {
		result := p.probeURL(ctx, url)
		if result.Healthy {
			log.Printf("[health] %s is healthy at %s (%.0fms)",
				workload.Name, result.Endpoint, result.ResponseTimeMs)
			return result, nil
		}
	}

	// All failed
	result := models.NewHealthResult(false)
	result.Error = "All endpoints failed"
	log.Printf("[health] %s is unhealthy: %s", workload.Name, result.Error)
	return result, nil
}

// probeURL probes a single URL.
func (p *HealthProcessor) probeURL(ctx context.Context, url string) *models.HealthResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result := models.NewHealthResult(false)
		result.Error = err.Error()
		return result
	}

	resp, err := p.client.Do(req)
	if err != nil {
		result := models.NewHealthResult(false)
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)

	result := models.NewHealthResult(resp.StatusCode >= 200 && resp.StatusCode < 400)
	result.Endpoint = url
	result.StatusCode = resp.StatusCode
	result.ResponseTimeMs = float64(elapsed.Milliseconds())

	return result
}
