// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"encoding/json"
	"fmt"

	apicore "github.com/agntcy/dir/api/core/v1alpha1"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type ExtensionBuilderType string

type AgentExtension struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Specs   any    `json:"specs,omitempty"`
}

func (a *AgentExtension) ToAPIExtension() (apicore.Extension, error) {
	// Marshal any to json
	jsonSpecs, err := json.Marshal(a.Specs)
	if err != nil {
		return apicore.Extension{}, fmt.Errorf("failed to marshal specs: %w", err)
	}

	// Unmarshal json to map[string]any
	var newSpecs map[string]interface{}
	if err := json.Unmarshal(jsonSpecs, &newSpecs); err != nil {
		return apicore.Extension{}, fmt.Errorf("failed to unmarshal specs: %w", err)
	}

	// Convert the specs to a Struct
	specsStruct, err := structpb.NewStruct(newSpecs)
	if err != nil {
		return apicore.Extension{}, fmt.Errorf("failed to convert specs to struct: %w", err)
	}

	return apicore.Extension{
		Name:    a.Name,
		Version: a.Version,
		Specs:   specsStruct,
	}, nil
}

type ExtensionBuilder interface {
	Build(ctx context.Context) (*AgentExtension, error)
}
