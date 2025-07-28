// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import "fmt"

// ErrorType represents the type of CID-related error.
type ErrorType string

const (
	// ErrorTypeInvalidInput indicates invalid input data.
	ErrorTypeInvalidInput ErrorType = "invalid_input"
	// ErrorTypeInvalidCID indicates an invalid CID format.
	ErrorTypeInvalidCID ErrorType = "invalid_cid"
	// ErrorTypeInvalidDigest indicates an invalid digest format.
	ErrorTypeInvalidDigest ErrorType = "invalid_digest"
	// ErrorTypeUnsupportedAlgorithm indicates an unsupported hash algorithm.
	ErrorTypeUnsupportedAlgorithm ErrorType = "unsupported_algorithm"
	// ErrorTypeHashCreation indicates a failure in hash creation.
	ErrorTypeHashCreation ErrorType = "hash_creation_failed"
)

// Error represents a CID-related error with additional context.
type Error struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if len(e.Details) == 0 {
		return fmt.Sprintf("CID error [%s]: %s", e.Type, e.Message)
	}

	return fmt.Sprintf("CID error [%s]: %s (details: %+v)", e.Type, e.Message, e.Details)
}

// Is checks if the error is of a specific type.
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Type == t.Type
	}

	return false
}
