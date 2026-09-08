// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package identity resolves and verifies claim signatures for dns/https/did
// subjects by looking up the subject's public key from an external,
// out-of-band source (a domain's JWKS well-known file, a DNS TXT record, or a
// DID document). SPIFFE subjects are verified directly against an embedded
// certificate (see api/identity/v1) and do not go through a Resolver here.
package identity

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
)

// Resolver verifies a claim signature for a specific identity URI scheme.
type Resolver interface {
	// Scheme returns the identity type this resolver handles: "dns", "https", or "did".
	Scheme() string

	// Verify resolves subject's public key(s) from an external source and
	// checks the detached JWS signature over payload. Returns false (not an
	// error) when no resolved key matches.
	Verify(ctx context.Context, subject string, signature, payload []byte) (bool, error)
}

// Registry dispatches claim verification to the Resolver registered for a
// subject's inferred URI scheme. It implements identityv1.KeyResolver.
type Registry struct {
	resolvers map[string]Resolver
}

// NewRegistry builds a Registry from the given resolvers, keyed by their Scheme().
func NewRegistry(resolvers ...Resolver) *Registry {
	r := &Registry{resolvers: make(map[string]Resolver, len(resolvers))}

	for _, resolver := range resolvers {
		r.resolvers[resolver.Scheme()] = resolver
	}

	return r
}

// Verify implements identityv1.KeyResolver.
func (r *Registry) Verify(ctx context.Context, subject string, signature, payload []byte) (bool, error) {
	scheme := corev1.InferIdentityType(subject)

	resolver, ok := r.resolvers[scheme]
	if !ok {
		return false, fmt.Errorf("no resolver registered for subject scheme %q", scheme)
	}

	return resolver.Verify(ctx, subject, signature, payload)
}

var _ identityv1.KeyResolver = (*Registry)(nil)
