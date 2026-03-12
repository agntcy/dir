// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"path/filepath"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/ociserver"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func (s *store) Server() (http.Handler, error) {
	logger.Info("creating OCI registry server", "address", s.config.RegistryAddress)

	// Create client
	reg, err := remote.NewRegistry(s.config.RegistryAddress)
	if err != nil {
		logger.Error("failed to create registry", "address", s.config.RegistryAddress, "error", err)
		return nil, fmt.Errorf("failed to create registry: %v", err)
	}

	// Configure registry client
	reg.RepositoryOptions.PlainHTTP = s.config.Insecure
	reg.RepositoryOptions.Client = &auth.Client{
		Client: retry.DefaultClient,
		Header: http.Header{
			"User-Agent": {"dir-client"},
		},
		Cache: auth.DefaultCache,
		Credential: auth.StaticCredential(
			s.config.RegistryAddress,
			auth.Credential{
				Username:     s.config.Username,
				Password:     s.config.Password,
				RefreshToken: s.config.RefreshToken,
				AccessToken:  s.config.AccessToken,
			},
		),
	}

	// Create server backend
	backend := &server{repo: reg}

	// Create registry server wrapped with auto cross-mount support
	return ociserver.New(backend, &ociserver.Options{}), nil
}

type server struct {
	ociregistry.Interface // conform to the interface, but delegate to the repo

	repo *remote.Registry
}

// errStopIteration is a sentinel returned from ORAS pagination callbacks to stop
// early when the iterator consumer (ociserver) has signalled it wants no more items.
var errStopIteration = errors.New("stop iteration")

// DeleteBlob implements [ociregistry.Interface].
func (s *server) DeleteBlob(ctx context.Context, repo string, digest ociregistry.Digest) error {
	logger.InfoContext(ctx, "deleting blob", "repo", repo, "digest", digest)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for DeleteBlob", "repo", repo, "digest", digest, "error", err)
		return fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	if err := r.Blobs().Delete(ctx, v1.Descriptor{Digest: digest}); err != nil {
		logger.ErrorContext(ctx, "failed to delete blob", "repo", repo, "digest", digest, "error", err)
		return fmt.Errorf("%w: %v", ociregistry.ErrBlobUnknown, err)
	}

	logger.InfoContext(ctx, "blob deleted", "repo", repo, "digest", digest)
	return nil
}

// DeleteManifest implements [ociregistry.Interface].
func (s *server) DeleteManifest(ctx context.Context, repo string, digest ociregistry.Digest) error {
	logger.InfoContext(ctx, "deleting manifest", "repo", repo, "digest", digest)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for DeleteManifest", "repo", repo, "digest", digest, "error", err)
		return fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	if err := r.Manifests().Delete(ctx, v1.Descriptor{Digest: digest}); err != nil {
		logger.ErrorContext(ctx, "failed to delete manifest", "repo", repo, "digest", digest, "error", err)
		return fmt.Errorf("%w: %v", ociregistry.ErrManifestUnknown, err)
	}

	logger.InfoContext(ctx, "manifest deleted", "repo", repo, "digest", digest)
	return nil
}

// DeleteTag implements [ociregistry.Interface].
func (s *server) DeleteTag(ctx context.Context, repo string, name string) error {
	logger.InfoContext(ctx, "deleting tag", "repo", repo, "tag", name)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for DeleteTag", "repo", repo, "tag", name, "error", err)
		return fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	// Resolve the tag to its manifest descriptor so we can delete by digest.
	desc, _, err := r.Manifests().FetchReference(ctx, name)
	if err != nil {
		logger.ErrorContext(ctx, "failed to resolve tag for DeleteTag", "repo", repo, "tag", name, "error", err)
		return fmt.Errorf("%w: %v", ociregistry.ErrManifestUnknown, err)
	}

	if err := r.Manifests().Delete(ctx, desc); err != nil {
		logger.ErrorContext(ctx, "failed to delete manifest for tag", "repo", repo, "tag", name, "digest", desc.Digest, "error", err)
		return fmt.Errorf("%w: %v", ociregistry.ErrManifestUnknown, err)
	}

	logger.InfoContext(ctx, "tag deleted", "repo", repo, "tag", name, "digest", desc.Digest)
	return nil
}

// GetBlob implements [ociregistry.Interface].
func (s *server) GetBlob(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.BlobReader, error) {
	logger.InfoContext(ctx, "getting blob", "repo", repo, "digest", digest)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for GetBlob", "repo", repo, "digest", digest, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch blob reference", "repo", repo, "digest", digest, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrBlobUnknown, err)
	}

	logger.InfoContext(ctx, "blob fetched", "repo", repo, "digest", digest, "media_type", desc.MediaType, "size", desc.Size)
	return newBlobReader(rd, desc), nil
}

// GetBlobRange implements [ociregistry.Interface].
func (s *server) GetBlobRange(ctx context.Context, repo string, digest ociregistry.Digest, offset0 int64, offset1 int64) (ociregistry.BlobReader, error) {
	logger.InfoContext(ctx, "getting blob range", "repo", repo, "digest", digest, "offset0", offset0, "offset1", offset1)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for GetBlobRange", "repo", repo, "digest", digest, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch blob for range", "repo", repo, "digest", digest, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrBlobUnknown, err)
	}

	// Skip to offset0.
	if offset0 > 0 {
		if _, err := io.CopyN(io.Discard, rd, offset0); err != nil {
			rd.Close()
			logger.ErrorContext(ctx, "failed to seek to range offset", "repo", repo, "digest", digest, "offset0", offset0, "error", err)
			return nil, fmt.Errorf("%w: failed to seek to offset %d: %v", ociregistry.ErrRangeInvalid, offset0, err)
		}
	}

	// Apply upper bound when it falls within the blob.
	if offset1 >= 0 && offset1 < desc.Size {
		logger.InfoContext(ctx, "blob range fetched with upper bound", "repo", repo, "digest", digest, "offset0", offset0, "offset1", offset1)
		return newBlobReader(&limitedReadCloser{Reader: io.LimitReader(rd, offset1-offset0), Closer: rd}, desc), nil
	}

	logger.InfoContext(ctx, "blob range fetched", "repo", repo, "digest", digest, "offset0", offset0, "offset1", offset1)
	return newBlobReader(rd, desc), nil
}

// GetManifest implements [ociregistry.Interface].
func (s *server) GetManifest(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.BlobReader, error) {
	logger.InfoContext(ctx, "getting manifest", "repo", repo, "digest", digest)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for GetManifest", "repo", repo, "digest", digest, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Manifests().FetchReference(ctx, digest.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch manifest reference", "repo", repo, "digest", digest, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrManifestUnknown, err)
	}

	logger.InfoContext(ctx, "manifest fetched", "repo", repo, "digest", digest, "media_type", desc.MediaType, "size", desc.Size)
	return newBlobReader(rd, desc), nil
}

// GetTag implements [ociregistry.Interface].
func (s *server) GetTag(ctx context.Context, repo string, tagName string) (ociregistry.BlobReader, error) {
	logger.InfoContext(ctx, "getting manifest by tag", "repo", repo, "tag", tagName)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for GetTag", "repo", repo, "tag", tagName, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Manifests().FetchReference(ctx, tagName)
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch manifest by tag", "repo", repo, "tag", tagName, "error", err)
		return nil, fmt.Errorf("%w: %v", ociregistry.ErrManifestUnknown, err)
	}

	logger.InfoContext(ctx, "manifest fetched by tag", "repo", repo, "tag", tagName, "digest", desc.Digest, "media_type", desc.MediaType, "size", desc.Size)
	return newBlobReader(rd, desc), nil
}

// MountBlob implements [ociregistry.Interface].
// ORAS does not expose a native cross-repo mount operation, so this falls back
// to fetching from the source repository and re-pushing to the destination.
func (s *server) MountBlob(ctx context.Context, fromRepo string, toRepo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	logger.InfoContext(ctx, "mounting blob", "from_repo", fromRepo, "to_repo", toRepo, "digest", digest)

	src, err := s.repo.Repository(ctx, fromRepo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get source repository for MountBlob", "from_repo", fromRepo, "digest", digest, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := src.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to fetch blob from source for MountBlob", "from_repo", fromRepo, "digest", digest, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrBlobUnknown, err)
	}
	defer rd.Close()

	dst, err := s.repo.Repository(ctx, toRepo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get destination repository for MountBlob", "to_repo", toRepo, "digest", digest, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	if err := dst.Blobs().Push(ctx, desc, rd); err != nil {
		logger.ErrorContext(ctx, "failed to push blob to destination for MountBlob", "to_repo", toRepo, "digest", digest, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrBlobUploadInvalid, err)
	}

	logger.InfoContext(ctx, "blob mounted", "from_repo", fromRepo, "to_repo", toRepo, "digest", desc.Digest, "size", desc.Size)
	return desc, nil
}

// PushBlob implements [ociregistry.Interface].
func (s *server) PushBlob(ctx context.Context, repo string, desc ociregistry.Descriptor, reader io.Reader) (ociregistry.Descriptor, error) {
	logger.InfoContext(ctx, "pushing blob", "repo", repo, "digest", desc.Digest, "media_type", desc.MediaType, "size", desc.Size)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for PushBlob", "repo", repo, "digest", desc.Digest, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	err = r.Blobs().Push(ctx, desc, reader)
	if err != nil {
		logger.ErrorContext(ctx, "failed to push blob", "repo", repo, "digest", desc.Digest, "media_type", desc.MediaType, "size", desc.Size, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrBlobUploadInvalid, err)
	}

	logger.InfoContext(ctx, "blob pushed", "repo", repo, "digest", desc.Digest, "media_type", desc.MediaType, "size", desc.Size)
	return desc, nil
}

// PushBlobChunked implements [ociregistry.Interface].
// A temp file named after the upload ID is created in os.TempDir().
// Data is appended across PATCH requests; the remote push happens atomically on Commit.
func (s *server) PushBlobChunked(ctx context.Context, repo string, chunkSize int) (ociregistry.BlobWriter, error) {
	logger.InfoContext(ctx, "starting chunked blob upload", "repo", repo)

	id := uuid.New().String()
	tmpPath := filepath.Join(os.TempDir(), id)

	f, err := os.Create(tmpPath)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create temp file for chunked upload", "repo", repo, "upload_id", id, "error", err)
		return nil, fmt.Errorf("%w: failed to create temp file: %v", ociregistry.ErrBlobUploadInvalid, err)
	}

	logger.InfoContext(ctx, "chunked blob upload session started", "repo", repo, "upload_id", id)
	return &blobWriter{id: id, tmpPath: tmpPath, file: f, ctx: ctx, srv: s, repo: repo, chunkSize: chunkSize}, nil
}

// PushBlobChunkedResume implements [ociregistry.Interface].
// The upload session is identified by the temp file at os.TempDir()/<id>.
// offset > 0: truncate-and-seek to that position (chunk retry/resume).
// offset == 0 or offset == -1: seek to end (append or info-query).
func (s *server) PushBlobChunkedResume(ctx context.Context, repo string, id string, offset int64, chunkSize int) (ociregistry.BlobWriter, error) {
	logger.InfoContext(ctx, "resuming chunked blob upload", "repo", repo, "upload_id", id, "offset", offset)

	tmpPath := filepath.Join(os.TempDir(), id)

	f, err := os.OpenFile(tmpPath, os.O_RDWR, 0o600)
	if err != nil {
		if os.IsNotExist(err) {
			logger.ErrorContext(ctx, "upload session not found", "repo", repo, "upload_id", id)
			return nil, fmt.Errorf("%w: upload session %q not found", ociregistry.ErrBlobUploadUnknown, id)
		}
		logger.ErrorContext(ctx, "failed to open upload session", "repo", repo, "upload_id", id, "error", err)
		return nil, fmt.Errorf("%w: failed to open upload session: %v", ociregistry.ErrBlobUploadInvalid, err)
	}

	if offset == -1 {
		// GET info query: seek to end to report current upload progress.
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, fmt.Errorf("%w: failed to seek to end: %v", ociregistry.ErrBlobUploadInvalid, err)
		}
	} else if offset == 0 && chunkSize == 0 {
		// Final PUT with no body/range (case 3 in ociserver): seek to end so that
		// whatever was accumulated by prior PATCH requests is committed as-is.
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, fmt.Errorf("%w: failed to seek to end: %v", ociregistry.ErrBlobUploadInvalid, err)
		}
	} else {
		// PATCH or PUT with content: the Content-Range start must exactly match the
		// number of bytes already accumulated. Any mismatch (out-of-order or retried
		// chunk) is rejected with 416 Range Not Satisfiable.
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("%w: failed to stat upload session: %v", ociregistry.ErrBlobUploadInvalid, err)
		}
		if offset != info.Size() {
			f.Close()
			return nil, fmt.Errorf("%w: expected offset %d, got %d", ociregistry.ErrRangeInvalid, info.Size(), offset)
		}
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, fmt.Errorf("%w: failed to seek to end: %v", ociregistry.ErrBlobUploadInvalid, err)
		}
	}

	pos, _ := f.Seek(0, io.SeekCurrent)
	logger.InfoContext(ctx, "chunked blob upload session resumed", "repo", repo, "upload_id", id, "position", pos)
	return &blobWriter{id: id, tmpPath: tmpPath, file: f, ctx: ctx, srv: s, repo: repo}, nil
}

// PushManifest implements [ociregistry.Interface].
func (s *server) PushManifest(ctx context.Context, repo string, tag string, contents []byte, mediaType string) (ociregistry.Descriptor, error) {
	logger.InfoContext(ctx, "pushing manifest", "repo", repo, "tag", tag, "media_type", mediaType, "content_size", len(contents))

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for PushManifest", "repo", repo, "tag", tag, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	// Build the descriptor from the manifest content.
	desc := v1.Descriptor{
		MediaType: mediaType,
		Digest:    digest.Canonical.FromBytes(contents),
		Size:      int64(len(contents)),
	}
	// When tag is empty the caller is pushing by digest only; use Push so that
	// ORAS sends PUT /v2/<repo>/manifests/<digest> with a well-formed URL.
	// PushReference with an empty string would produce a malformed URL.
	if tag == "" {
		err = r.Manifests().Push(ctx, desc, bytes.NewReader(contents))
	} else {
		err = r.Manifests().PushReference(ctx, desc, bytes.NewReader(contents), tag)
	}
	if err != nil {
		logger.ErrorContext(ctx, "failed to push manifest", "repo", repo, "tag", tag, "digest", desc.Digest, "media_type", mediaType, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrManifestInvalid, err)
	}

	logger.InfoContext(ctx, "manifest pushed", "repo", repo, "tag", tag, "digest", desc.Digest, "media_type", desc.MediaType, "size", desc.Size)
	return desc, nil
}

// Referrers implements [ociregistry.Interface].
func (s *server) Referrers(ctx context.Context, repo string, digest ociregistry.Digest, artifactType string) iter.Seq2[ociregistry.Descriptor, error] {
	return func(yield func(ociregistry.Descriptor, error) bool) {
		logger.InfoContext(ctx, "listing referrers", "repo", repo, "digest", digest, "artifact_type", artifactType)

		r, err := s.repo.Repository(ctx, repo)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get repository for Referrers", "repo", repo, "digest", digest, "error", err)
			yield(ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err))
			return
		}

		count := 0
		err = r.Referrers(ctx, v1.Descriptor{Digest: digest}, artifactType, func(desc []v1.Descriptor) error {
			count += len(desc)
			for _, d := range desc {
				logger.InfoContext(ctx, "referrer found", "repo", repo, "digest", digest, "referrer_digest", d.Digest, "referrer_media_type", d.MediaType, "referrer_artifact_type", d.ArtifactType)
				if !yield(d, nil) {
					return errStopIteration
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, errStopIteration) {
			logger.ErrorContext(ctx, "failed to list referrers", "repo", repo, "digest", digest, "artifact_type", artifactType, "error", err)
			yield(ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrManifestUnknown, err))
			return
		}

		logger.InfoContext(ctx, "referrers listed", "repo", repo, "digest", digest, "artifact_type", artifactType, "count", count)
	}
}

// Repositories implements [ociregistry.Interface].
func (s *server) Repositories(ctx context.Context, startAfter string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		logger.InfoContext(ctx, "listing repositories", "start_after", startAfter)

		count := 0
		err := s.repo.Repositories(ctx, startAfter, func(name []string) error {
			count += len(name)
			for _, n := range name {
				logger.InfoContext(ctx, "repository found", "name", n)
				if !yield(n, nil) {
					return errStopIteration
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, errStopIteration) {
			logger.ErrorContext(ctx, "failed to list repositories", "start_after", startAfter, "error", err)
			yield("", fmt.Errorf("%w: %v", ociregistry.ErrUnsupported, err))
			return
		}

		logger.InfoContext(ctx, "repositories listed", "start_after", startAfter, "count", count)
	}
}

// ResolveBlob implements [ociregistry.Interface].
func (s *server) ResolveBlob(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	logger.InfoContext(ctx, "resolving blob", "repo", repo, "digest", digest)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for ResolveBlob", "repo", repo, "digest", digest, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	desc, _, err := r.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		logger.ErrorContext(ctx, "failed to resolve blob reference", "repo", repo, "digest", digest, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrBlobUnknown, err)
	}

	logger.InfoContext(ctx, "blob resolved", "repo", repo, "digest", desc.Digest, "media_type", desc.MediaType, "size", desc.Size)
	return desc, nil
}

// ResolveManifest implements [ociregistry.Interface].
func (s *server) ResolveManifest(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	logger.InfoContext(ctx, "resolving manifest", "repo", repo, "digest", digest)
	return s.ResolveTag(ctx, repo, digest.String())
}

// ResolveTag implements [ociregistry.Interface].
func (s *server) ResolveTag(ctx context.Context, repo string, tagName string) (ociregistry.Descriptor, error) {
	logger.InfoContext(ctx, "resolving tag", "repo", repo, "tag", tagName)

	r, err := s.repo.Repository(ctx, repo)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get repository for ResolveTag", "repo", repo, "tag", tagName, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	desc, _, err := r.Manifests().FetchReference(ctx, tagName)
	if err != nil {
		logger.ErrorContext(ctx, "failed to resolve tag", "repo", repo, "tag", tagName, "error", err)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrManifestUnknown, err)
	}

	logger.InfoContext(ctx, "tag resolved", "repo", repo, "tag", tagName, "digest", desc.Digest, "media_type", desc.MediaType, "size", desc.Size)
	return desc, nil
}

// Tags implements [ociregistry.Interface].
func (s *server) Tags(ctx context.Context, repo string, startAfter string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		logger.InfoContext(ctx, "listing tags", "repo", repo, "start_after", startAfter)

		r, err := s.repo.Repository(ctx, repo)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get repository for Tags", "repo", repo, "start_after", startAfter, "error", err)
			yield("", fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err))
			return
		}

		count := 0
		err = r.Tags(ctx, startAfter, func(name []string) error {
			count += len(name)
			for _, n := range name {
				logger.InfoContext(ctx, "tag found", "repo", repo, "tag", n)
				if !yield(n, nil) {
					return errStopIteration
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, errStopIteration) {
			logger.ErrorContext(ctx, "failed to list tags", "repo", repo, "start_after", startAfter, "error", err)
			yield("", fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err))
			return
		}

		logger.InfoContext(ctx, "tags listed", "repo", repo, "start_after", startAfter, "count", count)
	}
}

type blobReader struct {
	io.ReadCloser
	desc v1.Descriptor
}

// Descriptor implements [ociregistry.BlobReader].
func (b *blobReader) Descriptor() ociregistry.Descriptor {
	return b.desc
}

func newBlobReader(r io.ReadCloser, desc v1.Descriptor) ociregistry.BlobReader {
	return &blobReader{
		ReadCloser: r,
		desc:       desc,
	}
}

// limitedReadCloser combines an io.LimitReader with the original closer.
type limitedReadCloser struct {
	io.Reader
	io.Closer
}

// blobWriter streams a chunked blob upload through a temp file.
// Each HTTP request (PATCH/PUT) opens a new blobWriter via PushBlobChunkedResume;
// Close() releases the file handle without deleting the file so the next request
// can resume. Commit() pushes the completed file to the remote registry and
// removes the temp file. Cancel() removes the temp file immediately.
type blobWriter struct {
	id        string
	tmpPath   string
	file      *os.File        // nil after Close; reopened by PushBlobChunkedResume
	ctx       context.Context //nolint:containedctx — refreshed each resume call
	chunkSize int
	srv       *server
	repo      string
}

func (w *blobWriter) Write(p []byte) (int, error) {
	return w.file.Write(p)
}

// Close releases the OS file handle. The temp file is kept on disk so that the
// next PushBlobChunkedResume call can reopen and append to it.
func (w *blobWriter) Close() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// Size returns the current write position, which equals the total bytes written.
func (w *blobWriter) Size() int64 {
	if w.file == nil {
		info, err := os.Stat(w.tmpPath)
		if err != nil {
			return 0
		}
		return info.Size()
	}
	pos, err := w.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0
	}
	return pos
}

func (w *blobWriter) ChunkSize() int { return w.chunkSize }

func (w *blobWriter) ID() string { return w.id }

// Commit closes the temp file, pushes its contents to the remote registry,
// then removes the temp file. Uses the context from the last PushBlobChunkedResume
// call (i.e. the live PUT request context), not the original POST context.
func (w *blobWriter) Commit(d ociregistry.Digest) (ociregistry.Descriptor, error) {
	if err := w.Close(); err != nil {
		os.Remove(w.tmpPath)
		return ociregistry.Descriptor{}, fmt.Errorf("%w: failed to close temp file: %v", ociregistry.ErrBlobUploadInvalid, err)
	}
	defer os.Remove(w.tmpPath)

	info, err := os.Stat(w.tmpPath)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: failed to stat temp file: %v", ociregistry.ErrBlobUploadInvalid, err)
	}

	f, err := os.Open(w.tmpPath)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: failed to open temp file for commit: %v", ociregistry.ErrBlobUploadInvalid, err)
	}
	defer f.Close()

	desc := v1.Descriptor{
		Digest: d,
		Size:   info.Size(),
	}

	r, err := w.srv.repo.Repository(w.ctx, w.repo)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrNameUnknown, err)
	}

	if err := r.Blobs().Push(w.ctx, desc, f); err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %v", ociregistry.ErrBlobUploadInvalid, err)
	}

	logger.InfoContext(w.ctx, "blob committed from chunked upload", "repo", w.repo, "upload_id", w.id, "digest", d, "size", info.Size())
	return desc, nil
}

// Cancel discards the temp file immediately.
func (w *blobWriter) Cancel() error {
	w.Close()            //nolint:errcheck
	os.Remove(w.tmpPath) //nolint:errcheck
	return nil
}
