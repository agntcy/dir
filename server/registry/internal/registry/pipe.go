// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cuelabs.dev/go/oci/ociregistry"
	"github.com/google/uuid"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// errSessionNotFound is returned when attempting to resume a chunked upload session that does not exist.
var errSessionNotFound = fmt.Errorf("upload session not found")

// reader implements ociregistry.BlobReader by wrapping an io.ReadCloser and its associated descriptor.
type reader struct {
	io.Reader
	close func() error
	desc  ociregistry.Descriptor
}

// NewReader returns an implementation of ociregistry.BlobReader.
func NewReader(rd io.ReadCloser, desc ociregistry.Descriptor) ociregistry.BlobReader {
	return &reader{Reader: rd, close: rd.Close, desc: desc}
}

// NewReader returns an implementation of ociregistry.BlobReader
// with custom close function.
func NewReadCloser(rd io.Reader, closer func() error, desc ociregistry.Descriptor) ociregistry.BlobReader {
	return &reader{Reader: rd, close: closer, desc: desc}
}

// Close delegates to the underlying reader's Close method.
func (r *reader) Close() error {
	if r.close != nil {
		return r.close()
	}

	return nil
}

// Descriptor returns the OCI descriptor associated with this reader.
func (r *reader) Descriptor() ociregistry.Descriptor {
	return r.desc
}

// writer implements ociregistry.BlobWriter using a temp file to accumulate written data.
type writer struct {
	id     string
	file   *os.File
	size   int64
	commit func(v1.Descriptor, io.Reader) error
}

// NewWriter returns an implementation of ociregistry.BlobWriter backed by a new
// temp file. The caller's commit function is invoked when Commit is called.
func NewWriter(commit func(v1.Descriptor, io.Reader) error) (ociregistry.BlobWriter, error) {
	id := uuid.New().String()

	// Create a temp session file
	file, err := os.Create(filepath.Join(os.TempDir(), id)) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to create request pipe: %w", err)
	}

	return &writer{
		id:     id,
		file:   file,
		size:   0,
		commit: commit,
	}, nil
}

// NewWriterForID resumes an existing chunked upload session identified by id.
// The returned writer's Size() reflects bytes already accumulated before this
// call. The caller's commit function is invoked when Commit is called.
func OpenWriter(id string, commit func(v1.Descriptor, io.Reader) error) (ociregistry.BlobWriter, error) {
	// Validate session ID format before attempting to open the file.
	// This avoids leaking file existence information and location escapes.
	if uuid.Validate(id) != nil {
		return nil, fmt.Errorf("invalid upload session ID: %s", id)
	}

	// Open existing session file.
	file, err := os.OpenFile(filepath.Join(os.TempDir(), id), os.O_RDWR|os.O_APPEND, 0o600) //nolint:mnd
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errSessionNotFound
		}

		return nil, fmt.Errorf("failed to open upload session: %w", err)
	}

	// Stat the file to get current size.
	// Required for chunked uploads for resume and offset validation.
	info, err := file.Stat()
	if err != nil {
		file.Close()

		return nil, fmt.Errorf("failed to stat upload session: %w", err)
	}

	return &writer{
		id:     id,
		file:   file,
		size:   info.Size(),
		commit: commit,
	}, nil
}

// Write appends p to the file and updates the running byte count.
func (b *writer) Write(p []byte) (int, error) {
	n, err := b.file.Write(p)
	b.size += int64(n)

	return n, err //nolint:wrapcheck
}

func (b *writer) Close() error {
	return b.file.Close() //nolint:wrapcheck
}

// Try to remove the session file on cancel, but ignore any errors since the file may
// have already been removed by a previous cancel.
func (b *writer) Cancel() error {
	_ = os.Remove(b.file.Name())

	return nil
}

// Size returns the total number of bytes accumulated in this upload session.
// It uses the tracked count to avoid stating the file on every call which can be expensive.
func (b *writer) Size() int64 {
	return b.size
}

// ChunkSize returns the chunk size to use for chunked uploads.
// This is a hint to the caller and does not need to be strictly enforced.
func (b *writer) ChunkSize() int {
	return 16 * 1024 //nolint:mnd
}

// ID returns the unique identifier for this upload session.
// It is used by the caller to resume chunked uploads.
func (b *writer) ID() string {
	return b.id
}

// Commit finalises the upload: seeks to the start of pipe so the
// commit function can read all accumulated data, then invokes it.
func (b *writer) Commit(dig ociregistry.Digest) (ociregistry.Descriptor, error) {
	desc := ociregistry.Descriptor{
		MediaType: "application/octet-stream",
		Size:      b.size,
		Digest:    dig,
	}

	// Seek to start so the commit function can read all accumulated data.
	if _, err := b.file.Seek(0, io.SeekStart); err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("failed to seek to start: %w", err)
	}

	// Commit the blob using the caller's commit function.
	// It is callers's responsibility to verify the content digest.
	if err := b.commit(desc, b.file); err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("failed to commit blob: %w", err)
	}

	return desc, nil
}
