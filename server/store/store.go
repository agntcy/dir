// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"

	"github.com/agntcy/dir/server/store/eventswrap"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/types"
)

// New constructs the OCI-backed store and wraps it with the event emitter.
// OCI is the only supported backend.
func New(opts types.APIOptions) (types.StoreAPI, error) {
	store, err := oci.New(opts.Config().Registry)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI store: %w", err)
	}

	return eventswrap.Wrap(store, opts.EventBus()), nil
}
