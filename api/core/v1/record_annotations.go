// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

// Reserved keys in the record's "annotations" map used to carry verifiable
// identity/ownership claims. These live in Record.data, so they are part of
// the record's content-addressed CID.
const (
	// AnnotationKeyIdentity holds the record's own claimed identity URI
	// (e.g. "did:web:acme.com:agents:finance", "spiffe://acme.com/agents/finance").
	AnnotationKeyIdentity = "agntcy.dir/identity"

	// AnnotationKeyIdentityType optionally overrides the inferred identity URI scheme.
	AnnotationKeyIdentityType = "agntcy.dir/identity-type"

	// AnnotationKeyOwner holds the claimed owning entity's identity URI
	// (e.g. "did:web:acme.com", "dns:acme.com").
	AnnotationKeyOwner = "agntcy.dir/owner"

	// AnnotationKeyOwnerType optionally overrides the inferred owner URI scheme.
	AnnotationKeyOwnerType = "agntcy.dir/owner-type"
)
