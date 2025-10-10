package client

import corev1 "github.com/agntcy/dir/api/core/v1"

// ReceiverFn is a callback function that processes records.
// It receives the record reference and any error that occurred.
type ReceiverFn func(*corev1.RecordRef, error) error

// ReceiverDataFn is a callback function that processes records.
// It receives the record reference, processed data, and any error that occurred.
// If the function returns an error, the stream processing will stop.
type ReceiverDataFn[T any] func(*corev1.RecordRef, T, error) error
