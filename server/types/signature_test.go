// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
)

func TestIsSignatureStatus(t *testing.T) {
	for _, s := range types.SignatureStatuses() {
		assert.True(t, types.IsSignatureStatus(s), s)
	}

	assert.Equal(t, []string{"verified", "failed", "unsigned"}, types.SignatureStatuses())
	assert.False(t, types.IsSignatureStatus("unknown"))
	assert.False(t, types.IsSignatureStatus(""))
}

func TestWithSignatureStatuses(t *testing.T) {
	var f types.RecordFilters

	types.WithSignatureStatuses(types.SignatureStatusFailed)(&f)
	types.WithSignatureStatuses(types.SignatureStatusUnsigned)(&f)

	assert.Equal(t, []string{types.SignatureStatusFailed, types.SignatureStatusUnsigned}, f.SignatureStatuses)
}
