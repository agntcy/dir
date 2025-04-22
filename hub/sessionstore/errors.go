// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sessionstore

import "errors"

var (
	ErrCouldNotOpenFile    = errors.New("could not open file")
	ErrCouldNotWriteFile   = errors.New("could not write file")
	ErrMalformedSecret     = errors.New("malformed secret")
	ErrMalformedSecretFile = errors.New("malformed secret file")
	ErrSessionNotFound     = errors.New("secret not found")
)
