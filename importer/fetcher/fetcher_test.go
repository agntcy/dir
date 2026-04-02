// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agntcy/dir/importer/types"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
)

func TestNewMCPRegistryFetcher_InvalidFilter(t *testing.T) {
	t.Parallel()

	_, err := NewMCPRegistryFetcher("https://registry.example.com", map[string]string{"bad": "v"}, 0)
	if err == nil {
		t.Fatal("expected error for unsupported filter key")
	}
}

func TestNewMCPRegistryFetcher_ValidFilters(t *testing.T) {
	t.Parallel()

	f, err := NewMCPRegistryFetcher("https://registry.example.com", map[string]string{
		"search": "foo",
		"limit":  "10",
	}, 0)
	if err != nil {
		t.Fatalf("NewMCPRegistryFetcher: %v", err)
	}

	if f.url.Host != "registry.example.com" {
		t.Errorf("host = %q", f.url.Host)
	}

	if f.url.Path != "/servers" {
		t.Errorf("path = %q, want /servers", f.url.Path)
	}
}

func TestFetch_SinglePage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers" {
			http.NotFound(w, r)

			return
		}

		_ = json.NewEncoder(w).Encode(mcpapiv0.ServerListResponse{
			Servers: []mcpapiv0.ServerResponse{
				{Server: mcpapiv0.ServerJSON{Name: "a", Version: "1"}},
				{Server: mcpapiv0.ServerJSON{Name: "b", Version: "2"}},
			},
			Metadata: mcpapiv0.Metadata{NextCursor: ""},
		})
	}))
	t.Cleanup(srv.Close)

	f, err := NewMCPRegistryFetcher(srv.URL, nil, 0)
	if err != nil {
		t.Fatalf("NewMCPRegistryFetcher: %v", err)
	}

	f.httpClient = srv.Client()

	ctx := context.Background()
	outCh, errCh := f.Fetch(ctx)

	var got int

	for range outCh {
		got++
	}

	for e := range errCh {
		if e != nil {
			t.Fatalf("unexpected err: %v", e)
		}
	}

	if got != 2 {
		t.Errorf("got %d servers, want 2", got)
	}
}

func TestFetch_RespectsLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mcpapiv0.ServerListResponse{
			Servers: []mcpapiv0.ServerResponse{
				{Server: mcpapiv0.ServerJSON{Name: "a", Version: "1"}},
				{Server: mcpapiv0.ServerJSON{Name: "b", Version: "1"}},
				{Server: mcpapiv0.ServerJSON{Name: "c", Version: "1"}},
			},
			Metadata: mcpapiv0.Metadata{NextCursor: ""},
		})
	}))
	t.Cleanup(srv.Close)

	f, err := NewMCPRegistryFetcher(srv.URL, nil, 2)
	if err != nil {
		t.Fatalf("NewMCPRegistryFetcher: %v", err)
	}

	f.httpClient = srv.Client()

	ctx := context.Background()
	outCh, errCh := f.Fetch(ctx)

	var got int

	for range outCh {
		got++
	}

	for e := range errCh {
		if e != nil {
			t.Fatalf("unexpected err: %v", e)
		}
	}

	if got != 2 {
		t.Errorf("got %d servers, want 2 (limit)", got)
	}
}

func TestFetch_Pagination(t *testing.T) {
	t.Parallel()

	calls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		if calls == 1 {
			_ = json.NewEncoder(w).Encode(mcpapiv0.ServerListResponse{
				Servers: []mcpapiv0.ServerResponse{
					{Server: mcpapiv0.ServerJSON{Name: "p1", Version: "1"}},
				},
				Metadata: mcpapiv0.Metadata{NextCursor: "next-page"},
			})

			return
		}

		if r.URL.Query().Get("cursor") != "next-page" {
			t.Errorf("second request missing cursor, query=%v", r.URL.Query())
		}

		_ = json.NewEncoder(w).Encode(mcpapiv0.ServerListResponse{
			Servers: []mcpapiv0.ServerResponse{
				{Server: mcpapiv0.ServerJSON{Name: "p2", Version: "1"}},
			},
			Metadata: mcpapiv0.Metadata{NextCursor: ""},
		})
	}))
	t.Cleanup(srv.Close)

	f, err := NewMCPRegistryFetcher(srv.URL, nil, 0)
	if err != nil {
		t.Fatalf("NewMCPRegistryFetcher: %v", err)
	}

	f.httpClient = srv.Client()

	ctx := context.Background()
	outCh, errCh := f.Fetch(ctx)

	var names []string

	for s := range outCh {
		if s.Kind != types.SourceKindMCP {
			t.Fatalf("Kind = %v, want MCP", s.Kind)
		}

		names = append(names, s.MCP.Server.Name)
	}

	for e := range errCh {
		if e != nil {
			t.Fatalf("unexpected err: %v", e)
		}
	}

	if calls != 2 {
		t.Errorf("HTTP calls = %d, want 2", calls)
	}

	if len(names) != 2 || names[0] != "p1" || names[1] != "p2" {
		t.Errorf("servers = %v", names)
	}
}

func TestFetch_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	f, err := NewMCPRegistryFetcher(srv.URL, nil, 0)
	if err != nil {
		t.Fatalf("NewMCPRegistryFetcher: %v", err)
	}

	f.httpClient = srv.Client()

	ctx := context.Background()
	outCh, errCh := f.Fetch(ctx)

	for range outCh {
		t.Fatal("expected no data")
	}

	var gotErr error

	for e := range errCh {
		if e != nil {
			gotErr = e
		}
	}

	if gotErr == nil {
		t.Fatal("expected error from non-200 response")
	}
}
