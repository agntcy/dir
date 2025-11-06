// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/decoding/v1"
	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha0"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	typesv1alpha2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// DecodedRecord is an interface representing a decoded OASF record.
// It provides methods to access the underlying record data.
type DecodedRecord interface {
	// GetRecord returns the underlying record data, which can be of supported type.
	GetRecord() any

	// HasV1Alpha0 checks if the record is of type V1Alpha0.
	HasV1Alpha0() bool
	GetV1Alpha0() *typesv1alpha0.Record

	// HasV1Alpha1 checks if the record is of type V1Alpha1.
	HasV1Alpha1() bool
	GetV1Alpha1() *typesv1alpha1.Record

	// HasV1Alpha2 checks if the record is of type V1Alpha2.
	HasV1Alpha2() bool
	GetV1Alpha2() *typesv1alpha2.Record
}

type decodedRecord struct {
	*decodingv1.DecodeRecordResponse
}

func (d *decodedRecord) GetRecord() any {
	if d == nil || d.DecodeRecordResponse == nil {
		return nil
	}

	switch data := d.DecodeRecordResponse.GetRecord().(type) {
	case *decodingv1.DecodeRecordResponse_V1Alpha0:
		return data.V1Alpha0
	case *decodingv1.DecodeRecordResponse_V1Alpha1:
		return data.V1Alpha1
	case *decodingv1.DecodeRecordResponse_V1Alpha2:
		return data.V1Alpha2
	default:
		return nil
	}
}

// New creates a Record for a supported OASF typed record.
func New[T typesv1alpha0.Record | typesv1alpha1.Record | typesv1alpha2.Record](record *T) *Record {
	data, _ := decoder.StructToProto(record)

	return &Record{
		Data: data,
	}
}
