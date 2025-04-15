// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"encoding/json"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"google.golang.org/protobuf/types/known/structpb"
)

func ToStruct(a any) (*structpb.Struct, error) {
	// Marshal any to json
	data, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Unmarshal json to map[string]any
	var mapData map[string]interface{}
	if err := json.Unmarshal(data, &mapData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data to map: %w", err)
	}

	// Convert the data to struct
	structData, err := structpb.NewStruct(mapData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to struct: %w", err)
	}

	return structData, nil
}

type Builder interface {
	Build(ctx context.Context) (*coretypes.Agent, error)
}
