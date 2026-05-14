// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	storev2 "github.com/agntcy/dir/api/store/v2"
	"github.com/agntcy/dir/server/controller/packaging"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"oras.land/oras-go/v2"
	oraserrs "oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
)

type objectType int

const (
	unknownObject objectType = iota
	blobObject
	manifestObject
)

type objstoreCtrl struct {
	_ storev2.UnimplementedObjectStoreServer

	target *remote.Repository
}

func NewObjStore(target *remote.Repository) storev2.ObjectStoreServer {
	return &objstoreCtrl{
		target: target,
	}
}

// Put implements [v2.ObjectStoreServer].
func (o *objstoreCtrl) Put(ctx context.Context, obj *storev2.Object) (*storev2.ObjectDescriptor, error) {
	// Patch object for managed media types (e.g. records) using registered packers
	packer, registered := packaging.GetPacker(obj.GetMediaType())
	if registered {
		// Package object as manifest
		manifest, err := packer.Pack(ctx, o.target, obj)
		if err != nil {
			return nil, fmt.Errorf("failed to pack object: %w", err)
		}

		// Get manifest bytes
		manifestBytes, err := json.Marshal(manifest)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal manifest: %w", err)
		}

		// Patch the object with manifest data
		obj = &storev2.Object{
			MediaType: manifest.MediaType,
			Size:      uint64(len(manifestBytes)),
			Data:      manifestBytes,
		}
	}

	// Push object to target repo
	desc, err := oras.PushBytes(ctx, o.target, obj.GetMediaType(), obj.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to push object: %w", err)
	}

	// Return object descriptor
	//
	// TODO(ramizpolic): we can extract the data here directly instead of doing a redundant
	// network lookup, but this is simpler for now. Change to avoid redundant call.
	return o.Lookup(ctx, &storev2.ObjectRef{
		Cid: desc.Digest.String(),
	})
}

// Get implements [v2.ObjectStoreServer].
func (o *objstoreCtrl) Get(ctx context.Context, ref *storev2.ObjectRef) (*storev2.Object, error) {
	// Pull data from target registry
	// If it's not a manifest, fallback to blob fetch
	manifest, _, err := o.getManifest(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	if manifest == nil {
		return o.GetRaw(ctx, ref)
	}

	// Get registered handlers for the manifest artifact type
	// If no handler is registered, fallback to blob fetch
	packer, registered := packaging.GetPacker(manifest.ArtifactType)
	if !registered {
		return o.GetRaw(ctx, ref)
	}

	// Unpack manifest into object
	obj, err := packer.Unpack(ctx, o.target, manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack object: %w", err)
	}

	return obj, nil
}

// GetRaw implements [v2.ObjectStoreServer].
func (o *objstoreCtrl) GetRaw(ctx context.Context, ref *storev2.ObjectRef) (*storev2.Object, error) {
	// Get descriptor for the object
	_, desc, err := o.resolveRef(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference: %w", err)
	}

	// Pull data from target registry
	reader, err := o.target.Blobs().Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob: %w", err)
	}
	defer reader.Close()

	// Read data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob data: %w", err)
	}

	// Return object
	return &storev2.Object{
		MediaType: desc.MediaType,
		Size:      uint64(desc.Size),
		Data:      data,
	}, nil
}

// Has implements [v2.ObjectStoreServer].
func (o *objstoreCtrl) Has(ctx context.Context, ref *storev2.ObjectRef) (*wrapperspb.BoolValue, error) {
	// Get descriptor for the object
	_, _, err := o.resolveRef(ctx, ref)
	if errors.Is(err, oraserrs.ErrNotFound) {
		return wrapperspb.Bool(false), nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference: %w", err)
	}

	return wrapperspb.Bool(true), nil
}

// Lookup implements [v2.ObjectStoreServer].
func (o *objstoreCtrl) Lookup(ctx context.Context, ref *storev2.ObjectRef) (*storev2.ObjectDescriptor, error) {
	// Fetch manifest from target registry
	manifest, desc, err := o.getManifest(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup object: %w", err)
	}

	// If it's not a manifest, return blob descriptor
	if manifest == nil {
		return storev2.NewDescriptor(desc), nil
	}

	// Return metadata for manifest
	return &storev2.ObjectDescriptor{
		Digest:       desc.Digest.String(),
		MediaType:    manifest.MediaType,
		ArtifactType: manifest.ArtifactType,
		Size:         uint64(desc.Size),
		Urls:         manifest.Config.URLs,
		Annotations:  manifest.Annotations,
	}, nil
}

// Delete implements [v2.ObjectStoreServer].
func (o *objstoreCtrl) Delete(ctx context.Context, ref *storev2.ObjectRef) (*emptypb.Empty, error) {
	// Try deleting manifest first
	// If manifest not found, try deleting blob
	desc := ocispec.Descriptor{Digest: digest.Digest(ref.GetCid())}
	err := o.target.Manifests().Delete(ctx, desc)
	if errors.Is(err, oraserrs.ErrNotFound) {
		err = o.target.Blobs().Delete(ctx, desc)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to delete object: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// ListReferrers implements [v2.ObjectStoreServer].
func (o *objstoreCtrl) ListReferrers(ctx context.Context, ref *storev2.ObjectRef) (*storev2.ObjectDescriptors, error) {
	// Get descriptor for the object
	objType, desc, err := o.resolveRef(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference: %w", err)
	}

	// If it's not a manifest, we can't lookup referrers since referrers are only supported for manifests in OCI spec
	if objType != manifestObject {
		return &storev2.ObjectDescriptors{}, nil
	}

	// Get referrers from target registry
	var refs []*storev2.ObjectDescriptor
	err = o.target.Referrers(ctx, desc, "", func(descs []ocispec.Descriptor) error {
		for _, desc := range descs {
			refs = append(refs, storev2.NewDescriptor(desc))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get referrers: %w", err)
	}

	return &storev2.ObjectDescriptors{
		Descriptors: refs,
	}, nil
}

func (o *objstoreCtrl) getManifest(ctx context.Context, ref *storev2.ObjectRef) (*ocispec.Manifest, ocispec.Descriptor, error) {
	// Fetch manifest from target registry
	// Fallback to blob lookup if manifest fetch fails, since the reference may be a blob
	desc, reader, err := o.target.Manifests().FetchReference(ctx, ref.GetCid())
	if errors.Is(err, oraserrs.ErrNotFound) {
		desc, reader, err = o.target.Blobs().FetchReference(ctx, ref.GetCid())
	}

	if err != nil {
		return nil, ocispec.Descriptor{}, err
	}

	defer reader.Close()

	// If its not a manifest, we can't lookup metadata.
	if desc.MediaType != "application/vnd.oci.image.manifest.v1+json" {
		return nil, desc, nil
	}

	// Read manifest data
	manifestData, err := io.ReadAll(reader)
	if err != nil {
		return nil, ocispec.Descriptor{}, err
	}

	// Parse manifest
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, ocispec.Descriptor{}, err
	}

	return &manifest, desc, nil
}

func (o *objstoreCtrl) resolveRef(ctx context.Context, ref *storev2.ObjectRef) (objectType, ocispec.Descriptor, error) {
	// Resolve manifest from target registry
	// Fallback to blob lookup if manifest fetch fails, since the reference may be a blob
	desc, err := o.target.Manifests().Resolve(ctx, ref.GetCid())
	if errors.Is(err, oraserrs.ErrNotFound) {
		desc, err = o.target.Blobs().Resolve(ctx, ref.GetCid())
	}

	if err != nil {
		return unknownObject, ocispec.Descriptor{}, err
	}

	// Determine if the reference is a manifest or blob based on media type
	if desc.MediaType == "application/vnd.oci.image.manifest.v1+json" {
		// It's a manifest reference
		return manifestObject, desc, nil
	}

	return blobObject, desc, nil
}
