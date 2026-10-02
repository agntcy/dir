// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func opaPolicy(file string) validatorsconfig.Validator {
	return validatorsconfig.Validator{
		Provider: validatorsconfig.ProviderOPA,
		Ops:      []string{validatorsconfig.OpEvaluate},
		Config:   map[string]any{validatorsconfig.ConfigKeyFile: file},
	}
}

func celPolicy(name string, expressions ...string) validatorsconfig.Validator {
	return validatorsconfig.Validator{
		Provider: validatorsconfig.ProviderCEL,
		Ops:      []string{validatorsconfig.OpEvaluate},
		Config:   map[string]any{validatorsconfig.ConfigKeyName: name, validatorsconfig.ConfigKeyExpressions: expressions},
	}
}

func policiesOf(t *testing.T, dir string, entries ...validatorsconfig.Validator) []Policy {
	t.Helper()

	reg, err := NewRegistry(context.Background(), entries, dir)
	require.NoError(t, err)

	return reg.Policies()
}

func TestRegistryPolicies(t *testing.T) {
	t.Parallel()

	dir := writePolicy(t, "require-name.rego", allowIfNamed)

	policies := policiesOf(t, dir, opaPolicy("require-name.rego"), celPolicy("has-description", `record.description != ""`))

	require.Len(t, policies, 2, "in config order")
	assert.Equal(t, "opa:require-name", policies[0].ID)
	assert.Equal(t, "cel:has-description", policies[1].ID)

	for _, p := range policies {
		assert.Len(t, p.Version, versionLength)
		require.NotNil(t, p.Validator)
	}

	t.Run("each validator decides on its own policy", func(t *testing.T) {
		t.Parallel()

		named := corev1.New(&typesv1alpha1.Record{Name: "agent"})

		ok, _, err := named.ValidateWith(context.Background(), policies[0].Validator)
		require.NoError(t, err)
		assert.True(t, ok, "OPA allows a named record")

		ok, _, err = named.ValidateWith(context.Background(), policies[1].Validator)
		require.NoError(t, err)
		assert.False(t, ok, "CEL rejects a record with no description")
	})
}

func TestRegistryPolicies_NoneConfigured(t *testing.T) {
	t.Parallel()

	var nilRegistry *Registry

	assert.Empty(t, nilRegistry.Policies())

	admission := validatorsconfig.Validator{
		Provider: validatorsconfig.ProviderCEL,
		Ops:      []string{validatorsconfig.OpPush},
		Config:   map[string]any{validatorsconfig.ConfigKeyExpressions: []string{`true`}},
	}

	assert.Empty(t, policiesOf(t, "", admission), "a validator without op evaluate defines no policy")
	assert.Empty(t, policiesOf(t, ""))
}

// The version follows the policy's content and nothing else, so an edit is
// evaluated again and a restart is not.
func TestPolicyVersion_FollowsContent(t *testing.T) {
	t.Parallel()

	const stricter = `
package dir.policy.require_name

default allow := false

allow if {
	input.name != ""
	input.version != ""
}
`

	t.Run("opa", func(t *testing.T) {
		t.Parallel()

		before := policiesOf(t, writePolicy(t, "p.rego", allowIfNamed), opaPolicy("p.rego"))[0].Version
		again := policiesOf(t, writePolicy(t, "p.rego", allowIfNamed), opaPolicy("p.rego"))[0].Version
		edited := policiesOf(t, writePolicy(t, "p.rego", stricter), opaPolicy("p.rego"))[0].Version

		assert.Equal(t, before, again, "the same file is the same version")
		assert.NotEqual(t, before, edited, "an edited file is a new one")
	})

	t.Run("opa ignores the file's name", func(t *testing.T) {
		t.Parallel()

		a := policiesOf(t, writePolicy(t, "a.rego", allowIfNamed), opaPolicy("a.rego"))[0].Version
		b := policiesOf(t, writePolicy(t, "b.rego", allowIfNamed), opaPolicy("b.rego"))[0].Version

		assert.Equal(t, a, b)
	})

	t.Run("cel", func(t *testing.T) {
		t.Parallel()

		before := policiesOf(t, "", celPolicy("p", `record.name != ""`))[0].Version
		again := policiesOf(t, "", celPolicy("p", `record.name != ""`))[0].Version
		edited := policiesOf(t, "", celPolicy("p", `record.name != "x"`))[0].Version
		renamed := policiesOf(t, "", celPolicy("q", `record.name != ""`))[0].Version
		added := policiesOf(t, "", celPolicy("p", `record.name != ""`, `true`))[0].Version

		assert.Equal(t, before, again)
		assert.NotEqual(t, before, edited)
		assert.Equal(t, before, renamed, "the name is part of the ID, not of the version")
		assert.NotEqual(t, before, added)
	})

	t.Run("parts do not run together", func(t *testing.T) {
		t.Parallel()

		// Moving the boundary between two parts must not give the same version.
		assert.NotEqual(t, contentVersion("ab", "c"), contentVersion("a", "bc"))
		assert.NotEqual(t, contentVersion("a", ""), contentVersion("", "a"))
	})
}
