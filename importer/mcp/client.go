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

// Client is a REST API client for the MCP registry.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new MCP registry client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, //nolint:mnd
		},
	}
}

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

// ListServersStream streams servers from the MCP registry as they are fetched.
func (c *Client) ListServersStream(ctx context.Context, filters map[string]string, limit int) (<-chan mcpapiv0.ServerResponse, <-chan error) {
	serverChan := make(chan mcpapiv0.ServerResponse)
	errChan := make(chan error, 1)

	// Validate filters
	for key := range filters {
		if !slices.Contains(supportedFilters, key) {
			close(serverChan)

			errChan <- fmt.Errorf("unsupported filter: %s", key)

			close(errChan)

			return serverChan, errChan
		}
	}

	go func() {
		defer close(serverChan)
		defer close(errChan)

		cursor := ""
		count := 0

		for {
			// Check context cancellation
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()

				return
			default:
			}

			// Fetch one page
			page, nextCursor, err := c.listServersPage(ctx, filters, cursor)
			if err != nil {
				errChan <- err

				return
			}

			// Stream each server as soon as it's available
			for _, server := range page {
				// Check if limit is reached (limit <= 0 means no limit)
				if limit > 0 && count >= limit {
					return
				}

				select {
				case <-ctx.Done():
					errChan <- ctx.Err()

					return
				case serverChan <- server:
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

	return serverChan, errChan
}

// listServersPage fetches a single page of servers from the MCP registry.
func (c *Client) listServersPage(ctx context.Context, filters map[string]string, cursor string) ([]mcpapiv0.ServerResponse, string, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL + "/servers")
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Add filters as query parameters
	query := u.Query()

	for key, value := range filters {
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

	u.RawQuery = query.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// TODO: Implement retry logic for transient failures
	// Execute request
	resp, err := c.httpClient.Do(req)
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
