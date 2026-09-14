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

	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
)

// Stage prefixes. Every error Verify returns starts with the stage that
// failed, so the error a claim row stores stays stable and comparable.
const (
	stageName        = "ans name"
	stageCertificate = "ans certificate"
	stageDNS         = "ans dns"
	stageBadgeURL    = "ans badge-url"
	stageLog         = "ans log"
	stageRootKeys    = "ans root-keys"
	stageStatusToken = "ans status-token"
	stageReceipt     = "ans receipt"
)

// stageError is a failure at one step of the trust path. Its text is stable
// and stage-prefixed; the cause is kept for inspection and never printed,
// because SDK messages carry resolver addresses and raw HTTP detail.
type stageError struct {
	stage string
	text  string
	cause error
}

func (e *stageError) Error() string {
	return e.stage + ": " + e.text
}

// Unwrap exposes the cause to errors.Is and errors.AsType.
func (e *stageError) Unwrap() error {
	return e.cause
}

// fail is a failure with no underlying cause.
func fail(stage, text string) error {
	return &stageError{stage: stage, text: text}
}

// failWith is a failure that keeps its cause in the chain.
func failWith(stage string, cause error, text string) error {
	return &stageError{stage: stage, text: text, cause: cause}
}

// unknownKeyID reports whether err says a log signature was made with a key
// the key set does not hold, and which one.
func unknownKeyID(err error) ([4]byte, bool) {
	sigErr, ok := errors.AsType[*scitt.SignatureError](err)
	if !ok || sigErr.Type != scitt.SigErrUnknownKeyID {
		return [4]byte{}, false
	}

	return sigErr.Kid, true
}

// isConnectionFailure reports whether err says the log produced no HTTP
// response at all: a TLS handshake failure, or a transport error carrying a
// cause and no status code.
func isConnectionFailure(err error) bool {
	if isTLSFailure(err) {
		return true
	}

	transportErr, ok := errors.AsType[*scitt.TransportError](err)

	return ok && transportErr.Type == scitt.TransportErrHTTPError && transportErr.StatusCode == 0 && transportErr.Cause != nil
}

// isTLSFailure reports whether err is a TLS handshake failure.
func isTLSFailure(err error) bool {
	if _, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
		return true
	}

	if _, ok := errors.AsType[x509.UnknownAuthorityError](err); ok {
		return true
	}

	_, ok := errors.AsType[x509.HostnameError](err)

	return ok
}

// describe renders a dependency failure as stable text without the stage. It
// never echoes SDK messages, which carry resolver and peer addresses.
func describe(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case isTLSFailure(err):
		return "TLS handshake failed"
	}

	if transportErr, ok := errors.AsType[*scitt.TransportError](err); ok {
		return describeTransport(transportErr)
	}

	if dnsErr, ok := errors.AsType[*verify.DNSError](err); ok {
		return describeDNS(dnsErr)
	}

	if tokenErr, ok := errors.AsType[*scitt.TokenError](err); ok {
		return describeToken(tokenErr)
	}

	if sigErr, ok := errors.AsType[*scitt.SignatureError](err); ok {
		return describeSignature(sigErr)
	}

	if _, ok := errors.AsType[*scitt.CoseError](err); ok {
		return "malformed COSE_Sign1 structure"
	}

	if _, ok := errors.AsType[*scitt.MerkleError](err); ok {
		return "invalid inclusion proof"
	}

	return "unexpected failure"
}

func describeTransport(err *scitt.TransportError) string {
	switch {
	case err.Type == scitt.TransportErrNotFound || err.StatusCode == http.StatusNotFound:
		return "not found on the transparency log (HTTP 404)"
	case err.StatusCode == http.StatusGone:
		return "agent is in a terminal state (HTTP 410)"
	case err.StatusCode != 0:
		return fmt.Sprintf("transparency log returned HTTP %d", err.StatusCode)
	case err.Type == scitt.TransportErrHTTPError && err.Cause != nil:
		return "transparency log unreachable"
	default:
		return "invalid response from the transparency log"
	}
}

func describeDNS(err *verify.DNSError) string {
	if err.Type == verify.DNSErrorTimeout {
		return "DNS lookup timed out"
	}

	if err.Type == verify.DNSErrorNotFound {
		return "DNS record not found"
	}

	return "DNS lookup failed"
}

func describeToken(err *scitt.TokenError) string {
	if err.Type == scitt.TokenErrExpired {
		return fmt.Sprintf("status token expired at %d, now %d", err.Exp, err.Now)
	}

	if err.Type == scitt.TokenErrTerminalStatus {
		return fmt.Sprintf("agent status %s is terminal", err.Status)
	}

	return "invalid status token payload"
}

func describeSignature(err *scitt.SignatureError) string {
	if err.Type == scitt.SigErrUnknownKeyID {
		return fmt.Sprintf("signed by unknown key id %x", err.Kid)
	}

	if err.Type == scitt.SigErrIssuerMismatch {
		return "issuer does not match the signing key"
	}

	return "signature verification failed"
}
