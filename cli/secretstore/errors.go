package secretstore

import "errors"

var (
	ErrCouldNotOpenFile    = errors.New("could not open file")
	ErrCouldNotWriteFile   = errors.New("could not write file")
	ErrMalformedSecret     = errors.New("malformed secret")
	ErrMalformedSecretFile = errors.New("malformed secret file")
	ErrSecretNotFound      = errors.New("secret not found")
	ErrInvalidSecret       = errors.New("invalid secret")
)
