// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package componenterror_test

import (
	"errors"
	"testing"

	"github.com/agntcy/dir/utils/componenterror"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

var (
	err       = errors.New("test error")
	err2      = errors.New("another test error")
	component = "HUB"
	message   = "custom error message"
	metadata  = map[string]string{
		"input_values": "hello world",
	}
)

func TestNewComponentError(t *testing.T) {
	cErr := componenterror.NewComponentError(err, component)

	assert.NotNil(t, cErr)

	assert.Equal(t, err, cErr.Err)
	assert.Equal(t, component, cErr.Component)
	assert.Nil(t, cErr.Metadata)
}

func TestError(t *testing.T) {
	cErr := componenterror.NewComponentError(err, component)

	assert.Equal(t, err.Error(), cErr.Error())
}

func TestWithMessage(t *testing.T) {
	cErr := componenterror.NewComponentError(err, component).WithMessage(message)

	assert.Equal(t, message, cErr.Message)
}

func TestWithMetadata(t *testing.T) {
	cErr := componenterror.NewComponentError(err, component).WithMetadata(metadata)
	assert.Equal(t, metadata, cErr.Metadata)
}

func TestWithErr(t *testing.T) {
	cErr := componenterror.NewComponentError(err, component).WithErr(err2)

	joinedErrors := errors.Join(err, err2)

	assert.Equal(t, joinedErrors, cErr.Err)
}

func TestIsComponent(t *testing.T) {
	cErr := componenterror.NewComponentError(err, component)
	assert.True(t, cErr.IsComponent(component))

	component2 := "API"
	assert.False(t, cErr.IsComponent(component2))
}

func TestToAPIError(t *testing.T) {
	cErr := componenterror.NewComponentError(err, component).WithMetadata(metadata)

	assert.Equal(t, cErr.ToAPIError().GetDomain(), component)
	assert.Equal(t, cErr.ToAPIError().GetReason(), cErr.Message)
	assert.Equal(t, cErr.ToAPIError().GetMetadata(), metadata)
}

func TestFromAPIError(t *testing.T) {
	apiErr := errdetails.ErrorInfo{
		Reason:   message,
		Domain:   component,
		Metadata: metadata,
	}

	cErr := componenterror.FromAPIError(&apiErr)

	assert.Equal(t, message, cErr.Message)
	assert.Equal(t, component, cErr.Component)
	assert.Equal(t, metadata, cErr.Metadata)
}
