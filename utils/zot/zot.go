// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package zot

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("utils/zot")

// buildRegistryURL constructs the registry URL with proper protocol.
func buildRegistryURL(registryURL string, insecure bool) string {
	// If URL already has a protocol, return as-is
	if strings.HasPrefix(registryURL, "http://") || strings.HasPrefix(registryURL, "https://") {
		return registryURL
	}

	// Add appropriate protocol based on insecure flag
	if insecure {
		return "http://" + registryURL
	}

	return "https://" + registryURL
}

// CheckReadiness checks if Zot is ready to serve traffic by querying its /readyz endpoint.
// Returns true if Zot responds with 200 OK, false otherwise.
func CheckReadiness(ctx context.Context, registryAddress string, insecure bool) bool {
	// Build URL to Zot's readiness endpoint
	registryURL := buildRegistryURL(registryAddress, insecure)
	readyzURL := registryURL + "/readyz"

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyzURL, nil)
	if err != nil {
		logger.Debug("Failed to create readiness check request", "error", err, "url", readyzURL)

		return false
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second, //nolint:mnd
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("Zot readiness check failed", "error", err, "url", readyzURL)

		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Debug("Zot readiness check passed", "address", registryAddress)

		return true
	}

	logger.Debug("Zot not ready", "address", registryAddress, "status", resp.StatusCode)

	return false
}
