// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"errors"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

var ErrDigestNotFound = errors.New("digest does not exist")

// StoreAPI handles management of content-addressable object storage.
type StoreAPI interface {
	// Push object to content store
	Push(context.Context, *coretypes.ObjectRef, io.Reader) (*coretypes.ObjectRef, error)

	// Pull object from content store
	Pull(context.Context, *coretypes.ObjectRef) (io.ReadCloser, error)

	// Lookup metadata about the object from digest
	Lookup(context.Context, *coretypes.ObjectRef) (*coretypes.ObjectRef, error)

	// Delete the object
	Delete(context.Context, *coretypes.ObjectRef) error

	// List all available objects
	// Needed for bootstrapping
	// List(context.Context, func(*coretypes.ObjectRef) error) error
}
