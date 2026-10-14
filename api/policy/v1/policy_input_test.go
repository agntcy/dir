// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"strings"
	"testing"

	policyv1 "github.com/agntcy/dir/api/policy/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// hasField reports whether m's descriptor declares a field of this name.
func hasField(m protoreflect.ProtoMessage, name string) bool {
	return m.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name)) != nil
}

// The evaluation layer must not see verdicts. An evaluation policy that could
// read another policy's verdict would turn a two-level dependency into a
// graph, bringing ordering, cycles, and verdicts computed against stale
// versions of their dependencies. The guarantee is the absence of the field,
// so it is worth asserting rather than trusting a comment.
func TestRecordStateCarriesNoVerdicts(t *testing.T) {
	t.Parallel()

	if hasField(&policyv1.RecordState{}, "verdicts") {
		t.Fatal("RecordState declares a verdicts field: evaluation policies could then read " +
			"each other's verdicts, and the dependency between the layers stops being acyclic")
	}
}

// The evaluation layer must not see wall-clock time. Verdicts are cached and
// invalidated when the facts underneath them change; the passage of time is
// not a change, so a time-dependent evaluation policy would be computed once
// and never revisited. Time comparisons belong in the operation layer.
func TestRecordStateCarriesNoNow(t *testing.T) {
	t.Parallel()

	if hasField(&policyv1.RecordState{}, "now") {
		t.Fatal("RecordState declares a now field: an evaluation policy could then depend on " +
			"wall-clock time, and nothing would ever mark its cached verdict stale")
	}
}

// The counterpart of the two rules above: both fields belong on the operation
// input, so testing only their absence would pass on an empty message.
func TestOperationInputCarriesVerdictsAndNow(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"verdicts", "now", "context", "record", "record_cid"} {
		if !hasField(&policyv1.OperationInput{}, name) {
			t.Fatalf("OperationInput is missing field %q", name)
		}
	}
}

// Evaluation is not an operation: it is decoupled from any point in a
// record's lifecycle and runs whenever the reconciler gets to it, which is
// why its results are cached, versioned and invalidated. Naming it as an
// operation is the conflation this schema exists to undo.
func TestOperationHasNoEvaluateValue(t *testing.T) {
	t.Parallel()

	values := policyv1.Operation(0).Descriptor().Values()
	for i := range values.Len() {
		if name := string(values.Get(i).Name()); strings.Contains(name, "EVALUATE") {
			t.Fatalf("Operation declares %s: evaluation produces facts and is configured as a "+
				"producer, not as an operation a policy decides on", name)
		}
	}
}

// ScanFacts.max_severity is documented as comparable with an ordinary
// threshold, which holds only while Severity ascends. Reordering it would
// silently invert every severity threshold in every policy, so the assumption
// is asserted here rather than left to the comment that relies on it.
func TestSeverityIsAscendingForThresholds(t *testing.T) {
	t.Parallel()

	ascending := []securityv1.Severity{
		securityv1.Severity_SEVERITY_UNSPECIFIED,
		securityv1.Severity_SEVERITY_NONE,
		securityv1.Severity_SEVERITY_INFO,
		securityv1.Severity_SEVERITY_LOW,
		securityv1.Severity_SEVERITY_MEDIUM,
		securityv1.Severity_SEVERITY_HIGH,
		securityv1.Severity_SEVERITY_CRITICAL,
	}

	for i := 1; i < len(ascending); i++ {
		if ascending[i] <= ascending[i-1] {
			t.Fatalf("Severity is not ascending: %s (%d) does not exceed %s (%d)",
				ascending[i], ascending[i], ascending[i-1], ascending[i-1])
		}
	}

	// An unscanned record must fail a threshold rather than pass it, which is
	// the safe direction for a prune criterion.
	if securityv1.Severity_SEVERITY_UNSPECIFIED >= securityv1.Severity_SEVERITY_MEDIUM {
		t.Fatal("SEVERITY_UNSPECIFIED must sort below any real severity, so that an unscanned " +
			"record fails a threshold rather than matching it")
	}
}
