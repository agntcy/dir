// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package consumer

import (
	"context"
	"io"
	"net/http"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// ARD registry endpoint exposed by a local ADS node.
const RegistryURL = "http://localhost:8889/v1/agents"

// Discover asks a local ADS node (via its ARD endpoint) for capabilities matching a task.
func Discover(ctx context.Context) ([]*catalogv1.CatalogEntry, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, RegistryURL, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// read the response
	outBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// unmarshal the response into a ListAgentsResponse
	// note: use protojson since the response is a protobuf message but we're communicating over HTTP with JSON encoding
	var out catalogv1.ListAgentsResponse
	if err := protojson.Unmarshal(outBytes, &out); err != nil {
		return nil, err
	}

	return out.GetResults(), nil
}
