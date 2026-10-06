// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
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

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers/agntcy"
	clientjws "github.com/agntcy/dir/client/utils/jws"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func compactBadge(t *testing.T, key ed25519.PrivateKey, value any) string {
	t.Helper()

	data, err := json.Marshal(value)
	require.NoError(t, err)
	sig, err := clientjws.Sign(key, data)
	require.NoError(t, err)

	parts := strings.Split(sig, ".")

	return parts[0] + "." + base64.RawURLEncoding.EncodeToString(data) + "." + parts[2]
}

func TestVerifierWithIdentityNode(t *testing.T) {
	agentPub, agentKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	publicJWK, err := jwk.FromRaw(agentPub)
	require.NoError(t, err)
	agentJWK, err := json.Marshal(publicJWK)
	require.NoError(t, err)

	definition := map[string]any{"name": "security-agent", "version": "1", "annotations": map[string]any{"agntcy.dir/identity": "agntcy://Agent-One"}}
	data, err := structpb.NewStruct(definition)
	require.NoError(t, err)

	record := &corev1.Record{Data: data}
	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: "agntcy://Agent-One", RecordCid: record.GetCid(), SignedAt: time.Now().UTC().Format(time.RFC3339)}
	payload, err := claim.GetPayload()
	require.NoError(t, err)
	claim.Signature, err = clientjws.Sign(agentKey, payload)
	require.NoError(t, err)

	badgeExpiry := time.Now().Add(5 * time.Minute).UTC().Truncate(time.Second)
	badgeSubject := map[string]any{"id": "Agent-One", "badge": definition}
	credential := map[string]any{"type": []string{"VerifiableCredential", "AgentBadge"}, "validUntil": badgeExpiry.Format(time.RFC3339), "credentialSubject": badgeSubject}

	var badge atomic.Value
	badge.Store(compactBadge(t, agentKey, credential))

	var identityAvailable atomic.Bool
	identityAvailable.Store(true)

	var badgeAccepted atomic.Bool
	badgeAccepted.Store(true)

	var identityKeysAvailable atomic.Bool
	identityKeysAvailable.Store(true)

	node := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !identityAvailable.Load() {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)

			return
		}

		switch r.URL.Path {
		case "/v1alpha1/id/resolve":
			methods := []map[string]any{}
			if identityKeysAvailable.Load() {
				methods = append(methods, map[string]any{"publicKeyJwk": json.RawMessage(agentJWK)})
			}

			writeJSON(w, map[string]any{"resolverMetadata": map[string]any{"id": "Agent-One", "verificationMethod": methods}})
		case "/v1alpha1/vc/Agent-One/.well-known/vcs.json":
			currentBadge, ok := badge.Load().(string)
			if !ok {
				t.Error("invalid badge fixture")

				return
			}

			writeJSON(w, wellKnownResponse{VCs: []credentialEnvelope{{EnvelopeType: "CREDENTIAL_ENVELOPE_TYPE_JOSE", Value: currentBadge}}})
		case "/v1alpha1/vc/verify":
			writeJSON(w, verifyResponse{Status: badgeAccepted.Load()})
		default:
			http.NotFound(w, r)
		}
	}))
	defer node.Close()

	resultKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	s := &server{identityURL: node.URL, verifierID: "agntcy-identity-verifier", privateKey: resultKey, client: node.Client(), token: "test-token", ttl: 30 * time.Minute}

	verifier := httptest.NewTLSServer(s.handler())
	defer verifier.Close()

	der, err := x509.MarshalPKIXPublicKey(&resultKey.PublicKey)
	require.NoError(t, err)
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "result.pem")
	tokenPath := filepath.Join(dir, "token")

	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(tokenPath, []byte("test-token"), 0o600))
	r, err := agntcy.New(agntcy.Config{VerifierURL: verifier.URL, VerifierTrustBundleFile: keyPath, BearerTokenFile: tokenPath}, verifier.Client())
	require.NoError(t, err)
	keys, err := r.Resolve(context.Background(), claim.GetSubject(), nil)
	require.NoError(t, err)
	_, err = clientidentity.Verify(claim, record.GetCid(), claim.GetSubject(), keys...)
	require.NoError(t, err)
	until, err := r.VerifyEvidence(context.Background(), claim)
	require.NoError(t, err)
	assert.True(t, until.Equal(badgeExpiry), "badge expiry must bound the signed observation")

	t.Run("different record CID", func(t *testing.T) {
		other, ok := proto.Clone(claim).(*identityv1.Claim)
		require.True(t, ok)

		other.RecordCid = "other-cid"
		_, err := r.VerifyEvidence(context.Background(), other)
		require.Error(t, err)
	})
	t.Run("wrong badge subject", func(t *testing.T) {
		subject := badgeSubject
		subject["id"] = "Other"

		badge.Store(compactBadge(t, agentKey, credential))

		_, err := r.VerifyEvidence(context.Background(), claim)
		require.Error(t, err)

		subject["id"] = "Agent-One"

		badge.Store(compactBadge(t, agentKey, credential))
	})
	t.Run("different definition", func(t *testing.T) {
		definition["version"] = "2"

		badge.Store(compactBadge(t, agentKey, credential))

		_, err := r.VerifyEvidence(context.Background(), claim)
		require.Error(t, err)

		definition["version"] = "1"

		badge.Store(compactBadge(t, agentKey, credential))
	})
	t.Run("expired badge", func(t *testing.T) {
		credential["validUntil"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
		badge.Store(compactBadge(t, agentKey, credential))

		_, err := r.VerifyEvidence(context.Background(), claim)
		require.Error(t, err)

		credential["validUntil"] = badgeExpiry.Format(time.RFC3339)
		badge.Store(compactBadge(t, agentKey, credential))
	})
	t.Run("node rejects badge", func(t *testing.T) {
		badgeAccepted.Store(false)

		_, err := r.VerifyEvidence(context.Background(), claim)
		require.Error(t, err)
		badgeAccepted.Store(true)
	})
	t.Run("withdrawn keys are authoritative rejection", func(t *testing.T) {
		identityKeysAvailable.Store(false)

		_, err := r.Resolve(context.Background(), claim.GetSubject(), nil)
		require.ErrorContains(t, err, "rejected")
		identityKeysAvailable.Store(true)
	})
	t.Run("temporary node failure", func(t *testing.T) {
		identityAvailable.Store(false)

		_, err := r.VerifyEvidence(context.Background(), claim)
		require.ErrorContains(t, err, "503")
		identityAvailable.Store(true)
	})
	t.Run("authentication required", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, verifier.URL+"/v1/resolve", bytes.NewReader([]byte(`{}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := verifier.Client().Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestCanonicalPayloadIncludesRoleAndExpiry(t *testing.T) {
	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, RecordCid: "cid", Subject: "agntcy://Agent", SignedAt: time.Now().UTC().Format(time.RFC3339)}
	until := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	claim.ExpiresAt = &until
	payload, err := claim.GetPayload()
	require.NoError(t, err)
	parsed, err := parseCanonicalPayload(payload, "signature")
	require.NoError(t, err)
	assert.Equal(t, claim.GetExpiresAt(), parsed.GetExpiresAt())
	assert.Equal(t, claim.GetRole(), parsed.GetRole())

	for _, invalid := range [][]byte{[]byte("cid|agntcy:Agent|timestamp"), append(payload, []byte(" {}")...), bytes.Replace(payload, []byte("CLAIM_ROLE_IDENTITY"), []byte("not-a-role"), 1)} {
		_, err := parseCanonicalPayload(invalid, "signature")
		require.Error(t, err)
	}
}
