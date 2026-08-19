// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package indexer

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// maxManifestSize bounds how much data is read while inspecting a manifest.
// Manifests are a few kilobytes at most, so anything larger is rejected rather
// than buffered.
const maxManifestSize = 1 << 20 // 1 MiB

// manifestReader is the subset of the OCI target API needed to read the
// manifest behind a tag. Both the local *oci.Store and the remote
// *remote.Repository that back the task's tag lister implement it.
type manifestReader interface {
	content.Resolver
	content.Fetcher
}

// isReferrerTag reports whether the manifest behind tag is a referrer artifact
// (signature, public key, scan report, ...) instead of a record.
//
// Referrers are stored in the same repository as their subject record and are
// tagged with their own CID, so a tag parsing as a CID does not make it a
// record. Referrer manifests are identified by the OCI subject field that links
// them to the record they describe, and by the Dir referrer type annotation.
//
// If the tag cannot be inspected, the error is returned and the caller decides
// how to proceed; an unreadable manifest is not assumed to be a referrer.
func (t *Task) isReferrerTag(ctx context.Context, tag string) (bool, error) {
	if t.manifests == nil {
		return false, nil
	}

	desc, err := t.manifests.Resolve(ctx, tag)
	if err != nil {
		return false, fmt.Errorf("failed to resolve tag: %w", err)
	}

	if desc.MediaType != ocispec.MediaTypeImageManifest {
		return false, nil
	}

	if desc.Size > maxManifestSize {
		return false, fmt.Errorf("manifest is too large: %d bytes", desc.Size)
	}

	// FetchAll verifies the content against the descriptor size and digest.
	manifestData, err := content.FetchAll(ctx, t.manifests, desc)
	if err != nil {
		return false, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return false, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return isReferrerManifest(&manifest), nil
}

// isReferrerManifest reports whether a manifest describes a referrer artifact.
func isReferrerManifest(manifest *ocispec.Manifest) bool {
	if manifest.Subject != nil {
		return true
	}

	_, hasReferrerType := manifest.Annotations[corev1.ReferrerTypeAnnotationKey]

	return hasReferrerType
}
