// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSignatureVerification_Identity pins the URI-style identifier produced
// for catalogv1.TrustManifest.identity. The DB columns are kept verbatim so
// the verifier output is preserved; Identity() is the single point that
// normalizes scheme noise (OIDC) and PEM armor (key) into a clean URI.
func TestSignatureVerification_Identity(t *testing.T) {
	t.Parallel()

	// SPKI for a fixed P-256 public key. Re-emitted in the "want" column as
	// the single-line base64 body — no BEGIN/END armor, no newlines.
	const samplePEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5GHp9pFg4MrwLZuVBX0rmfTJoVMP
kJFatF87H3DIEfYuwjfVTyO7NuFMwxy/m2kp7IOmtuj59ZJDlYQGcJeAfw==
-----END PUBLIC KEY-----
`

	const sampleSPKIBase64 = "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE5GHp9pFg4MrwLZuVBX0rmfTJoVMPkJFatF87H3DIEfYuwjfVTyO7NuFMwxy/m2kp7IOmtuj59ZJDlYQGcJeAfw=="

	tests := []struct {
		name string
		sv   *SignatureVerification
		want string
	}{
		{
			name: "oidc strips https:// from both issuer and subject",
			sv: &SignatureVerification{
				SignerType:    "oidc",
				SignerIssuer:  "https://token.actions.githubusercontent.com",
				SignerSubject: "https://github.com/agntcy/dir/.github/workflows/import-records.yaml@refs/heads/main",
			},
			want: "oidc:token.actions.githubusercontent.com:github.com/agntcy/dir/.github/workflows/import-records.yaml@refs/heads/main",
		},
		{
			name: "oidc strips http:// scheme",
			sv: &SignatureVerification{
				SignerType:    "oidc",
				SignerIssuer:  "http://accounts.example.com",
				SignerSubject: "release-bot@example.com",
			},
			want: "oidc:accounts.example.com:release-bot@example.com",
		},
		{
			name: "oidc passes through schemeless issuer + email subject",
			sv: &SignatureVerification{
				SignerType:    "oidc",
				SignerIssuer:  "accounts.acme.com",
				SignerSubject: "release-bot@acme.com",
			},
			want: "oidc:accounts.acme.com:release-bot@acme.com",
		},
		{
			name: "key strips PEM armor and collapses to single-line base64 SPKI",
			sv: &SignatureVerification{
				SignerType:      "key",
				SignerPublicKey: samplePEM,
			},
			want: "key:" + sampleSPKIBase64,
		},
		{
			name: "key with already-bare base64 input falls back cleanly",
			sv: &SignatureVerification{
				SignerType:      "key",
				SignerPublicKey: sampleSPKIBase64,
			},
			want: "key:" + sampleSPKIBase64,
		},
		{
			name: "key with stray whitespace and broken PEM falls back to armor strip",
			sv: &SignatureVerification{
				SignerType: "key",
				// Missing trailing dashes -> pem.Decode returns nil, exercising
				// the fallback path that drops armor lines and whitespace.
				SignerPublicKey: "  -----BEGIN PUBLIC KEY-----\n   abc\n   def\n-----END PUBLIC KEY\n",
				// "-----END PUBLIC KEY" (no trailing dashes) is intentionally malformed.
			},
			want: "key:abcdef",
		},
		{
			name: "key with empty body returns empty identity body",
			sv: &SignatureVerification{
				SignerType:      "key",
				SignerPublicKey: "",
			},
			want: "key:",
		},
		{
			name: "unknown signer type falls back to signer_key column",
			sv: &SignatureVerification{
				SignerType: "carrier-pigeon",
				SignerKey:  "row-pk-42",
			},
			want: "unknown:row-pk-42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.sv.Identity())
		})
	}
}
