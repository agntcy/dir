// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package extractor provisions and loads the OASF taxonomy extractor's local
// assets (the sentence-transformer model + embedded taxonomy) for dirctl. It
// wraps github.com/agntcy/oasf-sdk/pkg/extractor with dirctl-facing defaults,
// validation, a smoke check, and teardown, so `dirctl init` and future
// import/search consumers share one provisioning path.
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

// Config selects the OASF endpoint the taxonomy is pulled from and the local
// directory the provisioned assets are written to / loaded from.
type Config struct {
	OASFURL  string
	AssetDir string
}

// Resolve returns a copy of c with any empty field filled by its default.
func (c Config) Resolve() Config {
	if c.OASFURL == "" {
		c.OASFURL = DefaultOASFURL
	}

	if c.AssetDir == "" {
		c.AssetDir = DefaultAssetDir()
	}

	return c
}

// Validate reports whether the config is usable for provisioning: the OASF URL
// must be a non-empty absolute http(s) URL.
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

	return nil
}
