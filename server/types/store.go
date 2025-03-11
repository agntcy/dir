// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
)

// StoreAPI handles management of content-addressable object storage
type StoreAPI interface {
	// Push object to content store
	Push(context.Context, *storetypes.ObjectRef, io.Reader) (*storetypes.ObjectRef, error)

	// Pull object from content store
	Pull(context.Context, *storetypes.ObjectRef) (io.Reader, error)

	// Lookup metadata about the object from digest
	Lookup(context.Context, *storetypes.ObjectRef) (*storetypes.ObjectRef, error)

	// Delete the object
	Delete(context.Context, *storetypes.ObjectRef) error
}
