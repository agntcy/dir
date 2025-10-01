// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

const (
	// PublicKeyArtifactMediaType defines the media type for public key blobs.
	PublicKeyArtifactMediaType = "application/vnd.agntcy.dir.publickey.v1+pem"

	// SignatureArtifactType defines the media type for cosign signature layers.
	SignatureArtifactType = "application/vnd.dev.cosign.simplesigning.v1+json"

	// ReferrerArtifactMediaType defines the media type for referrer blobs.
	DefaultReferrerArtifactMediaType = "application/vnd.agntcy.dir.referrer.v1+json"
)
