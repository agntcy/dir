// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"sign-poc/zot"

	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

// Connect to two zot registries with oras
// Push a two artifacts to one of the registries
// Sync the artifacts to the other registry using regclient, zot or other sync extension
// Verify that the artifacts are present in both registries

const (
	sourceRegistry = "localhost:5000"
	targetRegistry = "localhost:5001"
	sourceRepo     = "demo"
	targetRepo     = "demo"
)

func main() {
	ctx := context.Background()

	// create client to source registry
	sourceRepoClient, err := remote.NewRepository(fmt.Sprintf("%s/%s", sourceRegistry, sourceRepo))
	if err != nil {
		fmt.Printf("failed to connect to source registry: %v\n", err)

		return
	}

	sourceRepoClient.PlainHTTP = true

	// create client to target registry
	targetRepoClient, err := remote.NewRepository(fmt.Sprintf("%s/%s", targetRegistry, targetRepo))
	if err != nil {
		fmt.Printf("failed to connect to target registry: %v\n", err)

		return
	}

	targetRepoClient.PlainHTTP = true

	// push artifact to source registry
	artifacts := []string{"artifact1", "artifact2"}
	for _, artifact := range artifacts {
		err = pushArtifact(ctx, sourceRepoClient, artifact, "")
		if err != nil {
			fmt.Printf("failed to push %s: %v\n", artifact, err)

			return
		}
	}

	// push empty artifact to target registry with same tag as artifact1
	content := "very important content"
	err = pushArtifact(ctx, targetRepoClient, artifacts[0], content)
	if err != nil {
		fmt.Printf("failed to push empty artifact: %v\n", err)

		return
	}

	// Verify artifacts exist in source registry
	for _, artifact := range artifacts {
		_, err := verifyArtifact(ctx, sourceRepoClient, artifact)
		if err != nil {
			fmt.Printf("failed to verify %s in source registry: %v\n", artifact, err)

			return
		}
	}

	// Sync artifacts to target registry
	err = zot.Sync(sourceRegistry, targetRegistry, sourceRepo, targetRepo)
	if err != nil {
		fmt.Printf("failed to sync artifacts: %v\n", err)
	}

	// Verify artifacts exist in target registry
	for _, artifact := range artifacts {
		_, err := verifyArtifact(ctx, targetRepoClient, artifact)
		if err != nil {
			fmt.Printf("failed to verify %s in target registry: %v\n", artifact, err)

			return
		}
	}

	// Check if tag points to empty blob content
	isEmpty, err := verifyBlob(ctx, targetRepoClient, artifacts[0])
	if err != nil {
		fmt.Printf("failed to verify blob: %v\n", err)

		return
	}

	if isEmpty {
		fmt.Printf("❌ FAILURE: %s is empty\n", artifacts[0])
	} else {
		fmt.Printf("✅ SUCCESS: %s is not empty\n", artifacts[0])
	}

	time.Sleep(10 * time.Minute)
}

func verifyBlob(ctx context.Context, targetRepoClient *remote.Repository, tag string) (bool, error) {
	fmt.Printf("Verifying blob content for tag: %s\n", tag)

	// Get the manifest descriptor for the tag
	manifestDesc, err := targetRepoClient.Resolve(ctx, tag)
	if err != nil {
		return false, fmt.Errorf("failed to resolve manifest for tag %s: %v", tag, err)
	}
	fmt.Printf("Resolved manifest digest: %s\n", manifestDesc.Digest)

	// Fetch the manifest content
	manifestReader, err := targetRepoClient.Fetch(ctx, manifestDesc)
	if err != nil {
		return false, fmt.Errorf("failed to fetch manifest: %v", err)
	}
	defer manifestReader.Close()

	// Parse the manifest to get blob references
	var manifest ocispec.Manifest
	manifestData, err := io.ReadAll(manifestReader)
	if err != nil {
		return false, fmt.Errorf("failed to read manifest: %v", err)
	}

	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return false, fmt.Errorf("failed to parse manifest: %v", err)
	}

	// Check if all blob layers are empty using OCI spec
	for i, layer := range manifest.Layers {
		fmt.Printf("Layer %d: digest=%s, size=%d, mediaType=%s\n", i, layer.Digest, layer.Size, layer.MediaType)

		// Check if this layer uses the empty JSON media type
		if layer.MediaType == ocispec.MediaTypeEmptyJSON {
			fmt.Printf("Found empty JSON media type in layer %d for tag: %s\n", i, tag)
			continue
		}

		// Check layer size
		if layer.Size == int64(len([]byte(""))) {
			fmt.Printf("Found zero-size blob in layer %d for tag: %s\n", i, tag)
			continue
		}

		// If we find any non-empty layer, the artifact is not "empty"
		fmt.Printf("Found non-empty layer %d for tag: %s (size: %d bytes)\n", i, tag, layer.Size)
		return false, nil
	}

	// All blobs are empty
	fmt.Printf("All blobs are empty for tag: %s\n", tag)
	return true, nil
}

func replaceBlobWithEmptyBlob(ctx context.Context, targetRepoClient *remote.Repository, tag string) error {
	fmt.Printf("Starting blob removal for tag: %s\n", tag)

	// Get the manifest descriptor
	manifestDesc, err := targetRepoClient.Resolve(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to resolve manifest: %v", err)
	}
	fmt.Printf("Original manifest digest: %s\n", manifestDesc.Digest)

	// Fetch the actual manifest content
	manifestReader, err := targetRepoClient.Fetch(ctx, manifestDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %v", err)
	}
	defer manifestReader.Close()

	// Parse the manifest to get blob references
	var manifest ocispec.Manifest
	manifestData, err := io.ReadAll(manifestReader)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %v", err)
	}

	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %v", err)
	}

	// Create a new manifest with empty blob content using OCI spec
	// This keeps the tag but replaces content with empty data
	for i := range manifest.Layers {
		// Create empty content using OCI empty JSON spec
		emptyContent := ""
		emptyBytes := []byte(emptyContent)

		// Push empty blob to registry using MediaTypeEmptyJSON
		emptyBlobDesc := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeEmptyJSON,
			Digest:    ocidigest.FromBytes(emptyBytes),
			Size:      int64(len(emptyBytes)),
		}

		err := targetRepoClient.Push(ctx, emptyBlobDesc, strings.NewReader(emptyContent))
		if err != nil {
			fmt.Printf("Warning: failed to push empty blob: %v (continuing anyway)\n", err)
		}

		fmt.Printf("Replacing blob %s with empty blob %s (MediaType: %s)\n",
			manifest.Layers[i].Digest, emptyBlobDesc.Digest, emptyBlobDesc.MediaType)
		manifest.Layers[i] = emptyBlobDesc
	}

	// Re-marshal the modified manifest
	modifiedManifestData, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal modified manifest: %v", err)
	}

	// Push the modified manifest directly to the tag (this replaces the existing tag)
	manifestDesc = ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    ocidigest.FromBytes(modifiedManifestData),
		Size:      int64(len(modifiedManifestData)),
	}

	// Push manifest directly to the tag - this overwrites the existing tag
	err = targetRepoClient.Manifests().PushReference(ctx, manifestDesc, strings.NewReader(string(modifiedManifestData)), tag)
	if err != nil {
		return fmt.Errorf("failed to push modified manifest to tag %s: %v", tag, err)
	}

	fmt.Printf("Successfully replaced artifact content for tag: %s\n", tag)
	fmt.Printf("New manifest digest: %s\n", manifestDesc.Digest)

	// Verify the replacement worked by re-reading the manifest
	fmt.Printf("Verifying replacement...\n")
	newManifestDesc, err := targetRepoClient.Resolve(ctx, tag)
	if err != nil {
		fmt.Printf("Warning: failed to verify replacement: %v\n", err)
		return nil
	}
	fmt.Printf("Tag %s now points to manifest: %s\n", tag, newManifestDesc.Digest)

	return nil
}

func pushArtifact(ctx context.Context, repo *remote.Repository, artifactName string, content string) error {
	// Create sample content for the artifact
	contentBytes := []byte(content)

	// Create blob descriptor
	blobDesc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    ocidigest.FromBytes(contentBytes),
		Size:      int64(len(contentBytes)),
	}

	// Push the blob
	err := repo.Push(ctx, blobDesc, strings.NewReader(content))
	if err != nil {
		fmt.Printf("Failed to push blob: %v\n", err)

		return err
	}

	// Create manifest with annotations
	annotations := map[string]string{
		"org.opencontainers.artifact.created":     "2024-01-01T00:00:00Z",
		"org.opencontainers.artifact.description": "Sample artifact: " + artifactName,
	}

	// Pack manifest
	manifestDesc, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ManifestAnnotations: annotations,
			Layers: []ocispec.Descriptor{
				blobDesc,
			},
		},
	)
	if err != nil {
		fmt.Printf("Failed to pack manifest: %v\n", err)

		return err
	}

	// Tag the manifest
	_, err = oras.Tag(ctx, repo, manifestDesc.Digest.String(), artifactName)
	if err != nil {
		fmt.Printf("Failed to tag manifest: %v\n", err)

		return err
	}

	fmt.Printf("Pushed artifact: %s\n", artifactName)

	return nil
}

func verifyArtifact(ctx context.Context, repo *remote.Repository, artifactName string) (bool, error) {
	// Try to resolve the tag to verify it exists
	_, err := repo.Resolve(ctx, artifactName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Printf("Artifact not found: %s\n", artifactName)

			return false, nil
		}

		return false, err
	}

	// If we got a descriptor, the artifact exists
	fmt.Printf("Found artifact: %s\n", artifactName)

	return true, nil
}
