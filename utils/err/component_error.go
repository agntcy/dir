// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package err

import (
	"errors"
	"maps"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

type ComponentError struct {
	Err error `json:"-"`

	Component   string            `json:"component"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Message     string            `json:"message"`
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

// TODO: decide how we want to format the component error message in the future, e.g. do we want to include annotations or not.
func (e *ComponentError) Error() string {
	return e.Message
}

func (e *ComponentError) WithAnnotations(annotations map[string]string) *ComponentError {
	if e.Annotations == nil {
		e.Annotations = make(map[string]string)
	}

	maps.Copy(e.Annotations, annotations)

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
		Metadata: e.Annotations,
	}
}

func FromAPIError(apiError *errdetails.ErrorInfo) *ComponentError {
	return NewComponentError(nil, apiError.GetDomain()).
		WithMessage(apiError.GetReason()).
		WithAnnotations(apiError.GetMetadata())
}
