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
// or the validation of an embedded certificate for spiffe:// and ans://) is a
// separate concern, not this package's job.
func Verify(claim *identityv1.Claim, recordCID, expectedSubject string, keys ...crypto.PublicKey) (bool, error) {
	if err := Check(claim, recordCID, expectedSubject); err != nil {
		return false, err
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

// Check runs the checks of Verify that need no key: that claim is for
// recordCID, that its subject is expectedSubject, and that it has not expired.
// A caller that must look keys up over the network can run it first, so a claim
// that cannot verify is rejected before any lookup.
func Check(claim *identityv1.Claim, recordCID, expectedSubject string) error {
	if claim == nil {
		return fmt.Errorf("claim is nil")
	}

	if claim.GetRecordCid() != recordCID {
		return fmt.Errorf("claim's record_cid does not match the record it's attached to")
	}

	subject := claim.GetSubject()

	if expectedSubject == "" {
		return fmt.Errorf("record does not declare an identity/owner annotation matching this claim")
	}

	if subject != expectedSubject {
		return fmt.Errorf("claim subject %q does not match record's declared annotation %q", subject, expectedSubject)
	}

	// The certificate sits outside the signed payload and is only resolved for
	// the subjects NeedsCertificate names, so one on any other claim could be
	// grafted on from elsewhere.
	needs, hasCert := NeedsCertificate(subject), claim.GetCertificate() != ""
	if needs && !hasCert {
		return fmt.Errorf("a claim for %q needs a certificate", subject)
	}

	if !needs && hasCert {
		return fmt.Errorf("a certificate is only used for %s subjects, not %q", certificateSchemes(), subject)
	}

	if signedAt := claim.GetSignedAt(); signedAt != "" {
		if _, err := time.Parse(time.RFC3339, signedAt); err != nil {
			return fmt.Errorf("invalid claim signed_at: %w", err)
		}
	}

	if expiresAt := claim.GetExpiresAt(); expiresAt != "" {
		if expiry, err := time.Parse(time.RFC3339, expiresAt); err != nil {
			return fmt.Errorf("invalid claim expiry: %w", err)
		} else if time.Now().After(expiry) {
			return fmt.Errorf("claim has expired")
		}
	}

	return nil
}

// NeedsCertificate reports whether a claim for subject must carry a
// certificate: see Scheme.NeedsCertificate. A subject in no known scheme needs
// none; the resolver refuses it anyway.
func NeedsCertificate(subject string) bool {
	scheme, ok := SchemeOf(subject)

	return ok && scheme.NeedsCertificate()
}
