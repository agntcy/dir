// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package customerror

import (
	"github.com/pkg/errors"
)

type Component uint32

const (
	ComponentAPI = iota + 1
	ComponentCLI
	ComponentHub
)

func (c Component) String() string {
	switch c {
	case ComponentAPI:
		return "API"
	case ComponentCLI:
		return "CLI"
	case ComponentHub:
		return "Hub"
	default:
		return "Unknown"
	}
}

type ComponentError struct {
	Err error `json:"-"`

	Component Component         `json:"component"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Message   string            `json:"message"`
}

func NewComponentError(component Component, message string) *ComponentError {
	return &ComponentError{
		Component: component,
		Message:   message,
	}
}

func (e *ComponentError) Error() string {
	return e.Message
}

func (e *ComponentError) IsComponent(component Component) bool {
	return e.Component == component
}

func (e *ComponentError) WithMetadata(metadata map[string]string) *ComponentError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	for key, value := range metadata {
		e.Metadata[key] = value
	}
	return e
}

func (e *ComponentError) WithMessage(message string) *ComponentError {
	e.Message = message
	return e
}

func (e *ComponentError) WithErr(err error) *ComponentError {
	if err != nil {
		e.Err = errors.Wrap(err, e)
	} else {
		e.Err = err
	}
	return e
}

func exampleErrHandler() error {
	e := NewComponentError(ComponentAPI, "An error occurred in the API component")

	// my operation failed (OP: 1)
	// NON-FATAL ERROR
	e.WithMetadata(map[string]string{
		"request_id": "12345",
		"user_id":    "67890",
		"operation":  "1",
	}).WithMessage("Failed to process request").WithErr(errors.New("request processing error"))

	// my operation failed (OP: 2)
	// FATAL ERROR
	e.WithMetadata(map[string]string{
		"operation": "2",
	})

	wrappedErr := errors.Wrap(e, e.Err)

	var extractedErr *ComponentError
	if !errors.As(wrappedErr, &extractedErr) {
		return errors.New("failed to extract ComponentError from wrapped error")
	}

	return nil
}
