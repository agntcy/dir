// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
)

// StoreAPI handles management of content-addressable object storage
type StoreAPI interface {
	// Push object to content store
	Push(context.Context, *coretypes.ObjectRef, io.Reader) (*coretypes.ObjectRef, error)

	// Pull object from content store
	Pull(context.Context, *coretypes.ObjectRef) (io.Reader, error)

	// Lookup metadata about the object from CID
	Lookup(context.Context, *coretypes.ObjectRef) (*coretypes.ObjectRef, error)

	// Delete the object
	Delete(context.Context, *coretypes.ObjectRef) error
}
