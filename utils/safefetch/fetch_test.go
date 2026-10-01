// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package safefetch

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsDisallowedAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"8.8.8.8", false},
		{"2606:4700:4700::1111", false},
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"fc00::1", true},
		{"169.254.169.254", true}, // cloud metadata
		{"fe80::1", true},
		{"100.100.100.200", true}, // shared address space, some clouds' metadata
		{"0.0.0.0", true},
		{"0.1.2.3", true},
		{"224.0.0.1", true},
		{"ff02::1", true},
		{"240.0.0.1", true},
		{"::ffff:10.0.0.1", true}, // IPv4-mapped private address
		{"::ffff:8.8.8.8", false},
		// IPv4-mapped forms of the ranges only the prefix list covers.
		{"::ffff:100.100.100.200", true},
		{"::ffff:0.1.2.3", true},
		{"::ffff:240.0.0.1", true},
		{"::ffff:198.18.0.1", true},
		// IPv4 documentation and relay ranges.
		{"192.0.2.1", true},
		{"198.51.100.1", true},
		{"203.0.113.1", true},
		{"192.88.99.1", true},
		// IPv6 transition ranges that can reach a blocked IPv4 target.
		{"64:ff9b::a9fe:a9fe", true}, // NAT64 to 169.254.169.254
		{"64:ff9b::808:808", true},   // NAT64 to 8.8.8.8, still refused
		{"64:ff9b:1::1", true},
		{"2002:a9fe:a9fe::1", true},                    // 6to4 embedding 169.254.169.254
		{"2001:0:4136:e378:8000:63bf:3fff:fdd2", true}, // Teredo
		{"2001:db8::1", true},
		{"100::1", true},
		{"::127.0.0.1", true}, // deprecated IPv4-compatible
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			require.Equal(t, tt.want, isDisallowedAddr(netip.MustParseAddr(tt.addr)))
		})
	}
}

func TestGet_BlocksNonPublicAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("secret"))
	}))
	defer srv.Close()

	// The test server listens on loopback, which the default client must refuse.
	_, err := New(WithAllowHTTP()).Get(t.Context(), srv.URL)
	require.ErrorIs(t, err, ErrDisallowedAddress)

	_, err = New().Get(t.Context(), "https://169.254.169.254/latest/meta-data")
	require.ErrorIs(t, err, ErrDisallowedAddress)

	_, err = New().Get(t.Context(), "https://[::1]/")
	require.ErrorIs(t, err, ErrDisallowedAddress)
}

// newTestClient returns a Client that may reach the loopback test server.
func newTestClient(t *testing.T, srv *httptest.Server, opts ...Option) *Client {
	t.Helper()

	c := newClient(func(netip.Addr) bool { return false }, opts...)

	transport, ok := c.httpClient.Transport.(*http.Transport)
	require.True(t, ok)

	srvTransport, ok := srv.Client().Transport.(*http.Transport)
	require.True(t, ok)

	transport.TLSClientConfig = srvTransport.TLSClientConfig

	return c
}

func TestGet(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("hello"))
		case "/big":
			_, _ = w.Write([]byte(strings.Repeat("x", 100)))
		case "/missing":
			http.NotFound(w, r)
		case "/redirect":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, "/loop", http.StatusFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv, WithMaxBytes(50))

	body, err := client.Get(t.Context(), srv.URL+"/ok")
	require.NoError(t, err)
	require.Equal(t, "hello", string(body))

	body, err = client.Get(t.Context(), srv.URL+"/redirect")
	require.NoError(t, err)
	require.Equal(t, "hello", string(body))

	_, err = client.Get(t.Context(), srv.URL+"/big")
	require.ErrorContains(t, err, "exceeds maximum size")

	_, err = client.Get(t.Context(), srv.URL+"/missing")
	require.ErrorContains(t, err, "unexpected status 404")

	_, err = client.Get(t.Context(), srv.URL+"/loop")
	require.ErrorContains(t, err, "too many redirects")
}

func TestGet_SchemeRules(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("plain"))
	}))
	defer plain.Close()

	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL, http.StatusFound)
	}))
	defer secure.Close()

	// http:// is refused by default, even to an otherwise reachable host.
	client := newTestClient(t, secure)
	_, err := client.Get(t.Context(), plain.URL)
	require.ErrorContains(t, err, `scheme "http" is not allowed`)

	// A redirect can't downgrade an https request to http.
	_, err = client.Get(t.Context(), secure.URL)
	require.ErrorContains(t, err, `scheme "http" is not allowed`)

	// WithAllowHTTP opts in.
	body, err := newTestClient(t, secure, WithAllowHTTP()).Get(t.Context(), plain.URL)
	require.NoError(t, err)
	require.Equal(t, "plain", string(body))

	_, err = client.Get(t.Context(), "ftp://example.com/x")
	require.ErrorContains(t, err, `scheme "ftp" is not allowed`)
}
