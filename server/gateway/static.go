// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	cacheControlImmutable = "public, max-age=31536000, immutable"
	cacheControlNoCache   = "no-cache"
	cacheControlLongLived = "public, max-age=31536000"
	maxCatalogTitleLen    = 128
)

var compressibleContentTypes = map[string]struct{}{
	"application/ecmascript": {},
	"application/javascript": {},
	"application/json":       {},
	"image/svg+xml":          {},
	"text/css":               {},
	"text/html":              {},
	"text/javascript":        {},
	"text/plain":             {},
}

func isStaticPathTraversal(name string) bool {
	for elem := range strings.SplitSeq(name, "/") {
		if elem == ".." {
			return true
		}
	}

	return false
}

func cacheControlForPath(path string) string {
	switch {
	case path == "/" || path == "/index.html" || path == "/_app/version.json" || path == "/ui/config.json":
		return cacheControlNoCache
	case strings.HasPrefix(path, "/_app/immutable/"):
		return cacheControlImmutable
	default:
		return cacheControlLongLived
	}
}

func acceptsGzip(r *http.Request) bool {
	return encodingAccepted(r.Header.Get("Accept-Encoding"), "gzip")
}

func encodingAccepted(header, encoding string) bool {
	if header == "" {
		return false
	}

	encoding = strings.ToLower(encoding)

	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		name := part
		q := 1.0

		if coding, params, ok := strings.Cut(part, ";"); ok {
			name = strings.TrimSpace(coding)

			if qVal, ok := parseEncodingQValue(params); ok {
				q = qVal
			}
		}

		if strings.ToLower(name) == encoding {
			return q > 0
		}
	}

	return false
}

func parseEncodingQValue(params string) (float64, bool) {
	for param := range strings.SplitSeq(params, ";") {
		param = strings.TrimSpace(param)
		if len(param) < 2 || !strings.EqualFold(param[:2], "q=") {
			continue
		}

		q, err := strconv.ParseFloat(strings.TrimSpace(param[2:]), 64)
		if err != nil {
			return 0, false
		}

		return q, true
	}

	return 1, false
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

	compressible := isCompressibleContentType(contentType)
	if compressible {
		w.Header().Add("Vary", "Accept-Encoding")
	}

	if compressible && acceptsGzip(r) {
		w.Header().Set("Content-Encoding", "gzip")

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data) //nolint:gosec // G705: static assets are compiled into the binary

			return
		}

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
		if errors.Is(err, fs.ErrNotExist) {
			return fs.ErrNotExist
		}

		return fmt.Errorf("read static file %q: %w", name, err)
	}

	writeStaticResponse(w, r, r.URL.Path, data, contentTypeForName(name))

	return nil
}

func serveIndexHTML(w http.ResponseWriter, r *http.Request, indexHTML []byte) {
	writeStaticResponse(w, r, "/", indexHTML, "text/html; charset=utf-8")
}

func serveUIConfig(w http.ResponseWriter, r *http.Request, uiConfigJSON []byte) {
	writeStaticResponse(w, r, "/ui/config.json", uiConfigJSON, "application/json; charset=utf-8")
}

func uiConfigJSON(ui UIConfig) []byte {
	title := normalizeCatalogTitle(ui.CatalogTitle)

	payload, err := json.Marshal(struct {
		CatalogTitle string `json:"catalogTitle"`
	}{
		CatalogTitle: title,
	})
	if err != nil {
		return []byte(`{"catalogTitle":"AI Catalog"}`)
	}

	return payload
}

func normalizeCatalogTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "AI Catalog"
	}

	var b strings.Builder

	for _, r := range title {
		if r < 0x20 || r == 0x7f {
			continue
		}

		b.WriteRune(r)
	}

	title = strings.TrimSpace(b.String())
	if title == "" {
		return "AI Catalog"
	}

	if len(title) > maxCatalogTitleLen {
		return title[:maxCatalogTitleLen]
	}

	return title
}
