// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package safefetch_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agntcy/dir/utils/safefetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Get_RejectsPlainHTTPByDefault(t *testing.T) {
	client := safefetch.New()

	_, err := client.Get(t.Context(), "http://example.com/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestClient_Get_RejectsLoopbackAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := safefetch.New(safefetch.WithAllowHTTP())

	_, err := client.Get(t.Context(), server.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, safefetch.ErrDisallowedAddress)
}

func TestClient_Get_RejectsLinkLocalMetadataAddress(t *testing.T) {
	client := safefetch.New(safefetch.WithAllowHTTP())

	_, err := client.Get(t.Context(), "http://169.254.169.254/latest/meta-data/")
	require.Error(t, err)
	assert.ErrorIs(t, err, safefetch.ErrDisallowedAddress)
}

func TestClient_Get_SchemeGate(t *testing.T) {
	tests := []struct {
		name      string
		opts      []safefetch.Option
		rawURL    string
		wantGated bool
	}{
		{name: "https passes the gate", rawURL: "https://127.0.0.1:9/"},
		{name: "http is gated by default", rawURL: "http://127.0.0.1:9/", wantGated: true},
		{name: "http passes the gate when allowed", opts: []safefetch.Option{safefetch.WithAllowHTTP()}, rawURL: "http://127.0.0.1:9/"},
		{name: "ftp is gated by default", rawURL: "ftp://127.0.0.1:9/", wantGated: true},
		{name: "ftp stays gated when http is allowed", opts: []safefetch.Option{safefetch.WithAllowHTTP()}, rawURL: "ftp://127.0.0.1:9/", wantGated: true},
		{name: "file stays gated when http is allowed", opts: []safefetch.Option{safefetch.WithAllowHTTP()}, rawURL: "file:///etc/hosts", wantGated: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := safefetch.New(tt.opts...)

			_, err := client.Get(t.Context(), tt.rawURL)
			require.Error(t, err)

			if tt.wantGated {
				assert.Contains(t, err.Error(), "scheme")

				return
			}

			assert.NotContains(t, err.Error(), "scheme")
			require.ErrorIs(t, err, safefetch.ErrDisallowedAddress)
		})
	}
}
