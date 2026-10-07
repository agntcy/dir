// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package agntcy implements authenticated key resolution and record-specific
// Agent Badge verification for agntcy:// subjects. The experimental protocol
// is adapted from agent-identity-demos at commit 2c891480a8af.
package agntcy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

const (
	ProtocolVersion = "agntcy.identity-verification.v2"
	Profile         = "agntcy-agent-badge.v1"
	KeyProfile      = "agntcy-agent-control.v1"
	// SubjectKeyPolicy explicitly describes the reference Identity Node's
	// subject-key-backed badges; it does not imply independent issuer trust.
	SubjectKeyPolicy = "subject-key-badge.v1"
)

type ResolutionRequest struct {
	Subject string `json:"subject"`
	Nonce   string `json:"nonce"`
}

type VerificationRequest struct {
	Profile   string `json:"profile"`
	Subject   string `json:"subject"`
	Signature string `json:"signature"`
	Payload   string `json:"payload"`
	Nonce     string `json:"nonce"`
}

type VerificationResponse struct {
	ResultJWS string `json:"resultJws"`
}

// VerificationChecks reports the checks under the selected profile. Badge is
// required only by the badge profile; Identity is required by both profiles.
type VerificationChecks struct {
	Identity bool `json:"identity"`
	Badge    bool `json:"badge"`
}

// VerificationResult is the signed response. Kind separates key resolution
// from evidence verification; RequestDigest binds every request field.
type VerificationResult struct {
	Version       string             `json:"version"`
	Kind          string             `json:"kind"`
	Verifier      string             `json:"verifier"`
	Profile       string             `json:"profile"`
	PolicyVersion string             `json:"policyVersion"`
	Checks        VerificationChecks `json:"checks"`
	Verified      bool               `json:"verified"`
	Subject       string             `json:"subject"`
	RecordCID     string             `json:"recordCid"`
	RequestDigest string             `json:"requestDigest"`
	CheckedAt     string             `json:"checkedAt"`
	ExpiresAt     string             `json:"expiresAt"`
	PublicKeys    []json.RawMessage  `json:"publicKeys,omitempty"`
	Error         string             `json:"error,omitempty"`
}

func DigestRequest(encoded []byte) string {
	digest := sha256.Sum256(encoded)

	return "sha256:" + hex.EncodeToString(digest[:])
}

// AgentID accepts one canonical spelling: an opaque, case-sensitive identifier
// containing ASCII letters, digits, '-', '_', '.', or '~'. There is no host
// normalization, escaping, path, port, query, fragment, or endpoint discovery.
func AgentID(subject string) (string, error) {
	if !strings.HasPrefix(subject, "agntcy://") {
		return "", errors.New("expected an agntcy:// identity")
	}

	id := strings.TrimPrefix(subject, "agntcy://")
	if len(id) == 0 || len(id) > 512 || id == "." || id == ".." {
		return "", errors.New("invalid AGNTCY Agent ID")
	}

	for _, c := range id {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && !strings.ContainsRune("-_.~", c) {
			return "", errors.New("non-canonical AGNTCY Agent ID")
		}
	}

	return id, nil
}
