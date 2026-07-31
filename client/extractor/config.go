// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package extractor resolves and wraps the OASF taxonomy extractor for both
// dirctl and the server gateway. The extractor turns free-form text into OASF
// skills, domains, modules, and keywords; it is reachable two ways — an
// in-process library over locally-provisioned assets, or a remote gRPC
// OASF-SDK server — and ResolveExtractor picks between them so callers never
// need to know where they run.
package extractor

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// DefaultOASFURL is the official OASF schema endpoint used when none is chosen.
const DefaultOASFURL = "https://schema.oasf.outshift.com"

// DefaultAssetDir returns the default local asset directory
// (~/.agntcy/oasf-sdk/extractor), matching the oasf-sdk default, and falling
// back to a temp dir when the home directory cannot be determined.
func DefaultAssetDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}

	return filepath.Join(home, ".agntcy", "oasf-sdk", "extractor")
}

// Config selects how the extractor is reached. When RemoteAddr is set, the
// resolver dials that gRPC OASF-SDK server; otherwise it loads the in-process
// assets provisioned under AssetDir from the taxonomy at OASFURL.
type Config struct {
	OASFURL    string
	AssetDir   string
	RemoteAddr string // gRPC OASF-SDK server address; empty => local assets
}

// Resolve returns a copy of c with any empty local field filled by its default.
// RemoteAddr has no default: it is empty unless explicitly configured.
func (c Config) Resolve() Config {
	if c.OASFURL == "" {
		c.OASFURL = DefaultOASFURL
	}

	if c.AssetDir == "" {
		c.AssetDir = DefaultAssetDir()
	}

	return c
}

// Validate reports whether the local-assets config is usable for provisioning:
// the OASF URL must be a non-empty absolute http(s) URL and the asset dir must
// be absolute. It does not validate RemoteAddr (a remote backend needs no local
// assets).
func (c Config) Validate() error {
	if c.OASFURL == "" {
		return fmt.Errorf("OASF URL is required")
	}

	u, err := url.Parse(c.OASFURL)
	if err != nil {
		return fmt.Errorf("invalid OASF URL %q: %w", c.OASFURL, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid OASF URL %q: scheme must be http or https", c.OASFURL)
	}

	if u.Host == "" {
		return fmt.Errorf("invalid OASF URL %q: missing host", c.OASFURL)
	}

	if c.AssetDir == "" {
		return fmt.Errorf("asset dir is required")
	}

	if !filepath.IsAbs(c.AssetDir) {
		return fmt.Errorf("asset dir must be an absolute path, got %q", c.AssetDir)
	}

	return nil
}
