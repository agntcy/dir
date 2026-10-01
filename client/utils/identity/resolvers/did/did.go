// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package didresolver

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/multiformats/go-multibase"
)

// Multicodec key-type codes used by did:key
// (https://github.com/multiformats/multicodec).
const (
	multicodecEd25519Pub = 0xed
	multicodecP256Pub    = 0x1200
	multicodecP384Pub    = 0x1201
	multicodecRSAPub     = 0x1205
)

// Resolver resolves did:web subjects from the DID document served over HTTPS,
// and did:key subjects locally: a did:key embeds its own key, so it needs no
// network access.
type Resolver struct {
	fetch resolvers.Fetcher
}

// New creates a Resolver. A nil fetch uses a default safefetch.Client.
func New(fetch resolvers.Fetcher) *Resolver {
	if fetch == nil {
		fetch = safefetch.New()
	}

	return &Resolver{fetch: fetch}
}

// Resolve implements resolvers.Resolver. It ignores certificate.
func (d *Resolver) Resolve(ctx context.Context, subject string, _ []byte) ([]crypto.PublicKey, error) {
	switch {
	case strings.HasPrefix(subject, "did:web:"):
		return d.resolveWeb(ctx, subject)
	case strings.HasPrefix(subject, "did:key:"):
		return resolveKey(subject)
	default:
		return nil, fmt.Errorf("unsupported did method in subject %q", subject)
	}
}

func (d *Resolver) resolveWeb(ctx context.Context, subject string) ([]crypto.PublicKey, error) {
	docURL, err := didWebURL(subject)
	if err != nil {
		return nil, err
	}

	body, err := d.fetch.Get(ctx, docURL)
	if err != nil {
		return nil, fmt.Errorf("fetch DID document from %s: %w", docURL, err)
	}

	return keysFromDIDDocument(body, subject, docURL)
}

// didDocument is the subset of a W3C DID document used here.
type didDocument struct {
	ID                 string `json:"id"`
	VerificationMethod []struct {
		PublicKeyJWK json.RawMessage `json:"publicKeyJwk"`
	} `json:"verificationMethod"`
}

// keysFromDIDDocument extracts the verification keys from a DID document and
// checks that the document is actually about subject.
func keysFromDIDDocument(body []byte, subject, source string) ([]crypto.PublicKey, error) {
	var doc didDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse DID document from %s: %w", source, err)
	}

	// did:web requires the document's id to be the DID that resolved to it,
	// otherwise one domain could publish keys on behalf of another identity.
	if doc.ID != subject {
		return nil, fmt.Errorf("DID document id %q does not match subject %q", doc.ID, subject)
	}

	keys := make([]crypto.PublicKey, 0, len(doc.VerificationMethod))

	for _, vm := range doc.VerificationMethod {
		if len(vm.PublicKeyJWK) == 0 {
			continue
		}

		key, err := jwk.ParseKey(vm.PublicKeyJWK)
		if err != nil {
			continue
		}

		if pub, ok := jws.PublicKeyFromJWK(key); ok {
			keys = append(keys, pub)
		}
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("%w at %s", resolvers.ErrNoKeys, source)
	}

	return keys, nil
}

// didWebURL implements the did:web method
// (https://w3c-ccg.github.io/did-method-web/): the domain, with an
// optionally percent-encoded port, is the first segment and the remaining
// segments are path segments. A bare domain resolves to /.well-known/did.json.
func didWebURL(subject string) (string, error) {
	segments := strings.Split(strings.TrimPrefix(subject, "did:web:"), ":")

	host, err := url.PathUnescape(segments[0])
	if err != nil {
		return "", fmt.Errorf("invalid did:web subject %q: %w", subject, err)
	}

	// Reparsing must yield the same host, so userinfo, a path, a query or a
	// fragment smuggled in through percent-encoding is refused.
	u, err := url.Parse("https://" + host)
	if err != nil || u.Host != host || u.Hostname() == "" {
		return "", fmt.Errorf("invalid did:web subject %q: bad host %q", subject, host)
	}

	docPath := "/.well-known/did.json"

	if len(segments) > 1 {
		decodedSegments := make([]string, 0, len(segments)-1)

		// A decoded segment must stay a single, plain path segment, so an
		// escaped "/" or ".." can't redirect the request elsewhere on the host.
		for _, seg := range segments[1:] {
			decoded, err := url.PathUnescape(seg)
			if err != nil || decoded == "" || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, `/\`) {
				return "", fmt.Errorf("invalid did:web subject %q: bad path segment %q", subject, seg)
			}

			decodedSegments = append(decodedSegments, decoded)
		}

		// URL.Path holds the decoded form; String() escapes it exactly once.
		docPath = "/" + strings.Join(decodedSegments, "/") + "/did.json"
	}

	return (&url.URL{Scheme: "https", Host: u.Host, Path: docPath}).String(), nil
}

// resolveKey decodes a self-certifying did:key subject.
func resolveKey(subject string) ([]crypto.PublicKey, error) {
	_, data, err := multibase.Decode(strings.TrimPrefix(subject, "did:key:"))
	if err != nil {
		return nil, fmt.Errorf("decode did:key %q: %w", subject, err)
	}

	code, n := binary.Uvarint(data)
	if n <= 0 {
		return nil, fmt.Errorf("decode did:key %q: bad multicodec prefix", subject)
	}

	pub, err := publicKeyFromMulticodec(code, data[n:])
	if err != nil {
		return nil, fmt.Errorf("decode did:key %q: %w", subject, err)
	}

	usable, ok := jws.ToPublicKey(pub)
	if !ok {
		return nil, fmt.Errorf("decode did:key %q: %w", subject, resolvers.ErrNoKeys)
	}

	return []crypto.PublicKey{usable}, nil
}

func publicKeyFromMulticodec(code uint64, raw []byte) (crypto.PublicKey, error) {
	switch code {
	case multicodecEd25519Pub:
		if len(raw) != ed25519.PublicKeySize {
			return nil, errors.New("invalid ed25519 key length")
		}

		return ed25519.PublicKey(raw), nil
	case multicodecP256Pub:
		return ecdsaFromCompressed(elliptic.P256(), raw)
	case multicodecP384Pub:
		return ecdsaFromCompressed(elliptic.P384(), raw)
	case multicodecRSAPub:
		pub, err := x509.ParsePKCS1PublicKey(raw)
		if err != nil {
			return nil, fmt.Errorf("parse rsa public key: %w", err)
		}

		return pub, nil
	default:
		return nil, fmt.Errorf("unsupported key codec 0x%x", code)
	}
}

// ecdsaFromCompressed decompresses a SEC1 point and builds the key through
// the uncompressed-point parser, which validates it is on the curve.
func ecdsaFromCompressed(curve elliptic.Curve, compressed []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.UnmarshalCompressed(curve, compressed)
	if x == nil {
		return nil, errors.New("invalid compressed ecdsa point")
	}

	size := len(compressed) - 1 // a compressed point is one prefix byte plus x
	uncompressed := make([]byte, 1+2*size)
	uncompressed[0] = 0x04

	x.FillBytes(uncompressed[1 : 1+size])
	y.FillBytes(uncompressed[1+size:])

	pub, err := ecdsa.ParseUncompressedPublicKey(curve, uncompressed)
	if err != nil {
		return nil, fmt.Errorf("parse ecdsa public key: %w", err)
	}

	return pub, nil
}
