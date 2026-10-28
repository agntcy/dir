// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func opaPolicy(file string) policyconfig.Evaluation {
	return policyconfig.Evaluation{
		Provider: policyconfig.ProviderOPA,
		File:     file,
	}
}

func celPolicy(name string, expressions ...string) policyconfig.Evaluation {
	return policyconfig.Evaluation{
		Provider:    policyconfig.ProviderCEL,
		Name:        name,
		Expressions: expressions,
	}
}

func policiesOf(t *testing.T, dir string, entries ...policyconfig.Evaluation) []Evaluator {
	t.Helper()

	reg, err := NewRegistry(context.Background(), entries, nil, dir)
	require.NoError(t, err)

	return reg.Evaluators(nil, nil)
}

type fixedSource struct {
	record *corev1.Record
}

func (s fixedSource) Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error) {
	return s.record, nil
}

// cidFacts returns the stored observations of one CID. A CID with no entry
// was looked up and has nothing stored.
type cidFacts struct {
	byCID map[string]types.PolicyFacts
}

func (c cidFacts) Facts(_ context.Context, cid string) (types.PolicyFacts, error) {
	return c.byCID[cid], nil
}

func TestRegistryPolicies(t *testing.T) {
	t.Parallel()

	dir := writePolicy(t, "require-name.rego", allowIfNamed)

	policies := policiesOf(t, dir, opaPolicy("require-name.rego"), celPolicy("has-description", `record.description != ""`))

	require.Len(t, policies, 2, "in config order")
	assert.Equal(t, "opa:require-name", policies[0].PolicyID())
	assert.Equal(t, "cel:has-description", policies[1].PolicyID())

	for _, p := range policies {
		assert.Len(t, p.PolicyVersion(), versionLength)
	}

	t.Run("each policy decides on its own rule", func(t *testing.T) {
		t.Parallel()

		reg, err := NewRegistry(context.Background(), policyconfig.Evaluations{
			opaPolicy("require-name.rego"),
			celPolicy("has-description", `record.description != ""`),
		}, nil, dir)
		require.NoError(t, err)

		named := corev1.New(&typesv1alpha1.Record{Name: "agent"})
		evs := reg.Evaluators(fixedSource{record: named}, nil)
		indexed := indexRecord{cid: named.GetCid()}

		ok, _, err := evs[0].Evaluate(context.Background(), indexed)
		require.NoError(t, err)
		assert.True(t, ok, "OPA allows a named record")

		ok, _, err = evs[1].Evaluate(context.Background(), indexed)
		require.NoError(t, err)
		assert.False(t, ok, "CEL rejects a record with no description")
	})
}

func TestRegistryPolicies_NoneConfigured(t *testing.T) {
	t.Parallel()

	var nilRegistry *Registry

	assert.Empty(t, nilRegistry.Evaluators(nil, nil))

	// An operation policy is not an evaluation policy: it decides at an
	// operation and produces no verdict.
	reg, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPush},
		Name:        "always",
		Expressions: []string{`true`},
	}}, "")
	require.NoError(t, err)

	assert.Empty(t, reg.Evaluators(nil, nil), "an operation policy defines no evaluation policy")
	assert.Len(t, reg.For(policyconfig.OpPush), 1)
	assert.Empty(t, policiesOf(t, ""))
}

// A configured evaluation policy decides on the meta the evaluator projects
// from stored facts. The expressions are the ones a deployment would write.
func TestRegistryPolicies_DecidesOnProjectedMeta(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), policyconfig.Evaluations{
		celPolicy("trusted",
			`meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED`,
			`meta.signature.source == FactSource.FACT_SOURCE_LOCAL`,
			`meta.scan.status == ScanStatus.SCAN_STATUS_COMPLETED`,
			`meta.scan.is_safe`,
			`meta.identity.identity.status == ClaimStatus.CLAIM_STATUS_VERIFIED`,
			`meta.identity.owner.status == ClaimStatus.CLAIM_STATUS_UNCLAIMED`,
		),
	}, nil, "")
	require.NoError(t, err)

	verifiedAt := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	trusted := annotatedRecord(t, map[string]string{
		corev1.AnnotationKeyIdentity: "did:web:acme.com",
	})
	bare := annotatedRecord(t, nil)

	src := &mapSource{records: map[string]*corev1.Record{
		trusted.GetCid(): trusted,
		bare.GetCid():    bare,
	}}
	evs := reg.Evaluators(src, cidFacts{byCID: map[string]types.PolicyFacts{
		trusted.GetCid(): {
			Signed: true,
			Signatures: []types.SignatureVerificationObject{
				sigRow{status: types.SignatureStatusVerified, subject: "did:web:acme.com", issuer: "https://issuer.example", updated: verifiedAt},
			},
			Scans: []types.ScanReportObject{
				scanRow{scanner: "MCP", status: types.ScanStatusCompleted, severity: "NONE", safe: true, updated: verifiedAt},
			},
			Claims: []types.IdentityClaimObject{
				claimRow{role: types.ClaimRoleIdentity, subject: "did:web:acme.com", status: types.ClaimStatusVerified, verified: verifiedAt},
			},
		},
	}})
	require.Len(t, evs, 1)

	ok, reason, err := evs[0].Evaluate(context.Background(), indexRecord{cid: trusted.GetCid()})
	require.NoError(t, err)
	assert.True(t, ok, reason)

	ok, reason, err = evs[0].Evaluate(context.Background(), indexRecord{cid: bare.GetCid()})
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Contains(t, reason, "meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED")
}

// The version follows the policy's content and nothing else, so an edit is
// evaluated again and a restart is not.
func TestPolicyVersion_FollowsContent(t *testing.T) {
	t.Parallel()

	const stricter = `
package dir.policy.require_name

default allow := false

allow if {
	input.record.name != ""
	input.record.version != ""
}
`

	t.Run("opa", func(t *testing.T) {
		t.Parallel()

		before := policiesOf(t, writePolicy(t, "p.rego", allowIfNamed), opaPolicy("p.rego"))[0].PolicyVersion()
		again := policiesOf(t, writePolicy(t, "p.rego", allowIfNamed), opaPolicy("p.rego"))[0].PolicyVersion()
		edited := policiesOf(t, writePolicy(t, "p.rego", stricter), opaPolicy("p.rego"))[0].PolicyVersion()

		assert.Equal(t, before, again, "the same file is the same version")
		assert.NotEqual(t, before, edited, "an edited file is a new one")
	})

	t.Run("opa ignores the file's name", func(t *testing.T) {
		t.Parallel()

		a := policiesOf(t, writePolicy(t, "a.rego", allowIfNamed), opaPolicy("a.rego"))[0].PolicyVersion()
		b := policiesOf(t, writePolicy(t, "b.rego", allowIfNamed), opaPolicy("b.rego"))[0].PolicyVersion()

		assert.Equal(t, a, b)
	})

	t.Run("cel", func(t *testing.T) {
		t.Parallel()

		before := policiesOf(t, "", celPolicy("p", `record.name != ""`))[0].PolicyVersion()
		again := policiesOf(t, "", celPolicy("p", `record.name != ""`))[0].PolicyVersion()
		edited := policiesOf(t, "", celPolicy("p", `record.name != "x"`))[0].PolicyVersion()
		renamed := policiesOf(t, "", celPolicy("q", `record.name != ""`))[0].PolicyVersion()
		added := policiesOf(t, "", celPolicy("p", `record.name != ""`, `true`))[0].PolicyVersion()

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
