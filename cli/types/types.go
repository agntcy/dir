// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"encoding/json"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"google.golang.org/protobuf/types/known/structpb"
)

type ExtensionBuilderType string

type AgentExtension struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func (a *AgentExtension) ToAPIExtension() (coretypes.Extension, error) {
	// Marshal any to json
	data, err := json.Marshal(a.Data)
	if err != nil {
		return coretypes.Extension{}, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Unmarshal json to map[string]any
	var mapData map[string]interface{}
	if err := json.Unmarshal(data, &mapData); err != nil {
		return coretypes.Extension{}, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	// Convert the data to struct
	structData, err := structpb.NewStruct(mapData)
	if err != nil {
		return coretypes.Extension{}, fmt.Errorf("failed to convert data to struct: %w", err)
	}

	return coretypes.Extension{
		Name:    a.Name,
		Version: a.Version,
		Data:    structData,
	}, nil
}

type Builder interface {
	Build(ctx context.Context) ([]*AgentExtension, error)
}
