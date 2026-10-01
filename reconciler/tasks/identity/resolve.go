// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"fmt"
	"strings"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	didresolver "github.com/agntcy/dir/client/utils/identity/resolvers/did"
	dnsresolver "github.com/agntcy/dir/client/utils/identity/resolvers/dns"
	wellknownresolver "github.com/agntcy/dir/client/utils/identity/resolvers/wellknown"
)

// resolverSet holds one key resolver per subject scheme.
type resolverSet struct {
	dns       resolvers.Resolver
	did       resolvers.Resolver
	wellknown resolvers.Resolver
	spiffe    resolvers.Resolver
}

// newNetworkResolvers returns the resolvers that need no per-run state. A nil
// fetcher makes the DID and well-known resolvers use an SSRF-guarded client.
func newNetworkResolvers() resolverSet {
	return resolverSet{
		dns:       dnsresolver.New(),
		did:       didresolver.New(nil),
		wellknown: wellknownresolver.New(nil),
	}
}

// forSubject picks the resolver for a claim's subject by its scheme:
//
//	did:web:..., did:key:...   did
//	spiffe://...               spiffe
//	https://...                wellknown
//	dns:acme.com, acme.com     dns
func (s resolverSet) forSubject(subject string) (resolvers.Resolver, error) {
	var r resolvers.Resolver

	switch {
	case strings.HasPrefix(subject, "did:"):
		r = s.did
	case strings.HasPrefix(subject, "spiffe://"):
		r = s.spiffe
	case strings.HasPrefix(subject, "https://"):
		r = s.wellknown
	case strings.HasPrefix(subject, "dns:"), !strings.Contains(subject, ":"):
		r = s.dns
	}

	if r == nil {
		return nil, fmt.Errorf("unsupported subject scheme in %q", subject)
	}

	return r, nil
}
