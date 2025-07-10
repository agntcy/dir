package main

import (
	"context"
	"fmt"
	"strings"

	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"

	"regsync-poc/regsync"
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
		err = pushArtifact(ctx, sourceRepoClient, artifact)
		if err != nil {
			fmt.Printf("failed to push %s: %v\n", artifact, err)
			return
		}
	}

	// Verify artifacts exist in source registry
	for _, artifact := range artifacts {
		_, err := verifyArtifact(ctx, sourceRepoClient, artifact)
		if err != nil {
			fmt.Printf("failed to verify %s in source registry: %v\n", artifact, err)
			return
		}
	}

	// TODO: Sync artifacts to dest registry
	syncType := "regclient"
	switch syncType {
	case "regclient":
		err := regsync.Sync(sourceRegistry, targetRegistry, sourceRepo, targetRepo)
		if err != nil {
			fmt.Printf("failed to sync artifacts: %v\n", err)
		}
		// case "zot":
		// 	sync := zot.Sync()
	}

	// Verify artifacts exist in source registry
	for _, artifact := range artifacts {
		_, err := verifyArtifact(ctx, targetRepoClient, artifact)
		if err != nil {
			fmt.Printf("failed to verify %s in target registry: %v\n", artifact, err)
			return
		}
	}
}

func pushArtifact(ctx context.Context, repo *remote.Repository, artifactName string) error {
	// Create sample content for the artifact
	content := fmt.Sprintf("This is sample content for %s", artifactName)
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
		"org.opencontainers.artifact.description": fmt.Sprintf("Sample artifact: %s", artifactName),
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
