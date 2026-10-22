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
	"google.golang.org/protobuf/types/known/structpb"
)

const allowQuerySfx = ".allow"

type opaValidator struct {
	name  string
	query rego.PreparedEvalQuery

	// version is a hash of the policy file as it was loaded.
	version string
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
		name:    policyName(mod, filename),
		version: contentVersion(string(src)),
		query:   prepared,
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

func (v *opaValidator) ValidateOperation(ctx context.Context, data *structpb.Struct) (bool, []string, []string, error) {
	if data == nil {
		return false, []string{"record data is required"}, nil, nil
	}

	return v.decide(ctx, data.AsMap())
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
