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

	"github.com/agntcy/dir/importer/types"
	"google.golang.org/protobuf/types/known/structpb"
)

// a2aFileFetcher reads one or more A2A AgentCard JSON objects from a local file.
type a2aFileFetcher struct {
	path string
}

// NewA2AFileFetcher creates a fetcher that reads A2A agent card(s) from a file.
// Supported formats:
//   - A JSON array of AgentCard objects
//   - A single AgentCard object
//
// Each card is emitted as types.SourceItem with Kind SourceKindA2A and the AgentCard as structpb.Struct.
func NewA2AFileFetcher(path string) (*a2aFileFetcher, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("file path is empty")
	}

	return &a2aFileFetcher{path: path}, nil
}

// Fetch reads the file and sends each AgentCard to the output channel.
// Invalid array elements are reported on errCh and skipped.
//
//nolint:cyclop // linear control flow with many ctx/errCh branches
func (f *a2aFileFetcher) Fetch(ctx context.Context) (<-chan types.SourceItem, <-chan error) {
	const chanBuf = 8

	outputCh := make(chan types.SourceItem, chanBuf)
	errCh := make(chan error, chanBuf)

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

		items, fatalErr := decodeA2ARoot(ctx, raw, errCh)
		if fatalErr != nil {
			select {
			case errCh <- fatalErr:
			case <-ctx.Done():
			}

			return
		}

		if len(items) == 0 {
			select {
			case errCh <- errors.New("no A2A agent cards found in file (check earlier errors if this was a JSON array)"):
			case <-ctx.Done():
			}

			return
		}

		for i, cardMap := range items {
			card, err := agentCardStructFromMap(cardMap)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("A2A card index %d: %w", i, err):
				case <-ctx.Done():
					return
				}

				continue
			}

			select {
			case <-ctx.Done():
				return
			case outputCh <- types.A2ASourceItem(card):
			}
		}
	}()

	return outputCh, errCh
}

func agentCardStructFromMap(card map[string]any) (*structpb.Struct, error) {
	if card == nil {
		return nil, errors.New("card is nil")
	}

	name, _ := card["name"].(string)
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("agent card missing non-empty \"name\"")
	}

	st, err := structpb.NewStruct(card)
	if err != nil {
		return nil, fmt.Errorf("agent card as struct: %w", err)
	}

	return st, nil
}

// decodeA2ARoot returns agent card maps. For a JSON array, invalid elements are skipped and
// errors are sent to errCh (best-effort; may drop an error if errCh blocks).
func decodeA2ARoot(ctx context.Context, raw []byte, errCh chan<- error) ([]map[string]any, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, errors.New("file is empty")
	}

	if raw[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("decode JSON array: %w", err)
		}

		out := make([]map[string]any, 0, len(arr))

		for i, elt := range arr {
			var m map[string]any
			if err := json.Unmarshal(elt, &m); err != nil {
				sendA2AErr(ctx, errCh, fmt.Errorf("array index %d: %w", i, err))

				continue
			}

			out = append(out, m)
		}

		return out, nil
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}

	return []map[string]any{obj}, nil
}

func sendA2AErr(ctx context.Context, errCh chan<- error, err error) {
	select {
	case errCh <- err:
	case <-ctx.Done():
	}
}
