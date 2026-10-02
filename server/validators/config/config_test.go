// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	require.NoError(t, (Config{{
		Provider: ProviderOPA,
		Ops:      []string{OpPush},
		Config:   map[string]any{ConfigKeyFile: "policy.rego"},
	}}).Validate())
	require.EqualError(t, (Config{{Provider: ProviderOPA}}).Validate(), `validators[0]: config.file is required for provider "opa"`)
	require.EqualError(t, (Config{{
		Provider: ProviderOPA,
		Config:   map[string]any{ConfigKeyFile: "../policy.rego"},
	}}).Validate(), `validators[0]: config.file "../policy.rego" must be a filename, not a path`)
	require.EqualError(t, (Config{{
		Provider: ProviderOPA,
		Config:   map[string]any{ConfigKeyFile: "policy.txt"},
	}}).Validate(), `validators[0]: config.file "policy.txt" must have a .rego extension`)
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

func TestConfigValidatePolicies(t *testing.T) {
	t.Parallel()

	opa := Validator{Provider: ProviderOPA, Ops: []string{OpEvaluate}, Config: map[string]any{ConfigKeyFile: "require-license.rego"}}
	cel := Validator{
		Provider: ProviderCEL,
		Ops:      []string{OpPush, OpEvaluate},
		Config:   map[string]any{ConfigKeyName: "has-description", ConfigKeyExpressions: []string{`record.description != ""`}},
	}

	require.NoError(t, (Config{opa, cel}).Validate())

	t.Run("names default to the OPA file", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "require-license", opa.PolicyName())
		assert.Equal(t, "opa:require-license", opa.PolicyID())
		assert.Equal(t, "cel:has-description", cel.PolicyID())

		named := opa
		named.Config = map[string]any{ConfigKeyFile: "require-license.rego", ConfigKeyName: "license"}
		assert.Equal(t, "opa:license", named.PolicyID())
	})

	t.Run("a CEL policy needs a name", func(t *testing.T) {
		t.Parallel()

		require.EqualError(t, (Config{{
			Provider: ProviderCEL,
			Ops:      []string{OpEvaluate},
			Config:   map[string]any{ConfigKeyExpressions: []string{`record.name != ""`}},
		}}).Validate(), `validators[0]: config.name is required for provider "cel" with op "evaluate"`)
	})

	t.Run("an OASF validator cannot define a policy", func(t *testing.T) {
		t.Parallel()

		require.EqualError(t, (Config{{
			Provider: ProviderOASF,
			Ops:      []string{OpEvaluate},
			Config:   map[string]any{ConfigKeySchemaURL: "https://schema.oasf.outshift.com"},
		}}).Validate(), `validators[0]: provider "oasf" cannot define a policy: op "evaluate" supports only "opa" and "cel"`)
	})

	t.Run("a name is limited to what an ID can carry", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"has space", "a,b", "-leading", "colon:name", "naïve"} {
			err := (Config{{
				Provider: ProviderCEL,
				Ops:      []string{OpEvaluate},
				Config:   map[string]any{ConfigKeyName: name, ConfigKeyExpressions: []string{`true`}},
			}}).Validate()

			require.ErrorContains(t, err, "must start with a letter or digit and contain only letters, digits, '.', '_' and '-'", name)
		}
	})

	t.Run("two entries cannot define the same policy", func(t *testing.T) {
		t.Parallel()

		dup := opa
		dup.Config = map[string]any{ConfigKeyFile: "other.rego", ConfigKeyName: "require-license"}

		require.EqualError(t, (Config{opa, dup}).Validate(), `validators[1]: policy "opa:require-license" is already defined by validators[0]`)
	})

	t.Run("a validator without the op defines no policy", func(t *testing.T) {
		t.Parallel()

		// No name is needed, and the same name can be used twice.
		admission := Validator{Provider: ProviderCEL, Ops: []string{OpPush}, Config: map[string]any{ConfigKeyExpressions: []string{`true`}}}

		require.NoError(t, (Config{admission, admission}).Validate())
	})
}
