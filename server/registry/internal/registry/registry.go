// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/ociserver"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry"
)

// NewServer returns an HTTP server that implements OCI Distribution API v1.1
// and serves content as a proxy using ORAS.
//
// https://github.com/opencontainers/distribution-spec
func NewServer(reg registry.Registry) (http.Handler, error) {
	return ociserver.New(&server{reg: reg}, &ociserver.Options{}), nil
}

type server struct {
	// Embed to conform to the private interface.
	// Use registry to implement public interface.
	ociregistry.Interface

	reg registry.Registry
}

// errStopIteration is a sentinel returned from ORAS pagination callbacks to stop
// early when the iterator consumer has signalled it wants no more items.
var errStopIteration = errors.New("stop iteration")

func (s *server) DeleteBlob(ctx context.Context, repo string, digest ociregistry.Digest) error {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	if err := r.Blobs().Delete(ctx, v1.Descriptor{Digest: digest}); err != nil {
		return fmt.Errorf("%w: %w", ociregistry.ErrBlobUnknown, err)
	}

	return nil
}

func (s *server) DeleteManifest(ctx context.Context, repo string, digest ociregistry.Digest) error {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	if err := r.Manifests().Delete(ctx, v1.Descriptor{Digest: digest}); err != nil {
		return fmt.Errorf("%w: %w", ociregistry.ErrManifestUnknown, err)
	}

	return nil
}

func (s *server) DeleteTag(ctx context.Context, repo string, name string) error {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	// Resolve the tag to its manifest descriptor so we can delete by digest.
	desc, err := r.Manifests().Resolve(ctx, name)
	if err != nil {
		return fmt.Errorf("%w: %w", ociregistry.ErrManifestUnknown, err)
	}

	if err := r.Manifests().Delete(ctx, desc); err != nil {
		return fmt.Errorf("%w: %w", ociregistry.ErrManifestUnknown, err)
	}

	return nil
}

func (s *server) GetBlob(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.BlobReader, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrBlobUnknown, err)
	}

	return NewReader(rd, desc), nil
}

func (s *server) GetBlobRange(ctx context.Context, repo string, digest ociregistry.Digest, offsetLow int64, offsetHigh int64) (ociregistry.BlobReader, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrBlobUnknown, err)
	}

	// Validate offsets before consuming the reader.
	if offsetLow < 0 || offsetLow > desc.Size {
		rd.Close()

		return nil, fmt.Errorf("%w: invalid offset %d for blob of size %d", ociregistry.ErrRangeInvalid, offsetLow, desc.Size)
	}

	if offsetHigh >= 0 && (offsetHigh < offsetLow || offsetHigh > desc.Size) {
		rd.Close()

		return nil, fmt.Errorf("%w: invalid offset %d for blob of size %d", ociregistry.ErrRangeInvalid, offsetHigh, desc.Size)
	}

	// Skip to lower offset.
	if offsetLow > 0 {
		if _, err := io.CopyN(io.Discard, rd, offsetLow); err != nil {
			rd.Close()

			return nil, fmt.Errorf("%w: failed to seek to offset %d: %w", ociregistry.ErrRangeInvalid, offsetLow, err)
		}
	}

	// Apply upper bound when it falls within the blob.
	// Return a reader that will close the underlying reader when closed,
	// so that ORAS can signal cancellation by closing the reader.
	if offsetHigh >= 0 && offsetHigh < desc.Size {
		return NewReadCloser(io.LimitReader(rd, offsetLow-offsetHigh), rd.Close, desc), nil
	}

	return NewReader(rd, desc), nil
}

func (s *server) GetManifest(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.BlobReader, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Manifests().FetchReference(ctx, digest.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrManifestUnknown, err)
	}

	return NewReader(rd, desc), nil
}

func (s *server) GetTag(ctx context.Context, repo string, tagName string) (ociregistry.BlobReader, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := r.Manifests().FetchReference(ctx, tagName)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrManifestUnknown, err)
	}

	return NewReader(rd, desc), nil
}

// ORAS does not expose a native cross-repo mount operation, so this falls back
// to fetching from the source repository and re-pushing to the destination.
func (s *server) MountBlob(ctx context.Context, fromRepo string, toRepo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	src, err := s.reg.Repository(ctx, fromRepo)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	desc, rd, err := src.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrBlobUnknown, err)
	}
	defer rd.Close()

	dst, err := s.reg.Repository(ctx, toRepo)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	if err := dst.Blobs().Push(ctx, desc, rd); err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrBlobUploadInvalid, err)
	}

	return desc, nil
}

func (s *server) PushBlob(ctx context.Context, repo string, desc ociregistry.Descriptor, reader io.Reader) (ociregistry.Descriptor, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	err = r.Blobs().Push(ctx, desc, reader)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrBlobUploadInvalid, err)
	}

	return desc, nil
}

func (s *server) PushBlobChunked(ctx context.Context, repo string, chunkSize int) (ociregistry.BlobWriter, error) {
	writer, err := NewWriter(s.blobCommiter(ctx, repo))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ociregistry.ErrBlobUploadInvalid, err)
	}

	return writer, nil
}

func (s *server) PushBlobChunkedResume(ctx context.Context, repo string, id string, offset int64, chunkSize int) (ociregistry.BlobWriter, error) {
	writer, err := OpenWriter(id, s.blobCommiter(ctx, repo))
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			return nil, fmt.Errorf("%w: upload session %q not found", ociregistry.ErrBlobUploadUnknown, id)
		}

		return nil, fmt.Errorf("%w: %w", ociregistry.ErrBlobUploadInvalid, err)
	}

	// Return preemptively for cases when:
	//   offset == -1: GET info query
	//   offset == 0 && chunkSize == 0: closing PUT with no body
	if offset == -1 || (offset == 0 && chunkSize == 0) {
		return writer, nil
	}

	// For all other cases (PATCH or PUT with content) the Content-Range start
	// must exactly equal the number of bytes already accumulated; any mismatch
	// (out-of-order or retried chunk) is rejected with 416 Range Not Satisfiable.
	if offset != writer.Size() {
		writer.Close()

		return nil, fmt.Errorf("%w: expected offset %d, got %d", ociregistry.ErrRangeInvalid, writer.Size(), offset)
	}

	return writer, nil
}

func (s *server) PushManifest(ctx context.Context, repo string, tag string, contents []byte, mediaType string) (ociregistry.Descriptor, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
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
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrManifestInvalid, err)
	}

	return desc, nil
}

func (s *server) Referrers(ctx context.Context, repo string, digest ociregistry.Digest, artifactType string) iter.Seq2[ociregistry.Descriptor, error] {
	return func(yield func(ociregistry.Descriptor, error) bool) {
		r, err := s.reg.Repository(ctx, repo)
		if err != nil {
			yield(ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err))

			return
		}

		err = r.Referrers(ctx, v1.Descriptor{Digest: digest}, artifactType, func(desc []v1.Descriptor) error {
			for _, d := range desc {
				if !yield(d, nil) {
					return errStopIteration
				}
			}

			return nil
		})
		if err != nil && !errors.Is(err, errStopIteration) {
			yield(ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrManifestUnknown, err))

			return
		}
	}
}

func (s *server) Repositories(ctx context.Context, startAfter string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		err := s.reg.Repositories(ctx, startAfter, func(name []string) error {
			for _, n := range name {
				if !yield(n, nil) {
					return errStopIteration
				}
			}

			return nil
		})
		if err != nil && !errors.Is(err, errStopIteration) {
			yield("", fmt.Errorf("%w: %w", ociregistry.ErrUnsupported, err))

			return
		}
	}
}

func (s *server) ResolveBlob(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	desc, err := r.Blobs().Resolve(ctx, digest.String())
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrBlobUnknown, err)
	}

	return desc, nil
}

func (s *server) ResolveManifest(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	return s.ResolveTag(ctx, repo, digest.String())
}

func (s *server) ResolveTag(ctx context.Context, repo string, tagName string) (ociregistry.Descriptor, error) {
	r, err := s.reg.Repository(ctx, repo)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
	}

	desc, err := r.Manifests().Resolve(ctx, tagName)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("%w: %w", ociregistry.ErrManifestUnknown, err)
	}

	return desc, nil
}

func (s *server) Tags(ctx context.Context, repo string, startAfter string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		r, err := s.reg.Repository(ctx, repo)
		if err != nil {
			yield("", fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err))

			return
		}

		err = r.Tags(ctx, startAfter, func(name []string) error {
			for _, n := range name {
				if !yield(n, nil) {
					return errStopIteration
				}
			}

			return nil
		})
		if err != nil && !errors.Is(err, errStopIteration) {
			yield("", fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err))

			return
		}
	}
}

func (s *server) blobCommiter(ctx context.Context, repo string) func(v1.Descriptor, io.Reader) error {
	return func(desc v1.Descriptor, reader io.Reader) error {
		r, err := s.reg.Repository(ctx, repo)
		if err != nil {
			return fmt.Errorf("%w: %w", ociregistry.ErrNameUnknown, err)
		}

		err = r.Blobs().Push(ctx, desc, reader)
		if err != nil {
			return fmt.Errorf("%w: %w", ociregistry.ErrBlobUploadInvalid, err)
		}

		return nil
	}
}
