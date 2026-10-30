// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	policyv1 "github.com/agntcy/dir/api/policy/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const allowQuerySfx = ".allow"

type opaValidator struct {
	name  string
	query rego.PreparedEvalQuery

	// version is a hash of the policy file as it was loaded.
	version string

	// factGroups are the metadata groups the module reads.
	factGroups []string

	// verdicts is what the module says about the verdicts it reads.
	verdicts verdictRefs

	// readsRecord is whether the module reads the record's content.
	readsRecord bool
}

func newOPAValidator(ctx context.Context, policyDir, filename string) (*opaValidator, error) {
	if err := policyconfig.ValidatePolicyFile(filename); err != nil {
		return nil, fmt.Errorf("invalid policy file: %w", err)
	}

	if strings.TrimSpace(policyDir) == "" {
		return nil, fmt.Errorf("policy directory is required for provider %q", policyconfig.ProviderOPA)
	}

	path := filepath.Join(policyDir, filename)

	src, err := os.ReadFile(path) //nolint:gosec // Policy path is operator-configured; filename is a bare basename.
	if err != nil {
		return nil, fmt.Errorf("read policy file %s: %w", filename, err)
	}

	mod, err := ast.ParseModuleWithOpts(filename, string(src), ast.ParserOptions{
		ProcessAnnotation: true,
		RegoVersion:       ast.RegoV1,
	})
	if err != nil {
		return nil, fmt.Errorf("parse policy file %s: %w", filename, err)
	}

	if mod.Package == nil {
		return nil, fmt.Errorf("policy file %s: package is required", filename)
	}

	query := mod.Package.Path.String() + allowQuerySfx

	prepared, err := rego.New(
		rego.Query(query),
		rego.ParsedModule(mod),
		rego.StrictBuiltinErrors(true),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile policy file %s: %w", filename, err)
	}

	return &opaValidator{
		name:        policyName(mod, filename),
		version:     contentVersion(string(src)),
		query:       prepared,
		factGroups:  opaFactGroups(mod),
		verdicts:    opaVerdictKeys(mod),
		readsRecord: opaReadsRecord(mod),
	}, nil
}

func (v *opaValidator) ValidateEvaluation(ctx context.Context, state *policyv1.RecordState) (bool, []string, []string, error) {
	if state.GetRecord() == nil {
		return false, []string{"record data is required"}, nil, nil
	}

	input, err := stateAsInput(state)
	if err != nil {
		return false, nil, nil, fmt.Errorf("encode input for policy %q: %w", v.name, err)
	}

	return v.decide(ctx, input)
}

func (v *opaValidator) ValidateOperation(ctx context.Context, opInput *policyv1.OperationInput) (bool, []string, []string, error) {
	// Only a policy that reads the record needs one; see the same check in
	// the CEL validator.
	if v.readsRecord && opInput.GetRecord() == nil {
		return false, []string{"record data is required"}, nil, nil
	}

	input, err := operationAsInput(opInput)
	if err != nil {
		return false, nil, nil, fmt.Errorf("encode input for policy %q: %w", v.name, err)
	}

	return v.decide(ctx, input)
}

func (v *opaValidator) decide(ctx context.Context, input map[string]any) (bool, []string, []string, error) {
	rs, err := v.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, nil, nil, fmt.Errorf("evaluate policy %q: %w", v.name, err)
	}

	if rs.Allowed() {
		return true, nil, nil, nil
	}

	return false, []string{fmt.Sprintf("record rejected by policy %q", v.name)}, nil, nil
}

// stateAsInput is the Rego input document for a RecordState.
//
// The record is copied with AsMap so its shape matches the struct the rest of
// the process already uses. meta is encoded from the proto, with proto field
// names, so a policy reads input.meta.signature.status.
func stateAsInput(state *policyv1.RecordState) (map[string]any, error) {
	input := map[string]any{
		"record": state.GetRecord().AsMap(),
	}

	if cid := state.GetRecordCid(); cid != "" {
		input["record_cid"] = cid
	}

	if ts := state.GetIndexedAt(); ts != nil {
		input["indexed_at"] = ts.AsTime().UTC().Format(time.RFC3339Nano)
	}

	if meta := state.GetMeta(); meta != nil {
		encoded, err := protoToMap(meta)
		if err != nil {
			return nil, err
		}

		input["meta"] = encoded
	}

	return input, nil
}

// operationAsInput is the Rego input document for an OperationInput.
//
// Shaped like stateAsInput so that a record field is reached the same way in
// both layers: the record is nested under its own key rather than merged into
// the root, which also keeps a record that happens to carry a top-level field
// named context or now from shadowing ours.
//
// context is encoded from the proto with proto field names, so a policy reads
// input.context.operation, and reads it as the enum's name: the string
// "OPERATION_PUSH", not a number. verdicts and indexed_at stay out for the
// same reason the operation CEL environment does not declare them.
// verdictsAsInput encodes the verdict map the way the rest of the input is
// encoded, so that a Rego policy reads VerdictStatus as its name rather than
// as a number.
//
// An empty map is encoded as an empty map rather than left out: input.verdicts
// is undefined if it is missing, and a rule reading it would simply not fire,
// which reads as a passing policy.
func verdictsAsInput(verdicts map[string]*policyv1.Verdict) (map[string]any, error) {
	out := make(map[string]any, len(verdicts))

	for id, verdict := range verdicts {
		encoded, err := protoToMap(verdict)
		if err != nil {
			return nil, err
		}

		out[id] = encoded
	}

	return out, nil
}

func operationAsInput(opInput *policyv1.OperationInput) (map[string]any, error) {
	if err := checkOperationInput(opInput); err != nil {
		return nil, err
	}

	encodedCtx, err := protoToMap(opInput.GetContext())
	if err != nil {
		return nil, err
	}

	verdicts, err := verdictsAsInput(opInput.GetVerdicts())
	if err != nil {
		return nil, err
	}

	input := map[string]any{
		"record":   opInput.GetRecord().AsMap(),
		"verdicts": verdicts,
		"context":  encodedCtx,
		"now":      opInput.GetNow().AsTime().UTC().Format(time.RFC3339Nano),
	}

	if cid := opInput.GetRecordCid(); cid != "" {
		input["record_cid"] = cid
	}

	return input, nil
}

func protoToMap(msg proto.Message) (map[string]any, error) {
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("encode %T: %w", msg, err)
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode %T: %w", msg, err)
	}

	return out, nil
}

func policyName(mod *ast.Module, filename string) string {
	for _, annot := range mod.Annotations {
		if annot == nil || annot.Scope != "package" {
			continue
		}

		if title := strings.TrimSpace(annot.Title); title != "" {
			return title
		}
	}

	return strings.TrimSuffix(filename, ".rego")
}
