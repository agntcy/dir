package routing

import (
	"fmt"

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
		return 0, fmt.Errorf("nothing to select")
	}
	return 0, nil
}
