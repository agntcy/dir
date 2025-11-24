// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	storev1 "github.com/agntcy/dir/api/store/v1"
)

// StoreAPI handles management of content-addressable object storage.
type StoreAPI interface {
	// PushData pushes raw data to the content store and returns its CID.
	PushData(ctx context.Context, reader io.ReadCloser) (*storev1.ObjectRef, error)

	// Pull object data from content store
	Pull(context.Context, *storev1.ObjectRef) (io.ReadCloser, error)

	// Push object to content store
	Push(context.Context, *storev1.Object) (*storev1.ObjectRef, error)

	// Lookup metadata about the object from reference
	Lookup(context.Context, *storev1.ObjectRef) (*storev1.Object, error)

	// Delete the object
	Delete(context.Context, *storev1.ObjectRef) error
}

// ReferrerStoreAPI handles management of generic object referrers.
type ReferrerStoreAPI interface {
	// Push referrer to content store
	PushReferrer(ctx context.Context, obj *storev1.ObjectRef, ref *storev1.ObjectRef) error

	// Walk referrers individually for a given object and optional type filter
	WalkReferrers(ctx context.Context, obj *storev1.ObjectRef, referrerType string, walkFn func(*storev1.Object) error) error
}
