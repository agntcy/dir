// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agntcy

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientjws "github.com/agntcy/dir/client/utils/jws"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func embeddedResult(t *testing.T, signer crypto.Signer, result VerificationResult) string {
	t.Helper()

	payload, err := json.Marshal(result)
	require.NoError(t, err)
	signature, err := clientjws.Sign(signer, payload)
	require.NoError(t, err)

	parts := strings.Split(signature, ".")

	return parts[0] + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + parts[2]
}

func authority(t *testing.T, mutate func(*VerificationResult)) (*Resolver, *identityv1.Claim, *atomic.Int32) {
	t.Helper()

	pub, signer, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	agentPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	agentJWK, err := jwk.FromRaw(agentPub)
	require.NoError(t, err)
	keyJSON, err := json.Marshal(agentJWK)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, RecordCid: "record-one", Subject: "agntcy://Agent-One", SignedAt: time.Now().UTC().Format(time.RFC3339), Signature: "claim-signature"}
	calls := &atomic.Int32{}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)

		var input json.RawMessage
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			t.Error(err)
			http.Error(w, "invalid request", http.StatusBadRequest)

			return
		}

		result := VerificationResult{Version: ProtocolVersion, Kind: "verify", Verifier: "agntcy-identity-verifier", Profile: Profile, PolicyVersion: SubjectKeyPolicy, Verified: true, Subject: claim.GetSubject(), RecordCID: claim.GetRecordCid(), RequestDigest: DigestRequest(input), CheckedAt: time.Now().UTC().Format(time.RFC3339), ExpiresAt: time.Now().Add(20 * time.Minute).UTC().Format(time.RFC3339)}
		result.PublicKeys = []json.RawMessage{keyJSON}
		result.Checks = VerificationChecks{Identity: true, Badge: true}

		if req.URL.Path == "/v1/verify" {
			var request VerificationRequest
			if err := json.Unmarshal(input, &request); err != nil {
				t.Error(err)
				http.Error(w, "invalid request", http.StatusBadRequest)

				return
			}

			result.Profile = request.Profile
			result.Checks.Badge = request.Profile == Profile
		}

		if req.URL.Path == "/v1/resolve" {
			result.Kind = "resolve"
			result.Profile = KeyProfile
			result.RecordCID = ""
			result.PublicKeys = []json.RawMessage{keyJSON}
		}

		if mutate != nil {
			mutate(&result)
		}

		_ = json.NewEncoder(w).Encode(VerificationResponse{ResultJWS: embeddedResult(t, signer, result)})
	}))
	t.Cleanup(srv.Close)

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "verifier.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o600))
	r, err := New(Config{VerifierURL: srv.URL, VerifierTrustBundleFile: path}, srv.Client())
	require.NoError(t, err)

	return r, claim, calls
}

func TestResolveClaimUsesOneRequest(t *testing.T) {
	r, claim, calls := authority(t, nil)
	resolution, err := r.ResolveClaim(context.Background(), claim)
	require.NoError(t, err)
	require.Len(t, resolution.PublicKeys, 1)
	assert.IsType(t, ed25519.PublicKey{}, resolution.PublicKeys[0])
	assert.True(t, resolution.ValidUntil.After(time.Now()))
	assert.EqualValues(t, 1, calls.Load(), "combined operation must not call /resolve first")

	other, ok := proto.Clone(claim).(*identityv1.Claim)
	require.True(t, ok)

	other.RecordCid = "record-two"
	_, err = r.ResolveClaim(context.Background(), other)
	require.ErrorContains(t, err, "bound")
	assert.EqualValues(t, 2, calls.Load(), "another CID must make its own request")
}

func TestRejectsSignedButInvalidResults(t *testing.T) {
	tests := map[string]func(*VerificationResult){
		"old protocol":   func(r *VerificationResult) { r.Version = "agntcy.identity-verification.v1" },
		"subject":        func(r *VerificationResult) { r.Subject = "agntcy://Other" },
		"cid":            func(r *VerificationResult) { r.RecordCID = "other-cid" },
		"digest":         func(r *VerificationResult) { r.RequestDigest = "sha256:other" },
		"profile":        func(r *VerificationResult) { r.Profile = KeyProfile },
		"policy":         func(r *VerificationResult) { r.PolicyVersion = "unaccepted" },
		"verifier":       func(r *VerificationResult) { r.Verifier = "unaccepted" },
		"kind":           func(r *VerificationResult) { r.Kind = "resolve" },
		"expired":        func(r *VerificationResult) { r.ExpiresAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339) },
		"stale":          func(r *VerificationResult) { r.CheckedAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339) },
		"future":         func(r *VerificationResult) { r.CheckedAt = time.Now().Add(time.Hour).UTC().Format(time.RFC3339) },
		"rejected":       func(r *VerificationResult) { r.Verified = false },
		"identity check": func(r *VerificationResult) { r.Checks.Identity = false },
		"badge check":    func(r *VerificationResult) { r.Checks.Badge = false },
		"missing keys":   func(r *VerificationResult) { r.PublicKeys = nil },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			r, claim, _ := authority(t, mutate)
			_, err := r.ResolveClaim(context.Background(), claim)
			require.Error(t, err)
		})
	}
}

func TestKeyResponseMustBeAccepted(t *testing.T) {
	r, claim, _ := authority(t, func(r *VerificationResult) { r.PublicKeys = nil })
	_, err := r.Resolve(context.Background(), claim.GetSubject(), nil)
	require.Error(t, err)
	_, err = r.Resolve(context.Background(), claim.GetSubject(), []byte("grafted-certificate"))
	require.ErrorContains(t, err, "certificates")
}

func TestControlOnlyProfileStillHasDeadline(t *testing.T) {
	r, claim, calls := authority(t, func(result *VerificationResult) {
		assert.Equal(t, KeyProfile, result.Profile)
		assert.False(t, result.Checks.Badge)
	})
	r.config.RequireAgentBadge = new(false)
	r.config.Profile = KeyProfile
	resolution, err := r.ResolveClaim(context.Background(), claim)
	require.NoError(t, err)
	assert.True(t, resolution.ValidUntil.After(time.Now()))
	assert.EqualValues(t, 1, calls.Load())
}

func TestSignatureMustUsePinnedAuthority(t *testing.T) {
	r, claim, _ := authority(t, nil)
	other, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	r.publicKeys = []crypto.PublicKey{other}
	_, err = r.ResolveClaim(context.Background(), claim)
	require.ErrorContains(t, err, "signature")
}

func TestMaximumAgeCapsResult(t *testing.T) {
	r, claim, _ := authority(t, func(r *VerificationResult) { r.ExpiresAt = time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339) })
	r.config.MaxVerificationAge = time.Minute
	resolution, err := r.ResolveClaim(context.Background(), claim)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Minute), resolution.ValidUntil, 2*time.Second)
}

func TestAgentIDCanonicalSyntax(t *testing.T) {
	for _, subject := range []string{"agntcy:Agent", "agntcy://", "agntcy://.", "agntcy://..", "agntcy://Agent/path", "agntcy://Agent?x", "agntcy://Agent#x", "agntcy://Agent:443", "agntcy://user@Agent", "agntcy://Agent%2Fother", "AGNTCY://Agent", "agntcy://unicode-ø"} {
		_, err := AgentID(subject)
		require.Error(t, err, subject)
	}

	id, err := AgentID("agntcy://AGNTCY-Agent_1.~")
	require.NoError(t, err)
	assert.Equal(t, "AGNTCY-Agent_1.~", id)
}

func TestVerifierURLMustBeHTTPS(t *testing.T) {
	for _, url := range []string{"http://localhost", "https://user@example.com", "https://example.com?x", "https://example.com#fragment"} {
		_, err := New(Config{VerifierURL: url}, nil)
		require.Error(t, err, url)
	}
}
