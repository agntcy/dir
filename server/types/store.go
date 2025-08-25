// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
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

// SignatureStoreAPI handles management of OCI signature artifacts.
type SignatureStoreAPI interface {
	// Push signature to content store
	PushSignature(context.Context, string, *signv1.Signature) error

	// Push public key to content store
	PushPublicKey(context.Context, string, string) error

	// Pull signature from content store
	PullSignature(context.Context, string) (*signv1.Signature, error)

	// Pull public key from content store
	PullPublicKey(context.Context, string) (string, error)
}
