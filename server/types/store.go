// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"io"

	corev1 "github.com/agntcy/dir/api/store/v1"
)

// StoreAPI handles management of content-addressable object storage.
type StoreAPI interface {
	// Push record to content store
	Push(context.Context, *corev1.Object, io.ReadCloser) (*corev1.ObjectRef, error)

	// Pull record from content store
	Pull(context.Context, *corev1.ObjectRef) (*corev1.Object, io.ReadCloser, error)

	// Lookup metadata about the record from reference
	Lookup(context.Context, *corev1.ObjectRef) (*corev1.Object, error)

	// Delete the record
	Delete(context.Context, *corev1.ObjectRef) error

	// Walk walks records individually
	Walk(ctx context.Context, head *corev1.ObjectRef, walkFn func(*corev1.Object) error, walkOpts ...func()) error

	// IsReady checks if the storage backend is ready to serve traffic.
	IsReady(context.Context) bool
}

// VerifierStore provides signature verification using Zot registry.
// This is implemented by OCI-backed stores that have access to a Zot registry
// with cosign/notation signature support.
//
// Implementations: oci.Store (when using Zot registry)
// Used by: sign.Controller.
type VerifierStore interface {
	// VerifyWithZot verifies a record signature using Zot registry GraphQL API
	VerifyWithZot(ctx context.Context, recordCID string) (bool, error)
}

// FullStore is the complete store interface with all optional capabilities.
// This is what the OCI store implementation provides.
type FullStore interface {
	StoreAPI
	VerifierStore
}
