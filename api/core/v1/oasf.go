// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/decoding/v1"
	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	typesv1alpha2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// DecodedRecord is an interface representing a decoded OASF record.
// It provides methods to access the underlying record data.
type DecodedRecord interface {
	// GetRecord returns the underlying record data, which can be of supported type.
	GetRecord() any

	// HasV1Alpha1 checks if the record is of type V1Alpha1.
	HasV1Alpha1() bool
	GetV1Alpha1() *typesv1alpha1.Record

	// HasV1Alpha2 checks if the record is of type V1Alpha2.
	HasV1Alpha2() bool
	GetV1Alpha2() *typesv1alpha2.Record

	// HasV1 checks if the record is of type V1 (OASF 1.0.0).
	HasV1() bool
	GetV1() *typesv1.Record
}

type decodedRecord struct {
	*decodingv1.DecodeRecordResponse
}

func (d *decodedRecord) GetRecord() any {
	if d == nil || d.DecodeRecordResponse == nil {
		return nil
	}

	switch data := d.DecodeRecordResponse.GetRecord().(type) {
	case *decodingv1.DecodeRecordResponse_V1Alpha1:
		return data.V1Alpha1
	case *decodingv1.DecodeRecordResponse_V1Alpha2:
		return data.V1Alpha2
	case *decodingv1.DecodeRecordResponse_V1:
		return data.V1
	default:
		return nil
	}
}

func (d *decodedRecord) HasV1Alpha1() bool {
	if d == nil || d.DecodeRecordResponse == nil {
		return false
	}

	_, ok := d.DecodeRecordResponse.GetRecord().(*decodingv1.DecodeRecordResponse_V1Alpha1)

	return ok
}

func (d *decodedRecord) GetV1Alpha1() *typesv1alpha1.Record {
	if d == nil || d.DecodeRecordResponse == nil {
		return nil
	}

	if v, ok := d.DecodeRecordResponse.GetRecord().(*decodingv1.DecodeRecordResponse_V1Alpha1); ok {
		return v.V1Alpha1
	}

	return nil
}

func (d *decodedRecord) HasV1Alpha2() bool {
	if d == nil || d.DecodeRecordResponse == nil {
		return false
	}

	_, ok := d.DecodeRecordResponse.GetRecord().(*decodingv1.DecodeRecordResponse_V1Alpha2)

	return ok
}

func (d *decodedRecord) GetV1Alpha2() *typesv1alpha2.Record {
	if d == nil || d.DecodeRecordResponse == nil {
		return nil
	}

	if v, ok := d.DecodeRecordResponse.GetRecord().(*decodingv1.DecodeRecordResponse_V1Alpha2); ok {
		return v.V1Alpha2
	}

	return nil
}

func (d *decodedRecord) HasV1() bool {
	if d == nil || d.DecodeRecordResponse == nil {
		return false
	}

	_, ok := d.DecodeRecordResponse.GetRecord().(*decodingv1.DecodeRecordResponse_V1)

	return ok
}

func (d *decodedRecord) GetV1() *typesv1.Record {
	if d == nil || d.DecodeRecordResponse == nil {
		return nil
	}

	if v, ok := d.DecodeRecordResponse.GetRecord().(*decodingv1.DecodeRecordResponse_V1); ok {
		return v.V1
	}

	return nil
}

// New creates a Record for a supported OASF typed record.
func New[T typesv1alpha1.Record | typesv1alpha2.Record | typesv1.Record](record *T) *Record {
	data, _ := decoder.StructToProto(record)

	return &Record{
		Data: data,
	}
}
