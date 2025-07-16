// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha2

import (
	"encoding/json"
	"fmt"
	"io"

	objectsv2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v2"
)

type Record struct {
	*objectsv2.AgentRecord
}

func (r *Record) LoadFromReader(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	err = json.Unmarshal(data, r)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return data, nil
}
