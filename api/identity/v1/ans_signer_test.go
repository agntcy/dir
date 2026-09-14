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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/lestrrat-go/jwx/v2/cert"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAnsName    = "ans://v1.0.0.agent.example.com"
	testRSABits    = 2048
	oversizedBytes = 17 << 10
)

// testKeyCert is a private key with a self-signed certificate over it.
type testKeyCert struct {
	key     crypto.Signer
	keyPEM  []byte
	certPEM []byte
	cert    *x509.Certificate
}

// newKey generates a private key of the named kind.
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
		key, err = rsa.GenerateKey(rand.Reader, testRSABits)
	case "ed25519":
		_, key, err = ed25519.GenerateKey(rand.Reader)
	default:
		t.Fatalf("unknown key kind %q", kind)
	}

	require.NoError(t, err)

	return key
}

// keyPEMOf encodes key in the PEM form parsePrivateKey accepts for its type.
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

// mintKeyCert self-signs a certificate over key carrying uris as URI SANs.
// padding adds an opaque extension of that many bytes to grow the DER.
func mintKeyCert(t *testing.T, key crypto.Signer, uris []string, padding int) testKeyCert {
	t.Helper()

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}

	for _, raw := range uris {
		u, err := url.Parse(raw)
		require.NoError(t, err)

		tmpl.URIs = append(tmpl.URIs, u)
	}

	if padding > 0 {
		tmpl.ExtraExtensions = []pkix.Extension{{
			Id:    asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1},
			Value: make([]byte, padding),
		}}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	require.NoError(t, err)

	parsed, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return testKeyCert{
		key:     key,
		keyPEM:  keyPEMOf(t, key),
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		cert:    parsed,
	}
}

func TestNewAnsSigner(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		uris    []string
		padding int
		// otherKey, when set, is the key whose PEM is passed instead of the
		// certificate's own key.
		otherKey string
		// garbageCert and garbageKey replace the PEM with text that is not one.
		garbageCert bool
		garbageKey  bool
		wantErr     string
	}{
		{name: "p-256 identity certificate", kind: "p256", uris: []string{testAnsName}},
		{name: "p-384 identity certificate", kind: "p384", uris: []string{testAnsName}},
		{name: "rsa identity certificate", kind: "rsa", uris: []string{testAnsName}},
		{name: "ed25519 identity certificate", kind: "ed25519", uris: []string{testAnsName}},
		{name: "one of several URI SANs names the agent", kind: "p256", uris: []string{"spiffe://acme.com/agent", testAnsName}},
		{name: "p-521 key cannot sign claims", kind: "p521", uris: []string{testAnsName}, wantErr: "unsupported"},
		{name: "certificate without an ans URI SAN", kind: "p256", uris: []string{"spiffe://acme.com/agent"}, wantErr: "ans://"},
		{name: "oversized certificate", kind: "p256", uris: []string{testAnsName}, padding: oversizedBytes, wantErr: "bytes"},
		{name: "key does not match the certificate", kind: "p256", uris: []string{testAnsName}, otherKey: "p256", wantErr: "does not match"},
		{name: "certificate is not a certificate", kind: "p256", uris: []string{testAnsName}, garbageCert: true, wantErr: "parse certificate"},
		{name: "key is not a key", kind: "p256", uris: []string{testAnsName}, garbageKey: true, wantErr: "parse private key"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kc := mintKeyCert(t, newKey(t, tc.kind), tc.uris, tc.padding)

			keyPEM := kc.keyPEM
			if tc.otherKey != "" {
				keyPEM = keyPEMOf(t, newKey(t, tc.otherKey))
			}

			if tc.garbageKey {
				keyPEM = []byte("not a key")
			}

			certPEM := kc.certPEM
			if tc.garbageCert {
				certPEM = []byte("not a certificate")
			}

			signer, err := identityv1.NewAnsSigner(keyPEM, certPEM)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.True(t, kc.cert.NotAfter.Equal(signer.NotAfter()))
		})
	}
}

func TestAnsSigner_SignCarriesTheCertificate(t *testing.T) {
	payload := identityv1.CanonicalBytes("baguqeera-cid", testAnsName, "2026-09-14T00:00:00Z")

	for _, kind := range []string{"p256", "p384", "rsa", "ed25519"} {
		t.Run(kind, func(t *testing.T) {
			kc := mintKeyCert(t, newKey(t, kind), []string{testAnsName}, 0)

			signer, err := identityv1.NewAnsSigner(kc.keyPEM, kc.certPEM)
			require.NoError(t, err)

			jwsCompact, err := signer.Sign(payload)
			require.NoError(t, err)

			carried, err := identityv1.CertificateFromJWS(jwsCompact)
			require.NoError(t, err)
			assert.Equal(t, kc.cert.Raw, carried.Raw)

			require.NoError(t, identityv1.VerifyJWS(jwsCompact, carried.PublicKey, payload))
			require.Error(t, identityv1.VerifyJWS(jwsCompact, carried.PublicKey, []byte("other payload")))

			assert.True(t, signer.SubjectMatchesCertificate(testAnsName))
			assert.False(t, signer.SubjectMatchesCertificate("ans://v1.0.0.AGENT.example.com"), "the signer compares exactly")
			assert.False(t, signer.SubjectMatchesCertificate("ans://v2.0.0.agent.example.com"))
		})
	}
}

func TestSignIdentityClaim_AnsSigner(t *testing.T) {
	kc := mintKeyCert(t, newKey(t, "p256"), []string{testAnsName}, 0)

	signer, err := identityv1.NewAnsSigner(kc.keyPEM, kc.certPEM)
	require.NoError(t, err)

	t.Run("claim carries the certificate in the jws only", func(t *testing.T) {
		claim := identityv1.NewIdentityClaim(testAnsName)
		require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

		assert.Empty(t, claim.GetCertificate(), "ans claims carry the certificate in x5c, not in field 5")

		carried, err := identityv1.CertificateFromJWS(claim.GetSignature())
		require.NoError(t, err)
		assert.Equal(t, kc.cert.Raw, carried.Raw)

		result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", testAnsName, &fakeResolver{verified: true}, nil)
		assert.True(t, result.Verified, result.Error)
		assert.Equal(t, "ans", result.SubjectType)
	})

	t.Run("subject the certificate does not name is refused before signing", func(t *testing.T) {
		claim := identityv1.NewIdentityClaim("ans://v2.0.0.agent.example.com")
		err := identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer)
		require.ErrorContains(t, err, "does not match")
		assert.Empty(t, claim.GetSignature())
	})
}

func TestSpiffeSigner_ReportsExpiry(t *testing.T) {
	kc := mintKeyCert(t, newKey(t, "p256"), []string{testSpiffeID}, 0)

	signer, err := identityv1.NewSpiffeSigner(kc.keyPEM, kc.certPEM)
	require.NoError(t, err)

	assert.True(t, kc.cert.NotAfter.Equal(signer.NotAfter()))
}

func TestCertificateFromJWS(t *testing.T) {
	kc := mintKeyCert(t, newKey(t, "p256"), []string{testAnsName}, 0)
	payload := identityv1.CanonicalBytes("baguqeera-cid", testAnsName, "2026-09-14T00:00:00Z")

	signWithHeaders := func(t *testing.T, hdrs jws.Headers, opts ...jws.SignOption) string {
		t.Helper()

		opts = append(opts, jws.WithKey(jwa.ES256, kc.key, jws.WithProtectedHeaders(hdrs)))

		sig, err := jws.Sign(nil, append(opts, jws.WithDetachedPayload(payload))...)
		require.NoError(t, err)

		return string(sig)
	}

	chainOf := func(t *testing.T, entries ...string) jws.Headers {
		t.Helper()

		var chain cert.Chain
		for _, entry := range entries {
			require.NoError(t, chain.AddString(entry))
		}

		hdrs := jws.NewHeaders()
		require.NoError(t, hdrs.Set(jws.X509CertChainKey, &chain))

		return hdrs
	}

	leaf := base64.StdEncoding.EncodeToString(kc.cert.Raw)
	oversized := mintKeyCert(t, kc.key, []string{testAnsName}, oversizedBytes)

	tests := []struct {
		name    string
		input   func(t *testing.T) string
		wantErr string
	}{
		{
			name: "leaf in x5c",
			input: func(t *testing.T) string {
				t.Helper()

				return signWithHeaders(t, chainOf(t, leaf))
			},
		},
		{
			name: "key signer jws carries no certificate",
			input: func(t *testing.T) string {
				t.Helper()

				signer, err := identityv1.NewKeySigner(kc.keyPEM)
				require.NoError(t, err)

				out, err := signer.Sign(payload)
				require.NoError(t, err)

				return out
			},
			wantErr: "no x5c",
		},
		{
			name: "json serialization is not compact",
			input: func(t *testing.T) string {
				t.Helper()

				sig, err := jws.Sign(payload, jws.WithKey(jwa.ES256, kc.key, jws.WithProtectedHeaders(chainOf(t, leaf))), jws.WithJSON())
				require.NoError(t, err)

				return string(sig)
			},
			wantErr: "compact",
		},
		{
			name:    "two segments",
			input:   func(*testing.T) string { return "a.b" },
			wantErr: "compact",
		},
		{
			name:    "header is not base64url",
			input:   func(*testing.T) string { return "!!!..sig" },
			wantErr: "header",
		},
		{
			name:    "header is not json",
			input:   func(*testing.T) string { return base64.RawURLEncoding.EncodeToString([]byte("nope")) + "..sig" },
			wantErr: "header",
		},
		{
			name: "x5c is not standard base64",
			input: func(t *testing.T) string {
				t.Helper()

				return signWithHeaders(t, chainOf(t, "!!!!"))
			},
			wantErr: "standard base64",
		},
		{
			name: "two certificates in the chain",
			input: func(t *testing.T) string {
				t.Helper()

				return signWithHeaders(t, chainOf(t, leaf, leaf))
			},
			wantErr: "leaf only",
		},
		{
			name: "oversized leaf",
			input: func(t *testing.T) string {
				t.Helper()

				return signWithHeaders(t, chainOf(t, base64.StdEncoding.EncodeToString(oversized.cert.Raw)))
			},
			wantErr: "bytes",
		},
		{
			name: "x5c is not a certificate",
			input: func(t *testing.T) string {
				t.Helper()

				return signWithHeaders(t, chainOf(t, base64.StdEncoding.EncodeToString([]byte("not a certificate"))))
			},
			wantErr: "parse x5c certificate",
		},
		{
			name: "compact text too large",
			input: func(*testing.T) string {
				return strings.Repeat("a", 65<<10) + ".." + "sig"
			},
			wantErr: "bytes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := identityv1.CertificateFromJWS(tc.input(t))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, kc.cert.Raw, got.Raw)
		})
	}
}

func TestCheckSigningKey(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		wantErr string
	}{
		{name: "p-256", kind: "p256"},
		{name: "p-384", kind: "p384"},
		{name: "rsa", kind: "rsa"},
		{name: "ed25519", kind: "ed25519"},
		{name: "p-521", kind: "p521", wantErr: "unsupported"},
		{name: "not a key", kind: "", wantErr: "unsupported"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pub crypto.PublicKey = "not a key"
			if tc.kind != "" {
				pub = newKey(t, tc.kind).Public()
			}

			err := identityv1.CheckSigningKey(pub)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

// A key signer checks nothing at load, so an unsupported key surfaces when it
// signs.
func TestKeySigner_UnsupportedKeyFailsAtSign(t *testing.T) {
	signer, err := identityv1.NewKeySigner(keyPEMOf(t, newKey(t, "p521")))
	require.NoError(t, err)

	_, err = signer.Sign([]byte("payload"))
	require.ErrorContains(t, err, "unsupported")
}

func TestNewCertificateSignerFromFile(t *testing.T) {
	writeFiles := func(t *testing.T, kc testKeyCert) (string, string) {
		t.Helper()

		dir := t.TempDir()
		keyPath := filepath.Join(dir, "key.pem")
		certPath := filepath.Join(dir, "cert.pem")

		require.NoError(t, os.WriteFile(keyPath, kc.keyPEM, 0o600))
		require.NoError(t, os.WriteFile(certPath, kc.certPEM, 0o600))

		return keyPath, certPath
	}

	tests := []struct {
		name     string
		uris     []string
		files    func(t *testing.T, kc testKeyCert) (string, string)
		wantType any
		wantErr  string
	}{
		{name: "spiffe certificate selects the spiffe signer", uris: []string{testSpiffeID}, files: writeFiles, wantType: &identityv1.SpiffeSigner{}},
		{name: "ans certificate selects the ans signer", uris: []string{testAnsName}, files: writeFiles, wantType: &identityv1.AnsSigner{}},
		{name: "certificate with neither scheme", uris: nil, files: writeFiles, wantErr: "spiffe:// or ans://"},
		{name: "certificate with both schemes is ambiguous", uris: []string{testSpiffeID, testAnsName}, files: writeFiles, wantErr: "both"},
		{
			name: "key file does not match the certificate",
			uris: []string{testAnsName},
			files: func(t *testing.T, kc testKeyCert) (string, string) {
				t.Helper()

				keyPath, certPath := writeFiles(t, kc)

				require.NoError(t, os.WriteFile(keyPath, keyPEMOf(t, newKey(t, "p256")), 0o600))

				return keyPath, certPath
			},
			wantErr: "does not match",
		},
		{
			name: "certificate file is not a certificate",
			uris: []string{testAnsName},
			files: func(t *testing.T, kc testKeyCert) (string, string) {
				t.Helper()

				keyPath, certPath := writeFiles(t, kc)

				require.NoError(t, os.WriteFile(certPath, []byte("not a certificate"), 0o600))

				return keyPath, certPath
			},
			wantErr: "parse certificate",
		},
		{
			name: "missing key file",
			uris: []string{testAnsName},
			files: func(t *testing.T, kc testKeyCert) (string, string) {
				t.Helper()

				_, certPath := writeFiles(t, kc)

				return filepath.Join(t.TempDir(), "missing.pem"), certPath
			},
			wantErr: "key",
		},
		{
			name: "missing certificate file",
			uris: []string{testAnsName},
			files: func(t *testing.T, kc testKeyCert) (string, string) {
				t.Helper()

				keyPath, _ := writeFiles(t, kc)

				return keyPath, filepath.Join(t.TempDir(), "missing.pem")
			},
			wantErr: "cert",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kc := mintKeyCert(t, newKey(t, "p256"), tc.uris, 0)
			keyPath, certPath := tc.files(t, kc)

			signer, err := identityv1.NewCertificateSignerFromFile(keyPath, certPath)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.IsType(t, tc.wantType, signer)
		})
	}
}
