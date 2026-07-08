// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- NewRemoteRunner / Name ---

func TestNewRemoteRunner_DefaultCLIPath(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})
	if r.cfg.CLIPath != DefaultMCPCLIPath {
		t.Errorf("empty CLIPath should default to %q, got %q", DefaultMCPCLIPath, r.cfg.CLIPath)
	}
}

func TestNewRemoteRunner_CustomCLIPath(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{CLIPath: "/usr/local/bin/mcp-scanner"})
	if r.cfg.CLIPath != "/usr/local/bin/mcp-scanner" {
		t.Errorf("custom CLIPath should be preserved, got %q", r.cfg.CLIPath)
	}
}

func TestRemoteRunner_Name(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})
	if got := r.Name(); got != "remote" {
		t.Errorf("Name() = %q, want %q", got, "remote")
	}
}

// --- Run / no endpoints ---

func TestRemoteRunner_Run_NilRecord_Skipped(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})

	got, err := r.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped || got.SkippedReason == "" {
		t.Errorf("nil record should be skipped with a reason: %+v", got)
	}
}

func TestRemoteRunner_Run_NoConnections_Skipped(t *testing.T) {
	t.Parallel()

	r := NewRemoteRunner(RemoteConfig{})

	st, err := structpb.NewStruct(map[string]any{"schema_version": "1.0.0"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	got, err := r.Run(context.Background(), &corev1.Record{Data: st})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped {
		t.Error("record with no remote-capable connection should be skipped")
	}
}

// --- extractConnectionURLs ---

func TestExtractConnectionURLs_Nil(t *testing.T) {
	t.Parallel()

	if got := extractConnectionURLs(nil); got != nil {
		t.Errorf("nil struct should return nil, got %v", got)
	}
}

func TestExtractConnectionURLs_NoMCPData(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{"other_data": "x"})
	if got := extractConnectionURLs(data); got != nil {
		t.Errorf("no mcp_data should return nil, got %v", got)
	}
}

func TestExtractConnectionURLs_StdioOnly_Excluded(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"mcp_data": map[string]any{
			"connections": []any{
				map[string]any{"type": "stdio", "command": "python server.py"},
			},
		},
	})

	if got := extractConnectionURLs(data); len(got) != 0 {
		t.Errorf("stdio-only connection should produce no URLs, got %v", got)
	}
}

func TestExtractConnectionURLs_SSEFound(t *testing.T) {
	t.Parallel()

	want := "https://example.com/sse"

	data, _ := structpb.NewStruct(map[string]any{
		"mcp_data": map[string]any{
			"connections": []any{
				map[string]any{"type": "sse", "url": want},
			},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 1 || got[0] != want {
		t.Errorf("want [%q], got %v", want, got)
	}
}

func TestExtractConnectionURLs_StreamableHTTPFound(t *testing.T) {
	t.Parallel()

	want := "https://example.com/mcp"

	data, _ := structpb.NewStruct(map[string]any{
		"mcp_data": map[string]any{
			"connections": []any{
				map[string]any{"type": "streamable-http", "url": want},
			},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 1 || got[0] != want {
		t.Errorf("want [%q], got %v", want, got)
	}
}

func TestExtractConnectionURLs_MixedTransports_OnlyRemoteReturned(t *testing.T) {
	t.Parallel()

	want := "https://example.com/mcp"

	data, _ := structpb.NewStruct(map[string]any{
		"mcp_data": map[string]any{
			"connections": []any{
				map[string]any{"type": "stdio", "command": "python server.py"},
				map[string]any{"type": "streamable-http", "url": want},
			},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 1 || got[0] != want {
		t.Errorf("want only the remote-capable URL [%q], got %v", want, got)
	}
}

func TestExtractConnectionURLs_RemoteTypeWithoutURL_Excluded(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"mcp_data": map[string]any{
			"connections": []any{
				map[string]any{"type": "sse"},
			},
		},
	})

	if got := extractConnectionURLs(data); len(got) != 0 {
		t.Errorf("sse connection without a url should produce no URLs, got %v", got)
	}
}

func TestExtractConnectionURLs_MultipleRemoteConnections(t *testing.T) {
	t.Parallel()

	data, _ := structpb.NewStruct(map[string]any{
		"mcp_data": map[string]any{
			"connections": []any{
				map[string]any{"type": "sse", "url": "https://a.example.com/sse"},
				map[string]any{"type": "streamable-http", "url": "https://b.example.com/mcp"},
			},
		},
	})

	got := extractConnectionURLs(data)
	if len(got) != 2 {
		t.Fatalf("want 2 URLs, got %d: %v", len(got), got)
	}
}

// --- tagFindings ---

func TestTagFindings_Empty(t *testing.T) {
	t.Parallel()

	if got := tagFindings("remote", "https://example.com", nil); got != nil {
		t.Errorf("empty findings should return nil, got %v", got)
	}
}

func TestTagFindings_PrefixesMessage(t *testing.T) {
	t.Parallel()

	in := []Finding{{Severity: SeverityError, Message: "prompt injection detected"}}

	got := tagFindings("prompts", "https://example.com/mcp", in)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}

	want := "[prompts https://example.com/mcp] prompt injection detected"
	if got[0].Message != want {
		t.Errorf("want %q, got %q", want, got[0].Message)
	}

	if got[0].Severity != SeverityError {
		t.Errorf("severity should be preserved, got %q", got[0].Severity)
	}
}

func TestTagFindings_PreservesOrderAndCount(t *testing.T) {
	t.Parallel()

	in := []Finding{
		{Severity: SeverityError, Message: "first"},
		{Severity: SeverityWarning, Message: "second"},
	}

	got := tagFindings("resources", "https://example.com", in)
	if len(got) != len(in) {
		t.Fatalf("want %d findings, got %d", len(in), len(got))
	}

	for i, f := range got {
		if f.Severity != in[i].Severity {
			t.Errorf("finding %d: severity changed, want %q got %q", i, in[i].Severity, f.Severity)
		}
	}
}

// --- extractRemoteEndpoints ---
//
// extractRemoteEndpoints itself is exercised indirectly via the Run() tests
// above. Like extractSourceInfo in mcp.go, it is a thin decode-and-delegate
// wrapper around record.Decode(); the interesting per-connection logic lives
// in extractConnectionURLs, which is unit tested directly above against
// hand-built structpb.Struct values (mirroring extractSubfolder's tests in
// mcp_test.go) rather than through a full OASF SDK decode of a synthetic record.

func TestExtractRemoteEndpoints_NilRecord(t *testing.T) {
	t.Parallel()

	if got := extractRemoteEndpoints(nil); got != nil {
		t.Errorf("nil record should return nil, got %v", got)
	}
}
