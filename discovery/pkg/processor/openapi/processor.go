// Package processor provides the OpenAPI discovery processor.
package openapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/discovery/pkg/types"
	"gopkg.in/yaml.v3"
)

// processor discovers OpenAPI specifications from workloads.
type processor struct {
	timeout time.Duration
	paths   []string
	client  *http.Client
}

// NewProcessor creates a new OpenAPI discovery processor.
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
	return "openapi"
}

// ShouldProcess returns whether to process the workload.
func (p *processor) ShouldProcess(workload *types.Workload) bool {
	return len(workload.Addresses) > 0 && len(workload.Ports) > 0
}

// Process discovers OpenAPI specs from the workload.
func (p *processor) Process(ctx context.Context, workload *types.Workload) (interface{}, error) {
	if !p.ShouldProcess(workload) {
		result := NewOpenAPIResult()
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

	log.Printf("[openapi] Probing URLs for workload %s", workload.Name)

	// Try each URL
	for _, url := range urls {
		result := p.fetchSpec(ctx, url)
		if result.Available {
			log.Printf("[openapi] Found spec for %s at %s (v%s, %d paths)",
				workload.Name, result.Endpoint, result.Version, result.PathCount)
			return result, nil
		}
	}

	// No spec found
	result := NewOpenAPIResult()
	result.Error = "No OpenAPI spec found"
	log.Printf("[openapi] No spec found for %s", workload.Name)
	return result, nil
}

// fetchSpec fetches and parses an OpenAPI spec from a URL.
func (p *processor) fetchSpec(ctx context.Context, url string) *OpenAPIResult {
	result := NewOpenAPIResult()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	resp, err := p.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Try to parse as JSON or YAML
	spec, err := p.parseSpec(body)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Extract OpenAPI information
	result.Available = true
	result.Endpoint = url
	result.Version = p.extractVersion(spec)
	result.Title = p.extractTitle(spec)
	result.Paths = p.extractPaths(spec)
	result.PathCount = len(result.Paths)

	return result
}

// parseSpec parses an OpenAPI spec from JSON or YAML.
func (p *processor) parseSpec(body []byte) (map[string]interface{}, error) {
	var spec map[string]interface{}

	// Try JSON first
	if err := json.Unmarshal(body, &spec); err == nil {
		return spec, nil
	}

	// Try YAML
	if err := yaml.Unmarshal(body, &spec); err == nil {
		return spec, nil
	}

	return nil, fmt.Errorf("failed to parse as JSON or YAML")
}

// extractVersion extracts the OpenAPI version from the spec.
func (p *processor) extractVersion(spec map[string]interface{}) string {
	// OpenAPI 3.x
	if v, ok := spec["openapi"].(string); ok {
		return v
	}
	// Swagger 2.x
	if v, ok := spec["swagger"].(string); ok {
		return v
	}
	return "unknown"
}

// extractTitle extracts the title from the spec.
func (p *processor) extractTitle(spec map[string]interface{}) string {
	if info, ok := spec["info"].(map[string]interface{}); ok {
		if title, ok := info["title"].(string); ok {
			return title
		}
	}
	return ""
}

// extractPaths extracts the paths from the spec.
func (p *processor) extractPaths(spec map[string]interface{}) []string {
	var paths []string
	if pathsMap, ok := spec["paths"].(map[string]interface{}); ok {
		for path := range pathsMap {
			// Skip paths with parameters for cleaner output
			if !strings.Contains(path, "{") {
				paths = append(paths, path)
			}
		}
	}
	return paths
}

// OpenAPIResult represents an OpenAPI discovery result.
type OpenAPIResult struct {
	Available    bool     `json:"available"`
	Endpoint     string   `json:"endpoint,omitempty"`
	Title        string   `json:"title,omitempty"`
	Version      string   `json:"version,omitempty"`
	Description  string   `json:"description,omitempty"`
	Paths        []string `json:"paths,omitempty"`
	PathCount    int      `json:"path_count,omitempty"`
	Error        string   `json:"error,omitempty"`
	DiscoveredAt string   `json:"discovered_at"`
}

// NewOpenAPIResult creates a new OpenAPI result with current timestamp.
func NewOpenAPIResult() *OpenAPIResult {
	return &OpenAPIResult{
		Available:    false,
		DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
	}
}
