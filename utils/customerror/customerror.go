// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package customerror

import (
	"encoding/json"
	"time"
)

type Component int

const (
	ComponentAPI = iota + 1
	ComponentCLI
	ComponentHub
)

type ComponentError interface {
	Error() string
	Marshal() ([]byte, error)
	Unmarshal(data []byte, v any) error
	IsFrom(component Component) bool
}

type componentError struct {
	OriginalErr error `json:"original_error"`

	Component Component `json:"component"`
	Message   string    `json:"message"`
	Time      time.Time `json:"time"`

	ComponentError
}

func NewCustomError(component Component, message string) componentError {
	return componentError{
		Component: component,
		Message:   message,
		Time:      time.Now(),
	}
}

func (e *componentError) Error() string {
	return e.Message
}

func (e *componentError) Marshal() ([]byte, error) {
	json, err := json.Marshal(e)

	return json, err //nolint:wrapcheck
}

func (e *componentError) Unmarshal(data []byte, v any) error {
	err := json.Unmarshal(data, v)

	return err //nolint:wrapcheck
}

func (e *componentError) IsFrom(component Component) bool {
	return e.Component == component
}
