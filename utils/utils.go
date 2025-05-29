// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"google.golang.org/protobuf/types/known/structpb"
)

func ToPtr[T any](v T) *T {
	return &v
}

func ToStructpb(data map[string]interface{}) (*structpb.Struct, error) {
	if data == nil {
		return nil, nil
	}

	structData, err := structpb.NewStruct(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to structpb.Struct: %w", err)
	}

	return structData, nil
}
