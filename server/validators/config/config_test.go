// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (Config{}).Validate())

	require.NoError(t, (Config{{
		Provider:  ProviderOASF,
		SchemaURL: "https://schema.oasf.outshift.com",
		Ops:       []string{OpPush, OpIndex},
	}}).Validate())

	require.EqualError(t, (Config{{}}).Validate(), "validators[0]: provider is required")
	require.EqualError(t, (Config{{Provider: "unknown"}}).Validate(), `validators[0]: unsupported provider "unknown"`)
	require.EqualError(t, (Config{{Provider: ProviderOASF}}).Validate(), `validators[0]: schema_url is required for provider "oasf"`)
	require.EqualError(t, (Config{{
		Provider:  ProviderOASF,
		SchemaURL: "https://schema.oasf.outshift.com",
	}}).Validate(), "validators[0]: at least one op is required")
	require.EqualError(t, (Config{{
		Provider:  ProviderOASF,
		SchemaURL: "https://schema.oasf.outshift.com",
		Ops:       []string{"unknown"},
	}}).Validate(), `validators[0]: unsupported op "unknown"`)
}

func TestValidatorHasOp(t *testing.T) {
	t.Parallel()

	v := Validator{Ops: []string{OpPush, OpIndex}}
	require.True(t, v.HasOp(OpPush))
	require.True(t, v.HasOp(OpIndex))
	require.False(t, v.HasOp(OpAutosync))
}
