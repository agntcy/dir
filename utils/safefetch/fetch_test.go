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
