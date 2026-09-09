// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/content/oci"
)

func TestDeleteFromOCIStore_NotFound(t *testing.T) {
	// Create an empty OCI store which will return errdef.ErrNotFound for any non-existent CID
	tmpDir := t.TempDir()
	ociStore, err := oci.New(tmpDir)
	require.NoError(t, err)

	s := &store{
		repo: ociStore,
	}

	// Attempting to delete a non-existent CID should return nil (success)
	err = s.deleteFromOCIStore(context.Background(), "bafybeib76j4m6pfr253a652y5z2ndifewfhztdv73cyb25yhtv4b3445nm")
	assert.NoError(t, err)
}

func TestIsReady_RemoteRegistry(t *testing.T) {
	tests := []struct {
		name       string
		baseStatus int
		wantReady  bool
	}{
		{name: "registry available", baseStatus: http.StatusOK, wantReady: true},
		{name: "repository absent", baseStatus: http.StatusNotFound, wantReady: true},
		{name: "registry unavailable", baseStatus: http.StatusInternalServerError, wantReady: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				mu    sync.Mutex
				paths []string
			)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()

				paths = append(paths, r.URL.Path)

				w.WriteHeader(tt.baseStatus)
			}))
			defer srv.Close()

			cfg := ociconfig.Config{
				RegistryAddress: strings.TrimPrefix(srv.URL, "http://"),
				RepositoryName:  "dir",
				Insecure:        true,
			}

			repo, err := NewORASRepository(cfg)
			require.NoError(t, err)

			s := &store{repo: repo, config: cfg}

			assert.Equal(t, tt.wantReady, s.IsReady(context.Background()))

			mu.Lock()
			defer mu.Unlock()

			// Readiness must stay a single request: the base endpoint only,
			// never the server-paginated tag list and never a retry storm.
			assert.Equal(t, []string{"/v2/"}, paths)
		})
	}
}
