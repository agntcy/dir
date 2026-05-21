// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	storev2 "github.com/agntcy/dir/api/store/v2"
	"github.com/agntcy/dir/server/store/packaging"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"oras.land/oras-go/v2"
	oraserrs "oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
)

type objstoreCtrl struct {
	_ storev2.UnimplementedObjectStoreServer

	target *remote.Repository
}

// TODO: fix status codes returned by the controller methods
// to be more specific and accurate instead of always returning codes.Internal.
func NewObjStore(target *remote.Repository) storev2.ObjectStoreServer {
	return &objstoreCtrl{
		target: target,
	}
}

func (o *objstoreCtrl) Put(ctx context.Context, obj *storev2.Object) (*storev2.ObjectDescriptor, error) {
	// Patch object for managed media types (e.g. records) using registered packers
	packer, registered := packaging.GetPacker(obj.GetMediaType())
	if registered {
		// Package object as manifest
		manifest, err := packer.Pack(ctx, o.target, obj)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to pack object: %v", err)
		}

		// Get manifest bytes
		manifestBytes, err := json.Marshal(manifest)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal manifest: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to push object: %v", err)
	}

	// Return object descriptor
	//
	// TODO: we can extract the data here directly instead of doing a redundant
	// network lookup, but this is simpler for now. Change to avoid network call.
	return o.Lookup(ctx, &storev2.ObjectRef{
		Cid: desc.Digest.String(),
	})
}

func (o *objstoreCtrl) Get(ctx context.Context, ref *storev2.ObjectRef) (*storev2.Object, error) {
	// Pull data from target registry
	// If it's not a manifest, fallback to blob fetch
	manifest, _, err := o.getManifest(ctx, ref)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get manifest: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to unpack object: %v", err)
	}

	return obj, nil
}

func (o *objstoreCtrl) GetRaw(ctx context.Context, ref *storev2.ObjectRef) (*storev2.Object, error) {
	// Fetch reader from target registry
	// Fallback to blob lookup if manifest fetch fails, since the reference may be a blob
	desc, reader, err := o.target.Manifests().FetchReference(ctx, ref.GetCid())
	if errors.Is(err, oraserrs.ErrNotFound) {
		desc, reader, err = o.target.Blobs().FetchReference(ctx, ref.GetCid())
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch object: %v", err)
	}

	defer reader.Close()

	// Read data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read object data: %v", err)
	}

	// Some OCI registries use "text/plain" for blobs, use "application/octet-stream"
	mediaType := desc.MediaType
	if mediaType == "text/plain" {
		mediaType = "application/octet-stream"
	}

	// Return object
	return &storev2.Object{
		MediaType: mediaType,
		Size:      uint64(desc.Size), //nolint:gosec
		Data:      data,
	}, nil
}

func (o *objstoreCtrl) Has(ctx context.Context, ref *storev2.ObjectRef) (*wrapperspb.BoolValue, error) {
	// Get descriptor for the object
	_, err := o.resolveRef(ctx, ref)
	if errors.Is(err, oraserrs.ErrNotFound) {
		return wrapperspb.Bool(false), nil
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve reference: %v", err)
	}

	return wrapperspb.Bool(true), nil
}

func (o *objstoreCtrl) Lookup(ctx context.Context, ref *storev2.ObjectRef) (*storev2.ObjectDescriptor, error) {
	// Fetch manifest from target registry
	manifest, desc, err := o.getManifest(ctx, ref)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to lookup object: %v", err)
	}

	// If it's not a manifest, return blob descriptor
	if manifest == nil {
		return toObjectDescriptor(desc), nil
	}

	// Return metadata for manifest
	return &storev2.ObjectDescriptor{
		Digest:       desc.Digest.String(),
		MediaType:    manifest.MediaType,
		ArtifactType: manifest.ArtifactType,
		Size:         uint64(desc.Size), //nolint:gosec
		Urls:         manifest.Config.URLs,
		Annotations:  manifest.Annotations,
	}, nil
}

func (o *objstoreCtrl) Delete(ctx context.Context, ref *storev2.ObjectRef) (*emptypb.Empty, error) {
	// Try deleting manifest first
	// If manifest not found, try deleting blob
	desc := ocispec.Descriptor{Digest: digest.Digest(ref.GetCid())}

	err := o.target.Manifests().Delete(ctx, desc)
	if errors.Is(err, oraserrs.ErrNotFound) {
		err = o.target.Blobs().Delete(ctx, desc)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete object: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (o *objstoreCtrl) ListReferrers(ctx context.Context, req *storev2.ListReferrersRequest) (*storev2.ListReferrersResponse, error) {
	// Get descriptor for the object
	subjectDesc, err := o.resolveRef(ctx, req.GetSubject())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve reference: %v", err)
	}

	// If it's not a manifest, we can't lookup referrers since referrers are only supported for manifests in OCI spec
	if subjectDesc.MediaType != ocispec.MediaTypeImageManifest {
		return &storev2.ListReferrersResponse{}, nil
	}

	// Get referrers from target registry
	var refs []*storev2.ObjectDescriptor

	if err = o.target.Referrers(ctx, subjectDesc,
		req.GetFilterMediaType(),
		func(descs []ocispec.Descriptor) error {
			for _, desc := range descs {
				refs = append(refs, toObjectDescriptor(desc))
			}

			return nil
		},
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get referrers: %v", err)
	}

	return &storev2.ListReferrersResponse{
		Referrers: refs,
	}, nil
}

//nolint:wrapcheck
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
	if desc.MediaType != ocispec.MediaTypeImageManifest {
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

//nolint:wrapcheck
func (o *objstoreCtrl) resolveRef(ctx context.Context, ref *storev2.ObjectRef) (ocispec.Descriptor, error) {
	// Resolve manifest from target registry
	// Fallback to blob lookup if manifest fetch fails, since the reference may be a blob
	desc, err := o.target.Manifests().Resolve(ctx, ref.GetCid())
	if errors.Is(err, oraserrs.ErrNotFound) {
		desc, err = o.target.Blobs().Resolve(ctx, ref.GetCid())
	}

	if err != nil {
		return ocispec.Descriptor{}, err
	}

	return desc, nil
}

func toObjectDescriptor(desc ocispec.Descriptor) *storev2.ObjectDescriptor {
	return &storev2.ObjectDescriptor{
		Digest:       desc.Digest.String(),
		MediaType:    desc.MediaType,
		ArtifactType: desc.ArtifactType,
		Size:         uint64(desc.Size), //nolint:gosec
		Urls:         desc.URLs,
		Annotations:  desc.Annotations,
	}
}
