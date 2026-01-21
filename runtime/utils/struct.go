// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

func InterfaceToStruct(input any) (*structpb.Struct, error) {
	// Convert to JSON
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input to JSON: %w", err)
	}

	// Convert to map
	var dataMap map[string]any
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	// Convert to structpb
	structData, err := structpb.NewStruct(dataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to structpb: %w", err)
	}

	return structData, nil
}
