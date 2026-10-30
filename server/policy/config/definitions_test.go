// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluationsValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (Evaluations{}).Validate())

	require.NoError(t, (Evaluations{{
		Provider: ProviderOPA,
		File:     "require-license.rego",
	}}).Validate())

	require.NoError(t, (Evaluations{{
		Provider:    ProviderCEL,
		Name:        "has-description",
		Expressions: []string{`record.description != ""`},
	}}).Validate())

	require.EqualError(t, (Evaluations{{}}).Validate(), "evaluations[0]: provider is required")
	require.EqualError(t, (Evaluations{{Provider: "unknown"}}).Validate(),
		`evaluations[0]: unsupported provider "unknown", expected "oasf", "opa" or "cel"`)
}

// An evaluation policy's verdict is stored against the version of the content
// that reached it, so a provider whose content is behind an HTTP endpoint
// cannot be one.
func TestEvaluationsValidate_RejectsOASF(t *testing.T) {
	t.Parallel()

	require.EqualError(t, (Evaluations{{Provider: ProviderOASF}}).Validate(),
		`evaluations[0]: provider "oasf" cannot define an evaluation policy, because its content `+
			`is not known when it is loaded: only "opa" and "cel" can`)
}

func TestOperationsValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (Operations{}).Validate())

	require.NoError(t, (Operations{{
		Provider:  ProviderOASF,
		Ops:       []string{OpPush, OpAutosync, OpIndex},
		Name:      "schema",
		SchemaURL: "https://schema.oasf.outshift.com",
	}}).Validate())

	require.NoError(t, (Operations{{
		Provider:    ProviderCEL,
		Ops:         []string{OpPush},
		Mode:        ModeEnforce,
		Name:        "require-identity",
		Expressions: []string{`record.name != ""`},
	}}).Validate())

	require.EqualError(t, (Operations{{
		Provider:    ProviderCEL,
		Name:        "require-identity",
		Expressions: []string{`record.name != ""`},
	}}).Validate(), "operations[0]: at least one op is required")
}

// evaluate was an op only because producing and deciding shared one list. It
// is not a point in a record's lifecycle, and it is no longer an op.
func TestOperationsValidate_EvaluateIsNotAnOp(t *testing.T) {
	t.Parallel()

	require.EqualError(t, (Operations{{
		Provider:    ProviderCEL,
		Ops:         []string{"evaluate"},
		Name:        "has-description",
		Expressions: []string{`record.description != ""`},
	}}).Validate(), `operations[0]: unsupported op "evaluate", expected one of autosync, index, pull, push`)
}

func TestOperationsValidate_Ops(t *testing.T) {
	t.Parallel()

	require.EqualError(t, (Operations{{
		Provider:    ProviderCEL,
		Ops:         []string{OpPush, OpPush},
		Name:        "require-identity",
		Expressions: []string{`true`},
	}}).Validate(), `operations[0]: op "push" is listed more than once`)
}

func TestOperationsValidate_Mode(t *testing.T) {
	t.Parallel()

	for _, mode := range []Mode{"", ModeOff, ModeShadow, ModeEnforce} {
		require.NoError(t, (Operations{{
			Provider:    ProviderCEL,
			Ops:         []string{OpPush},
			Mode:        mode,
			Name:        "require-identity",
			Expressions: []string{`true`},
		}}).Validate(), "mode %q", mode)
	}

	require.EqualError(t, (Operations{{
		Provider:    ProviderCEL,
		Ops:         []string{OpPush},
		Mode:        "on",
		Name:        "require-identity",
		Expressions: []string{`true`},
	}}).Validate(), `operations[0]: unknown mode "on", expected "off", "shadow" or "enforce"`)
}

// What the rest of the server derives per op is checked against KnownOps, so
// it has to agree with the ops a policy can actually be configured at.
func TestKnownOps_AgreesWithValidation(t *testing.T) {
	t.Parallel()

	ops := KnownOps()
	require.NotEmpty(t, ops)

	for _, op := range ops {
		require.NoError(t, (Operations{{
			Provider:    ProviderCEL,
			Ops:         []string{op},
			Name:        "require-identity",
			Expressions: []string{`true`},
		}}).Validate(), "op %q", op)
	}
}

// An expression that is only whitespace is a line someone left behind, not a
// rule, and it would otherwise compile to nothing.
func TestOperationsValidate_BlankExpression(t *testing.T) {
	t.Parallel()

	require.EqualError(t, (Operations{{
		Provider:    ProviderCEL,
		Ops:         []string{OpPush},
		Name:        "require-identity",
		Expressions: []string{`true`, "  "},
	}}).Validate(), "operations[0]: expressions[1]: value is empty")
}

// Each provider's settings are its own. A setting meant for another one is a
// mistake, and silently ignoring it would leave a policy doing nothing like
// what its author wrote.
func TestValidate_ProviderFields(t *testing.T) {
	t.Parallel()

	require.EqualError(t, (Operations{{Provider: ProviderOASF, Ops: []string{OpPush}, Name: "schema"}}).Validate(),
		`operations[0]: schema_url is required for provider "oasf"`)
	require.EqualError(t, (Operations{{Provider: ProviderCEL, Ops: []string{OpPush}, Name: "x"}}).Validate(),
		`operations[0]: expressions is required for provider "cel"`)
	require.EqualError(t, (Operations{{Provider: ProviderOPA, Ops: []string{OpPush}}}).Validate(),
		`operations[0]: file is required for provider "opa"`)

	require.EqualError(t, (Evaluations{{
		Provider:    ProviderOPA,
		File:        "require-license.rego",
		Expressions: []string{`true`},
	}}).Validate(), `evaluations[0]: expressions does not apply to provider "opa", which takes file`)

	require.EqualError(t, (Evaluations{{
		Provider:    ProviderCEL,
		Name:        "has-description",
		File:        "require-license.rego",
		Expressions: []string{`true`},
	}}).Validate(), `evaluations[0]: file does not apply to provider "cel", which takes expressions`)

	require.EqualError(t, (Operations{{
		Provider:  ProviderOASF,
		Ops:       []string{OpPush},
		Name:      "schema",
		SchemaURL: "https://schema.oasf.outshift.com",
		File:      "require-license.rego",
	}}).Validate(), `operations[0]: file does not apply to provider "oasf", which takes schema_url`)
}

func TestValidate_PolicyFile(t *testing.T) {
	t.Parallel()

	require.EqualError(t, (Evaluations{{Provider: ProviderOPA, File: "../policy.rego"}}).Validate(),
		`evaluations[0]: file "../policy.rego" must be a filename, not a path`)
	require.EqualError(t, (Evaluations{{Provider: ProviderOPA, File: "policy.txt"}}).Validate(),
		`evaluations[0]: file "policy.txt" must have a .rego extension`)
}

func TestPolicyName(t *testing.T) {
	t.Parallel()

	opa := Evaluation{Provider: ProviderOPA, File: "require-license.rego"}
	assert.Equal(t, "require-license", opa.PolicyName(), "the OPA file names the policy")
	assert.Equal(t, "opa:require-license", opa.PolicyID())

	named := opa
	named.Name = "license"
	assert.Equal(t, "opa:license", named.PolicyID(), "an explicit name wins")

	cel := Operation{Provider: ProviderCEL, Ops: []string{OpPush}, Name: "require-identity"}
	assert.Equal(t, "cel:require-identity", cel.PolicyID())

	require.EqualError(t, (Evaluations{{
		Provider:    ProviderCEL,
		Expressions: []string{`record.name != ""`},
	}}).Validate(), "evaluations[0]: name is required", "nothing else names a CEL policy")

	require.EqualError(t, (Operations{{
		Provider:    ProviderCEL,
		Ops:         []string{OpPush},
		Expressions: []string{`true`},
	}}).Validate(), "operations[0]: name is required")
}

// The name reaches an ID that is listed in a comma-separated setting and used
// as a metric label, so it is limited to what those can carry.
func TestPolicyName_Pattern(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"has space", "a,b", "-leading", "colon:name", "naïve"} {
		err := (Evaluations{{Provider: ProviderCEL, Name: name, Expressions: []string{`true`}}}).Validate()
		require.ErrorContains(t, err,
			"must start with a letter or digit and contain only letters, digits, '.', '_' and '-'", name)
	}
}

func TestValidate_DuplicateWithinABlock(t *testing.T) {
	t.Parallel()

	opa := Evaluation{Provider: ProviderOPA, File: "require-license.rego"}
	sameName := Evaluation{Provider: ProviderOPA, File: "other.rego", Name: "require-license"}

	require.EqualError(t, (Evaluations{opa, sameName}).Validate(),
		`evaluations[1]: policy "opa:require-license" is already defined by evaluations[0]`)

	push := Operation{Provider: ProviderCEL, Ops: []string{OpPush}, Name: "x", Expressions: []string{`true`}}
	index := Operation{Provider: ProviderCEL, Ops: []string{OpIndex}, Name: "x", Expressions: []string{`true`}}

	require.EqualError(t, (Operations{push, index}).Validate(),
		`operations[1]: policy "cel:x" is already defined by operations[0]`)
}

// Nothing resolves an operation policy by ID, so this is not ambiguity in the
// code. It is ambiguity in the logs, which are the audit surface.
func TestValidateNames_AcrossBlocks(t *testing.T) {
	t.Parallel()

	evaluations := Evaluations{{Provider: ProviderCEL, Name: "shared", Expressions: []string{`true`}}}
	operations := Operations{{
		Provider:    ProviderCEL,
		Ops:         []string{OpPush},
		Name:        "shared",
		Expressions: []string{`true`},
	}}

	require.EqualError(t, ValidateNames(evaluations, operations),
		`operations[0]: policy "cel:shared" is also defined by evaluations[0]: a name must identify one policy`)

	// The same name under a different provider is a different policy.
	operations[0].Provider = ProviderOPA
	operations[0].Expressions = nil
	operations[0].File = "shared.rego"
	require.NoError(t, ValidateNames(evaluations, operations))

	require.NoError(t, ValidateNames(nil, nil))
}

func TestOperationHasOp(t *testing.T) {
	t.Parallel()

	o := Operation{Ops: []string{OpPush, OpIndex}}
	require.True(t, o.HasOp(OpPush))
	require.True(t, o.HasOp(OpIndex))
	require.False(t, o.HasOp(OpAutosync))
}

func TestModeActive(t *testing.T) {
	t.Parallel()

	require.True(t, ModeShadow.Active())
	require.True(t, ModeEnforce.Active())
	require.False(t, ModeOff.Active())
	require.False(t, Mode("").Active())
}
