// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

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

// newPolicyEnv builds the environment CEL policy expressions are compiled in.
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
// in would be more precise and would break that.
func newPolicyEnv() (*cel.Env, error) {
	opts := []cel.EnvOption{
		cel.Container(policyContainer),
		// Registering one message per file registers every type that file
		// declares, which is how the enums reached through meta become
		// resolvable.
		cel.Types(
			&policyv1.RecordMeta{},
			&securityv1.ScanReport{},
			&identityv1.Claim{},
		),
		cel.Abbrevs(celEnumAbbrevs...),
		cel.Variable(celVarRecord, cel.DynType),
		cel.Variable(celVarMeta, cel.ObjectType(recordMetaType)),
	}

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, fmt.Errorf("create CEL environment: %w", err)
	}

	return env, nil
}
