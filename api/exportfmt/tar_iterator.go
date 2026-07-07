// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
)

type TarIteratorOptions struct {
	typeflags []byte
}

type TarIteratorOption func(*TarIteratorOptions)

func WithTypeflag(typeflag byte) TarIteratorOption {
	return func(opts *TarIteratorOptions) {
		opts.typeflags = append(opts.typeflags, typeflag)
	}
}

type TarIterator iter.Seq2[*TarEntry, error]

type TarEntry struct {
	header  *tar.Header
	payload []byte
}

func NewTarIterator(raw []byte, opts ...TarIteratorOption) (TarIterator, error) {
	var o TarIteratorOptions
	for _, opt := range opts {
		opt(&o)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip: %w", err)
	}

	tarReader := tar.NewReader(gzipReader)

	return func(yield func(*TarEntry, error) bool) {
		defer gzipReader.Close()

		for {
			header, err := tarReader.Next()
			if errors.Is(err, io.EOF) {
				return
			}

			if err != nil {
				yield(nil, err)

				return
			}

			entry, err := readTarEntry(tarReader, header)
			if err != nil {
				yield(nil, err)

				return
			}

			if o.typeflags != nil && !slices.Contains(o.typeflags, header.Typeflag) {
				continue
			}

			if !yield(entry, nil) {
				return
			}
		}
	}, nil
}

func readTarEntry(tarReader *tar.Reader, header *tar.Header) (*TarEntry, error) {
	reader := io.Reader(tarReader)
	if header.Size >= 0 {
		reader = io.LimitReader(tarReader, header.Size)
	}

	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read tar entry: %w", err)
	}

	return &TarEntry{
		header:  header,
		payload: payload,
	}, nil
}
