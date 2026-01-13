// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"

	"github.com/agntcy/dir/server/store/eventswrap"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/types"
)

type Provider string

const (
	OCI = Provider("oci")
)

// TODO: add options for adding cache.
func New(opts types.APIOptions) (types.StoreAPI, error) {
	switch provider := Provider(opts.Config().Store.Provider); provider {
	case OCI:
		store, err := oci.New(opts.Config().Store.OCI)
		if err != nil {
			return nil, fmt.Errorf("failed to create OCI store: %w", err)
		}

		// Wrap with event emitter
		store = eventswrap.Wrap(store, opts.EventBus())

		return store, nil

	default:
		return nil, fmt.Errorf("unsupported provider=%s", provider)
	}
}
