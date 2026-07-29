// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import "fmt"

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
