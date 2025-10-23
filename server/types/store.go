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

	// IsReady checks if the storage backend is ready to serve traffic.
	IsReady(context.Context) bool
}

// ReferrerStoreAPI handles management of generic record referrers.
// This implements the OCI Referrers API for attaching artifacts to records.
//
// Implementations: oci.Store
// Used by: store.Controller, sync.Monitor.
type ReferrerStoreAPI interface {
	// PushReferrer pushes a referrer to content store
	PushReferrer(context.Context, string, *corev1.RecordReferrer) error

	// WalkReferrers walks referrers individually for a given record CID and optional type filter
	WalkReferrers(ctx context.Context, recordCID string, referrerType string, walkFn func(*corev1.RecordReferrer) error) error
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
	ReferrerStoreAPI
	VerifierStore
}
