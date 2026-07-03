// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ReferrerType returns the fully-qualified proto type name used as the referrer type tag.
func (r *ScanReport) ReferrerType() string {
	return string((&ScanReport{}).ProtoReflect().Descriptor().FullName())
}

// MarshalReferrer encodes the ScanReport into a RecordReferrer.
func (r *ScanReport) MarshalReferrer() (*corev1.RecordReferrer, error) {
	if r == nil {
		return nil, errors.New("scan report is nil")
	}

	jsonBytes, err := protojson.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshal scan report to json: %w", err)
	}

	var m map[string]any
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return nil, fmt.Errorf("unmarshal json to map: %w", err)
	}

	data, err := structpb.NewStruct(m)
	if err != nil {
		return nil, fmt.Errorf("convert map to struct: %w", err)
	}

	return &corev1.RecordReferrer{
		Type: r.ReferrerType(),
		Data: data,
	}, nil
}

// UnmarshalReferrer loads a ScanReport from a RecordReferrer.
func (r *ScanReport) UnmarshalReferrer(ref *corev1.RecordReferrer) error {
	if ref == nil || ref.GetData() == nil {
		return errors.New("referrer or data is nil")
	}

	jsonBytes, err := json.Marshal(ref.GetData().AsMap())
	if err != nil {
		return fmt.Errorf("marshal struct to json: %w", err)
	}

	if err := protojson.Unmarshal(jsonBytes, r); err != nil {
		return fmt.Errorf("unmarshal json to scan report: %w", err)
	}

	return nil
}
