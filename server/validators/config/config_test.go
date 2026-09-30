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
		Provider: ProviderOASF,
		Ops:      []string{OpPush, OpIndex},
		Config:   map[string]any{ConfigKeySchemaURL: "https://schema.oasf.outshift.com"},
	}}).Validate())

	require.NoError(t, (Config{{
		Provider: ProviderCEL,
		Ops:      []string{OpPush},
		Config:   map[string]any{ConfigKeyExpressions: []string{`record.name != ""`}},
	}}).Validate())

	require.EqualError(t, (Config{{}}).Validate(), "validators[0]: provider is required")
	require.EqualError(t, (Config{{Provider: "unknown"}}).Validate(), `validators[0]: unsupported provider "unknown"`)
	require.EqualError(t, (Config{{Provider: ProviderOASF}}).Validate(), `validators[0]: config.schema_url is required for provider "oasf"`)
	require.EqualError(t, (Config{{Provider: ProviderCEL}}).Validate(), "validators[0]: config.expressions: expressions is required")
	require.EqualError(t, (Config{{
		Provider: ProviderCEL,
		Config:   map[string]any{ConfigKeyExpressions: []string{`record.name != ""`}},
	}}).Validate(), "validators[0]: at least one op is required")
	require.EqualError(t, (Config{{
		Provider: ProviderOASF,
		Config:   map[string]any{ConfigKeySchemaURL: "https://schema.oasf.outshift.com"},
	}}).Validate(), "validators[0]: at least one op is required")
	require.EqualError(t, (Config{{
		Provider: ProviderOASF,
		Ops:      []string{"unknown"},
		Config:   map[string]any{ConfigKeySchemaURL: "https://schema.oasf.outshift.com"},
	}}).Validate(), `validators[0]: unsupported op "unknown"`)
}

func TestValidatorHasOp(t *testing.T) {
	t.Parallel()

	v := Validator{Ops: []string{OpPush, OpIndex}}
	require.True(t, v.HasOp(OpPush))
	require.True(t, v.HasOp(OpIndex))
	require.False(t, v.HasOp(OpAutosync))
}

func TestValidatorConfigString(t *testing.T) {
	t.Parallel()

	require.Empty(t, (Validator{}).ConfigString(ConfigKeySchemaURL))
	require.Empty(t, (Validator{Config: map[string]any{ConfigKeySchemaURL: 1}}).ConfigString(ConfigKeySchemaURL))
	require.Equal(t, "https://schema.example.com", (Validator{
		Config: map[string]any{ConfigKeySchemaURL: "https://schema.example.com", "extra": true},
	}).ConfigString(ConfigKeySchemaURL))
}

func TestValidatorConfigStrings(t *testing.T) {
	t.Parallel()

	_, err := (Validator{}).ConfigStrings(ConfigKeyExpressions)
	require.EqualError(t, err, "expressions is required")

	got, err := (Validator{
		Config: map[string]any{ConfigKeyExpressions: []string{`record.name != ""`, " true "}},
	}).ConfigStrings(ConfigKeyExpressions)
	require.NoError(t, err)
	require.Equal(t, []string{`record.name != ""`, "true"}, got)

	got, err = (Validator{
		Config: map[string]any{ConfigKeyExpressions: []any{`record.name != ""`, `record.version != ""`}},
	}).ConfigStrings(ConfigKeyExpressions)
	require.NoError(t, err)
	require.Equal(t, []string{`record.name != ""`, `record.version != ""`}, got)

	_, err = (Validator{
		Config: map[string]any{ConfigKeyExpressions: []string{}},
	}).ConfigStrings(ConfigKeyExpressions)
	require.EqualError(t, err, "expressions must contain at least one value")

	_, err = (Validator{
		Config: map[string]any{ConfigKeyExpressions: []any{1}},
	}).ConfigStrings(ConfigKeyExpressions)
	require.EqualError(t, err, "[0]: must be a string")

	_, err = (Validator{
		Config: map[string]any{ConfigKeyExpressions: "record.name != \"\""},
	}).ConfigStrings(ConfigKeyExpressions)
	require.EqualError(t, err, "must be a list of strings")
}
