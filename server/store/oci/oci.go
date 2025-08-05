// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck,nilerr,gosec
package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/store/cache"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// OCI-specific constants for signature artifacts.
const (
	// SignatureArtifactMediaType is the media type for signature artifacts.
	SignatureArtifactMediaType = "application/vnd.dev.cosign.artifact.sig.v1+json"

	// SignatureManifestMediaType is the media type for signature manifests.
	SignatureManifestMediaType = "application/vnd.oci.image.manifest.v1+json"

	// Cosign simple signing media type for signature layers.
	CosignSimpleSigningMediaType = "application/vnd.dev.cosign.simplesigning.v1+json"

	// Signature annotations.
	SignatureTypeAnnotation = "org.opencontainers.artifact.type"

	// Cosign-specific annotations.
	CosignSignatureAnnotation = "dev.cosignproject.cosign/signature"
	CosignBundleAnnotation    = "dev.sigstore.cosign/bundle"
)

var logger = logging.Logger("store/oci")

type store struct {
	repo   oras.GraphTarget
	config ociconfig.Config
}

func New(cfg ociconfig.Config) (types.StoreAPI, error) {
	logger.Debug("Creating OCI store with config", "config", cfg)

	// if local dir used, return client for that local path.
	// allows mounting of data via volumes
	// allows S3 usage for backup store
	if repoPath := cfg.LocalDir; repoPath != "" {
		repo, err := oci.New(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create local repo: %w", err)
		}

		return &store{
			repo:   repo,
			config: cfg,
		}, nil
	}

	// create remote client
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", cfg.RegistryAddress, cfg.RepositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// configure client to remote
	repo.PlainHTTP = cfg.Insecure
	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Header: http.Header{
			"User-Agent": {"dir-client"},
		},
		Cache: auth.DefaultCache,
		Credential: auth.StaticCredential(
			cfg.RegistryAddress,
			auth.Credential{
				Username:     cfg.Username,
				Password:     cfg.Password,
				RefreshToken: cfg.RefreshToken,
				AccessToken:  cfg.AccessToken,
			},
		),
	}

	// Create store API
	store := &store{
		repo:   repo,
		config: cfg,
	}

	// If no cache requested, return.
	// Do not use in memory cache as it can get large.
	if cfg.CacheDir == "" {
		return store, nil
	}

	// Create cache datastore
	cacheDS, err := datastore.New(datastore.WithFsProvider(cfg.CacheDir))
	if err != nil {
		return nil, fmt.Errorf("failed to create cache store: %w", err)
	}

	// Return cached store
	return cache.Wrap(store, cacheDS), nil
}

// Push record to the OCI registry
//
// This creates a blob, a manifest that points to that blob, and a tagged release for that manifest.
// The tag for the manifest is: <CID of digest>.
// The tag for the blob is needed to link the actual record with its associated metadata.
// Note that metadata can be stored in a different store and only wrap this store.
//
// Ref: https://github.com/oras-project/oras-go/blob/main/docs/Modeling-Artifacts.md
func (s *store) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	logger.Debug("Pushing record to OCI store", "record", record)

	// Marshal the record using canonical JSON marshaling first
	// This ensures consistent bytes for both CID calculation and storage
	recordBytes, err := record.MarshalOASF()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal record: %v", err)
	}

	// Step 1: Use oras.PushBytes to push the record data and get Layer Descriptor
	layerDesc, err := oras.PushBytes(ctx, s.repo, "application/json", recordBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push record bytes: %v", err)
	}

	// Step 2: Calculate CID from Layer Descriptor's digest using our new utility function
	recordCID, err := corev1.ConvertDigestToCID(layerDesc.Digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert digest to CID: %v", err)
	}

	// Validate consistency: CID from ORAS digest should match CID from record
	expectedCID := record.GetCid()
	if recordCID != expectedCID {
		return nil, status.Errorf(codes.Internal,
			"CID mismatch: OCI digest CID (%s) != Record CID (%s)",
			recordCID, expectedCID)
	}

	logger.Debug("CID validation successful",
		"cid", recordCID,
		"digest", layerDesc.Digest.String(),
		"validation", "ORAS digest CID matches Record CID")

	logger.Debug("Calculated CID from ORAS digest", "cid", recordCID, "digest", layerDesc.Digest.String())

	// Create record reference
	recordRef := &corev1.RecordRef{Cid: recordCID}

	// Check if record already exists
	if _, err := s.Lookup(ctx, recordRef); err == nil {
		logger.Info("Record already exists in OCI store", "cid", recordCID)

		return recordRef, nil
	}

	// Step 3: Construct manifest annotations and add CID to annotations
	manifestAnnotations := extractManifestAnnotations(record)
	// Add the calculated CID to manifest annotations for discovery
	manifestAnnotations[ManifestKeyCid] = recordCID

	// Step 4: Pack manifest (in-memory only)
	manifestDesc, err := oras.PackManifest(ctx, s.repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ManifestAnnotations: manifestAnnotations,
			Layers: []ocispec.Descriptor{
				layerDesc,
			},
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pack manifest: %v", err)
	}

	// Step 5: Create CID tag for content-addressable storage
	cidTag := recordCID
	logger.Debug("Generated CID tag", "cid", recordCID, "tag", cidTag)

	// Step 6: Tag the manifest with CID tag
	// => resolve manifest to record which can be looked up (lookup)
	// => allows pulling record directly (pull)
	if _, err := oras.Tag(ctx, s.repo, manifestDesc.Digest.String(), cidTag); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CID tag: %v", err)
	}

	logger.Info("Record pushed to OCI store successfully", "cid", recordCID, "tag", cidTag)

	// Return record reference
	return recordRef, nil
}

// Lookup checks if the ref exists as a tagged record.
func (s *store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	// Input validation using shared helper
	if err := validateRecordRef(ref); err != nil {
		return nil, err
	}

	logger.Debug("Starting record lookup", "cid", ref.GetCid())

	// Use shared helper to fetch and parse manifest (eliminates code duplication)
	manifest, _, err := s.fetchAndParseManifest(ctx, ref.GetCid())
	if err != nil {
		return nil, err // Error already has proper context from helper
	}

	// Extract and validate record type from manifest metadata
	recordType, ok := manifest.Annotations[manifestDirObjectTypeKey]
	if !ok {
		return nil, status.Errorf(codes.Internal, "record type not found in manifest annotations for CID %s: missing key %s",
			ref.GetCid(), manifestDirObjectTypeKey)
	}

	// Extract comprehensive metadata from manifest annotations using our enhanced parser
	recordMeta := parseManifestAnnotations(manifest.Annotations)

	// Set the CID from the request (this is the primary identifier)
	recordMeta.Cid = ref.GetCid()

	logger.Debug("Record metadata retrieved successfully",
		"cid", ref.GetCid(),
		"type", recordType,
		"annotationCount", len(manifest.Annotations))

	return recordMeta, nil
}

func (s *store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	// Input validation using shared helper
	if err := validateRecordRef(ref); err != nil {
		return nil, err
	}

	logger.Debug("Starting record pull", "cid", ref.GetCid())

	// Use shared helper to fetch and parse manifest (eliminates code duplication)
	manifest, manifestDesc, err := s.fetchAndParseManifest(ctx, ref.GetCid())
	if err != nil {
		return nil, err // Error already has proper context from helper
	}

	// Validate manifest has layers
	if len(manifest.Layers) == 0 {
		return nil, status.Errorf(codes.Internal, "manifest has no layers for CID %s", ref.GetCid())
	}

	// Handle multiple layers with warning
	if len(manifest.Layers) > 1 {
		logger.Warn("Manifest has multiple layers, using first layer",
			"cid", ref.GetCid(),
			"layerCount", len(manifest.Layers))
	}

	// Get the blob descriptor from the first layer
	blobDesc := manifest.Layers[0]

	// Validate layer media type
	if blobDesc.MediaType != "application/json" {
		logger.Warn("Unexpected blob media type",
			"cid", ref.GetCid(),
			"expected", "application/json",
			"actual", blobDesc.MediaType)
	}

	logger.Debug("Fetching record blob",
		"cid", ref.GetCid(),
		"blobDigest", blobDesc.Digest.String(),
		"blobSize", blobDesc.Size,
		"mediaType", blobDesc.MediaType)

	// Fetch the record data using the correct blob descriptor from the manifest
	reader, err := s.repo.Fetch(ctx, blobDesc)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "record blob not found for CID %s: %v", ref.GetCid(), err)
	}
	defer reader.Close()

	// Read all data from the reader
	recordData, err := io.ReadAll(reader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read record data for CID %s: %v", ref.GetCid(), err)
	}

	// Validate blob size matches descriptor
	if blobDesc.Size > 0 && int64(len(recordData)) != blobDesc.Size {
		logger.Warn("Blob size mismatch",
			"cid", ref.GetCid(),
			"expected", blobDesc.Size,
			"actual", len(recordData))
	}

	// Unmarshal canonical JSON data back to Record
	record, err := corev1.UnmarshalOASF(recordData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal record for CID %s: %v", ref.GetCid(), err)
	}

	logger.Debug("Record pulled successfully",
		"cid", ref.GetCid(),
		"blobSize", len(recordData),
		"blobDigest", blobDesc.Digest.String(),
		"manifestDigest", manifestDesc.Digest.String())

	return record, nil
}

func (s *store) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	logger.Debug("Deleting record from OCI store", "ref", ref)

	// Input validation using shared helper
	if err := validateRecordRef(ref); err != nil {
		return err
	}

	switch s.repo.(type) {
	case *oci.Store:
		return s.deleteFromOCIStore(ctx, ref)
	case *remote.Repository:
		return s.deleteFromRemoteRepository(ctx, ref)
	default:
		return status.Errorf(codes.FailedPrecondition, "unsupported repo type: %T", s.repo)
	}
}

// PushSignature stores OCI signature artifacts for a record.
func (s *store) PushSignature(ctx context.Context, recordCID string, signature *signv1.Signature) error {
	logger.Debug("Pushing signature artifact to OCI store", "recordCID", recordCID)

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required")
	}

	// Create signature blob
	signatureDesc, err := s.pushSignatureBlob(ctx, signature)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to push signature blob: %v", err)
	}

	// Create signature manifest that references the signed record
	signatureManifestDesc, err := s.createSignatureManifest(ctx, signatureDesc, recordCID, signature)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create signature manifest: %v", err)
	}

	logger.Debug("Signature artifact pushed successfully", "digest", signatureManifestDesc.Digest.String())

	return nil
}

// PullSignature retrieves signature associated with a record.
func (s *store) PullSignature(ctx context.Context, recordCID string) (*signv1.Signature, error) {
	logger.Debug("Pulling signature from OCI store", "recordCID", recordCID)

	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required")
	}

	// Find signature manifest for this record using OCI Referrers API
	signatureManifestDesc, err := s.findSignatureManifest(ctx, recordCID)
	if err != nil {
		logger.Debug("Failed to find signature manifest", "error", err, "recordCID", recordCID)

		return nil, status.Errorf(codes.NotFound, "signature not found: %s", recordCID)
	}

	// Fetch signature manifest
	manifestReader, err := s.repo.Fetch(ctx, *signatureManifestDesc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch signature manifest: %v", err)
	}
	defer manifestReader.Close()

	manifestData, err := io.ReadAll(manifestReader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read signature manifest: %v", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal signature manifest: %v", err)
	}

	// Extract signature artifact from manifest layers
	if len(manifest.Layers) == 0 {
		return nil, status.Errorf(codes.Internal, "signature manifest has no layers")
	}

	// Fetch signature blob data
	signatureBlob := manifest.Layers[0]

	signatureReader, err := s.repo.Fetch(ctx, signatureBlob)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch signature blob: %v", err)
	}
	defer signatureReader.Close()

	signatureData, err := io.ReadAll(signatureReader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read signature blob: %v", err)
	}

	// Unmarshal the complete signature object from blob content
	var signature signv1.Signature
	if err := json.Unmarshal(signatureData, &signature); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal signature: %v", err)
	}

	// Restore original content type from annotations if available
	if signatureBlob.Annotations != nil {
		if originalContentType, ok := signatureBlob.Annotations["original.content.type"]; ok && originalContentType != "" {
			signature.ContentType = originalContentType
		}
	}

	logger.Debug("Retrieved signature artifact", "recordCID", recordCID)

	return &signature, nil
}

// DeleteSignature removes signature associated with a record.
func (s *store) DeleteSignature(ctx context.Context, recordCID string) error {
	logger.Debug("Deleting signature from OCI store", "recordCID", recordCID)

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required")
	}

	// Find signature manifest for this record using OCI Referrers API
	signatureManifestDesc, err := s.findSignatureManifest(ctx, recordCID)
	if err != nil {
		logger.Debug("Failed to find signature manifest for deletion", "error", err, "recordCID", recordCID)

		return nil
	}

	// Delete signature manifest and associated blobs
	switch store := s.repo.(type) {
	case *oci.Store:
		// Delete manifest
		if err := store.Delete(ctx, *signatureManifestDesc); err != nil {
			return fmt.Errorf("failed to delete signature manifest %s: %w", signatureManifestDesc.Digest.String(), err)
		}
	case *remote.Repository:
		// Delete manifest
		if err := store.Manifests().Delete(ctx, *signatureManifestDesc); err != nil {
			return fmt.Errorf("failed to delete signature manifest %s: %w", signatureManifestDesc.Digest.String(), err)
		}
	default:
		return fmt.Errorf("unsupported repo type for signature deletion: %T", s.repo)
	}

	logger.Info("Signature deletion completed", "recordCID", recordCID)

	return nil
}

// pushSignatureBlob pushes a signature artifact as a blob and returns its descriptor.
func (s *store) pushSignatureBlob(ctx context.Context, signature *signv1.Signature) (ocispec.Descriptor, error) {
	// Store the original signature data as the blob content so it can be reconstructed
	signatureJSON, err := json.Marshal(signature)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal signature: %w", err)
	}

	// Create annotations for cosign compatibility
	annotations := make(map[string]string)

	// Add cosign signature annotation if signature data is available
	if signature.GetSignature() != "" {
		annotations[CosignSignatureAnnotation] = signature.GetSignature()
	}

	// Add cosign bundle annotation if content bundle is available
	if signature.GetContentBundle() != "" {
		annotations[CosignBundleAnnotation] = signature.GetContentBundle()
	}

	// Use cosign simple signing media type for zot compatibility
	mediaType := CosignSimpleSigningMediaType

	if signature.GetContentType() != "" {
		// Store the original content type in annotations for reconstruction
		annotations["original.content.type"] = signature.GetContentType()
	}

	// Push the signature blob with original signature data but cosign media type
	blobDesc, err := oras.PushBytes(ctx, s.repo, mediaType, signatureJSON)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push signature blob: %w", err)
	}

	// Update descriptor with annotations for cosign compatibility
	blobDesc.Annotations = annotations

	return blobDesc, nil
}

// createSignatureManifest creates a signature manifest that references the signed record using OCI subject field.
func (s *store) createSignatureManifest(ctx context.Context, signatureDesc ocispec.Descriptor, recordCID string, signature *signv1.Signature) (ocispec.Descriptor, error) {
	// First, resolve the record manifest to get its descriptor for the subject field
	recordManifestDesc, err := s.repo.Resolve(ctx, recordCID)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to resolve record manifest for subject: %w", err)
	}

	// Create annotations for the signature manifest
	annotations := make(map[string]string)

	// Copy signature annotations
	for k, v := range signature.GetAnnotations() {
		annotations[k] = v
	}

	// Add OCI-required signature artifact type annotation
	annotations[SignatureTypeAnnotation] = SignatureArtifactMediaType

	// Add cosign-specific annotations for better zot compatibility
	if signature.GetSignature() != "" {
		annotations[CosignSignatureAnnotation] = signature.GetSignature()
	}

	if signature.GetContentBundle() != "" {
		annotations[CosignBundleAnnotation] = signature.GetContentBundle()
	}

	// Create the signature manifest with proper OCI subject field
	manifestDesc, err := oras.PackManifest(ctx, s.repo, oras.PackManifestVersion1_1, SignatureManifestMediaType,
		oras.PackManifestOptions{
			ManifestAnnotations: annotations,
			Subject:             &recordManifestDesc, // OCI 1.1 subject field
			Layers: []ocispec.Descriptor{
				signatureDesc, // Cosign simple signing layer
			},
		},
	)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to pack signature manifest: %w", err)
	}

	return manifestDesc, nil
}

// ReferrersLister interface for repositories that support the OCI Referrers API.
type ReferrersLister interface {
	Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error
}

// findSignatureManifest finds the signature manifest for a given record using OCI Referrers API.
func (s *store) findSignatureManifest(ctx context.Context, recordCID string) (*ocispec.Descriptor, error) {
	logger.Debug("Finding signature manifest for record", "recordCID", recordCID)

	// First, resolve the record manifest to get its descriptor
	recordManifestDesc, err := s.repo.Resolve(ctx, recordCID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve record manifest: %w", err)
	}

	// Check if the repository supports the referrers API
	referrerLister, ok := s.repo.(ReferrersLister)
	if !ok {
		logger.Debug("Repository does not support Referrers API, falling back to tag schema")

		return nil, errors.New("repository does not support Referrers API")
	}

	// Use the Referrers API to find signature manifests
	var signatureManifestDesc *ocispec.Descriptor

	err = referrerLister.Referrers(ctx, recordManifestDesc, SignatureManifestMediaType, func(referrers []ocispec.Descriptor) error {
		for _, referrer := range referrers {
			if s.isSignatureManifest(referrer) {
				signatureManifestDesc = &referrer
				logger.Debug("Found signature manifest using Referrers API", "digest", referrer.Digest.String())

				return nil // Found our signature manifest
			}
		}

		return nil // no signature manifest found
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query referrers: %w", err)
	}

	if signatureManifestDesc != nil {
		return signatureManifestDesc, nil
	}

	return nil, fmt.Errorf("no signature manifest found for record %s", recordCID)
}

// isSignatureManifest checks if a descriptor represents a signature manifest.
func (s *store) isSignatureManifest(desc ocispec.Descriptor) bool {
	// Check media type
	if desc.MediaType != SignatureManifestMediaType {
		return false
	}

	// Check signature type annotation
	if artifactType, ok := desc.Annotations[SignatureTypeAnnotation]; !ok || artifactType != SignatureArtifactMediaType {
		return false
	}

	return true
}

// VerifyWithZot queries zot's verification API to check if a signature is valid.
func (s *store) VerifyWithZot(ctx context.Context, recordCID string) (bool, error) {
	// Skip if using local storage
	if s.config.LocalDir != "" {
		logger.Debug("Skipping zot verification for local storage")

		return false, errors.New("zot verification not available for local storage")
	}

	// Build zot search endpoint URL
	registryURL := s.config.RegistryAddress
	if !strings.HasPrefix(registryURL, "http://") && !strings.HasPrefix(registryURL, "https://") {
		if s.config.Insecure {
			registryURL = "http://" + registryURL
		} else {
			registryURL = "https://" + registryURL
		}
	}

	searchEndpoint := registryURL + "/v2/_zot/ext/search"
	logger.Debug("Querying zot for signature verification", "endpoint", searchEndpoint, "recordCID", recordCID)

	// Create GraphQL query for signature verification
	query := fmt.Sprintf(`{
		Image(image: "%s:%s") {
			Digest
			IsSigned
			Tag
			SignatureInfo {
				Tool
				IsTrusted
				Author
			}
		}
	}`, s.config.RepositoryName, recordCID)

	graphqlQuery := map[string]interface{}{
		"query": query,
	}

	jsonData, err := json.Marshal(graphqlQuery)
	if err != nil {
		return false, fmt.Errorf("failed to marshal GraphQL query: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create verification request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if s.config.Username != "" && s.config.Password != "" {
		req.SetBasicAuth(s.config.Username, s.config.Password)
	} else if s.config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	}

	// Create HTTP client
	client := &http.Client{}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to query zot verification: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read verification response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Debug("Verification query returned non-success status", "status", resp.StatusCode, "body", string(body))

		return false, fmt.Errorf("verification query returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse GraphQL response
	var graphqlResp struct {
		Data struct {
			Image struct {
				Digest        string `json:"Digest"`
				IsSigned      bool   `json:"IsSigned"`
				Tag           string `json:"Tag"`
				SignatureInfo []struct {
					Tool      string `json:"Tool"`
					IsTrusted bool   `json:"IsTrusted"`
					Author    string `json:"Author"`
				} `json:"SignatureInfo"`
			} `json:"Image"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return false, fmt.Errorf("failed to decode verification response: %w", err)
	}

	// Check if the image is signed according to zot
	isSigned := graphqlResp.Data.Image.IsSigned
	logger.Debug("Zot verification result", "recordCID", recordCID, "isSigned", isSigned)

	return isSigned, nil
}
