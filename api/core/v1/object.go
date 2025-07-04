package corev1

import (
	"encoding/json"
	"fmt"
	"io"
)

func (obj *Object) LoadFromReader(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	err = json.Unmarshal(data, obj)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return data, nil
}
