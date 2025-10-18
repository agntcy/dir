// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

const (
	// defaultPageLimit is the default number of servers to fetch per page.
	defaultPageLimit = 30
)

// Supported filters https://registry.modelcontextprotocol.io/docs#/operations/list-servers#Query-Parameters
//   - search: Filter by server name (substring match)
//   - version: Filter by version ('latest' for latest version, or an exact version like '1.2.3')
//   - updated_since: Filter by updated time (RFC3339 datetime)
//   - limit: Number of servers per page (default 30)
//   - cursor: Pagination cursor
var supportedFilters = []string{
	"search",
	"version",
	"updated_since",
	"limit",
	"cursor",
}

// Fetcher implements the pipeline.Fetcher interface for MCP registry.
type Fetcher struct {
	url        *url.URL
	httpClient *http.Client
	filters    map[string]string
	limit      int
}

// NewFetcher creates a new MCP fetcher.
func NewFetcher(baseURL string, filters map[string]string, limit int) (*Fetcher, error) {
	// Parse and validate base URL
	u, err := url.Parse(baseURL + "/servers")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Validate filters
	for key := range filters {
		if !slices.Contains(supportedFilters, key) {
			return nil, fmt.Errorf("unsupported filter: %s", key)
		}
	}

	return &Fetcher{
		url: u,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, //nolint:mnd
		},
		filters: filters,
		limit:   limit,
	}, nil
}

// Fetch retrieves servers from the MCP registry and sends them to the output channel.
func (f *Fetcher) Fetch(ctx context.Context) (<-chan interface{}, <-chan error) {
	// Use buffered channel to allow fetcher to work ahead of transformers
	outputCh := make(chan interface{}, 50) //nolint:mnd
	errCh := make(chan error, 1)

	go func() {
		defer close(outputCh)
		defer close(errCh)

		cursor := ""
		count := 0

		for {
			// Check context cancellation
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()

				return
			default:
			}

			// Fetch one page
			page, nextCursor, err := f.listServersPage(ctx, cursor)
			if err != nil {
				errCh <- err

				return
			}

			// Stream each server as soon as it's available
			for _, server := range page {
				// Check if limit is reached (limit <= 0 means no limit)
				if f.limit > 0 && count >= f.limit {
					return
				}

				select {
				case <-ctx.Done():
					errCh <- ctx.Err()

					return
				case outputCh <- server:
					count++
				}
			}

			// Check if there are more pages
			if nextCursor == "" {
				break
			}

			cursor = nextCursor
		}
	}()

	return outputCh, errCh
}

// listServersPage fetches a single page of servers from the MCP registry.
func (f *Fetcher) listServersPage(ctx context.Context, cursor string) ([]mcpapiv0.ServerResponse, string, error) {
	// Add filters as query parameters
	query := f.url.Query()

	for key, value := range f.filters {
		if value != "" {
			query.Set(key, value)
		}
	}

	// Add cursor if provided
	if cursor != "" {
		query.Set("cursor", cursor)
	}

	// Add limit parameter to control page size
	query.Set("limit", strconv.Itoa(defaultPageLimit))

	f.url.RawQuery = query.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// TODO: Implement retry logic for transient failures
	// Execute request
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var registryResp mcpapiv0.ServerListResponse
	if err := json.NewDecoder(resp.Body).Decode(&registryResp); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return registryResp.Servers, registryResp.Metadata.NextCursor, nil
}

// ServerResponseFromInterface converts an interface{} back to ServerResponse.
// This is a helper for the transformer stage.
func ServerResponseFromInterface(i interface{}) (mcpapiv0.ServerResponse, bool) {
	resp, ok := i.(mcpapiv0.ServerResponse)

	return resp, ok
}
