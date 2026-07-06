// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
)

// smokeSampleText exercises both skill and domain matching so SmokeCheck
// confirms a real extraction round-trip rather than an empty result.
const smokeSampleText = "real-time fraud detection for banking transactions using natural language processing"

// Provision downloads and caches the extractor's assets (model + embedded OASF
// taxonomy) under cfg.AssetDir, pulling the taxonomy from cfg.OASFURL. It is
// idempotent: the SDK skips the model load and re-embed when the on-disk assets
// already match the endpoint and taxonomy. Extra opts are forwarded to the SDK
// (e.g. a test embedder).
func Provision(ctx context.Context, cfg Config, opts ...sdk.Option) error {
	cfg = cfg.Resolve()
	if err := cfg.Validate(); err != nil {
		return err
	}

	base := []sdk.Option{
		sdk.WithOASFURL(cfg.OASFURL),
		sdk.WithAssetDir(cfg.AssetDir),
	}

	if err := sdk.Provision(ctx, append(base, opts...)...); err != nil {
		return fmt.Errorf("provision extractor assets: %w", err)
	}

	return nil
}

// SmokeCheck loads the provisioned assets and runs one extraction, returning an
// error if the client can't be built or the round-trip yields no skills and no
// domains. It confirms the assets are usable in-process by consumers.
func SmokeCheck(ctx context.Context, cfg Config, opts ...sdk.Option) error {
	e, err := Load(cfg, opts...)
	if err != nil {
		return err
	}

	res, err := e.Extract(ctx, smokeSampleText)
	if err != nil {
		return fmt.Errorf("extractor smoke check: %w", err)
	}

	if len(res.Skills) == 0 && len(res.Domains) == 0 {
		return errors.New("extractor smoke check returned no skills or domains")
	}

	return nil
}
