// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func TestClassifyPredicates(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantConnection bool
		wantDescribe   string
	}{
		{
			name:         "dns record not found",
			err:          verify.ErrRecordNotFound,
			wantDescribe: "unexpected failure",
		},
		{
			name:         "dns timeout",
			err:          &verify.DNSError{Type: verify.DNSErrorTimeout, Fqdn: "_ans-badge.agent.example.com"},
			wantDescribe: "DNS lookup timed out",
		},
		{
			name:         "dns lookup failed hides the resolver address",
			err:          &verify.DNSError{Type: verify.DNSErrorLookupFailed, Fqdn: "_ans-badge.agent.example.com", Reason: "read udp 127.0.0.1:53: connection refused"},
			wantDescribe: "DNS lookup failed",
		},
		{
			name:         "dns not found",
			err:          &verify.DNSError{Type: verify.DNSErrorNotFound},
			wantDescribe: "DNS record not found",
		},
		{
			name:           "connection failure",
			err:            &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom},
			wantConnection: true,
			wantDescribe:   "transparency log unreachable",
		},
		{
			name:         "invalid response without status",
			err:          &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "no valid keys found in root keys response"},
			wantDescribe: "invalid response from the transparency log",
		},
		{
			name:         "http 503",
			err:          &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable},
			wantDescribe: "transparency log returned HTTP 503",
		},
		{
			name:         "http 429",
			err:          &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusTooManyRequests},
			wantDescribe: "transparency log returned HTTP 429",
		},
		{
			name:         "http 302",
			err:          &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusFound},
			wantDescribe: "transparency log returned HTTP 302",
		},
		{
			name:         "http 403 from a proxy",
			err:          &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusForbidden},
			wantDescribe: "transparency log returned HTTP 403",
		},
		{
			name:         "http 404",
			err:          &scitt.TransportError{Type: scitt.TransportErrNotFound, StatusCode: http.StatusNotFound},
			wantDescribe: "not found on the transparency log (HTTP 404)",
		},
		{
			name:         "not found without a status code",
			err:          &scitt.TransportError{Type: scitt.TransportErrNotFound},
			wantDescribe: "not found on the transparency log (HTTP 404)",
		},
		{
			name:         "http 410",
			err:          &scitt.TransportError{Type: scitt.TransportErrAgentTerminal, StatusCode: http.StatusGone},
			wantDescribe: "agent is in a terminal state (HTTP 410)",
		},
		{
			name:         "http 501",
			err:          &scitt.TransportError{Type: scitt.TransportErrNotSupported, StatusCode: http.StatusNotImplemented},
			wantDescribe: "transparency log returned HTTP 501",
		},
		{
			name:         "base64 decode",
			err:          &scitt.TransportError{Type: scitt.TransportErrBase64Decode},
			wantDescribe: "invalid response from the transparency log",
		},
		{
			name: "tls verification through the transport error",
			err: &scitt.TransportError{Type: scitt.TransportErrHTTPError, Cause: &url.Error{
				Op: "Get", URL: "https://log.example.com/v1/agents/x/status-token",
				Err: &tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}},
			}},
			wantConnection: true,
			wantDescribe:   "TLS handshake failed",
		},
		{
			name:           "unknown authority",
			err:            x509.UnknownAuthorityError{},
			wantConnection: true,
			wantDescribe:   "TLS handshake failed",
		},
		{
			name:           "hostname mismatch",
			err:            x509.HostnameError{Host: "log.example.com"},
			wantConnection: true,
			wantDescribe:   "TLS handshake failed",
		},
		{
			name:         "deadline",
			err:          context.DeadlineExceeded,
			wantDescribe: "timed out",
		},
		{
			name:         "wrapped deadline",
			err:          fmt.Errorf("fetch: %w", context.DeadlineExceeded),
			wantDescribe: "timed out",
		},
		{
			name:         "canceled",
			err:          context.Canceled,
			wantDescribe: "canceled",
		},
		{
			name: "canceled inside a transport error is connection-shaped",
			err: &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: &url.Error{
				Op: "Get", URL: "https://log.example.com/v1/agents/x/status-token", Err: context.Canceled,
			}},
			wantConnection: true,
			wantDescribe:   "canceled",
		},
		{
			name:         "token expired",
			err:          &scitt.TokenError{Type: scitt.TokenErrExpired, Exp: 100, Now: 200},
			wantDescribe: "status token expired at 100, now 200",
		},
		{
			name:         "terminal status",
			err:          &scitt.TokenError{Type: scitt.TokenErrTerminalStatus, Status: scitt.StatusRevoked},
			wantDescribe: "agent status REVOKED is terminal",
		},
		{
			name:         "missing field",
			err:          &scitt.TokenError{Type: scitt.TokenErrMissingField, Message: "agent_id"},
			wantDescribe: "invalid status token payload",
		},
		{
			name:         "unknown key id",
			err:          &scitt.SignatureError{Type: scitt.SigErrUnknownKeyID, Kid: [4]byte{0x0a, 0x0b, 0x0c, 0x0d}},
			wantDescribe: "signed by unknown key id 0a0b0c0d",
		},
		{
			name:         "issuer mismatch",
			err:          &scitt.SignatureError{Type: scitt.SigErrIssuerMismatch},
			wantDescribe: "issuer does not match the signing key",
		},
		{
			name:         "bad signature",
			err:          &scitt.SignatureError{Type: scitt.SigErrSignatureInvalid},
			wantDescribe: "signature verification failed",
		},
		{
			name:         "cose",
			err:          &scitt.CoseError{Type: scitt.CoseErrNotACoseSign1},
			wantDescribe: "malformed COSE_Sign1 structure",
		},
		{
			name:         "merkle",
			err:          &scitt.MerkleError{Type: scitt.MerkleErrInvalidProof},
			wantDescribe: "invalid inclusion proof",
		},
		{
			name:         "stage error describes its cause",
			err:          failWith(stageDNS, &verify.DNSError{Type: verify.DNSErrorTimeout}, "DNS lookup timed out"),
			wantDescribe: "DNS lookup timed out",
		},
		{
			name:         "unknown error",
			err:          errBoom,
			wantDescribe: "unexpected failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantConnection, isConnectionFailure(tt.err), "isConnectionFailure")
			assert.Equal(t, tt.wantDescribe, describe(tt.err), "describe")
		})
	}
}

func TestStageError(t *testing.T) {
	cause := &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable}
	err := failWith(stageReceipt, cause, "transparency log returned HTTP 503")

	require.EqualError(t, err, "ans receipt: transparency log returned HTTP 503")

	got, ok := errors.AsType[*scitt.TransportError](err)
	require.True(t, ok, "failWith() did not keep the cause in the chain")
	assert.Same(t, cause, got)

	require.EqualError(t, fail(stageBadgeURL, "x"), "ans badge-url: x")
	require.NoError(t, errors.Unwrap(fail(stageBadgeURL, "x")))
}

func TestUnknownKeyID(t *testing.T) {
	kid := [4]byte{0x0a, 0x0b, 0x0c, 0x0d}

	tests := []struct {
		name    string
		err     error
		wantKid [4]byte
		want    bool
	}{
		{name: "unknown key id", err: &scitt.SignatureError{Type: scitt.SigErrUnknownKeyID, Kid: kid}, wantKid: kid, want: true},
		{name: "wrapped in a stage error", err: failWith(stageStatusToken, &scitt.SignatureError{Type: scitt.SigErrUnknownKeyID, Kid: kid}, "signed by unknown key id"), wantKid: kid, want: true},
		{name: "other signature error", err: &scitt.SignatureError{Type: scitt.SigErrSignatureInvalid, Kid: kid}},
		{name: "not a signature error", err: errBoom},
		{name: "nil", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKid, got := unknownKeyID(tt.err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantKid, gotKid)
		})
	}
}
