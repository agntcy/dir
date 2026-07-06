// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"errors"
	"fmt"

	clientconfig "github.com/agntcy/dir/client/config"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
)

// Load builds a ready-to-use extractor client from the assets a prior Provision
// (dirctl init) wrote to cfg.AssetDir. It NEVER provisions: it errors when the
// extractor has not been set up, so read-path consumers (import enrichment,
// search) fail clearly instead of implicitly triggering an ~89 MB download.
func Load(cfg Config, opts ...sdk.Option) (*sdk.Extractor, error) {
	cfg = cfg.Resolve()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if !IsProvisioned(cfg) {
		return nil, fmt.Errorf("OASF extractor not provisioned at %s; run `dirctl init` first", cfg.AssetDir)
	}

	base := []sdk.Option{
		sdk.WithOASFURL(cfg.OASFURL),
		sdk.WithAssetDir(cfg.AssetDir),
	}

	e, err := sdk.New(append(base, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("load provisioned extractor: %w", err)
	}

	return e, nil
}

// LoadConfigured loads the extractor using the OASF URL / asset dir persisted by
// dirctl init, erroring clearly when init has not been run. This is the entry
// point for read-path consumers (import enrichment, search): they get a ready
// client or an actionable error, and never provision implicitly.
func LoadConfigured(opts ...sdk.Option) (*sdk.Extractor, error) {
	saved, err := clientconfig.LoadExtractor("")
	if err != nil {
		return nil, fmt.Errorf("load extractor config: %w", err)
	}

	if saved == nil {
		return nil, errors.New("OASF extractor not configured; run `dirctl init` first")
	}

	return Load(Config{OASFURL: saved.OASFURL, AssetDir: saved.AssetDir}, opts...)
}
