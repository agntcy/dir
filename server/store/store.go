// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"

	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/store/localfs"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/types"
)

type Provider string

const (
	LocalFS = Provider("localfs")
	OCI     = Provider("oci")
)

func New(config *config.Config) (types.StoreService, error) {
	switch provider := Provider(config.Provider); provider {
	case OCI:
		store, err := oci.New(config.OCI)
		if err != nil {
			return nil, fmt.Errorf("failed to create OCI store: %w", err)
		}

		return store, nil
	case LocalFS:
		store, err := localfs.New(config.LocalFS.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to create localfs store: %w", err)
		}

		return store, nil
	default:
		return nil, fmt.Errorf("unsupported provider=%s", provider)
	}
}
