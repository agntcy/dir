// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"compress/gzip"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	cacheControlImmutable = "public, max-age=31536000, immutable"
	cacheControlNoCache   = "no-cache"
	cacheControlLongLived = "public, max-age=31536000"
)

var compressibleContentTypes = map[string]struct{}{
	"application/javascript": {},
	"application/json":       {},
	"image/svg+xml":          {},
	"text/css":               {},
	"text/html":              {},
	"text/plain":             {},
}

func cacheControlForPath(path string) string {
	switch {
	case path == "/" || path == "/index.html":
		return cacheControlNoCache
	case strings.HasPrefix(path, "/_app/immutable/"):
		return cacheControlImmutable
	default:
		return cacheControlLongLived
	}
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func isCompressibleContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	_, ok := compressibleContentTypes[mediaType]

	return ok
}

func contentTypeForName(name string) string {
	if contentType := mime.TypeByExtension(filepath.Ext(name)); contentType != "" {
		return contentType
	}

	return "application/octet-stream"
}

func writeStaticResponse(w http.ResponseWriter, r *http.Request, path string, data []byte, contentType string) {
	w.Header().Set("Cache-Control", cacheControlForPath(path))
	w.Header().Set("Content-Type", contentType)

	if acceptsGzip(r) && isCompressibleContentType(contentType) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")

		gz := gzip.NewWriter(w)

		defer func() { _ = gz.Close() }()

		w.WriteHeader(http.StatusOK)

		_, _ = gz.Write(data)

		return
	}

	w.WriteHeader(http.StatusOK)
	// Embedded build artifacts only; not user-controlled content.
	_, _ = w.Write(data) //nolint:gosec // G705: static assets are compiled into the binary
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, static fs.FS, name string) error {
	data, err := fs.ReadFile(static, name)
	if err != nil {
		return fmt.Errorf("read static file %q: %w", name, err)
	}

	writeStaticResponse(w, r, r.URL.Path, data, contentTypeForName(name))

	return nil
}

func serveIndexHTML(w http.ResponseWriter, r *http.Request, indexHTML []byte) {
	writeStaticResponse(w, r, "/", indexHTML, "text/html; charset=utf-8")
}
