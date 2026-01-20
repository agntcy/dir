// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
)

var _ types.Record = (*RecordAdapter)(nil)

// RecordAdapter adapts corev1.Record to types.Record interface.
type RecordAdapter struct {
	record *corev1.Record
}

// NewRecordAdapter creates a new RecordAdapter.
func NewRecordAdapter(record *corev1.Record) *RecordAdapter {
	return &RecordAdapter{record: record}
}

// GetCid implements types.Record interface.
func (r *RecordAdapter) GetCid() string {
	return r.record.GetCid()
}

// GetRecordData implements types.Record interface.
func (r *RecordAdapter) GetRecordData() (types.RecordData, error) {
	// Decode record
	decoded, err := r.record.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}

	// Determine record type and create appropriate adapter
	switch {
	case decoded.HasV1Alpha0():
		return NewV1Alpha0Adapter(decoded.GetV1Alpha0()), nil
	case decoded.HasV1Alpha1():
		return NewV1Alpha1Adapter(decoded.GetV1Alpha1()), nil
	case decoded.HasV1Alpha2():
		return NewV1Alpha2Adapter(decoded.GetV1Alpha2()), nil
	default:
		return nil, fmt.Errorf("unsupported record type: %T", decoded.GetRecord())
	}
}
