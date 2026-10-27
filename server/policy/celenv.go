// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"fmt"

	"cel.dev/cel-go/cel"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
)

const (
	// celVarRecord is the OASF record, bound as a map of its fields.
	celVarRecord = "record"

	// celVarRecordCID is the record's content identifier.
	celVarRecordCID = "record_cid"

	// celVarIndexedAt is when this node first indexed the record.
	// Absent until it has, so the binding is optional.
	celVarIndexedAt = "indexed_at"

	// celVarMeta is what this node has observed about the record.
	celVarMeta = "meta"

	// celVarContext is the circumstances of the operation being decided.
	celVarContext = "context"

	// celVarNow is the instant the operation is being decided at, the same one
	// for every policy deciding it.
	celVarNow = "now"

	// policyContainer is the package unqualified names resolve in, so that an
	// enum constant is written SignatureStatus.SIGNATURE_STATUS_VERIFIED
	// rather than with the package repeated on every comparison.
	policyContainer = "agntcy.dir.policy.v1"

	recordMetaType = policyContainer + ".RecordMeta"

	operationContextType = policyContainer + ".OperationContext"
)

// celEnumAbbrevs are the enums a policy reads that are declared outside
// policyContainer, and so are not covered by it. Abbreviating them keeps one
// spelling for every enum constant regardless of which package declares it.
var celEnumAbbrevs = []string{
	"agntcy.dir.security.v1.Severity",
	"agntcy.dir.security.v1.ScannerType",
	"agntcy.dir.identity.v1.ClaimRole",
}

// celEnv builds the environment a CEL policy in l is compiled in.
//
// The layer has to be settled here, at load time, because CEL resolves every
// name against the environment it compiles in: a policy that reads a variable
// its layer is not given fails to load instead of failing per record. That is
// the enforcement mechanism for the separation, not a convention on top of it.
func celEnv(l layer) (*cel.Env, error) {
	switch l {
	case layerEvaluation:
		return newEvaluationEnv()
	case layerOperation:
		return newOperationEnv()
	default:
		return nil, fmt.Errorf("unknown policy layer %q", l)
	}
}

// newEvaluationEnv builds the environment an evaluation policy is compiled in.
//
// Typed where there is a schema, dynamic where there is not. The facts in meta
// come from policy/v1 protos, so registering their descriptors makes a
// misspelled field a load-time error rather than an evaluation-time one, and
// makes an ordered enum comparison such as
// meta.scan.max_severity >= Severity.SEVERITY_MEDIUM check out.
//
// indexed_at is optional because a record that has not been indexed has no
// such time.
//
// verdicts, now and context are deliberately absent. An evaluation that could
// read another policy's verdict, the wall clock, or the caller is what
// RecordState is shaped to prevent.
func newEvaluationEnv() (*cel.Env, error) {
	return newEnv(
		// The facts in meta reach enums declared outside policy_input.proto,
		// so those files are registered too.
		cel.Types(
			&securityv1.ScanReport{},
			&identityv1.Claim{},
		),
		cel.Abbrevs(celEnumAbbrevs...),
		cel.Variable(celVarIndexedAt, cel.OptionalType(cel.TimestampType)),
		cel.Variable(celVarMeta, cel.ObjectType(recordMetaType)),
	)
}

// newOperationEnv builds the environment an operation policy is compiled in.
//
// meta is deliberately absent, and is the difference that defines the layer.
// An operation decides on what the evaluations concluded, not on the facts
// they concluded it from, so a policy here that wants a fact has to ask for
// the verdict that reads it.
//
// verdicts and indexed_at are absent because reaching either needs a database
// read that Decide does not have. Declaring them before they can be answered
// truthfully would compile policies that silently decide on nothing.
func newOperationEnv() (*cel.Env, error) {
	return newEnv(
		cel.Variable(celVarContext, cel.ObjectType(operationContextType)),
		cel.Variable(celVarNow, cel.TimestampType),
	)
}

// newEnv builds a CEL environment holding what both layers share, plus the
// options that distinguish one.
//
// Both are given the record and its CID, because every policy is about a
// record whichever question it answers. record stays dynamic on purpose: an
// OASF record has no compile-time schema here, so a policy must be able to
// read fields of schema versions this build does not know about. Declaring it
// as the google.protobuf.Struct it travels in would be more precise and would
// break that.
func newEnv(layerOpts ...cel.EnvOption) (*cel.Env, error) {
	opts := []cel.EnvOption{
		cel.OptionalTypes(),
		cel.Container(policyContainer),
		// Registering one message per file registers every type that file
		// declares, which is how the enums reached through the inputs become
		// resolvable. Registering a type is not declaring a variable, so this
		// gives neither layer anything its own variables cannot reach.
		cel.Types(&policyv1.RecordMeta{}),
		cel.Variable(celVarRecord, cel.DynType),
		cel.Variable(celVarRecordCID, cel.StringType),
	}

	env, err := cel.NewEnv(append(opts, layerOpts...)...)
	if err != nil {
		return nil, fmt.Errorf("create CEL environment: %w", err)
	}

	return env, nil
}
