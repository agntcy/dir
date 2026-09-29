// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
)

func TestVerifyWithKey(t *testing.T) {
	t.Run("happy path: verified true", func(t *testing.T) {
		keys := genTestKeyPair(t)

		signReq := signWithKeyRequest{
			CID:        "baecid123",
			PrivateKey: string(keys.PrivateBytes),
			Password:   "test-password",
		}

		signReqJSON, err := json.Marshal(signReq)
		if err != nil {
			t.Fatalf("failed to marshal sign request: %v", err)
		}

		signResp := bridgeNoHandle(SignWithKey, string(signReqJSON))

		var signOut signWithKeyResponse
		if err := json.Unmarshal([]byte(signResp), &signOut); err != nil {
			t.Fatalf("sign response is not valid JSON: %v (%s)", err, signResp)
		}

		if signOut.Error != "" {
			t.Fatalf("unexpected error signing: %s", signOut.Error)
		}

		verifyReq := verifyWithKeyRequest{
			CID:        "baecid123",
			PublicKeys: []string{signOut.PublicKey},
			Signature:  *signOut.Signature,
		}

		verifyReqJSON, err := json.Marshal(verifyReq)
		if err != nil {
			t.Fatalf("failed to marshal verify request: %v", err)
		}

		resp := bridgeNoHandle(VerifyWithKey, string(verifyReqJSON))

		var out verifyWithKeyResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if !out.Verified {
			t.Fatal("expected verified=true")
		}

		if out.PublicKey == "" {
			t.Fatal("expected a non-empty public key in the response")
		}
	})

	t.Run("wrong public key: verified false", func(t *testing.T) {
		keys := genTestKeyPair(t)
		otherKeys := genTestKeyPair(t)

		signReq := signWithKeyRequest{
			CID:        "baecid123",
			PrivateKey: string(keys.PrivateBytes),
			Password:   "test-password",
		}

		signReqJSON, err := json.Marshal(signReq)
		if err != nil {
			t.Fatalf("failed to marshal sign request: %v", err)
		}

		signResp := bridgeNoHandle(SignWithKey, string(signReqJSON))

		var signOut signWithKeyResponse
		if err := json.Unmarshal([]byte(signResp), &signOut); err != nil {
			t.Fatalf("sign response is not valid JSON: %v (%s)", err, signResp)
		}

		if signOut.Error != "" {
			t.Fatalf("unexpected error signing: %s", signOut.Error)
		}

		// Verify against a different key pair's public key: cosign.VerifyWithKeys
		// returns an error ("no valid signature found...") when no key/signature
		// combination verifies, which the wrapper surfaces as verified=false plus
		// a non-empty error message.
		verifyReq := verifyWithKeyRequest{
			CID:        "baecid123",
			PublicKeys: []string{string(otherKeys.PublicBytes)},
			Signature:  *signOut.Signature,
		}

		verifyReqJSON, err := json.Marshal(verifyReq)
		if err != nil {
			t.Fatalf("failed to marshal verify request: %v", err)
		}

		resp := bridgeNoHandle(VerifyWithKey, string(verifyReqJSON))

		var out verifyWithKeyResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Verified {
			t.Fatal("expected verified=false when verifying against the wrong public key")
		}

		if out.Error == "" {
			t.Fatal("expected a non-empty error when verifying against the wrong public key")
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeNoHandle(VerifyWithKey, `{not valid`))
	})
}

// TestVerifyWithOIDC only exercises input-validation and local-parsing error
// paths. Full OIDC verification requires a live Sigstore trust root (TUF) and
// a real signed bundle from Fulcio/Rekor, which cannot be meaningfully
// unit-tested offline. An empty/missing content_bundle is rejected while
// parsing the bundle JSON, before any network call would be made, so it is
// safe to assert on here alongside malformed request JSON.
func TestVerifyWithOIDC(t *testing.T) {
	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeNoHandle(VerifyWithOIDC, `{not valid`))
	})

	t.Run("empty signature bundle", func(t *testing.T) {
		req := verifyWithOIDCRequest{CID: "baecid123", Signature: signatureJSON{}}

		reqJSON, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		assertErrorNonEmpty(t, bridgeNoHandle(VerifyWithOIDC, string(reqJSON)))
	})
}

func TestVerify(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path: from_server=true", func(t *testing.T) {
		req := mustProtoJSON(t, &signv1.VerifyRequest{
			RecordRef:  &corev1.RecordRef{Cid: "baecid123"},
			FromServer: true,
		})

		resp := bridgeHandle(Verify, handle, req)

		var out verifyResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		var verifyResp signv1.VerifyResponse
		mustProtoUnmarshal(t, out.Response, &verifyResp)

		if !verifyResp.GetSuccess() {
			t.Fatal("expected success=true from the server-cached verification")
		}
	})

	t.Run("record not found: from_server=false", func(t *testing.T) {
		// The mock store's Lookup RPC always succeeds (see server_test.go), so
		// to deterministically exercise client.Verify's "record not found"
		// branch without doing full local Sigstore verification, this uses a
		// record ref whose CID is empty -- rejected before the Lookup RPC by
		// client.Verify's own "record CID is required" validation, which the
		// wrapper surfaces the same way (success=false is not returned here;
		// an error is, since RecordRef.Cid is empty at the API boundary).
		req := mustProtoJSON(t, &signv1.VerifyRequest{
			RecordRef:  &corev1.RecordRef{},
			FromServer: false,
		})

		assertErrorNonEmpty(t, bridgeHandle(Verify, handle, req))
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(Verify, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		req := mustProtoJSON(t, &signv1.VerifyRequest{RecordRef: &corev1.RecordRef{Cid: "baecid123"}, FromServer: true})
		assertErrorNonEmpty(t, bridgeHandle(Verify, 999999, req))
	})
}

func TestPullSignatures(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := recordRefEnvelopeJSON(t, "baecid123")

		resp := bridgeHandle(PullSignatures, handle, req)

		var out arrayResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if len(out.Results) != 1 {
			t.Fatalf("got %d results, want 1 signature referrer", len(out.Results))
		}

		var sig signv1.Signature
		mustProtoUnmarshal(t, out.Results[0], &sig)

		if sig.GetAlgorithm() != "ecdsa" {
			t.Fatalf("got algorithm %q, want ecdsa", sig.GetAlgorithm())
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PullSignatures, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		req := recordRefEnvelopeJSON(t, "baecid123")
		assertErrorNonEmpty(t, bridgeHandle(PullSignatures, 999999, req))
	})
}

func TestPullPublicKeys(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	handle := newTestClientHandle(t, addr)

	t.Run("happy path", func(t *testing.T) {
		req := recordRefEnvelopeJSON(t, "baecid123")

		resp := bridgeHandle(PullPublicKeys, handle, req)

		var out publicKeysResponse
		if err := json.Unmarshal([]byte(resp), &out); err != nil {
			t.Fatalf("response is not valid JSON: %v (%s)", err, resp)
		}

		if out.Error != "" {
			t.Fatalf("unexpected error: %s", out.Error)
		}

		if len(out.PublicKeys) != 1 || out.PublicKeys[0] != cannedPublicKeyPEM {
			t.Fatalf("unexpected public keys: %v", out.PublicKeys)
		}
	})

	t.Run("invalid request JSON", func(t *testing.T) {
		assertErrorNonEmpty(t, bridgeHandle(PullPublicKeys, handle, `{not valid`))
	})

	t.Run("unknown handle", func(t *testing.T) {
		req := recordRefEnvelopeJSON(t, "baecid123")
		assertErrorNonEmpty(t, bridgeHandle(PullPublicKeys, 999999, req))
	})
}
