// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	objectmanager "github.com/agntcy/dir/api/objectmanager"
)

// StoreAPI handles management of content-addressable object storage.
type StoreAPI interface {
	// Push object to content store
	Push(context.Context, *objectmanager.RecordObject, io.Reader) (*objectmanager.RecordObject, error)

	// Pull object from content store
	Pull(context.Context, *objectmanager.RecordObject) (io.ReadCloser, error)

	// Lookup metadata about the object from digest
	Lookup(context.Context, *objectmanager.RecordObject) (*objectmanager.RecordObject, error)

	// Delete the object
	Delete(context.Context, *objectmanager.RecordObject) error

	// List all available objects
	// Needed for bootstrapping
	// List(context.Context, func(*objectmanager.RecordObject) error) error
}
