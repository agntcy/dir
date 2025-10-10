// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import corev1 "github.com/agntcy/dir/api/core/v1"

// ErrReceiverFn is a callback function that processes errors.
type ErrReceiverFn func(error) error

// RefReceiverFn is a callback function that processes records refs.
// It receives the record reference and any error that occurred.
type RefReceiverFn func(*corev1.RecordRef, error) error

// DataReceiverFn is a callback function that processes records, and returns processed data.
// It receives the record reference, processed data, and any error that occurred.
type DataReceiverFn[T any] func(*corev1.RecordRef, T, error) error
