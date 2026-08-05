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
