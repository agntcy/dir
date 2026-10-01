// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package resolvers defines what an identity key resolver is. Each scheme
// lives in its own subpackage: dns, did, wellknown and spiffe.
//
// A resolver finds the public keys a claim's subject publishes; it never
// checks a signature. Pass its result to identity.Verify (or jws.Verify),
// which does all of the cryptography and succeeds if any one key validates.
package resolvers

import (
	"context"
	"crypto"
	"errors"
)

// ErrNoKeys is returned when a subject publishes no usable public key.
var ErrNoKeys = errors.New("no usable public keys found")

// Resolver finds the candidate public keys for a claim's subject.
type Resolver interface {
	// Resolve returns at least one key, or an error. certificate is the
	// claim's DER-encoded X.509 certificate, empty when the claim has none;
	// only SPIFFE resolution reads it.
	Resolve(ctx context.Context, subject string, certificate []byte) ([]crypto.PublicKey, error)
}

// Fetcher retrieves a document over HTTPS. safefetch.Client implements it.
type Fetcher interface {
	Get(ctx context.Context, url string) ([]byte, error)
}
