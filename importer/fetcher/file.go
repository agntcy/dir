// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

// fileFetcher reads MCP registry-style server definitions from a local JSON file.
type fileFetcher struct {
	path string
}

// NewFileFetcher creates a fetcher that reads one or more MCP servers from a file.
// Supported formats:
//   - A JSON array of ServerResponse
//   - A single bare ServerJSON object (wrapped as ServerResponse)
func NewFileFetcher(path string) (*fileFetcher, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("file path is empty")
	}

	return &fileFetcher{path: path}, nil
}

// Fetch reads the file and sends each decoded server to the output channel.
func (f *fileFetcher) Fetch(ctx context.Context) (<-chan mcpapiv0.ServerResponse, <-chan error) {
	outputCh := make(chan mcpapiv0.ServerResponse, 8) //nolint:mnd
	errCh := make(chan error, 1)

	go func() {
		defer close(outputCh)
		defer close(errCh)

		raw, err := os.ReadFile(f.path)
		if err != nil {
			select {
			case errCh <- fmt.Errorf("read file: %w", err):
			case <-ctx.Done():
			}

			return
		}

		raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")) // UTF-8 BOM

		servers, err := decodeServerResponses(raw)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}

			return
		}

		if len(servers) == 0 {
			select {
			case errCh <- errors.New("no MCP servers found in file"):
			case <-ctx.Done():
			}

			return
		}

		for _, srv := range servers {
			select {
			case <-ctx.Done():
				return
			case outputCh <- srv:
			}
		}
	}()

	return outputCh, errCh
}

func decodeServerResponses(raw []byte) ([]mcpapiv0.ServerResponse, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, errors.New("file is empty")
	}

	// JSON array of ServerResponse
	if raw[0] == '[' {
		var batch []mcpapiv0.ServerResponse
		if err := json.Unmarshal(raw, &batch); err != nil {
			return nil, fmt.Errorf("decode JSON array: %w", err)
		}

		out := make([]mcpapiv0.ServerResponse, 0, len(batch))
		for _, item := range batch {
			if item.Server.Name == "" {
				continue
			}

			out = append(out, item)
		}

		if len(out) == 0 {
			return nil, errors.New("no valid servers in JSON array (missing server.name)")
		}

		return out, nil
	}

	// Single JSON object: bare ServerJSON only (registry server.json shape)
	var bare mcpapiv0.ServerJSON
	if err := json.Unmarshal(raw, &bare); err == nil && bare.Name != "" {
		return []mcpapiv0.ServerResponse{{Server: bare}}, nil
	}

	return nil, errors.New("could not parse file as JSON array of servers or bare server.json")
}
