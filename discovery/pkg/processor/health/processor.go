package health

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/discovery/pkg/types"
)

// processor probes workloads for health endpoints.
type processor struct {
	timeout time.Duration
	paths   []string
	client  *http.Client
}

// NewProcessor creates a new health check processor.
func NewProcessor(cfg Config) types.WorkloadProcessor {
	return &processor{
		timeout: cfg.Timeout,
		paths:   cfg.Paths,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Name returns the processor name.
func (p *processor) Name() string {
	return "health"
}

// ShouldProcess returns whether to process the workload.
func (p *processor) ShouldProcess(workload *types.Workload) bool {
	return len(workload.Addresses) > 0 && len(workload.Ports) > 0
}

// Process probes health endpoints on the workload.
func (p *processor) Process(ctx context.Context, workload *types.Workload) (interface{}, error) {
	if !p.ShouldProcess(workload) {
		result := NewHealthResult(false)
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
	result := NewHealthResult(false)
	result.Error = "All endpoints failed"
	log.Printf("[health] %s is unhealthy: %s", workload.Name, result.Error)
	return result, nil
}

// probeURL probes a single URL.
func (p *processor) probeURL(ctx context.Context, url string) *HealthResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result := NewHealthResult(false)
		result.Error = err.Error()
		return result
	}

	resp, err := p.client.Do(req)
	if err != nil {
		result := NewHealthResult(false)
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)

	result := NewHealthResult(resp.StatusCode >= 200 && resp.StatusCode < 400)
	result.Endpoint = url
	result.StatusCode = resp.StatusCode
	result.ResponseTimeMs = float64(elapsed.Milliseconds())

	return result
}

// HealthResult represents a health check result.
type HealthResult struct {
	Healthy        bool    `json:"healthy"`
	Endpoint       string  `json:"endpoint,omitempty"`
	StatusCode     int     `json:"status_code,omitempty"`
	ResponseTimeMs float64 `json:"response_time_ms,omitempty"`
	Error          string  `json:"error,omitempty"`
	CheckedAt      string  `json:"checked_at"`
}

// NewHealthResult creates a new health result with current timestamp.
func NewHealthResult(healthy bool) *HealthResult {
	return &HealthResult{
		Healthy:   healthy,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
