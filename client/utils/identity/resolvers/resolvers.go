// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package resolvers defines what an identity key resolver is. Each scheme
// lives in its own subpackage: dns, did, wellknown, spiffe and ans.
//
// A resolver finds the public keys a claim's subject publishes; it never
// checks a signature. Pass its result to identity.Verify (or jws.Verify),
// which does all of the cryptography and succeeds if any one key validates.
package resolvers

import (
	"context"
	"crypto"
	"errors"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
)

// ErrNoKeys is returned when a subject publishes no usable public key.
var ErrNoKeys = errors.New("no usable public keys found")

// ErrFinal marks a failure as a verdict about the claim. A caller that keeps
// an earlier result through failures that say nothing about the claim (a
// timeout, a refused connection) classifies them by their cause; a failure
// wrapped with Final is never one of those, whatever its cause, so the caller
// must check errors.Is(err, ErrFinal) before looking at the cause. A resolver
// uses it for what it can judge: the subject, what the subject publishes, and
// what a trusted party states or refuses to state. A failure to reach the
// network, the caller's own deadline included, is left unmarked.
var ErrFinal = errors.New("final")

// Final marks err as a verdict about the claim (see ErrFinal). The text and
// the cause of err stay as they are for errors.Is and errors.As; a nil err
// stays nil.
func Final(err error) error {
	if err == nil {
		return nil
	}

	return &finalError{err: err}
}

type finalError struct {
	err error
}

func (e *finalError) Error() string {
	return e.err.Error()
}

func (e *finalError) Unwrap() error {
	return e.err
}

func (e *finalError) Is(target error) bool {
	return target == ErrFinal //nolint:err113 // the sentinel is compared by identity on purpose
}

// Resolver finds the candidate public keys for a claim's subject.
type Resolver interface {
	// Resolve returns at least one key, or an error. certificate is the
	// claim's DER-encoded X.509 certificate, empty when the claim has none;
	// only SPIFFE and ANS resolution read it.
	Resolve(ctx context.Context, subject string, certificate []byte, opts ...Option) ([]crypto.PublicKey, error)
}

// Options carries optional record-specific input and an output validity bound.
// Resolvers that need only subject keys may ignore these options.
type Options struct {
	Claim      *identityv1.Claim
	ValidUntil *time.Time
}

type Option func(*Options)

// WithClaim supplies the complete signed claim to a record-aware resolver.
func WithClaim(claim *identityv1.Claim) Option {
	return func(o *Options) { o.Claim = claim }
}

// WithValidUntil receives the deadline of accepted evidence, when bounded.
func WithValidUntil(until *time.Time) Option {
	return func(o *Options) { o.ValidUntil = until }
}

// ApplyOptions collects the options for a resolution request.
func ApplyOptions(opts ...Option) Options {
	var out Options
	for _, opt := range opts {
		opt(&out)
	}

	return out
}

// Fetcher retrieves a document over HTTPS. safefetch.Client implements it.
type Fetcher interface {
	Get(ctx context.Context, url string) ([]byte, error)
}
