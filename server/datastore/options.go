// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package datastore

import (
	"fmt"
	"os"
)

type Option func(*options) error

type options struct {
	localDir string
}

// WithFsProvider sets the filesystem as the datastore provider.
// It creates a local directory if it doesn't exist.
func WithFsProvider(dir string) Option {
	return func(o *options) error {
		// create local dir if it doesn't exist
		if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:mnd
			return fmt.Errorf("failed to create local dir: %w", err)
		}

		// set local dir
		o.localDir = dir

		return nil
	}
}
