// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package did verifies "did:" subjects. Supports did:web (fetches the DID
// document over HTTPS and checks verificationMethod publicKeyJwk entries)
// and did:key (self-certifying, decoded locally with no network lookup).
package did

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/multiformats/go-multibase"
)

// Multicodec key-type codes used by did:key (https://github.com/multiformats/multicodec).
// Only the codes matching the algorithms api/identity/v1 supports (ECDSA P-256, RSA) are handled.
const (
	multicodecP256Pub = 0x1200
	multicodecRSAPub  = 0x1205
)

const didWebWellKnownSuffix = "/.well-known/did.json"

// didDocument is the minimal subset of a W3C DID document this resolver needs.
type didDocument struct {
	VerificationMethod []struct {
		PublicKeyJWK json.RawMessage `json:"publicKeyJwk"`
	} `json:"verificationMethod"`
}

// Resolver verifies claim signatures for did:web and did:key subjects.
type Resolver struct {
	fetch *safefetch.Client
}

// New creates a new DID resolver.
func New(fetch *safefetch.Client) *Resolver {
	if fetch == nil {
		fetch = safefetch.New()
	}

	return &Resolver{fetch: fetch}
}

// Scheme implements identity.Resolver.
func (*Resolver) Scheme() string {
	return "did"
}

// Verify implements identity.Resolver.
func (r *Resolver) Verify(ctx context.Context, subject string, signature, payload []byte) (bool, error) {
	switch {
	case strings.HasPrefix(subject, "did:web:"):
		return r.verifyDIDWeb(ctx, subject, signature, payload)
	case strings.HasPrefix(subject, "did:key:"):
		return verifyDIDKey(subject, signature, payload)
	default:
		return false, fmt.Errorf("unsupported did method in subject %q", subject)
	}
}

func (r *Resolver) verifyDIDWeb(ctx context.Context, subject string, signature, payload []byte) (bool, error) {
	docURL, err := didWebURL(subject)
	if err != nil {
		return false, err
	}

	body, err := r.fetch.Get(ctx, docURL)
	if err != nil {
		return false, fmt.Errorf("fetch DID document from %s: %w", docURL, err)
	}

	return verifyDIDDocument(body, signature, payload)
}

// verifyDIDDocument checks signature against each verificationMethod's
// publicKeyJwk in a raw DID document body. Split out from verifyDIDWeb so it
// can be unit-tested without a network fetch.
func verifyDIDDocument(body, signature, payload []byte) (bool, error) {
	var doc didDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return false, fmt.Errorf("parse DID document: %w", err)
	}

	for _, vm := range doc.VerificationMethod {
		if len(vm.PublicKeyJWK) == 0 {
			continue
		}

		key, err := jwk.ParseKey(vm.PublicKeyJWK)
		if err != nil {
			continue
		}

		var rawKey any
		if err := key.Raw(&rawKey); err != nil {
			continue
		}

		if identityv1.VerifyJWS(string(signature), rawKey, payload) == nil {
			return true, nil
		}
	}

	return false, nil
}

// didWebURL implements the did:web resolution algorithm
// (https://w3c-ccg.github.io/did-method-web/): the identifier's colon-
// separated segments after the domain become URL path segments, and a
// bare-domain identifier resolves to the domain's well-known path.
func didWebURL(subject string) (string, error) {
	id := strings.TrimPrefix(subject, "did:web:")
	if id == "" {
		return "", fmt.Errorf("invalid did:web subject: %q", subject)
	}

	parts := strings.Split(id, ":")
	for i, p := range parts {
		if decoded, err := url.PathUnescape(p); err == nil {
			parts[i] = decoded
		}
	}

	domain := parts[0]
	if domain == "" {
		return "", fmt.Errorf("invalid did:web subject: %q", subject)
	}

	if len(parts) == 1 {
		return "https://" + domain + didWebWellKnownSuffix, nil
	}

	return "https://" + domain + "/" + strings.Join(parts[1:], "/") + "/did.json", nil
}

// verifyDIDKey decodes a self-certifying did:key subject (no network lookup)
// and verifies signature against its embedded public key.
func verifyDIDKey(subject string, signature, payload []byte) (bool, error) {
	id := strings.TrimPrefix(subject, "did:key:")
	if id == "" {
		return false, fmt.Errorf("invalid did:key subject: %q", subject)
	}

	_, data, err := multibase.Decode(id)
	if err != nil {
		return false, fmt.Errorf("decode did:key multibase: %w", err)
	}

	code, n := binary.Uvarint(data)
	if n <= 0 {
		return false, errors.New("decode did:key multicodec prefix")
	}

	keyBytes := data[n:]

	var pub any

	switch code {
	case multicodecP256Pub:
		x, y := elliptic.UnmarshalCompressed(elliptic.P256(), keyBytes)
		if x == nil {
			return false, errors.New("invalid did:key p256 compressed point")
		}

		pub = &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
	case multicodecRSAPub:
		rsaPub, err := x509.ParsePKCS1PublicKey(keyBytes)
		if err != nil {
			return false, fmt.Errorf("parse did:key rsa public key: %w", err)
		}

		pub = rsaPub
	default:
		return false, fmt.Errorf("unsupported did:key codec 0x%x", code)
	}

	return identityv1.VerifyJWS(string(signature), pub, payload) == nil, nil
}
