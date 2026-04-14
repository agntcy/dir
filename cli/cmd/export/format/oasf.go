// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package format

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	RegisterFormatter("oasf", &oasfFormatter{})
}

type oasfFormatter struct{}

func (f *oasfFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	raw, err := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OASF record: %w", err)
	}

	raw = append(raw, '\n')

	return raw, nil
}

func (f *oasfFormatter) FileExtension() string {
	return ".json"
}
