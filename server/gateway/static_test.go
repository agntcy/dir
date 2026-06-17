// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/fs"
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
		{path: "/_app/version.json", want: cacheControlNoCache},
		{path: "/ui/config.json", want: cacheControlNoCache},
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
	tests := []struct {
		name        string
		fileName    string
		requestPath string
		data        []byte
		wantBody    string
		wantMIME    string
	}{
		{
			name:        "css",
			fileName:    "app.css",
			requestPath: "/_app/immutable/assets/app.css",
			data:        []byte("body { color: red; }"),
			wantBody:    "body { color: red; }",
			wantMIME:    "text/css; charset=utf-8",
		},
		{
			name:        "js",
			fileName:    "app.js",
			requestPath: "/_app/immutable/chunks/app.js",
			data:        []byte("export {}"),
			wantBody:    "export {}",
			wantMIME:    "text/javascript; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			static := fstest.MapFS{
				tt.fileName: &fstest.MapFile{Data: tt.data},
			}

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tt.requestPath, nil)
			req.Header.Set("Accept-Encoding", "gzip")

			rec := httptest.NewRecorder()

			require.NoError(t, serveStaticFile(rec, req, static, tt.fileName))
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, cacheControlImmutable, rec.Header().Get("Cache-Control"))
			assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
			assert.Equal(t, tt.wantMIME, rec.Header().Get("Content-Type"))
			assert.Contains(t, rec.Header().Get("Vary"), "Accept-Encoding")

			reader, err := gzip.NewReader(rec.Body)
			require.NoError(t, err)

			body, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())
			assert.Equal(t, tt.wantBody, string(body))
		})
	}
}

func TestEncodingAccepted(t *testing.T) {
	tests := []struct {
		header   string
		encoding string
		want     bool
	}{
		{header: "", encoding: "gzip", want: false},
		{header: "gzip", encoding: "gzip", want: true},
		{header: "gzip, deflate", encoding: "gzip", want: true},
		{header: "deflate, gzip;q=0.8", encoding: "gzip", want: true},
		{header: "gzip;q=0", encoding: "gzip", want: false},
		{header: "gzip;q=0.0", encoding: "gzip", want: false},
		{header: "deflate", encoding: "gzip", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			assert.Equal(t, tt.want, encodingAccepted(tt.header, tt.encoding))
		})
	}
}

func TestIsStaticPathTraversal(t *testing.T) {
	assert.True(t, isStaticPathTraversal("../secret"))
	assert.True(t, isStaticPathTraversal("foo/../bar"))
	assert.False(t, isStaticPathTraversal("agents/"))
	assert.False(t, isStaticPathTraversal("_app/immutable/chunks/app.js"))
}

func TestWriteStaticResponse_VaryWithoutGzip(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/_app/immutable/assets/app.css", nil)
	rec := httptest.NewRecorder()

	writeStaticResponse(rec, req, "/_app/immutable/assets/app.css", []byte("body { }"), "text/css; charset=utf-8")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))
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
	handler := withStaticFallback(mux, static, UIConfig{})

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

	t.Run("returns 500 on static read failure", func(t *testing.T) {
		handler := withStaticFallback(mux, errorFS{}, UIConfig{})

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/broken.css", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("returns 404 for traversal static path", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/../secret", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("spa fallback for trailing slash route", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/agents/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, cacheControlNoCache, rec.Header().Get("Cache-Control"))
	})

	t.Run("returns 404 for missing app asset", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/_app/immutable/chunks/missing.js", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("serves ui config without cache", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ui/config.json", nil)
		rec := httptest.NewRecorder()

		withStaticFallback(mux, static, UIConfig{CatalogTitle: "Cisco AI Catalog"}).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, cacheControlNoCache, rec.Header().Get("Cache-Control"))
		assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"catalogTitle":"Cisco AI Catalog"}`, rec.Body.String())
	})
}

func TestNormalizeCatalogTitle(t *testing.T) {
	assert.Equal(t, "AI Catalog", normalizeCatalogTitle(""))
	assert.Equal(t, "Cisco AI Catalog", normalizeCatalogTitle("  Cisco AI Catalog  "))
	assert.Equal(t, "badtitle", normalizeCatalogTitle("bad\x00title"))
	assert.Equal(t, "AI Catalog", normalizeCatalogTitle("\x00\x1f"))
}

func TestEmbeddedFaviconIsICO(t *testing.T) {
	data, err := fs.ReadFile(staticFS, "static/favicon.ico")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 4, "favicon.ico should have ICO header")
	assert.Equal(t, []byte{0x00, 0x00, 0x01, 0x00}, data[:4], "favicon.ico must be a Windows ICO file, not a PNG renamed with .ico")
}

type errorFS struct{}

func (errorFS) Open(string) (fs.File, error) {
	return nil, errors.New("static filesystem read failure")
}
