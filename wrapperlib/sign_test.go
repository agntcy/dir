// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	cosignpkg "github.com/sigstore/cosign/v3/pkg/cosign"
)

// genTestKeyPair generates a real cosign key pair encrypted with the fixed
// test password "test-password", for use by both sign_test.go and
// verify_test.go.
func genTestKeyPair(t *testing.T) *cosignpkg.KeysBytes {
	t.Helper()

	keys, err := cosignpkg.GenerateKeyPair(func(bool) ([]byte, error) { return []byte("test-password"), nil })
	if err != nil {
		t.Fatalf("failed to generate test key pair: %v", err)
	}

	return keys
}

func TestSignWithKey(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		keys := genTestKeyPair(t)

		req := signWithKeyRequest{
			CID:        "baecid123",
			PrivateKey: string(keys.PrivateBytes),
			Password:   "test-password",
		}

		reqJSON, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		resp := bridgeNoHandle(SignWithKey, string(reqJSON))

		var out signWithKeyResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if out.Signature == nil || out.Signature.Signature == "" {
			t.Fatal("expected a non-empty signature")
		}

		if out.PublicKey == "" {
			t.Fatal("expected a non-empty public key")
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeNoHandle(SignWithKey, `{not valid`))
	})

	t.Run("bad private key", func(t *testing.T) {
		req := signWithKeyRequest{CID: "baecid123", PrivateKey: "not a real key"}

		reqJSON, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		assertErrorNonEmpty(t, bridgeNoHandle(SignWithKey, string(reqJSON)))
	})
}

// TestSignWithOIDC only exercises input-validation error paths. Signing with
// OIDC requires live Fulcio/Rekor/OIDC-provider network calls (ephemeral key
// generation, certificate issuance, transparency log submission), which
// cannot be meaningfully unit-tested without a live network and real OIDC
// identity token -- there is no local/offline path through this code, unlike
// SignWithKey. Malformed request JSON is rejected before any network call is
// attempted, so it is safe to assert on here.
func TestSignWithOIDC(t *testing.T) {
	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeNoHandle(SignWithOIDC, `{not valid`))
	})
}

func TestSign(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path with key provider", func(t *testing.T) {
		keys := genTestKeyPair(t)

		req := mustProtoJSON(t, &signv1.SignRequest{
			RecordRef: &corev1.RecordRef{Cid: "baecid123"},
			Provider: &signv1.SignRequestProvider{
				Request: &signv1.SignRequestProvider_Key{
					Key: &signv1.SignWithKey{
						PrivateKey: string(keys.PrivateBytes),
						Password:   []byte("test-password"),
					},
				},
			},
		})

		resp := bridgeHandle(Sign, handle, req)

		var out signResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var signResp signv1.SignResponse
		mustProtoUnmarshal(t, out.Response, &signResp)

		if signResp.GetSignature().GetSignature() == "" {
			t.Fatal("expected a non-empty signature in the response")
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Sign, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		req := mustProtoJSON(t, &signv1.SignRequest{RecordRef: &corev1.RecordRef{Cid: "baecid123"}})
		assertErrorNonEmpty(t, bridgeHandle(Sign, 999999, req))
	})

	// Missing provider/record ref are validated locally by client.Sign before
	// any signing method (key or OIDC) is dispatched, so these are safe,
	// fast, deterministic error-path checks that also cover part of the
	// OIDC-routing code without making a real OIDC/network call.
	t.Run("missing provider", func(t *testing.T) {
		req := mustProtoJSON(t, &signv1.SignRequest{RecordRef: &corev1.RecordRef{Cid: "baecid123"}})
		assertErrorNonEmpty(t, bridgeHandle(Sign, handle, req))
	})

	t.Run("missing record ref", func(t *testing.T) {
		keys := genTestKeyPair(t)
		req := mustProtoJSON(t, &signv1.SignRequest{
			Provider: &signv1.SignRequestProvider{
				Request: &signv1.SignRequestProvider_Key{
					Key: &signv1.SignWithKey{PrivateKey: string(keys.PrivateBytes), Password: []byte("test-password")},
				},
			},
		})
		assertErrorNonEmpty(t, bridgeHandle(Sign, handle, req))
	})
}
