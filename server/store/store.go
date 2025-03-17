// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"

	"github.com/agntcy/dir/server/store/localfs"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/types"
)

type Provider string

const (
	LocalFS = Provider("localfs")
	OCI     = Provider("oci")
)

// TODO: add options for adding cache.
func New(opts types.APIOptions) (types.StoreAPI, error) {
	switch provider := Provider(opts.Config().Provider); provider {
	case LocalFS:
		store, err := localfs.New(opts.Config().LocalFS)
		if err != nil {
			return nil, fmt.Errorf("failed to create OCI store: %w", err)
		}

		return store, nil

	case OCI:
		store, err := oci.New(opts.Config().OCI)
		if err != nil {
			return nil, fmt.Errorf("failed to create OCI store: %w", err)
		}

		return store, nil

	default:
		return nil, fmt.Errorf("unsupported provider=%s", provider)
	}
}
