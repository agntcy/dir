// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"errors"
	"fmt"

	extractorv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/extractor/v1/extractorv1grpc"
	clientconfig "github.com/agntcy/dir/client/config"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// backendKind identifies which extractor backend a Config resolves to.
type backendKind int

const (
	backendRemote backendKind = iota // a gRPC OASF-SDK server (Config.RemoteAddr)
	backendLocal                     // in-process assets under Config.AssetDir
)

// chooseBackend decides which backend a Config selects, without dialing or
// loading anything: a configured RemoteAddr wins (remote-first), else
// provisioned local assets, else an actionable error naming both fixes.
func chooseBackend(cfg Config) (backendKind, error) {
	if cfg.RemoteAddr != "" {
		return backendRemote, nil
	}

	if IsProvisioned(cfg) {
		return backendLocal, nil
	}

	return 0, fmt.Errorf(
		"OASF extractor unavailable: no assets provisioned at %s (run `dirctl init`) "+
			"and no OASF-SDK server configured (set a remote address)",
		cfg.Resolve().AssetDir,
	)
}

// ResolveExtractor returns a ready Extractor for cfg: the remote gRPC backend
// when RemoteAddr is set, otherwise the in-process local backend loaded from
// provisioned assets. localOpts tune only the local backend (e.g. embedding
// weights); they are ignored by the remote backend, whose weighting is fixed by
// the server. It errors when neither backend is available.
func ResolveExtractor(cfg Config, localOpts ...sdk.Option) (Extractor, error) {
	kind, err := chooseBackend(cfg)
	if err != nil {
		return nil, err
	}

	switch kind {
	case backendRemote:
		conn, err := grpc.NewClient(cfg.RemoteAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("dial OASF-SDK server %q: %w", cfg.RemoteAddr, err)
		}

		return &remoteExtractor{client: extractorv1grpc.NewExtractorServiceClient(conn)}, nil
	case backendLocal:
		ext, err := Load(cfg, localOpts...)
		if err != nil {
			return nil, err
		}

		return &localExtractor{ext: ext}, nil
	}

	return nil, fmt.Errorf("unknown extractor backend %d", kind)
}

// ResolveConfigured resolves the extractor from the machine-wide config saved by
// `dirctl init`, returning the Extractor interface. It honors a persisted
// RemoteAddr (remote backend) and otherwise loads the provisioned local assets.
// localOpts tune the local backend only. It errors clearly when init has not run.
func ResolveConfigured(localOpts ...sdk.Option) (Extractor, error) {
	saved, err := clientconfig.LoadExtractor("")
	if err != nil {
		return nil, fmt.Errorf("load extractor config: %w", err)
	}

	if saved == nil {
		return nil, errors.New("OASF extractor not configured; run `dirctl init` first")
	}

	return ResolveExtractor(Config{
		OASFURL:    saved.OASFURL,
		AssetDir:   saved.AssetDir,
		RemoteAddr: saved.RemoteAddr,
	}, localOpts...)
}
