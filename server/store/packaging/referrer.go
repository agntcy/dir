// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// This file defines the schema for DIR managed links between objects.
// It serves to simplify the management of relationships between objects in the DIR store,
// such as build artifacts and their signatures/sboms/etc.
//
// Directory relies on this object type to manage referrers, but other referrers are also
// possible, although store.Get will return their raw format (blob/manifest) instead
// of a structured ObjectReferrer data.

package packaging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	storev2 "github.com/agntcy/dir/api/store/v2"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

// Media type for referrer object.
const ReferrerMediaType = "application/vnd.agntcy.dir.objects.referrer+json"

type referrer struct{}

func init() {
	RegisterPacker(ReferrerMediaType, &referrer{})
}

func (p *referrer) Pack(ctx context.Context, repo *remote.Repository, obj *storev2.Object) (*ocispec.Manifest, error) {
	// Unpack into ObjectReferrer struct
	var referrer storev2.ObjectReferrer
	if err := json.Unmarshal(obj.GetData(), &referrer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal object data into referrer: %w", err)
	}

	// Get subject descriptor
	subjectDesc, err := repo.Manifests().Resolve(ctx, referrer.GetSubject().GetCid())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subject descriptor: %w", err)
	}

	// Push the referrer bytes as a blob
	referrerDesc, err := oras.PushBytes(ctx, repo, referrer.GetMediaType(), referrer.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to push referrer bytes: %w", err)
	}

	return &ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: int(oras.PackManifestVersion1_1),
		},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: ReferrerMediaType,
		Config:       referrerDesc,
		Layers:       []ocispec.Descriptor{ocispec.DescriptorEmptyJSON},
		Subject:      &subjectDesc,
		Annotations:  referrer.GetAnnotations(),
	}, nil
}

func (p *referrer) Unpack(ctx context.Context, repo *remote.Repository, manifest *ocispec.Manifest) (*storev2.Object, error) {
	// Get data from config layer
	reader, err := repo.Blobs().Fetch(ctx, manifest.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch referrer blob: %w", err)
	}
	defer reader.Close()

	// Read the config blob (referrer bytes)
	referrerBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read referrer data: %w", err)
	}

	// Create referrer object
	referrer := &storev2.ObjectReferrer{
		Subject:     &storev2.ObjectRef{Cid: manifest.Subject.Digest.String()},
		Annotations: manifest.Annotations,
		MediaType:   manifest.Config.MediaType,
		Size:        uint64(len(referrerBytes)),
		Data:        referrerBytes,
	}

	objectBytes, err := json.Marshal(referrer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal referrer object: %w", err)
	}

	// Return as Object
	return &storev2.Object{
		MediaType: ReferrerMediaType,
		Size:      uint64(len(objectBytes)),
		Data:      objectBytes,
	}, nil
}
