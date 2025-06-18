// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package converter

import (
	corev1alpha1 "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/api/objectmanager"
	storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"
)

// CoreV1Alpha1ToRecordObject converts a core.v1alpha1.Object to objectmanager.RecordObject
func CoreV1Alpha1ToRecordObject(obj *corev1alpha1.Object) (*objectmanager.RecordObject, error) {
	// TODO: implement conversion logic
	return nil, nil
}

// RecordObjectToCoreV1Alpha1 converts an objectmanager.RecordObject to core.v1alpha1.Object
func RecordObjectToCoreV1Alpha1(obj *objectmanager.RecordObject) (*corev1alpha1.Object, error) {
	// TODO: implement conversion logic
	return nil, nil
}

// StoreV1Alpha2ToRecordObject converts a store.v1alpha2.ObjectRef to objectmanager.RecordObject
func StoreV1Alpha2ToRecordObject(obj *storev1alpha2.ObjectRef) (*objectmanager.RecordObject, error) {
	// TODO: implement conversion logic
	return nil, nil
}

// RecordObjectToStoreV1Alpha2 converts an objectmanager.RecordObject to store.v1alpha2.ObjectRef
func RecordObjectToStoreV1Alpha2(obj *objectmanager.RecordObject) (*storev1alpha2.ObjectRef, error) {
	// TODO: implement conversion logic
	return nil, nil
}

// StoreV1Alpha2ObjectToRecordObject converts a store.v1alpha2.Object to objectmanager.RecordObject
func StoreV1Alpha2ObjectToRecordObject(obj *storev1alpha2.Object) (*objectmanager.RecordObject, error) {
	// TODO: implement conversion logic
	return nil, nil
}

// RecordObjectToStoreV1Alpha2Object converts an objectmanager.RecordObject to store.v1alpha2.Object
func RecordObjectToStoreV1Alpha2Object(obj *objectmanager.RecordObject) (*storev1alpha2.Object, error) {
	// TODO: implement conversion logic
	return nil, nil
}
