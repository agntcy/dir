// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"crypto"
	"fmt"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/jws"
)

// Verify checks that claim was validly signed for recordCID and that its
// subject matches expectedSubject (the record's own "agntcy.dir/identity"
// or "agntcy.dir/owner" annotation, matching claim's role).
//
// keys are the already-resolved candidate public keys for claim's subject
// (e.g. a DNS TXT record set or a JWKS); the claim verifies if any one of
// them validates its signature. Resolving those keys (a DNS/DID/JWKS lookup,
// or SPIFFE trust-bundle validation of an embedded certificate) is a
// separate concern, not this package's job.
func Verify(claim *identityv1.Claim, recordCID, expectedSubject string, keys ...crypto.PublicKey) (bool, error) {
	if claim == nil {
		return false, fmt.Errorf("claim is nil")
	}

	if claim.GetRecordCid() != recordCID {
		return false, fmt.Errorf("claim's record_cid does not match the record it's attached to")
	}

	subject := claim.GetSubject()

	if expectedSubject == "" {
		return false, fmt.Errorf("record does not declare an identity/owner annotation matching this claim")
	}

	if subject != expectedSubject {
		return false, fmt.Errorf("claim subject %q does not match record's declared annotation %q", subject, expectedSubject)
	}

	if expiresAt := claim.GetExpiresAt(); expiresAt != "" {
		if expiry, err := time.Parse(time.RFC3339, expiresAt); err == nil && time.Now().After(expiry) {
			return false, fmt.Errorf("claim has expired")
		}
	}

	payload, err := claim.GetPayload()
	if err != nil {
		return false, fmt.Errorf("get claim payload: %w", err)
	}

	if err := jws.Verify(claim.GetSignature(), payload, keys...); err != nil {
		return false, fmt.Errorf("signature does not verify: %w", err)
	}

	return true, nil
}
