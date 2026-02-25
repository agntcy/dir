// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
	"github.com/agntcy/dir/importer/config"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestExtractNameVersionFromSource(t *testing.T) {
	tests := []struct {
		name     string
		source   any
		expected string
	}{
		{
			name: "valid ServerResponse",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server",
					Version: "1.0.0",
				},
			},
			expected: "test-server@1.0.0",
		},
		{
			name: "ServerResponse with empty name",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "",
					Version: "1.0.0",
				},
			},
			expected: "", // Empty name should return empty string
		},
		{
			name: "ServerResponse with empty version",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server",
					Version: "",
				},
			},
			expected: "", // Empty version should return empty string
		},
		{
			name: "ServerResponse with both empty",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "",
					Version: "",
				},
			},
			expected: "",
		},
		{
			name: "pointer to ServerResponse",
			source: &mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "pointer-server",
					Version: "2.0.0",
				},
			},
			expected: "pointer-server@2.0.0",
		},
		{
			name:     "nil pointer to ServerResponse",
			source:   (*mcpapiv0.ServerResponse)(nil),
			expected: "",
		},
		{
			name:     "wrong type - string",
			source:   "not a server response",
			expected: "",
		},
		{
			name:     "wrong type - int",
			source:   42,
			expected: "",
		},
		{
			name:     "wrong type - nil",
			source:   nil,
			expected: "",
		},
		{
			name: "ServerResponse with special characters in name",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "test-server-v2",
					Version: "1.2.3-beta",
				},
			},
			expected: "test-server-v2@1.2.3-beta",
		},
		{
			name: "ServerResponse with long version",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "server",
					Version: "1.2.3.4.5.6",
				},
			},
			expected: "server@1.2.3.4.5.6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNameVersionFromSource(tt.source)

			if result != tt.expected {
				t.Errorf("extractNameVersionFromSource() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// mockDedupStreamResult is a StreamResult that yields one CID then closes.
type mockDedupStreamResult struct {
	resCh  chan *searchv1.SearchCIDsResponse
	errCh  chan error
	doneCh chan struct{}
}

func (m *mockDedupStreamResult) ResCh() <-chan *searchv1.SearchCIDsResponse { return m.resCh }
func (m *mockDedupStreamResult) ErrCh() <-chan error                           { return m.errCh }
func (m *mockDedupStreamResult) DoneCh() <-chan struct{}                       { return m.doneCh }

// mockDedupClient implements config.ClientInterface for MCPDuplicateChecker tests.
// buildCache calls SearchCIDs once per module (integration/mcp, runtime/mcp).
// For first call return one CID so PullBatch is used; for second call return empty.
type mockDedupClient struct {
	searchCallCount int
	cidsFirstCall   []string
	pullRecords     []*corev1.Record
}

func (m *mockDedupClient) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	return nil, nil
}

func (m *mockDedupClient) SearchCIDs(ctx context.Context, req *searchv1.SearchCIDsRequest) (streaming.StreamResult[searchv1.SearchCIDsResponse], error) {
	m.searchCallCount++
	var cids []string
	if m.searchCallCount == 1 {
		cids = m.cidsFirstCall
	}
	// Unbuffered resCh so each send blocks until buildCache receives; then close doneCh so it exits
	resCh := make(chan *searchv1.SearchCIDsResponse)
	errCh := make(chan error, 1)
	doneCh := make(chan struct{})

	go func() {
		for _, cid := range cids {
			resCh <- &searchv1.SearchCIDsResponse{RecordCid: cid}
		}
		close(doneCh)
		// Do not close resCh or errCh - buildCache exits on doneCh; closing would send nil to consumer
	}()

	return &mockDedupStreamResult{resCh: resCh, errCh: errCh, doneCh: doneCh}, nil
}

func (m *mockDedupClient) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
	return m.pullRecords, nil
}

func TestNewMCPDuplicateChecker_Success(t *testing.T) {
	ctx := context.Background()
	client := &mockDedupClient{
		cidsFirstCall: []string{"cid-server1"},
		pullRecords: []*corev1.Record{
			{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name":    structpb.NewStringValue("server1"),
						"version": structpb.NewStringValue("1.0.0"),
					},
				},
			},
		},
	}

	checker, err := NewMCPDuplicateChecker(ctx, client, false)
	if err != nil {
		t.Fatalf("NewMCPDuplicateChecker() error = %v", err)
	}
	if checker == nil {
		t.Fatal("NewMCPDuplicateChecker() returned nil checker")
	}

	// FilterDuplicates: send duplicate (server1@1.0.0) and non-duplicate (server2@2.0.0)
	result := &Result{}
	inputCh := make(chan any, 2)
	inputCh <- mcpapiv0.ServerResponse{
		Server: mcpapiv0.ServerJSON{Name: "server1", Version: "1.0.0"},
	}
	inputCh <- mcpapiv0.ServerResponse{
		Server: mcpapiv0.ServerJSON{Name: "server2", Version: "2.0.0"},
	}
	close(inputCh)

	outputCh := checker.FilterDuplicates(ctx, inputCh, result)

	outCount := 0
	for v := range outputCh {
		if v != nil {
			outCount++
		}
	}

	// Duplicate checker only increments TotalRecords for items it sees (and SkippedCount for duplicates)
	if result.TotalRecords != 1 {
		t.Errorf("TotalRecords = %d, want 1 (one duplicate counted)", result.TotalRecords)
	}
	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1 (one duplicate)", result.SkippedCount)
	}
	if outCount != 1 {
		t.Errorf("output count = %d, want 1 (only non-duplicate passes through)", outCount)
	}
}

func TestNewMCPDuplicateChecker_EmptyCache(t *testing.T) {
	ctx := context.Background()
	client := &mockDedupClient{
		cidsFirstCall: []string{},
		pullRecords:   []*corev1.Record{},
	}

	checker, err := NewMCPDuplicateChecker(ctx, client, false)
	if err != nil {
		t.Fatalf("NewMCPDuplicateChecker() error = %v", err)
	}

	result := &Result{}
	inputCh := make(chan any, 1)
	inputCh <- mcpapiv0.ServerResponse{
		Server: mcpapiv0.ServerJSON{Name: "server1", Version: "1.0.0"},
	}
	close(inputCh)

	outputCh := checker.FilterDuplicates(ctx, inputCh, result)
	outCount := 0
	for range outputCh {
		outCount++
	}

	if result.SkippedCount != 0 {
		t.Errorf("SkippedCount = %d, want 0 (empty cache)", result.SkippedCount)
	}
	if outCount != 1 {
		t.Errorf("output count = %d, want 1", outCount)
	}
}

// Ensure mockDedupClient implements config.ClientInterface.
var _ config.ClientInterface = (*mockDedupClient)(nil)
