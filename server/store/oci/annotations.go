// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types/adapters"
)

// extractManifestAnnotations extracts manifest annotations from record using adapter pattern.
//
//nolint:cyclop // Function handles multiple annotation types with justified complexity
func extractManifestAnnotations(record *corev1.Record) map[string]string {
	annotations := make(map[string]string)

	// Always set the type
	annotations[manifestDirObjectTypeKey] = "record"

	// Use adapter pattern to get version-agnostic access to record data
	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		// Return minimal annotations if no valid data
		return annotations
	}

	// Add version details
	annotations[ManifestKeyOASFVersion] = record.GetSchemaVersion()

	// Core identity fields (version-agnostic via adapter)
	if name := recordData.GetName(); name != "" {
		annotations[ManifestKeyName] = name
	}

	if version := recordData.GetVersion(); version != "" {
		annotations[ManifestKeyVersion] = version
	}

	// Lifecycle metadata
	if schemaVersion := recordData.GetSchemaVersion(); schemaVersion != "" {
		annotations[ManifestKeySchemaVersion] = schemaVersion
	}

	if createdAt := recordData.GetCreatedAt(); createdAt != "" {
		annotations[ManifestKeyCreatedAt] = createdAt
	}

	// Versioning (v1 specific)
	if previousCid := recordData.GetPreviousRecordCid(); previousCid != "" {
		annotations[ManifestKeyPreviousCid] = previousCid
	}

	// Custom annotations from record data -> manifest custom annotations
	if customAnnotations := recordData.GetAnnotations(); len(customAnnotations) > 0 {
		for key, value := range customAnnotations {
			annotations[ManifestKeyCustomPrefix+key] = value
		}
	}

	return annotations
}

// parseManifestAnnotations extracts structured metadata from manifest annotations.
//
//nolint:cyclop // Function handles multiple metadata extraction paths with justified complexity
func parseManifestAnnotations(annotations map[string]string) *corev1.RecordMeta {
	recordMeta := &corev1.RecordMeta{
		Annotations: make(map[string]string),
	}

	// Set fallback schema version first for error recovery scenarios
	recordMeta.SchemaVersion = FallbackSchemaVersion

	if annotations == nil {
		return recordMeta
	}

	// Extract schema version from stored data (override fallback if present)
	if schemaVersion := annotations[ManifestKeySchemaVersion]; schemaVersion != "" {
		recordMeta.SchemaVersion = schemaVersion
	}

	// Extract created time from stored data (no more empty strings!)
	if createdAt := annotations[ManifestKeyCreatedAt]; createdAt != "" {
		recordMeta.CreatedAt = createdAt
	}

	// Copy structured metadata into annotations for easy access
	// Core identity - these will be easily accessible to consumers
	if name := annotations[ManifestKeyName]; name != "" {
		recordMeta.Annotations[MetadataKeyName] = name
	}

	if version := annotations[ManifestKeyVersion]; version != "" {
		recordMeta.Annotations[MetadataKeyVersion] = version
	}

	if oasfVersion := annotations[ManifestKeyOASFVersion]; oasfVersion != "" {
		recordMeta.Annotations[MetadataKeyOASFVersion] = oasfVersion
	}

	// Versioning information
	if previousCid := annotations[ManifestKeyPreviousCid]; previousCid != "" {
		recordMeta.Annotations[MetadataKeyPreviousCid] = previousCid
	}

	// Custom annotations (those with our custom prefix) - clean namespace
	for key, value := range annotations {
		if after, ok := strings.CutPrefix(key, ManifestKeyCustomPrefix); ok {
			customKey := after
			recordMeta.Annotations[customKey] = value
		}
	}

	return recordMeta
}
