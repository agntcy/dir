// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"crypto/x509"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
)

// checkCertificate applies the checks that need no network to the untrusted
// leaf a claim carries: a URI SAN names the agent, now falls inside the
// validity window, and the key is one claims are verified with.
func checkCertificate(cert *x509.Certificate, want agentName, now time.Time) error {
	if !certNames(cert, want) {
		return fail(stageCertificate, "certificate does not name this agent")
	}

	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fail(stageCertificate, "certificate expired or not yet valid")
	}

	if err := identityv1.CheckSigningKey(cert.PublicKey); err != nil {
		return failWith(stageCertificate, err, "unsupported key type: "+err.Error())
	}

	return nil
}

// certNames reports whether one of cert's URI SANs names want.
func certNames(cert *x509.Certificate, want agentName) bool {
	for _, uri := range cert.URIs {
		if want.matches(uri.String()) {
			return true
		}
	}

	return false
}
