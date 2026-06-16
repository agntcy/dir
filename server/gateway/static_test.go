// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheControlForPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/", want: cacheControlNoCache},
		{path: "/index.html", want: cacheControlNoCache},
		{path: "/_app/immutable/assets/app.css", want: cacheControlImmutable},
		{path: "/_app/immutable/chunks/start.js", want: cacheControlImmutable},
		{path: "/favicon.ico", want: cacheControlLongLived},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, cacheControlForPath(tt.path))
		})
	}
}

func TestServeStaticFile_ImmutableCacheAndGzip(t *testing.T) {
	static := fstest.MapFS{
		"app.css": &fstest.MapFile{
			Data: []byte("body { color: red; }"),
		},
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/_app/immutable/assets/app.css", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()

	require.NoError(t, serveStaticFile(rec, req, static, "app.css"))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, cacheControlImmutable, rec.Header().Get("Cache-Control"))
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "text/css; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Vary"), "Accept-Encoding")

	reader, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)

	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "body { color: red; }", string(body))
}

func TestServeIndexHTML_NoCache(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	serveIndexHTML(rec, req, []byte("<!doctype html><html></html>"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, cacheControlNoCache, rec.Header().Get("Cache-Control"))
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
}

func TestWithStaticFallback(t *testing.T) {
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!doctype html><html></html>"),
		},
		"_app/immutable/assets/app.css": &fstest.MapFile{
			Data: []byte("body { color: blue; }"),
		},
	}

	mux := runtime.NewServeMux()
	handler := withStaticFallback(mux, static)

	t.Run("serves immutable asset", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/_app/immutable/assets/app.css", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, cacheControlImmutable, rec.Header().Get("Cache-Control"))
	})

	t.Run("serves index without cache", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, cacheControlNoCache, rec.Header().Get("Cache-Control"))
	})

	t.Run("spa fallback uses index without cache", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/agents/example", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, cacheControlNoCache, rec.Header().Get("Cache-Control"))
	})
}
