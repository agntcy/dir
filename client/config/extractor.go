// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"

	extractor "github.com/agntcy/dir/utils/extractor"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
)

// ResolveConfigured resolves the extractor from the machine-wide config saved by
// `dirctl init`, returning the Extractor interface. It honors a persisted
// RemoteAddr (remote backend) and otherwise loads the provisioned local assets.
// localOpts tune the local backend only. It errors clearly when init has not run.
func ResolveConfigured(localOpts ...sdk.Option) (extractor.Extractor, error) {
	saved, err := LoadExtractor("")
	if err != nil {
		return nil, fmt.Errorf("load extractor config: %w", err)
	}

	if saved == nil {
		return nil, errors.New("OASF extractor not configured; run `dirctl init` first")
	}

	ext, err := extractor.ResolveExtractor(extractor.Config{
		OASFURL:    saved.OASFURL,
		AssetDir:   saved.AssetDir,
		RemoteAddr: saved.RemoteAddr,
	}, localOpts...)
	if err != nil {
		return nil, fmt.Errorf("resolve extractor: %w", err)
	}

	return ext, nil
}

// LoadConfigured loads the local in-process extractor using the OASF URL / asset
// dir persisted by dirctl init, erroring clearly when init has not been run. It
// returns the concrete *sdk.Extractor, so it is local-only: a persisted
// RemoteAddr is rejected rather than silently ignored (callers that need the
// remote backend should use ResolveConfigured, which returns the Extractor
// interface). Read-path consumers get a ready client or an actionable error, and
// never provision implicitly.
func LoadConfigured(opts ...sdk.Option) (*sdk.Extractor, error) {
	saved, err := LoadExtractor("")
	if err != nil {
		return nil, fmt.Errorf("load extractor config: %w", err)
	}

	if saved == nil {
		return nil, errors.New("OASF extractor not configured; run `dirctl init` first")
	}

	if saved.RemoteAddr != "" {
		return nil, fmt.Errorf("a remote OASF extractor (%s) is configured, but this operation uses the local "+
			"in-process extractor; provision local assets with `dirctl init` (without --extractor-remote-addr)", saved.RemoteAddr)
	}

	ext, err := extractor.Load(extractor.Config{OASFURL: saved.OASFURL, AssetDir: saved.AssetDir}, opts...)
	if err != nil {
		return nil, fmt.Errorf("load provisioned extractor: %w", err)
	}

	return ext, nil
}
