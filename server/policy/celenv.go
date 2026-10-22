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

	// policyContainer is the package unqualified names resolve in, so that an
	// enum constant is written SignatureStatus.SIGNATURE_STATUS_VERIFIED
	// rather than with the package repeated on every comparison.
	policyContainer = "agntcy.dir.policy.v1"

	recordMetaType = policyContainer + ".RecordMeta"
)

// celEnumAbbrevs are the enums a policy reads that are declared outside
// policyContainer, and so are not covered by it. Abbreviating them keeps one
// spelling for every enum constant regardless of which package declares it.
var celEnumAbbrevs = []string{
	"agntcy.dir.security.v1.Severity",
	"agntcy.dir.security.v1.ScannerType",
	"agntcy.dir.identity.v1.ClaimRole",
}

// newRecordEnv builds the environment a CEL policy is compiled in.
//
// Typed where there is a schema, dynamic where there is not. The facts in meta
// come from policy/v1 protos, so registering their descriptors makes a
// misspelled field a load-time error rather than an evaluation-time one, and
// makes an ordered enum comparison such as
// meta.scan.max_severity >= Severity.SEVERITY_MEDIUM check out.
//
// record stays dynamic on purpose. An OASF record has no compile-time schema
// here, so a policy must be able to read fields of schema versions this build
// does not know about. Declaring it as the google.protobuf.Struct it travels
// in would be more precise and would break that. indexed_at is optional
// because a record that has not been indexed has no such time.
//
// verdicts and now are deliberately absent. An evaluation that could read
// another policy's verdict, or the wall clock, is what RecordState is shaped
// to prevent.
func newRecordEnv() (*cel.Env, error) {
	env, err := cel.NewEnv(
		cel.OptionalTypes(),
		cel.Container(policyContainer),
		// Registering one message per file registers every type that file
		// declares, which is how the enums reached through the inputs become
		// resolvable.
		cel.Types(
			&policyv1.RecordMeta{},
			&securityv1.ScanReport{},
			&identityv1.Claim{},
		),
		cel.Abbrevs(celEnumAbbrevs...),
		cel.Variable(celVarRecord, cel.DynType),
		cel.Variable(celVarRecordCID, cel.StringType),
		cel.Variable(celVarIndexedAt, cel.OptionalType(cel.TimestampType)),
		cel.Variable(celVarMeta, cel.ObjectType(recordMetaType)),
	)
	if err != nil {
		return nil, fmt.Errorf("create CEL environment: %w", err)
	}

	return env, nil
}
