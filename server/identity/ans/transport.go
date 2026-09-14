// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/agentnameservice/ans-sdk-go/models"
	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
)

// badgeFinder finds the badge record of one agent version in DNS. The SDK's
// resolvers satisfy it; tests script it.
type badgeFinder interface {
	FindBadgeForVersion(ctx context.Context, fqdn models.Fqdn, version models.Version) (*verify.AnsBadgeRecord, error)
}

// newTransport clones the default transport with the system roots plus the
// certificates in caFile, so a mixed list of public and private logs works.
func newTransport(caFile string) (*http.Transport, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("ans: system certificate pool: %w", err)
	}

	if caFile != "" {
		pemBytes, err := os.ReadFile(caFile) //nolint:gosec // operator-supplied path from configuration
		if err != nil {
			return nil, fmt.Errorf("ans: ca_file: %w", err)
		}

		if !pool.AppendCertsFromPEM(pemBytes) {
			return nil, fmt.Errorf("ans: ca_file %q contains no PEM certificates", caFile)
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert // http.DefaultTransport is documented as an *http.Transport
	transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}

	return transport, nil
}

// defaultResolver builds the SDK resolver, pointed at server (host:port) for
// both UDP and TCP when one is configured.
func defaultResolver(server string) *verify.StandardDNSResolver {
	resolver := verify.NewStandardDNSResolver()
	if server == "" {
		return resolver
	}

	dialer := &net.Dialer{}

	return resolver.WithResolver(&net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, server)
			if err != nil {
				return nil, fmt.Errorf("ans dns: dial %s: %w", network, err)
			}

			return conn, nil
		},
	})
}

// httpLogClient builds the SDK client for one log origin: the shared
// transport, no redirects, and the verification's time budget as the request
// timeout.
func (r *Resolver) httpLogClient(base string) (scitt.Client, error) {
	httpClient := &http.Client{
		Transport: r.transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	client, err := scitt.NewHTTPClient(base, scitt.WithHTTPClient(httpClient), scitt.WithTimeout(r.cfg.GetTimeout()))
	if err != nil {
		return nil, fmt.Errorf("ans log client: %w", err)
	}

	return client, nil
}

// logClient returns the client for the target's log, built once per origin
// and wrapped so that every fetch is bounded and reported to the host's
// circuit breaker. The allow-list bounds the number of origins.
func (r *Resolver) logClient(target badgeTarget) (scitt.Client, error) {
	r.mu.Lock()
	client, ok := r.clients[target.LogBase]
	r.mu.Unlock()

	if ok {
		return client, nil
	}

	inner, err := r.newLogClient(target.LogBase)
	if err != nil {
		return nil, failWith(stageLog, err, "cannot create a client for the transparency log")
	}

	client = &breakerClient{
		inner:   inner,
		host:    target.LogHost,
		breaker: r.breaker,
		clock:   r.clock,
		timeout: r.cfg.GetTimeout() / fetchBudgetDivisor,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.clients[target.LogBase]; ok {
		return existing, nil
	}

	r.clients[target.LogBase] = client

	return client, nil
}
