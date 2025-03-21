package routing

import (
	record "github.com/libp2p/go-libp2p-record"
)

var _ record.Validator = &validator{}

type validator struct {
}

// Validate implements record.Validator.
func (v *validator) Validate(key string, value []byte) error {
	panic("unimplemented")
}

// Select implements record.Validator.
func (v *validator) Select(key string, values [][]byte) (int, error) {
	panic("unimplemented")
}
