// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	corev1 "github.com/agntcy/dir/api/core/v1"
)

// Internal OCI artifact media types used for storage implementation.
// These are mapped from proto full names at the server boundary.
const (
	// PublicKeyArtifactMediaType defines the internal OCI media type for public key blobs.
	PublicKeyArtifactMediaType = "application/vnd.agntcy.dir.publickey.v1+pem"

	// SignatureArtifactType defines the internal OCI media type for signature layers.
	SignatureArtifactType = "application/vnd.dev.cosign.simplesigning.v1+json"

	// DefaultReferrerArtifactMediaType defines the default internal OCI media type for referrer blobs.
	DefaultReferrerArtifactMediaType = "application/vnd.agntcy.dir.referrer.v1+json"
)

// apiToOCIType maps Dir API types to internal OCI artifact types.
func apiToOCIType(apiType string) string {
	switch apiType {
	case corev1.SignatureReferrerType:
		return SignatureArtifactType
	case corev1.PublicKeyReferrerType:
		return PublicKeyArtifactMediaType
	default:
		return DefaultReferrerArtifactMediaType
	}
}

// ociToAPIType maps internal OCI artifact types back to Dir API types.
func ociToAPIType(ociType string) string {
	switch ociType {
	case SignatureArtifactType:
		return corev1.SignatureReferrerType
	case PublicKeyArtifactMediaType:
		return corev1.PublicKeyReferrerType
	default:
		return ociType // Return the original OCI type if not found
	}
}
