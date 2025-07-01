// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package converter

import (
	"encoding/json"

	corev1alpha1 "github.com/agntcy/dir/api/core/v1alpha1"
	corev1alpha2 "github.com/agntcy/dir/api/core/v1alpha2"
	"github.com/agntcy/dir/api/objectmanager"
	storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"
)

// Global converter instances for store.v1alpha2 conversions
var (
	// StoreV1Alpha2Converter converts between store.v1alpha2.Object and objectmanager.RecordObject
	StoreV1Alpha2Converter ObjectConverter[*storev1alpha2.Object, *objectmanager.RecordObject] = NewStoreV1Alpha2ToRecordObjectConverter()
)

// StoreV1Alpha2ToRecordObjectConverter implements conversion between store.v1alpha2.Object and objectmanager.RecordObject
type StoreV1Alpha2ToRecordObjectConverter struct{}

// NewStoreV1Alpha2ToRecordObjectConverter creates a new converter instance
func NewStoreV1Alpha2ToRecordObjectConverter() *StoreV1Alpha2ToRecordObjectConverter {
	return &StoreV1Alpha2ToRecordObjectConverter{}
}

// From converts from objectmanager.RecordObject to store.v1alpha2.Object
func (c *StoreV1Alpha2ToRecordObjectConverter) From(recordObj *objectmanager.RecordObject) (*storev1alpha2.Object, error) {
	if recordObj == nil {
		return nil, NewConversionError("RecordObject", "store.v1alpha2.Object", "input is nil", nil)
	}

	// Serialize record data based on type
	var data []byte
	var err error

	if recordObj.GetRecord() != nil && recordObj.GetRecord().GetData() != nil {
		switch recordObj.GetType() {
		case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON:
			if agent := recordObj.GetRecord().GetRecordV1Alpha1(); agent != nil {
				data, err = json.Marshal(agent)
				if err != nil {
					return nil, NewConversionError("RecordObject", "store.v1alpha2.Object",
						"failed to marshal v1alpha1 Agent", err)
				}
			}
		case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON:
			if record := recordObj.GetRecord().GetRecordV1Alpha2(); record != nil {
				data, err = json.Marshal(record)
				if err != nil {
					return nil, NewConversionError("RecordObject", "store.v1alpha2.Object",
						"failed to marshal v1alpha2 Record", err)
				}
			}
		default:
			// For unknown types, data remains nil
		}
	}

	// Create the store object
	storeObject := &storev1alpha2.Object{
		Cid:         recordObj.GetCid(),
		Type:        recordObjectTypeToStoreType(recordObj.GetType()),
		Size:        uint64(len(data)),       // Set actual size based on serialized data
		CreatedAt:   "",                      // TODO: implement timestamp if needed
		Data:        data,                    // Set the serialized data
		Annotations: make(map[string]string), // TODO: implement annotations if needed
	}

	return storeObject, nil
}

// To converts from store.v1alpha2.Object to objectmanager.RecordObject
func (c *StoreV1Alpha2ToRecordObjectConverter) To(storeObj *storev1alpha2.Object) (*objectmanager.RecordObject, error) {
	if storeObj == nil {
		return nil, NewConversionError("store.v1alpha2.Object", "RecordObject", "input is nil", nil)
	}

	// Determine the record type based on store object type
	recordType := storeTypeToRecordObjectType(storeObj.GetType())

	// Create record data
	recordData := &objectmanager.RecordObjectData{}

	// Extract data based on store object type
	if storeObj.GetData() != nil && len(storeObj.GetData()) > 0 {
		// Try to unmarshal the data based on the record type
		switch recordType {
		case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON:
			// Try to unmarshal as v1alpha1 Agent
			var agent corev1alpha1.Agent
			if err := json.Unmarshal(storeObj.GetData(), &agent); err != nil {
				return nil, NewConversionError("store.v1alpha2.Object", "RecordObject",
					"failed to unmarshal data as v1alpha1 Agent", err)
			}
			recordData.Data = &objectmanager.RecordObjectData_RecordV1Alpha1{
				RecordV1Alpha1: &agent,
			}
		case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON:
			// Try to unmarshal as v1alpha2 Record
			var record corev1alpha2.Record
			if err := json.Unmarshal(storeObj.GetData(), &record); err != nil {
				return nil, NewConversionError("store.v1alpha2.Object", "RecordObject",
					"failed to unmarshal data as v1alpha2 Record", err)
			}
			recordData.Data = &objectmanager.RecordObjectData_RecordV1Alpha2{
				RecordV1Alpha2: &record,
			}
		default:
			// For unknown types, we don't try to unmarshal the data
			// The recordData.Data remains nil
		}
	}

	// Create the record object
	recordObject := &objectmanager.RecordObject{
		Cid:    storeObj.GetCid(),
		Type:   recordType,
		Record: recordData,
	}

	return recordObject, nil
}

// Helper function to convert RecordObjectType to Store ObjectType
func recordObjectTypeToStoreType(recordType objectmanager.RecordObjectType) storev1alpha2.ObjectType {
	switch recordType {
	case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON:
		return storev1alpha2.ObjectType_OBJECT_TYPE_RAW // TODO: define proper mapping
	case objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA2_JSON:
		return storev1alpha2.ObjectType_OBJECT_TYPE_RAW // TODO: define proper mapping
	default:
		return storev1alpha2.ObjectType_OBJECT_TYPE_UNSPECIFIED
	}
}

// Helper function to convert Store ObjectType to RecordObjectType
func storeTypeToRecordObjectType(storeType storev1alpha2.ObjectType) objectmanager.RecordObjectType {
	switch storeType {
	case storev1alpha2.ObjectType_OBJECT_TYPE_RAW:
		return objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON // TODO: define proper mapping
	default:
		return objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_UNSPECIFIED
	}
}
