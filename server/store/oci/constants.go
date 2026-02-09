// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

// This file defines the complete metadata schema for OCI annotations.
// It serves as the single source of truth for all annotation keys used
// in manifest and descriptor annotations for record storage.

const (
	// Used for dir-specific annotations.
	manifestDirObjectKeyPrefix = "org.agntcy.dir"
	manifestDirObjectTypeKey   = manifestDirObjectKeyPrefix + "/type"

	// THESE ARE THE SOURCE OF TRUTH for field names.

	// Core Identity (simple keys).
	MetadataKeyName        = "name"
	MetadataKeyVersion     = "version"
	MetadataKeyDescription = "description"
	MetadataKeyOASFVersion = "oasf-version"
	MetadataKeyCid         = "cid"

	// Lifecycle (simple keys).
	MetadataKeySchemaVersion = "schema-version"
	MetadataKeyCreatedAt     = "created-at"
	MetadataKeyAuthors       = "authors"

	// Versioning (simple keys).
	MetadataKeyPreviousCid = "previous-cid"

	// Count metadata (simple keys).
	MetadataKeyAuthorsCount = "authors-count"

	// Derived from MetadataKey constants to ensure consistency.

	// Core Identity (derived from MetadataKey constants).
	ManifestKeyName        = manifestDirObjectKeyPrefix + "/" + MetadataKeyName
	ManifestKeyVersion     = manifestDirObjectKeyPrefix + "/" + MetadataKeyVersion
	ManifestKeyDescription = manifestDirObjectKeyPrefix + "/" + MetadataKeyDescription
	ManifestKeyOASFVersion = manifestDirObjectKeyPrefix + "/" + MetadataKeyOASFVersion
	ManifestKeyCid         = manifestDirObjectKeyPrefix + "/" + MetadataKeyCid

	// Lifecycle Metadata (mixed: some derived, some standalone).
	ManifestKeySchemaVersion = manifestDirObjectKeyPrefix + "/" + MetadataKeySchemaVersion
	ManifestKeyCreatedAt     = manifestDirObjectKeyPrefix + "/" + MetadataKeyCreatedAt
	ManifestKeyAuthors       = manifestDirObjectKeyPrefix + "/" + MetadataKeyAuthors

	// Versioning & Linking (standalone - no simple key equivalents).
	ManifestKeyPreviousCid = manifestDirObjectKeyPrefix + "/" + MetadataKeyPreviousCid

	// Custom annotations prefix.
	ManifestKeyCustomPrefix = manifestDirObjectKeyPrefix + "/custom."

	// Fallback values for error recovery scenarios.
	// Used when parsing corrupted storage, legacy records, or external modifications.
	FallbackSchemaVersion = "0.7.0"
)
