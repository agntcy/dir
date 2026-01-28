package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/discovery/pkg/types"
)

// processor probes workloads for A2A endpoints.
type processor struct {
	timeout    time.Duration
	paths      []string
	client     *http.Client
	labelKey   string
	labelValue string
}

// NewProcessor creates a new A2A processor.
func NewProcessor(cfg Config) types.WorkloadProcessor {
	return &processor{
		timeout: cfg.Timeout,
		paths:   cfg.Paths,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		labelKey:   cfg.LabelKey,
		labelValue: cfg.LabelValue,
	}
}

// Name returns the processor name.
func (p *processor) Name() string {
	return "a2a"
}

// ShouldProcess returns whether to process the workload.
func (p *processor) ShouldProcess(workload *types.Workload) bool {
	// If workload does not have a label key with expected value, skip it
	if val, ok := workload.Labels[p.labelKey]; ok {
		if !strings.Contains(strings.ToLower(val), strings.ToLower(p.labelValue)) {
			return false
		}
	} else {
		return false
	}

	// Only process workloads with addresses and ports
	return len(workload.Addresses) > 0 && len(workload.Ports) > 0
}

// Process probes health endpoints on the workload.
func (p *processor) Process(ctx context.Context, workload *types.Workload) (interface{}, error) {
	// Build list of URLs to try
	var urls []string
	for _, addr := range workload.Addresses {
		for _, port := range workload.Ports {
			for _, path := range p.paths {
				urls = append(urls, fmt.Sprintf("http://%s:%s%s", addr, port, path))
			}
		}
	}

	log.Printf("[a2a] Probing (%s) URLs for workload %s", strings.Join(urls, ","), workload.Name)

	// Try each URL
	for _, url := range urls {
		result := p.probeURL(ctx, url)
		if result != nil {
			log.Printf("[a2a] %s scraped successfully", workload.Name)
			return result, nil
		}
	}

	// All failed
	return nil, fmt.Errorf("[a2a] no reachable A2A endpoints found for workload %s", workload.Name)
}

// probeURL probes a single URL.
func (p *processor) probeURL(ctx context.Context, url string) map[string]any {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Read returned body as metadata
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	if len(result) == 0 {
		return nil
	}

	return result
}
