// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// StoreAPI handles management of content-addressable object storage.
type StoreAPI interface {
	// PushData pushes raw data to the content store and returns its CID.
	PushData(ctx context.Context, reader io.ReadCloser) (*corev1.ObjectRef, error)

	// Pull object data from content store
	Pull(context.Context, *corev1.ObjectRef) (io.ReadCloser, error)

	// Push object to content store
	Push(context.Context, *corev1.Object) (*corev1.ObjectRef, error)

	// Lookup metadata about the object from reference
	Lookup(context.Context, *corev1.ObjectRef) (*corev1.Object, error)

	// Delete the object
	Delete(context.Context, *corev1.ObjectRef) error
}

// ReferrerStoreAPI handles management of generic object referrers.
type ReferrerStoreAPI interface {
	// Push referrer to content store
	PushReferrer(ctx context.Context, obj *corev1.ObjectRef, ref *corev1.ObjectRef) error

	// Walk referrers individually for a given object and optional type filter
	WalkReferrers(ctx context.Context, obj *corev1.ObjectRef, referrerType string, walkFn func(*corev1.Object) error) error
}
