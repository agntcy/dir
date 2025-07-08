// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package err_test

import (
	"errors"
	"testing"

	errutils "github.com/agntcy/dir/utils/err"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

var (
	err         = errors.New("test error")
	err2        = errors.New("another test error")
	component   = "HUB"
	message     = "custom error message"
	annotations = map[string]string{
		"input_values": "hello world",
	}
)

func TestNewerr(t *testing.T) {
	cErr := errutils.NewComponentError(err, component)

	assert.NotNil(t, cErr)

	assert.Equal(t, err, cErr.Err)
	assert.Equal(t, component, cErr.Component)
	assert.Nil(t, cErr.Annotations)
}

func TestError(t *testing.T) {
	cErr := errutils.NewComponentError(err, component)

	assert.Equal(t, err.Error(), cErr.Error())
}

func TestWithMessage(t *testing.T) {
	cErr := errutils.NewComponentError(err, component).WithMessage(message)

	assert.Equal(t, message, cErr.Message)
}

func TestWithAnnotations(t *testing.T) {
	cErr := errutils.NewComponentError(err, component).WithAnnotations(annotations)
	assert.Equal(t, annotations, cErr.Annotations)
}

func TestWithErr(t *testing.T) {
	cErr := errutils.NewComponentError(err, component).WithErr(err2)

	joinedErrors := errors.Join(err, err2)

	assert.Equal(t, joinedErrors, cErr.Err)
}

func TestIsComponent(t *testing.T) {
	cErr := errutils.NewComponentError(err, component)
	assert.True(t, cErr.IsComponent(component))

	component2 := "API"
	assert.False(t, cErr.IsComponent(component2))
}

func TestToAPIError(t *testing.T) {
	cErr := errutils.NewComponentError(err, component).WithAnnotations(annotations)

	assert.Equal(t, cErr.ToAPIError().GetDomain(), component)
	assert.Equal(t, cErr.ToAPIError().GetReason(), cErr.Message)
	assert.Equal(t, cErr.ToAPIError().GetMetadata(), annotations)
}

func TestFromAPIError(t *testing.T) {
	apiErr := errdetails.ErrorInfo{
		Reason:   message,
		Domain:   component,
		Metadata: annotations,
	}

	cErr := errutils.FromAPIError(&apiErr)

	assert.Equal(t, message, cErr.Message)
	assert.Equal(t, component, cErr.Component)
	assert.Equal(t, annotations, cErr.Annotations)
}
