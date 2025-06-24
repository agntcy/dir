// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package componenterror

import (
	"errors"
	"maps"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

type ComponentError struct {
	Err error `json:"-"`

	Component string            `json:"component"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Message   string            `json:"message"`
}

func NewComponentError(err error, component string) *ComponentError {
	cErr := &ComponentError{
		Err:       err,
		Component: component,
	}

	if err != nil {
		cErr.Message = err.Error()
	}

	return cErr
}

func (e *ComponentError) Error() string {
	return e.Message
}

func (e *ComponentError) WithMetadata(metadata map[string]string) *ComponentError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}

	maps.Copy(e.Metadata, metadata)

	return e
}

func (e *ComponentError) WithMessage(message string) *ComponentError {
	e.Message = message

	return e
}

func (e *ComponentError) WithErr(err error) *ComponentError {
	e.Err = errors.Join(e.Err, err)
	e.Message = e.Err.Error()

	return e
}

func (e *ComponentError) IsComponent(component string) bool {
	return strings.EqualFold(e.Component, component)
}

func (e *ComponentError) ToAPIError() *errdetails.ErrorInfo {
	return &errdetails.ErrorInfo{
		Reason:   e.Message,
		Domain:   e.Component,
		Metadata: e.Metadata,
	}
}

func FromAPIError(apiError *errdetails.ErrorInfo) *ComponentError {
	return NewComponentError(nil, apiError.GetDomain()).
		WithMessage(apiError.GetReason()).
		WithMetadata(apiError.GetMetadata())
}
