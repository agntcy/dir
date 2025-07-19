// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// StoreAPI handles management of content-addressable object storage.
type StoreAPI interface {
	// Push record to content store
	Push(context.Context, *corev1.Record) (*corev1.RecordRef, error)

	// Pull record from content store
	Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error)

	// Lookup metadata about the record from reference
	Lookup(context.Context, *corev1.RecordRef) (*corev1.RecordMeta, error)

	// Delete the record
	Delete(context.Context, *corev1.RecordRef) error

	// List all available records
	// Needed for bootstrapping
	// List(context.Context, func(*corev1.RecordRef) error) error
}
