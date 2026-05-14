// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package packaging

import (
	"context"
	"encoding/json"

	storev2 "github.com/agntcy/dir/api/store/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
)

var packers = make(map[string]Packer)

type Packer interface {
	// Pack takes an object and returns a packed version of it in a manifest format.
	Pack(ctx context.Context, repo *remote.Repository, obj *storev2.Object) (*ocispec.Manifest, error)

	// Unpack takes a packed manifest and returns the original object.
	Unpack(ctx context.Context, repo *remote.Repository, manifest *ocispec.Manifest) (*storev2.Object, error)
}

func RegisterPacker(mediaType string, packer Packer) {
	if _, exists := packers[mediaType]; exists {
		panic("packer for media type " + mediaType + " already registered")
	}

	packers[mediaType] = packer
}

func GetPacker(mediaType string) (Packer, bool) {
	packer, exists := packers[mediaType]

	return packer, exists
}

func Pack(ctx context.Context, repo *remote.Repository, obj *storev2.Object) (*storev2.Object, error) {
	// Get registered handlers for the media type
	packer, registered := GetPacker(obj.GetMediaType())
	if !registered {
		// If no handler is registered, return the original object
		return obj, nil
	}

	// Package as manifest
	manifest, err := packer.Pack(ctx, repo, obj)
	if err != nil {
		return nil, err
	}

	// Convert manifest to bytes
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	// Patch the object with manifest bytes
	return &storev2.Object{
		MediaType: manifest.MediaType,
		Size:      uint64(len(manifestBytes)),
		Data:      manifestBytes,
	}, nil
}
