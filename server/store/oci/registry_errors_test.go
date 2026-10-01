// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

// testManifest is the manifest the test registries hold, and its digest.
const (
	testManifest       = "{}"
	testManifestDigest = "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
)

// errorResponse is the error oras-go returns for a registry response with
// status and, when set, an OCI error code, from a registry at address.
func errorResponse(t *testing.T, address string, status int, code string) error {
	t.Helper()

	u, err := url.Parse("http://" + address + "/v2/dir/manifests/latest")
	require.NoError(t, err)

	resp := &errcode.ErrorResponse{Method: http.MethodDelete, URL: u, StatusCode: status}
	if code != "" {
		resp.Errors = errcode.Errors{{Code: code, Message: "test"}}
	}

	return resp
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"missing manifest or blob", fmt.Errorf("latest: %w", errdef.ErrNotFound), true},
		{"404 response", errorResponse(t, "127.0.0.1:5000", http.StatusNotFound, errcode.ErrorCodeNameUnknown), true},
		{"500 response at an address containing 404", errorResponse(t, "127.0.0.1:40404", http.StatusInternalServerError, ""), false},
		{"text saying 404", errors.New("GET http://127.0.0.1:40404/v2/: 404 name unknown"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsNotFound(tt.err))
		})
	}
}

func TestIsDeleteUnsupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"oras unsupported", fmt.Errorf("delete: %w", errdef.ErrUnsupported), true},
		{"405 response", errorResponse(t, "127.0.0.1:5000", http.StatusMethodNotAllowed, ""), true},
		{"UNSUPPORTED code", errorResponse(t, "127.0.0.1:5000", http.StatusBadRequest, errcode.ErrorCodeUnsupported), true},
		{"500 response at an address containing 405", errorResponse(t, "127.0.0.1:40505", http.StatusInternalServerError, ""), false},
		{"text saying unsupported", errors.New("405 unsupported"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, isDeleteUnsupported(tt.err))
		})
	}
}

// registryAnswering answers every request with status and, when set, an OCI
// error body carrying code.
func registryAnswering(status int, code string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if code != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = fmt.Fprintf(w, `{"errors":[{"code":%q,"message":"test"}]}`, code)

			return
		}

		w.WriteHeader(status)
	})
}

// registryHolding holds one manifest and answers its deletion with
// deleteStatus.
func registryHolding(deleteStatus int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(deleteStatus)

			return
		}

		w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
		w.Header().Set("Docker-Content-Digest", testManifestDigest)
		w.Header().Set("Content-Length", strconv.Itoa(len(testManifest)))
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(testManifest))
		}
	})
}

// The reconciler's indexer reads a missing repository as an empty one: the
// tag listing's 404 is recognized, a refusal is not.
func TestIsNotFound_TagListing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		registry http.Handler
		want     bool
	}{
		{"repository not created yet", registryAnswering(http.StatusNotFound, errcode.ErrorCodeNameUnknown), true},
		{"registry refusing", registryAnswering(http.StatusForbidden, errcode.ErrorCodeDenied), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := remoteStoreOnPortContaining(t, "404", tt.registry)

			repo, ok := s.repo.(*remote.Repository)
			require.True(t, ok)

			err := repo.Tags(t.Context(), "", func([]string) error { return nil })
			require.Error(t, err)
			assert.Equal(t, tt.want, IsNotFound(err))
		})
	}
}

// A registry that refuses is not a deletion: the record must not be reported
// deleted while it is still there. Refusals (403) stand for every failure:
// oras-go retries 5xx responses, which only slows the test.
func TestDeleteFromRemoteRepository_Outcomes(t *testing.T) {
	tests := []struct {
		name     string
		digits   string
		registry http.Handler
		code     codes.Code
	}{
		{"manifest already gone", "404", registryAnswering(http.StatusNotFound, ""), codes.OK},
		{"registry refusing at an address containing 404", "404", registryAnswering(http.StatusForbidden, errcode.ErrorCodeDenied), codes.Internal},
		{"deleted", "404", registryHolding(http.StatusAccepted), codes.OK},
		{"deletion not allowed", "404", registryHolding(http.StatusMethodNotAllowed), codes.Unimplemented},
		{"deletion refused at an address containing 405", "405", registryHolding(http.StatusForbidden), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := remoteStoreOnPortContaining(t, tt.digits, tt.registry)

			err := s.deleteFromRemoteRepository(t.Context(), "latest")
			assert.Equal(t, tt.code, status.Code(err), "error: %v", err)
		})
	}
}

func TestDeleteReferrerManifest_Outcomes(t *testing.T) {
	manifest := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.Digest(testManifestDigest), Size: int64(len(testManifest))}

	tests := []struct {
		name         string
		digits       string
		deleteStatus int
		code         codes.Code
	}{
		{"deleted", "404", http.StatusAccepted, codes.OK},
		{"already gone", "404", http.StatusNotFound, codes.OK},
		{"refused at an address containing 404", "404", http.StatusForbidden, codes.Internal},
		{"deletion not allowed", "404", http.StatusMethodNotAllowed, codes.Unimplemented},
		{"refused at an address containing 405", "405", http.StatusForbidden, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := remoteStoreOnPortContaining(t, tt.digits, registryHolding(tt.deleteStatus))

			err := s.deleteReferrerManifest(t.Context(), "referrer", manifest)
			assert.Equal(t, tt.code, status.Code(err), "error: %v", err)
		})
	}
}
