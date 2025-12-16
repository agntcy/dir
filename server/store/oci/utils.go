// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func stringPtr(s string) *string {
	return &s
}

// NewORASRepository creates a new ORAS repository client configured with authentication.
func NewORASRepository(cfg ociconfig.Config) (*remote.Repository, error) {
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", cfg.RegistryAddress, cfg.RepositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// Configure repository
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

	return repo, nil
}

func (s *store) descLookupMany(ctx context.Context, cids []*storev1.ObjectRef) (map[string]ocispec.Descriptor, error) {
	results := make(map[string]ocispec.Descriptor)
	resultsMu := sync.Mutex{}

	// Use errgroup for concurrent fetching with controlled parallelism
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // Limit concurrent requests

	for _, cid := range cids {
		cid := cid // Capture loop variable
		g.Go(func() error {
			// Convert CID to digest
			digest, err := corev1.ConvertCIDToDigest(cid.GetCid())
			if err != nil {
				return fmt.Errorf("invalid CID %s: %w", cid.GetCid(), err)
			}

			// Fetch content
			descriptor, err := s.repo.Resolve(ctx, digest.String())
			if err != nil {
				return fmt.Errorf("failed to fetch %s: %w", cid, err)
			}

			// Store result
			resultsMu.Lock()
			results[cid.GetCid()] = descriptor
			resultsMu.Unlock()

			return nil
		})
	}

	// Wait for all fetches to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// pushOrSkip pushes data to the OCI registry if it does not already exist
func (s *store) pushOrSkip(ctx context.Context, reader io.Reader, mediaType string) (ocispec.Descriptor, error) {
	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to read data from reader: %w", err)
	}

	// Compute digest
	dgst := digest.FromBytes(data)

	// Check if data already exists
	exists, err := s.repo.Exists(ctx, ocispec.Descriptor{Digest: dgst})
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to check blob existence: %w", err)
	}
	if !exists {
		_, err := oras.PushBytes(ctx, s.repo, mediaType, data)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to push object bytes: %w", err)
		}
	}

	return ocispec.Descriptor{Digest: dgst, MediaType: mediaType, Size: int64(len(data))}, nil
}

// tagWithRetry attempts to tag a manifest with exponential backoff retry logic.
// This is necessary because under concurrent load, oras.PackManifest may push the manifest
// to the registry, but it might not be immediately available when oras.Tag is called.
func (s *store) tagWithRetry(ctx context.Context, manifestDigest, tag string) error {
	var lastErr error

	delay := initialRetryDelay

	for attempt := 0; attempt <= maxTagRetries; attempt++ {
		if attempt > 0 {
			logger.Debug("Retrying Tag operation",
				"attempt", attempt,
				"max_retries", maxTagRetries,
				"delay", delay,
				"manifest_digest", manifestDigest,
				"tag", tag)

			// Wait before retrying
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during tag retry: %w", ctx.Err())
			case <-time.After(delay):
			}

			// Exponential backoff with cap
			delay *= 2
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
		}

		// Attempt to tag the manifest
		_, err := oras.Tag(ctx, s.repo, manifestDigest, tag)
		if err == nil {
			if attempt > 0 {
				logger.Info("Tag operation succeeded after retry",
					"attempt", attempt,
					"manifest_digest", manifestDigest,
					"tag", tag)
			}

			return nil
		}

		lastErr = err

		// Log the retryable error
		logger.Debug("Tag operation failed with retryable error",
			"attempt", attempt,
			"error", err,
			"manifest_digest", manifestDigest,
			"tag", tag)
	}

	// All retries exhausted
	logger.Warn("Tag operation failed after all retries",
		"max_retries", maxTagRetries,
		"last_error", lastErr,
		"manifest_digest", manifestDigest,
		"tag", tag)

	return lastErr
}
