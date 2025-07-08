// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package converter

import (
	corev1alpha1 "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/api/objectmanager"
)

// Global converter instance for core.v1alpha1.Object <-> objectmanager.RecordObject
var CoreV1Alpha1Converter ObjectConverter[*corev1alpha1.Object, *objectmanager.RecordObject] = NewCoreV1Alpha1ToRecordObjectConverter()

// CoreV1Alpha1ToRecordObjectConverter implements conversion between core.v1alpha1.Object and objectmanager.RecordObject
type CoreV1Alpha1ToRecordObjectConverter struct{}

// NewCoreV1Alpha1ToRecordObjectConverter creates a new converter instance
func NewCoreV1Alpha1ToRecordObjectConverter() *CoreV1Alpha1ToRecordObjectConverter {
	return &CoreV1Alpha1ToRecordObjectConverter{}
}

// From converts from objectmanager.RecordObject to core.v1alpha1.Object
func (c *CoreV1Alpha1ToRecordObjectConverter) From(recordObj *objectmanager.RecordObject) (*corev1alpha1.Object, error) {
	if recordObj == nil {
		return nil, NewConversionError("RecordObject", "core.v1alpha1.Object", "input is nil", nil)
	}

	// Create the core object
	coreObject := &corev1alpha1.Object{
		Ref: &corev1alpha1.ObjectRef{
			Digest: recordObj.GetCid(),
			Type:   recordObjectTypeToString(recordObj.GetType()),
			Size:   0, // TODO: implement size calculation if needed
		},
		Data: []byte{}, // TODO: implement data extraction if needed
	}

	// Extract agent data if present
	if recordObj.GetRecord() != nil {
		if agent := recordObj.GetRecord().GetRecordV1Alpha1(); agent != nil {
			coreObject.Agent = agent
		}
	}

	return coreObject, nil
}

// To converts from core.v1alpha1.Object to objectmanager.RecordObject
func (c *CoreV1Alpha1ToRecordObjectConverter) To(coreObj *corev1alpha1.Object) (*objectmanager.RecordObject, error) {
	if coreObj == nil {
		return nil, NewConversionError("core.v1alpha1.Object", "RecordObject", "input is nil", nil)
	}

	// Determine the record type based on content
	recordType := objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON

	// Create record data
	recordData := &objectmanager.RecordObjectData{}

	// Set agent data if present
	if coreObj.GetAgent() != nil {
		recordData.Data = &objectmanager.RecordObjectData_RecordV1Alpha1{
			RecordV1Alpha1: coreObj.GetAgent(),
		}
	}

	// Create the record object
	recordObject := &objectmanager.RecordObject{
		Cid:    getCidFromRef(coreObj.GetRef()),
		Type:   recordType,
		Record: recordData,
	}

	return recordObject, nil
}

// Helper function to convert RecordObjectType to string
func recordObjectTypeToString(recordType objectmanager.RecordObjectType) string {
	switch recordType {
	case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON:
		return "oasf-v1alpha1-json"
	case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON:
		return "oasf-v1alpha2-json"
	default:
		return "unknown"
	}
}

// Helper function to extract CID from ObjectRef
func getCidFromRef(ref *corev1alpha1.ObjectRef) string {
	if ref == nil {
		return ""
	}
	return ref.GetDigest()
}
