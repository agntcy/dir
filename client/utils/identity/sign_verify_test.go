// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/url"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/stretchr/testify/require"
)

const (
	testRecordCID   = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	otherRecordCID  = "bafybeih25dih6ug3xtj73vswccw423b56qh4pj5awfj6oyf2u2x7cwaczy"
	testSubjectDNS  = "dns:acme.com"
	testSubjectDID  = "did:web:acme.com"
	testSubjectSVID = "spiffe://acme.com/agents/finance"
)

func generateKey(t *testing.T, kind string) crypto.Signer {
	t.Helper()

	switch kind {
	case "ES256":
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		return key
	case "ES384":
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)

		return key
	case "EdDSA":
		_, key, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		return key
	case "RS256":
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		return key
	default:
		t.Fatalf("unsupported key kind %q", kind)

		return nil
	}
}

func keySigner(t *testing.T, key crypto.Signer) *jws.KeySigner {
	t.Helper()

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	signer, err := jws.NewKeySigner(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil)
	require.NoError(t, err)

	return signer
}

// certPEM returns a self-signed certificate for key carrying subject as a URI SAN.
func certPEM(t *testing.T, key crypto.Signer, subject string) []byte {
	t.Helper()

	uri, err := url.Parse(subject)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: subject},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestSignVerify_RoundTrip(t *testing.T) {
	for _, kind := range []string{"ES256", "ES384", "EdDSA", "RS256"} {
		t.Run(kind, func(t *testing.T) {
			for _, role := range []identityv1.ClaimRole{identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, identityv1.ClaimRole_CLAIM_ROLE_OWNER} {
				key := generateKey(t, kind)

				claim := &identityv1.Claim{Role: role, Subject: testSubjectDNS}
				require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))
				require.NotEmpty(t, claim.GetSignature())
				require.Equal(t, testRecordCID, claim.GetRecordCid())
				require.Nil(t, claim.Certificate)

				ok, err := Verify(claim, testRecordCID, testSubjectDNS, key.Public())
				require.NoError(t, err)
				require.True(t, ok)
			}
		})
	}
}

func TestSignVerify_WithCertificate(t *testing.T) {
	key := generateKey(t, "ES256")
	cert := certPEM(t, key, testSubjectSVID)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key), WithCertificate(cert)))
	require.NotEmpty(t, claim.GetCertificate())

	ok, err := Verify(claim, testRecordCID, testSubjectSVID, key.Public())
	require.NoError(t, err)
	require.True(t, ok)

	// A raw DER certificate is accepted as well.
	block, _ := pem.Decode(cert)
	require.NotNil(t, block)

	claim = &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key), WithCertificate(block.Bytes)))
	require.NotEmpty(t, claim.GetCertificate())
}

func TestSign_RejectsInvalidInput(t *testing.T) {
	signer := keySigner(t, generateKey(t, "ES256"))

	require.Error(t, Sign(nil, testRecordCID, signer), "nil claim")
	require.Error(t, Sign(&identityv1.Claim{Subject: testSubjectDNS}, testRecordCID, signer), "unspecified role")
	require.Error(t, Sign(&identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_OWNER}, testRecordCID, signer), "empty subject")
	require.Error(t, Sign(&identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_OWNER, Subject: testSubjectDNS}, testRecordCID, nil), "nil signer")
}

func TestVerify_TamperedPayload(t *testing.T) {
	key := generateKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))

	claim.SignedAt = time.Now().Add(time.Hour).UTC().Format(time.RFC3339) // tamper after signing

	ok, err := Verify(claim, testRecordCID, testSubjectDNS, key.Public())
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_WrongKey(t *testing.T) {
	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, generateKey(t, "ES256"))))

	ok, err := Verify(claim, testRecordCID, testSubjectDNS, generateKey(t, "ES256").Public())
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_MultipleKeys(t *testing.T) {
	key := generateKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := Verify(claim, testRecordCID, testSubjectDNS, generateKey(t, "EdDSA").Public(), key.Public())
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = Verify(claim, testRecordCID, testSubjectDNS)
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_WrongSubject(t *testing.T) {
	key := generateKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := Verify(claim, testRecordCID, testSubjectDID, key.Public())
	require.False(t, ok)
	require.ErrorContains(t, err, "does not match")
}

func TestVerify_MismatchedRole(t *testing.T) {
	key := generateKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))

	// Replay the same claim, relabeled as an ownership claim for the same subject/record.
	claim.Role = identityv1.ClaimRole_CLAIM_ROLE_OWNER

	ok, err := Verify(claim, testRecordCID, testSubjectDNS, key.Public())
	require.Error(t, err)
	require.False(t, ok)
}

func TestVerify_Expired(t *testing.T) {
	key := generateKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	expired := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	claim.ExpiresAt = &expired
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := Verify(claim, testRecordCID, testSubjectDNS, key.Public())
	require.False(t, ok)
	require.ErrorContains(t, err, "expired")
}

func TestVerify_WrongRecordCID(t *testing.T) {
	key := generateKey(t, "ES256")

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
	require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))

	ok, err := Verify(claim, otherRecordCID, testSubjectDNS, key.Public())
	require.False(t, ok)
	require.ErrorContains(t, err, "record_cid")
}

type failingSigner struct{}

func (failingSigner) Sign([]byte) (string, error) { return "", errors.New("hsm unavailable") }

func (failingSigner) Public() crypto.PublicKey { return nil }

// requireUntouched asserts Sign left every field it manages as it found it.
func requireUntouched(t *testing.T, claim *identityv1.Claim) {
	t.Helper()

	require.Empty(t, claim.GetRecordCid())
	require.Empty(t, claim.GetSignedAt())
	require.Empty(t, claim.GetSignature())
	require.Nil(t, claim.Certificate)
}

func TestSign_FailureLeavesClaimUntouched(t *testing.T) {
	key := generateKey(t, "ES256")
	signer := keySigner(t, key)
	cert := certPEM(t, key, testSubjectSVID)

	tests := map[string]struct {
		claim   *identityv1.Claim
		signer  jws.Signer
		opts    []SignOption
		wantErr string
	}{
		"signer fails": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS},
			signer:  failingSigner{},
			wantErr: "hsm unavailable",
		},
		"subject not covered": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: "spiffe://acme.com/agents/other"},
			signer:  signer,
			opts:    []SignOption{WithCertificate(cert)},
			wantErr: "does not cover",
		},
		"certificate of another key": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  keySigner(t, generateKey(t, "ES256")),
			opts:    []SignOption{WithCertificate(cert)},
			wantErr: "does not match",
		},
		"certificate of another key type": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  keySigner(t, generateKey(t, "EdDSA")),
			opts:    []SignOption{WithCertificate(cert)},
			wantErr: "does not match",
		},
		"empty certificate": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  signer,
			opts:    []SignOption{WithCertificate(nil)},
			wantErr: "parse certificate",
		},
		"garbage certificate": {
			claim:   &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID},
			signer:  signer,
			opts:    []SignOption{WithCertificate([]byte("not a certificate"))},
			wantErr: "parse certificate",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := Sign(tt.claim, testRecordCID, tt.signer, tt.opts...)
			require.ErrorContains(t, err, tt.wantErr)
			requireUntouched(t, tt.claim)
		})
	}
}

func TestSign_ResigningReplacesStaleFields(t *testing.T) {
	key := generateKey(t, "ES256")
	signer := keySigner(t, key)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectSVID}
	require.NoError(t, Sign(claim, testRecordCID, signer, WithCertificate(certPEM(t, key, testSubjectSVID))))
	require.NotEmpty(t, claim.GetCertificate())

	// Signed again without a certificate, the old one must not linger.
	require.NoError(t, Sign(claim, otherRecordCID, signer))
	require.Equal(t, otherRecordCID, claim.GetRecordCid())
	require.Nil(t, claim.Certificate)
}

func TestVerify_RejectsInvalidInput(t *testing.T) {
	key := generateKey(t, "ES256")

	signed := func(mutate func(*identityv1.Claim)) *identityv1.Claim {
		claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: testSubjectDNS}
		mutate(claim)
		require.NoError(t, Sign(claim, testRecordCID, keySigner(t, key)))

		return claim
	}

	malformed := "next tuesday"

	tests := map[string]struct {
		claim    *identityv1.Claim
		expected string
		keys     []crypto.PublicKey
		wantErr  string
	}{
		"nil claim":           {nil, testSubjectDNS, []crypto.PublicKey{key.Public()}, "claim is nil"},
		"no declared subject": {signed(func(*identityv1.Claim) {}), "", []crypto.PublicKey{key.Public()}, "does not declare"},
		"no keys":             {signed(func(*identityv1.Claim) {}), testSubjectDNS, nil, "no public keys"},
		"malformed expiry":    {signed(func(c *identityv1.Claim) { c.ExpiresAt = &malformed }), testSubjectDNS, []crypto.PublicKey{key.Public()}, "invalid claim expiry"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ok, err := Verify(tt.claim, testRecordCID, tt.expected, tt.keys...)
			require.False(t, ok)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
