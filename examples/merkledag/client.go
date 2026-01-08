package main

import (
	"fmt"
	"os"

	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	registryAddress = "localhost:5000"
	repositoryName  = "my-merkledag-repo"

	localRepoPath = "./store"
)

// NewLocalClient creates a new local ORAS repository client.
func NewLocalClient() (*oci.Store, error) {
	// remove existing store for clean state (for demo purposes)
	if err := os.RemoveAll(localRepoPath); err != nil {
		return nil, fmt.Errorf("failed to remove existing local repo: %w", err)
	}

	repo, err := oci.New(localRepoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to local repo: %w", err)
	}

	return repo, nil
}

// NewRemoteClient creates a new ORAS repository client configured with authentication.
func NewRemoteClient() (*remote.Repository, error) {
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", registryAddress, repositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// Configure repository
	repo.PlainHTTP = true
	repo.Client = &auth.Client{Client: retry.DefaultClient}

	return repo, nil
}
