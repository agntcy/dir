// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"
)

// StoreAPI handles management of content-addressable object storage.
type StoreAPI interface {
	// Push object to content store
	Push(context.Context, Object, io.Reader) (Object, error)

	// Pull object from content store
	Pull(context.Context, ObjectRef) (io.ReadCloser, error)

	// Lookup metadata about the object from digest
	Lookup(context.Context, ObjectRef) (Object, error)

	// Delete the object
	Delete(context.Context, ObjectRef) error

	// List all available objects
	// Needed for bootstrapping
	// List(context.Context, func(*coretypes.ObjectRef) error) error
}
