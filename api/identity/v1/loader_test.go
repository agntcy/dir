// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net/url"
	"strings"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxCertificateDERSize = 16 << 10
	rsaKeyBits            = 2048
	weakRSAKeyBits        = 1024
)

type testKeyCert struct {
	keyPEM  []byte
	certPEM []byte
	cert    *x509.Certificate
}

type certTemplate struct {
	uris      []string
	notBefore time.Time
	notAfter  time.Time
	padding   int
}

func newKey(t *testing.T, kind string) crypto.Signer {
	t.Helper()

	var (
		key crypto.Signer
		err error
	)

	switch kind {
	case "p256":
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "p384":
		key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "p521":
		key, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	case "rsa":
		key, err = rsa.GenerateKey(rand.Reader, rsaKeyBits)
	case "rsa-weak":
		key, err = rsa.GenerateKey(rand.Reader, weakRSAKeyBits)
	case "ed25519":
		_, key, err = ed25519.GenerateKey(rand.Reader)
	default:
		t.Fatalf("unknown key kind %q", kind)
	}

	require.NoError(t, err)

	return key
}

func keyPEMOf(t *testing.T, key crypto.Signer) []byte {
	t.Helper()

	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		der, err := x509.MarshalECPrivateKey(k)
		require.NoError(t, err)

		return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	case *rsa.PrivateKey:
		return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
	case ed25519.PrivateKey:
		der, err := x509.MarshalPKCS8PrivateKey(k)
		require.NoError(t, err)

		return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	default:
		t.Fatalf("unsupported key type %T", key)

		return nil
	}
}

// rsaPublicKeyOfBits builds a public key whose modulus has exactly bits bits;
// it is only good for size checks, never for signing.
func rsaPublicKeyOfBits(bits int) *rsa.PublicKey {
	return &rsa.PublicKey{N: new(big.Int).Lsh(big.NewInt(1), uint(bits-1)), E: 65537}
}

func mintKeyCert(t *testing.T, key crypto.Signer, tmpl certTemplate) testKeyCert {
	t.Helper()

	if tmpl.notBefore.IsZero() {
		tmpl.notBefore = time.Now().Add(-time.Minute)
	}

	if tmpl.notAfter.IsZero() {
		tmpl.notAfter = time.Now().Add(time.Hour)
	}

	x509Tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    tmpl.notBefore,
		NotAfter:     tmpl.notAfter,
	}

	for _, raw := range tmpl.uris {
		u, err := url.Parse(raw)
		require.NoError(t, err)

		x509Tmpl.URIs = append(x509Tmpl.URIs, u)
	}

	if tmpl.padding > 0 {
		x509Tmpl.ExtraExtensions = []pkix.Extension{{
			Id:    asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1},
			Value: make([]byte, tmpl.padding),
		}}
	}

	der, err := x509.CreateCertificate(rand.Reader, x509Tmpl, x509Tmpl, key.Public(), key)
	require.NoError(t, err)

	parsed, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return testKeyCert{
		keyPEM:  keyPEMOf(t, key),
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		cert:    parsed,
	}
}

func TestNewSpiffeSigner(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		kind        string
		tmpl        certTemplate
		otherKey    string
		garbageKey  bool
		garbageCert bool
		wantErr     string
		wantWindow  bool
	}{
		{name: "p-256 certificate", kind: "p256"},
		{name: "p-384 certificate", kind: "p384"},
		{name: "rsa-2048 certificate", kind: "rsa"},
		{name: "ed25519 certificate", kind: "ed25519"},
		{name: "p-521 key cannot sign claims", kind: "p521", wantErr: "unsupported signing key"},
		{name: "certificate without a spiffe URI SAN", kind: "p256", tmpl: certTemplate{uris: []string{"https://acme.com/agent"}}, wantErr: "spiffe://"},
		{name: "oversized certificate", kind: "p256", tmpl: certTemplate{padding: maxCertificateDERSize}, wantErr: "at most 16384"},
		{name: "key does not match the certificate", kind: "p256", otherKey: "p256", wantErr: "does not match"},
		{name: "expired certificate", kind: "p256", tmpl: certTemplate{notBefore: now.Add(-2 * time.Hour), notAfter: now.Add(-time.Hour)}, wantErr: "not valid now", wantWindow: true},
		{name: "certificate not yet valid", kind: "p256", tmpl: certTemplate{notBefore: now.Add(time.Hour), notAfter: now.Add(2 * time.Hour)}, wantErr: "not valid now", wantWindow: true},
		{name: "certificate is not a certificate", kind: "p256", garbageCert: true, wantErr: "parse certificate"},
		{name: "key is not a key", kind: "p256", garbageKey: true, wantErr: "parse private key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := tt.tmpl
			if tmpl.uris == nil {
				tmpl.uris = []string{testSpiffeID}
			}

			kc := mintKeyCert(t, newKey(t, tt.kind), tmpl)

			keyPEM := kc.keyPEM
			if tt.otherKey != "" {
				keyPEM = keyPEMOf(t, newKey(t, tt.otherKey))
			}

			if tt.garbageKey {
				keyPEM = []byte("not a key")
			}

			certPEM := kc.certPEM
			if tt.garbageCert {
				certPEM = []byte("not a certificate")
			}

			signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				if tt.wantWindow {
					require.ErrorContains(t, err, kc.cert.NotBefore.UTC().Format(time.RFC3339))
					require.ErrorContains(t, err, kc.cert.NotAfter.UTC().Format(time.RFC3339))
				}

				return
			}

			require.NoError(t, err)
			assert.True(t, kc.cert.NotAfter.Equal(signer.NotAfter()))
			assert.Equal(t, kc.cert.Raw, signer.CertificateDER())
			assert.True(t, signer.SubjectMatchesCertificate(testSpiffeID))
		})
	}
}

func TestCheckSigningKey(t *testing.T) {
	tests := []struct {
		name    string
		pub     crypto.PublicKey
		wantErr string
	}{
		{name: "p-256", pub: newKey(t, "p256").Public()},
		{name: "p-384", pub: newKey(t, "p384").Public()},
		{name: "ed25519", pub: newKey(t, "ed25519").Public()},
		{name: "rsa 2048 bits", pub: rsaPublicKeyOfBits(2048)},
		{name: "rsa 8192 bits", pub: rsaPublicKeyOfBits(8192)},
		{name: "rsa 2047 bits", pub: rsaPublicKeyOfBits(2047), wantErr: "unsupported RSA key size: 2047 bits"},
		{name: "rsa 8193 bits", pub: rsaPublicKeyOfBits(8193), wantErr: "unsupported RSA key size: 8193 bits"},
		{name: "p-521", pub: newKey(t, "p521").Public(), wantErr: "unsupported ECDSA curve"},
		{name: "not a key", pub: "not a key", wantErr: "unsupported public key type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := identityv1.CheckSigningKey(tt.pub)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

// A key signer checks nothing at load, so an unsupported key surfaces when a
// claim is signed with it.
func TestNewKeySigner_UnsupportedKey(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		wantErr string
	}{
		{name: "p-521", kind: "p521", wantErr: "unsupported ECDSA curve"},
		{name: "rsa-1024", kind: "rsa-weak", wantErr: "unsupported RSA key size: 1024 bits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := identityv1.NewKeySigner(keyPEMOf(t, newKey(t, tt.kind)))
			require.ErrorContains(t, err, "unsupported signing key: ")
			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, signer)
		})
	}
}

func TestVerifyOwnershipClaim_BoundsTheEmbeddedCertificate(t *testing.T) {
	_, _, anchor := generateTestSpiffeKeyCert(t, testSpiffeID)

	tests := []struct {
		name        string
		certificate string
		wantErr     string
	}{
		{name: "encoded certificate longer than the bound", certificate: strings.Repeat("A", base64.StdEncoding.EncodedLen(maxCertificateDERSize)+4), wantErr: "exceeds"},
		{name: "decoded certificate longer than the bound", certificate: base64.StdEncoding.EncodeToString(make([]byte, maxCertificateDERSize+1)), wantErr: "at most"},
		{name: "certificate at the bound is decoded and parsed", certificate: base64.StdEncoding.EncodeToString(make([]byte, maxCertificateDERSize)), wantErr: "parse certificate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claim := &identityv1.OwnershipClaim{
				Subject:     testSpiffeID,
				SignedAt:    "2026-09-14T00:00:00Z",
				Signature:   "not-a-jws",
				Certificate: &tt.certificate,
			}

			result := identityv1.VerifyOwnershipClaim(t.Context(), claim, "baguqeera-cid", testSpiffeID, nil, []*x509.Certificate{anchor})
			assert.False(t, result.Verified)
			assert.Contains(t, result.Error, tt.wantErr)
		})
	}
}
