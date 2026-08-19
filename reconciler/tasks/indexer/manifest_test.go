// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeManifestReader serves a single manifest for any tag, mimicking the
// Resolve/Fetch pair implemented by local and remote OCI targets.
type fakeManifestReader struct {
	manifest   ocispec.Manifest
	mediaType  string
	resolveErr error
	fetchErr   error
}

func (f *fakeManifestReader) descriptor() (ocispec.Descriptor, []byte) {
	data, err := json.Marshal(f.manifest)
	if err != nil {
		panic(err)
	}

	mediaType := f.mediaType
	if mediaType == "" {
		mediaType = ocispec.MediaTypeImageManifest
	}

	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    ocidigest.FromBytes(data),
		Size:      int64(len(data)),
	}, data
}

func (f *fakeManifestReader) Resolve(_ context.Context, _ string) (ocispec.Descriptor, error) {
	if f.resolveErr != nil {
		return ocispec.Descriptor{}, f.resolveErr
	}

	desc, _ := f.descriptor()

	return desc, nil
}

func (f *fakeManifestReader) Fetch(_ context.Context, _ ocispec.Descriptor) (io.ReadCloser, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}

	_, data := f.descriptor()

	return io.NopCloser(bytes.NewReader(data)), nil
}

func TestIsReferrerManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest ocispec.Manifest
		want     bool
	}{
		{
			name:     "record manifest",
			manifest: ocispec.Manifest{Annotations: map[string]string{"org.agntcy.dir/type": "record"}},
			want:     false,
		},
		{
			name:     "referrer with subject",
			manifest: ocispec.Manifest{Subject: &ocispec.Descriptor{Digest: "sha256:abc"}},
			want:     true,
		},
		{
			name: "referrer with type annotation only",
			manifest: ocispec.Manifest{
				Annotations: map[string]string{corev1.ReferrerTypeAnnotationKey: corev1.ScanReportReferrerType},
			},
			want: true,
		},
		{
			name:     "manifest without annotations",
			manifest: ocispec.Manifest{},
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isReferrerManifest(&tt.manifest))
		})
	}
}

func TestIsReferrerTag(t *testing.T) {
	recordManifest := ocispec.Manifest{Annotations: map[string]string{"org.agntcy.dir/type": "record"}}
	referrerManifest := ocispec.Manifest{
		Subject:     &ocispec.Descriptor{Digest: "sha256:abc"},
		Annotations: map[string]string{corev1.ReferrerTypeAnnotationKey: corev1.SignatureReferrerType},
	}

	tests := []struct {
		name    string
		reader  manifestReader
		want    bool
		wantErr bool
	}{
		{
			name:   "no manifest reader available",
			reader: nil,
			want:   false,
		},
		{
			name:   "record",
			reader: &fakeManifestReader{manifest: recordManifest},
			want:   false,
		},
		{
			name:   "referrer",
			reader: &fakeManifestReader{manifest: referrerManifest},
			want:   true,
		},
		{
			name:   "non manifest media type",
			reader: &fakeManifestReader{manifest: referrerManifest, mediaType: ocispec.MediaTypeImageIndex},
			want:   false,
		},
		{
			name:    "resolve fails",
			reader:  &fakeManifestReader{manifest: recordManifest, resolveErr: errors.New("not found")},
			wantErr: true,
		},
		{
			name:    "fetch fails",
			reader:  &fakeManifestReader{manifest: recordManifest, fetchErr: errors.New("connection reset")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{manifests: tt.reader}

			got, err := task.isReferrerTag(t.Context(), "baeareihwzcbx3ymdhvbgpzialtketsctkkikubksllegiqyhrouwlrdnz4")
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsReferrerTag_RejectsOversizedManifest(t *testing.T) {
	reader := &fakeManifestReader{manifest: ocispec.Manifest{}}
	desc, _ := reader.descriptor()
	desc.Size = maxManifestSize + 1

	task := &Task{manifests: &oversizedManifestReader{desc: desc}}

	_, err := task.isReferrerTag(t.Context(), "tag")
	require.ErrorContains(t, err, "too large")
}

// oversizedManifestReader reports a manifest larger than the read limit.
type oversizedManifestReader struct {
	desc ocispec.Descriptor
}

func (o *oversizedManifestReader) Resolve(_ context.Context, _ string) (ocispec.Descriptor, error) {
	return o.desc, nil
}

func (o *oversizedManifestReader) Fetch(_ context.Context, _ ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, errors.New("should not be fetched")
}
