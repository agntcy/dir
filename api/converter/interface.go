// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package converter

import (
	"fmt"
)

// ObjectConverter defines a generic interface for converting between two object types
type ObjectConverter[TypeA, TypeB any] interface {
	// To converts from TypeA to TypeB
	To(a TypeA) (TypeB, error)
	// From converts from TypeB to TypeA
	From(b TypeB) (TypeA, error)
}

// ConversionError represents an error that occurred during type conversion
type ConversionError struct {
	FromType string
	ToType   string
	Message  string
	Cause    error
}

func (e *ConversionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("conversion from %s to %s failed: %s (cause: %v)",
			e.FromType, e.ToType, e.Message, e.Cause)
	}
	return fmt.Sprintf("conversion from %s to %s failed: %s",
		e.FromType, e.ToType, e.Message)
}

func (e *ConversionError) Unwrap() error {
	return e.Cause
}

// NewConversionError creates a new conversion error
func NewConversionError(fromType, toType, message string, cause error) *ConversionError {
	return &ConversionError{
		FromType: fromType,
		ToType:   toType,
		Message:  message,
		Cause:    cause,
	}
}
