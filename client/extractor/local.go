// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"errors"
	"fmt"
	"io"

	clientconfig "github.com/agntcy/dir/client/config"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// smokeSampleText exercises both skill and domain matching so SmokeCheck
// confirms a real extraction round-trip rather than an empty result.
const smokeSampleText = "real-time fraud detection for banking transactions using natural language processing"

// localExtractor implements Extractor with an in-process SDK extractor loaded
// from provisioned assets. It carries the ~89 MB model and taxonomy in memory.
type localExtractor struct {
	ext *sdk.Extractor
}

// Extract runs the in-process extractor, translating the backend-agnostic
// ExtractOptions into the SDK's per-query options.
func (l *localExtractor) Extract(ctx context.Context, text string, opts ExtractOptions) (Result, error) {
	var qopts []sdk.QueryOption
	if len(opts.Versions) > 0 {
		qopts = append(qopts, sdk.Versions(opts.Versions...))
	}

	res, err := l.ext.Extract(ctx, text, qopts...)
	if err != nil {
		return Result{}, fmt.Errorf("local extract: %w", err)
	}

	return res, nil
}

// Load builds a ready-to-use in-process extractor from the assets a prior
// Provision (dirctl init) wrote to cfg.AssetDir. It NEVER provisions: it errors
// when the extractor has not been set up, so read-path consumers fail clearly
// instead of implicitly triggering an ~89 MB download.
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

	prev := zlog.Logger
	zlog.Logger = zerolog.New(io.Discard)
	e, err := sdk.New(append(base, opts...)...)
	zlog.Logger = prev

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
