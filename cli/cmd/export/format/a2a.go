// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"encoding/json"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/translator"
)

func init() {
	RegisterFormatter("a2a", &a2aFormatter{})
}

type a2aFormatter struct{}

func (f *a2aFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	a2aCard, err := translator.RecordToA2A(data)
	if err != nil {
		return nil, fmt.Errorf("failed to translate record to A2A AgentCard: %w", err)
	}

	raw, err := json.MarshalIndent(a2aCard, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal A2A AgentCard to JSON: %w", err)
	}

	raw = append(raw, '\n')

	return raw, nil
}

func (f *a2aFormatter) FileExtension() string {
	return ".json"
}
