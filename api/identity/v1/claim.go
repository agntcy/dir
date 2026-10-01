// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"fmt"
)

// CanonicalPayload represents the deterministic payload of a claim
// that is used for signing and verification.
type CanonicalPayload struct {
	RecordCID string `json:"record_cid"`
	Role      string `json:"role"`
	Subject   string `json:"subject"`
	SignedAt  string `json:"signed_at"`
	ExpiresAt string `json:"expires_at"`
}

// GetPayload returns the deterministic bytes that are signed and
// verified for c. signature and certificate are never included: they
// don't exist yet at signing time, and aren't part of what the claim
// asserts.
func (c *Claim) GetPayload() ([]byte, error) {
	data, err := json.Marshal(CanonicalPayload{
		RecordCID: c.GetRecordCid(),
		Role:      c.GetRole().String(),
		Subject:   c.GetSubject(),
		SignedAt:  c.GetSignedAt(),
		ExpiresAt: c.GetExpiresAt(),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal claim payload: %w", err)
	}

	return data, nil
}
