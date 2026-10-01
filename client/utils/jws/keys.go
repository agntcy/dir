// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"slices"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// ToPublicKey returns v as a public key this package can verify with, or
// false for anything else (unsupported type, curve, or a too-small RSA key).
// Value-typed keys are returned as pointers.
func ToPublicKey(v any) (crypto.PublicKey, bool) {
	pub := v

	switch k := v.(type) {
	case ecdsa.PublicKey:
		pub = &k
	case rsa.PublicKey:
		pub = &k
	}

	if _, err := jwsAlgorithmFor(pub); err != nil {
		return nil, false
	}

	return pub, true
}

// PublicKeyFromJWK returns the signature-verification public key of key,
// skipping encryption-only keys and keys ToPublicKey rejects.
func PublicKeyFromJWK(key jwk.Key) (crypto.PublicKey, bool) {
	if key.KeyUsage() == string(jwk.ForEncryption) {
		return nil, false
	}

	// A key restricted to other operations (e.g. encrypt) must not verify.
	if ops := key.KeyOps(); len(ops) > 0 && !slices.Contains(ops, jwk.KeyOpVerify) {
		return nil, false
	}

	pub, err := jwk.PublicKeyOf(key)
	if err != nil {
		return nil, false
	}

	var raw any
	if err := pub.Raw(&raw); err != nil {
		return nil, false
	}

	return ToPublicKey(raw)
}
