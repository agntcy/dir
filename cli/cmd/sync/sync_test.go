// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"bytes"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSearchOutput_EmptyInput tests parsing empty input.
func TestParseSearchOutput_EmptyInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "empty array",
			input:       "[]",
			expectError: false,
		},
		{
			name:        "null",
			input:       "null",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parseSearchOutput(reader)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				if tt.input == "[]" {
					assert.Empty(t, result)
				}
			}
		})
	}
}

// TestParseSearchOutput_ValidJSON tests parsing valid JSON input.
func TestParseSearchOutput_ValidJSON(t *testing.T) {
	input := `[
		{
			"record_ref": {"cid": "cid1"},
			"peer": {
				"addrs": ["http://peer1.example.com"]
			}
		},
		{
			"record_ref": {"cid": "cid2"},
			"peer": {
				"addrs": ["http://peer2.example.com"]
			}
		}
	]`

	reader := strings.NewReader(input)
	result, err := parseSearchOutput(reader)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "cid1", result[0].GetRecordRef().GetCid())
	assert.Equal(t, "cid2", result[1].GetRecordRef().GetCid())
}

// TestParseSearchOutput_InvalidJSON tests error handling for invalid JSON.
func TestParseSearchOutput_InvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "malformed JSON",
			input: `{"invalid": "json"`,
		},
		{
			name:  "not an array",
			input: `{"key": "value"}`,
		},
		{
			name:  "random text",
			input: "not json at all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := parseSearchOutput(reader)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "failed to parse JSON")
		})
	}
}

// TestParseSearchOutput_ComplexJSON tests parsing complex search results.
func TestParseSearchOutput_ComplexJSON(t *testing.T) {
	input := `[
		{
			"record_ref": {"cid": "bafyabc123"},
			"peer": {
				"id": "peer1",
				"addrs": ["http://api1.example.com", "http://api2.example.com"]
			},
			"queries": ["skill:AI"],
			"score": 2
		}
	]`

	reader := strings.NewReader(input)
	result, err := parseSearchOutput(reader)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "bafyabc123", result[0].GetRecordRef().GetCid())
	assert.Equal(t, "peer1", result[0].GetPeer().GetId())
	assert.Len(t, result[0].GetPeer().GetAddrs(), 2)
}

// TestGroupResultsByAPIAddress_EmptyInput tests grouping empty results.
func TestGroupResultsByAPIAddress_EmptyInput(t *testing.T) {
	result := groupResultsByAPIAddress(nil)
	assert.Empty(t, result)

	result = groupResultsByAPIAddress([]*routingv1.SearchResponse{})
	assert.Empty(t, result)
}

// TestGroupResultsByAPIAddress_SingleResult tests grouping single result.
func TestGroupResultsByAPIAddress_SingleResult(t *testing.T) {
	results := []*routingv1.SearchResponse{
		{
			RecordRef: &corev1.RecordRef{Cid: "cid1"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://api1.example.com"},
			},
		},
	}

	grouped := groupResultsByAPIAddress(results)

	assert.Len(t, grouped, 1)
	assert.Contains(t, grouped, "http://api1.example.com")
	assert.Equal(t, "http://api1.example.com", grouped["http://api1.example.com"].APIAddress)
	assert.Equal(t, []string{"cid1"}, grouped["http://api1.example.com"].CIDs)
}

// TestGroupResultsByAPIAddress_MultipleSamePeer tests grouping multiple records from same peer.
func TestGroupResultsByAPIAddress_MultipleSamePeer(t *testing.T) {
	results := []*routingv1.SearchResponse{
		{
			RecordRef: &corev1.RecordRef{Cid: "cid1"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://api1.example.com"},
			},
		},
		{
			RecordRef: &corev1.RecordRef{Cid: "cid2"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://api1.example.com"},
			},
		},
		{
			RecordRef: &corev1.RecordRef{Cid: "cid3"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://api1.example.com"},
			},
		},
	}

	grouped := groupResultsByAPIAddress(results)

	assert.Len(t, grouped, 1)
	peerInfo := grouped["http://api1.example.com"]
	assert.Equal(t, "http://api1.example.com", peerInfo.APIAddress)
	assert.Equal(t, []string{"cid1", "cid2", "cid3"}, peerInfo.CIDs)
}

// TestGroupResultsByAPIAddress_MultipleDifferentPeers tests grouping records from different peers.
func TestGroupResultsByAPIAddress_MultipleDifferentPeers(t *testing.T) {
	results := []*routingv1.SearchResponse{
		{
			RecordRef: &corev1.RecordRef{Cid: "cid1"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://api1.example.com"},
			},
		},
		{
			RecordRef: &corev1.RecordRef{Cid: "cid2"},
			Peer: &routingv1.Peer{
				Id:    "peer2",
				Addrs: []string{"http://api2.example.com"},
			},
		},
		{
			RecordRef: &corev1.RecordRef{Cid: "cid3"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://api1.example.com"},
			},
		},
	}

	grouped := groupResultsByAPIAddress(results)

	assert.Len(t, grouped, 2)

	peer1Info := grouped["http://api1.example.com"]
	assert.Equal(t, "http://api1.example.com", peer1Info.APIAddress)
	assert.Equal(t, []string{"cid1", "cid3"}, peer1Info.CIDs)

	peer2Info := grouped["http://api2.example.com"]
	assert.Equal(t, "http://api2.example.com", peer2Info.APIAddress)
	assert.Equal(t, []string{"cid2"}, peer2Info.CIDs)
}

// TestGroupResultsByAPIAddress_NoPeerInfo tests skipping results without peer info.
func TestGroupResultsByAPIAddress_NoPeerInfo(t *testing.T) {
	results := []*routingv1.SearchResponse{
		{
			RecordRef: &corev1.RecordRef{Cid: "cid1"},
			Peer:      nil, // No peer info
		},
		{
			RecordRef: &corev1.RecordRef{Cid: "cid2"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{}, // No addresses
			},
		},
		{
			RecordRef: &corev1.RecordRef{Cid: "cid3"},
			Peer: &routingv1.Peer{
				Id:    "peer2",
				Addrs: []string{"http://api1.example.com"},
			},
		},
	}

	grouped := groupResultsByAPIAddress(results)

	// Only the third result should be included
	assert.Len(t, grouped, 1)
	assert.Contains(t, grouped, "http://api1.example.com")
	assert.Equal(t, []string{"cid3"}, grouped["http://api1.example.com"].CIDs)
}

// TestGroupResultsByAPIAddress_MultipleAddresses tests using first address only.
func TestGroupResultsByAPIAddress_MultipleAddresses(t *testing.T) {
	results := []*routingv1.SearchResponse{
		{
			RecordRef: &corev1.RecordRef{Cid: "cid1"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://api1.example.com", "http://api2.example.com", "http://api3.example.com"},
			},
		},
	}

	grouped := groupResultsByAPIAddress(results)

	// Should use the first address
	assert.Len(t, grouped, 1)
	assert.Contains(t, grouped, "http://api1.example.com")
	assert.NotContains(t, grouped, "http://api2.example.com")
	assert.NotContains(t, grouped, "http://api3.example.com")
}

// TestGroupResultsByAPIAddress_MixedScenario tests complex real-world scenario.
func TestGroupResultsByAPIAddress_MixedScenario(t *testing.T) {
	results := []*routingv1.SearchResponse{
		// Peer 1 - multiple CIDs
		{
			RecordRef: &corev1.RecordRef{Cid: "cid1"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://peer1.com"},
			},
		},
		{
			RecordRef: &corev1.RecordRef{Cid: "cid2"},
			Peer: &routingv1.Peer{
				Id:    "peer1",
				Addrs: []string{"http://peer1.com"},
			},
		},
		// Peer 2 - single CID
		{
			RecordRef: &corev1.RecordRef{Cid: "cid3"},
			Peer: &routingv1.Peer{
				Id:    "peer2",
				Addrs: []string{"http://peer2.com"},
			},
		},
		// No peer info - should be skipped
		{
			RecordRef: &corev1.RecordRef{Cid: "cid4"},
			Peer:      nil,
		},
		// Peer 3 - single CID
		{
			RecordRef: &corev1.RecordRef{Cid: "cid5"},
			Peer: &routingv1.Peer{
				Id:    "peer3",
				Addrs: []string{"http://peer3.com"},
			},
		},
	}

	grouped := groupResultsByAPIAddress(results)

	assert.Len(t, grouped, 3)
	assert.Equal(t, []string{"cid1", "cid2"}, grouped["http://peer1.com"].CIDs)
	assert.Equal(t, []string{"cid3"}, grouped["http://peer2.com"].CIDs)
	assert.Equal(t, []string{"cid5"}, grouped["http://peer3.com"].CIDs)
}

// TestPeerSyncInfo_Structure tests the PeerSyncInfo structure.
func TestPeerSyncInfo_Structure(t *testing.T) {
	info := PeerSyncInfo{
		APIAddress: "http://example.com",
		CIDs:       []string{"cid1", "cid2", "cid3"},
	}

	assert.Equal(t, "http://example.com", info.APIAddress)
	assert.Len(t, info.CIDs, 3)
	assert.Contains(t, info.CIDs, "cid1")
	assert.Contains(t, info.CIDs, "cid2")
	assert.Contains(t, info.CIDs, "cid3")
}

// TestParseSearchOutput_LargeDataset tests parsing large number of results.
func TestParseSearchOutput_LargeDataset(t *testing.T) {
	// Build a large JSON array
	var builder strings.Builder
	builder.WriteString("[")

	for i := range 100 {
		if i > 0 {
			builder.WriteString(",")
		}

		builder.WriteString(`{
			"recordRef": {"cid": "cid`)
		builder.WriteString(strings.Repeat("a", i))
		builder.WriteString(`"},
			"peer": {"addrs": ["http://peer`)
		builder.WriteString(strings.Repeat("a", i%10))
		builder.WriteString(`.com"]}
		}`)
	}

	builder.WriteString("]")

	reader := strings.NewReader(builder.String())
	result, err := parseSearchOutput(reader)

	require.NoError(t, err)
	assert.Len(t, result, 100)
}

// TestGroupResultsByAPIAddress_LargeDataset tests grouping large number of results.
func TestGroupResultsByAPIAddress_LargeDataset(t *testing.T) {
	results := make([]*routingv1.SearchResponse, 100)

	// Create 100 results distributed across 10 peers
	for i := range 100 {
		peerNum := i % 10
		results[i] = &routingv1.SearchResponse{
			RecordRef: &corev1.RecordRef{Cid: "cid" + strings.Repeat("a", i)},
			Peer: &routingv1.Peer{
				Id:    "peer" + strings.Repeat("a", peerNum),
				Addrs: []string{"http://peer" + strings.Repeat("a", peerNum) + ".com"},
			},
		}
	}

	grouped := groupResultsByAPIAddress(results)

	// Should have 10 unique API addresses
	assert.Len(t, grouped, 10)

	// Each peer should have 10 CIDs
	for _, info := range grouped {
		assert.Len(t, info.CIDs, 10)
	}
}

// TestParseSearchOutput_SpecialCharacters tests handling special characters in JSON.
func TestParseSearchOutput_SpecialCharacters(t *testing.T) {
	input := `[
		{
			"record_ref": {"cid": "cid-with-special-chars-!@#$%^&*()"},
			"peer": {
				"id": "peer/with\\slashes\"and'quotes",
				"addrs": ["http://api.example.com/path?query=value&foo=bar"]
			}
		}
	]`

	reader := strings.NewReader(input)
	result, err := parseSearchOutput(reader)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result[0].GetRecordRef().GetCid(), "special-chars")
}

// TestCommand_Initialization tests that sync command is properly initialized.
func TestCommand_Initialization(t *testing.T) {
	assert.NotNil(t, Command)
	assert.Equal(t, "sync", Command.Use)
	assert.NotEmpty(t, Command.Short)
	assert.NotEmpty(t, Command.Long)
}

// TestCreateCmd_Initialization tests create subcommand initialization.
func TestCreateCmd_Initialization(t *testing.T) {
	assert.NotNil(t, createCmd)
	assert.Equal(t, "create <remote-directory-url>", createCmd.Use)
	assert.NotEmpty(t, createCmd.Short)
	assert.NotEmpty(t, createCmd.Long)
	assert.NotNil(t, createCmd.Args)
	assert.NotNil(t, createCmd.RunE)

	// Check that examples are in the Long description
	assert.Contains(t, createCmd.Long, "dirctl routing search")
	assert.Contains(t, createCmd.Long, "--output json")
}

// TestListCmd_Initialization tests list subcommand initialization.
func TestListCmd_Initialization(t *testing.T) {
	assert.NotNil(t, listCmd)
	assert.Equal(t, "list", listCmd.Use)
	assert.NotEmpty(t, listCmd.Short)
	assert.NotNil(t, listCmd.RunE)
}

// TestStatusCmd_Initialization tests status subcommand initialization.
func TestStatusCmd_Initialization(t *testing.T) {
	assert.NotNil(t, statusCmd)
	assert.Equal(t, "status <sync-id>", statusCmd.Use)
	assert.NotEmpty(t, statusCmd.Short)
	assert.NotNil(t, statusCmd.Args)
	assert.NotNil(t, statusCmd.RunE)
}

// TestDeleteCmd_Initialization tests delete subcommand initialization.
func TestDeleteCmd_Initialization(t *testing.T) {
	assert.NotNil(t, deleteCmd)
	assert.Equal(t, "delete <sync-id>", deleteCmd.Use)
	assert.NotEmpty(t, deleteCmd.Short)
	assert.NotNil(t, deleteCmd.Args)
	assert.NotNil(t, deleteCmd.RunE)
}

// TestCreateCmd_ArgsValidation tests argument validation for create command.
func TestCreateCmd_ArgsValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		stdin       bool
		expectError bool
	}{
		{
			name:        "no args with stdin",
			args:        []string{},
			stdin:       true,
			expectError: false,
		},
		{
			name:        "one arg without stdin",
			args:        []string{"http://example.com"},
			stdin:       false,
			expectError: false,
		},
		{
			name:        "no args without stdin",
			args:        []string{},
			stdin:       false,
			expectError: true,
		},
		{
			name:        "multiple args without stdin",
			args:        []string{"http://example.com", "extra"},
			stdin:       false,
			expectError: true,
		},
		{
			name:        "args with stdin flag",
			args:        []string{"http://example.com"},
			stdin:       true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore opts
			oldStdin := opts.Stdin

			defer func() { opts.Stdin = oldStdin }()

			opts.Stdin = tt.stdin

			cmd := &cobra.Command{}
			err := createCmd.Args(cmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseSearchOutput_ReadError tests handling of read errors.
func TestParseSearchOutput_ReadError(t *testing.T) {
	// Create a reader that always errors
	reader := &errorReader{}
	result, err := parseSearchOutput(reader)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "error reading input")
}

// errorReader is a test helper that always returns an error.
type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, assert.AnError
}

// TestGroupResultsByAPIAddress_EdgeCases tests edge cases in grouping.
func TestGroupResultsByAPIAddress_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		results  []*routingv1.SearchResponse
		expected int // expected number of groups
	}{
		{
			name: "empty CID",
			results: []*routingv1.SearchResponse{
				{
					RecordRef: &corev1.RecordRef{Cid: ""},
					Peer: &routingv1.Peer{
						Addrs: []string{"http://api.example.com"},
					},
				},
			},
			expected: 1,
		},
		{
			name: "nil recordRef",
			results: []*routingv1.SearchResponse{
				{
					RecordRef: nil,
					Peer: &routingv1.Peer{
						Addrs: []string{"http://api.example.com"},
					},
				},
			},
			expected: 1, // Creates group with empty CID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			require.NotPanics(t, func() {
				grouped := groupResultsByAPIAddress(tt.results)
				if tt.expected >= 0 {
					assert.Len(t, grouped, tt.expected)
				}
			})
		})
	}
}

// TestParseSearchOutput_WithBytes tests using bytes.Buffer.
func TestParseSearchOutput_WithBytes(t *testing.T) {
	// Use snake_case JSON field names (record_ref, not recordRef)
	input := []byte(`[{"record_ref": {"cid": "test-cid-123"}, "peer": {"addrs": ["http://test.com"]}}]`)
	buffer := bytes.NewBuffer(input)

	result, err := parseSearchOutput(buffer)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	// The CID should be set
	assert.NotNil(t, result[0].GetRecordRef())
	assert.Equal(t, "test-cid-123", result[0].GetRecordRef().GetCid())
}
