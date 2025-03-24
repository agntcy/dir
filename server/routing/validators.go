// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive
package routing

import (
	"errors"

	record "github.com/libp2p/go-libp2p-record"
)

var _ record.Validator = &validator{}

// validator validates namespaced KV ops for DHT GetValue and PutValue methods.
type validator struct{}

func (v *validator) Validate(key string, value []byte) error {
	return nil
}

func (v *validator) Select(key string, values [][]byte) (int, error) {
	if len(values) == 0 {
		return 0, errors.New("nothing to select")
	}

	return 0, nil
}
