// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package sessionstore provides session and token storage for the Agent Hub CLI and related applications.
package sessionstore

import "errors"

var (
	// ErrCouldNotOpenFile indicates a failure to open the session file.
	ErrCouldNotOpenFile = errors.New("could not open file")
	// ErrCouldNotWriteFile indicates a failure to write to the session file.
	ErrCouldNotWriteFile = errors.New("could not write file")
	// ErrMalformedSecret indicates a malformed secret in the session file.
	ErrMalformedSecret = errors.New("malformed secret")
	// ErrMalformedSecretFile indicates a malformed secret file.
	ErrMalformedSecretFile = errors.New("malformed secret file")
	// ErrSessionNotFound indicates that the requested session was not found.
	ErrSessionNotFound = errors.New("secret not found")
)
