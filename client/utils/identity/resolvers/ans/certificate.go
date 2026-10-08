// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/agntcy/dir/client/utils/jws"
)

// checkCertificate applies the checks that need no network to the untrusted
// certificate a claim carries: a URI SAN is exactly the subject (the same rule
// the signer applies), now falls inside the validity window, and the key is one
// claims are verified with. It returns that key.
func checkCertificate(cert *x509.Certificate, subject string, now time.Time) (crypto.PublicKey, error) {
	if !slices.ContainsFunc(cert.URIs, func(u *url.URL) bool { return u.String() == subject }) {
		return nil, fmt.Errorf("ans certificate: certificate does not name %q as a URI SAN", subject)
	}

	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, fmt.Errorf("ans certificate: certificate is not valid at %s (valid from %s to %s)",
			now.UTC().Format(time.RFC3339), cert.NotBefore.UTC().Format(time.RFC3339), cert.NotAfter.UTC().Format(time.RFC3339))
	}

	key, ok := jws.ToPublicKey(cert.PublicKey)
	if !ok {
		return nil, fmt.Errorf("ans certificate: unsupported public key type %T", cert.PublicKey)
	}

	return key, nil
}
