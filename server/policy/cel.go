// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type celRule struct {
	expr string
	prg  cel.Program
}

// celProgram is a set of CEL expressions compiled in the record environment.
type celProgram struct {
	rules []celRule

	// version is a hash of the expressions, in order.
	version string
}

// celValidator evaluates a CEL policy. ValidateEvaluation reads a RecordState.
// ValidateOperation reads the record struct.
type celValidator struct {
	*celProgram
}

func newCELValidator(exprs []string) (*celValidator, error) {
	env, err := newRecordEnv()
	if err != nil {
		return nil, err
	}

	prog, err := compileCEL(exprs, env)
	if err != nil {
		return nil, err
	}

	return &celValidator{prog}, nil
}

func compileCEL(exprs []string, env *cel.Env) (*celProgram, error) {
	rules := make([]celRule, 0, len(exprs))
	for i, expr := range exprs {
		ast, iss := env.Compile(expr)
		if iss.Err() != nil {
			return nil, fmt.Errorf("compile CEL expressions[%d]: %w", i, iss.Err())
		}

		if ast.OutputType() != cel.BoolType {
			return nil, fmt.Errorf("CEL expressions[%d] must evaluate to bool, got %s", i, ast.OutputType())
		}

		prg, err := env.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("create CEL program for expressions[%d]: %w", i, err)
		}

		rules = append(rules, celRule{expr: expr, prg: prg})
	}

	return &celProgram{rules: rules, version: contentVersion(exprs...)}, nil
}

func (v *celValidator) ValidateEvaluation(ctx context.Context, state *policyv1.RecordState) (bool, []string, []string, error) {
	if err := ctx.Err(); err != nil {
		return false, nil, nil, fmt.Errorf("CEL validation canceled: %w", err)
	}

	if state.GetRecord() == nil {
		return false, []string{"record data is nil"}, nil, nil
	}

	return v.eval(ctx, recordActivation(state))
}

func (v *celValidator) ValidateOperation(ctx context.Context, data *structpb.Struct) (bool, []string, []string, error) {
	if err := ctx.Err(); err != nil {
		return false, nil, nil, fmt.Errorf("CEL validation canceled: %w", err)
	}

	if data == nil {
		return false, []string{"record data is nil"}, nil, nil
	}

	return v.eval(ctx, map[string]any{
		celVarRecord:    data.AsMap(),
		celVarRecordCID: "",
		celVarIndexedAt: types.OptionalNone,
		celVarMeta:      &policyv1.RecordMeta{},
	})
}

func (p *celProgram) eval(ctx context.Context, vars map[string]any) (bool, []string, []string, error) {
	var failures []string

	for _, rule := range p.rules {
		if err := ctx.Err(); err != nil {
			return false, nil, nil, fmt.Errorf("CEL validation canceled: %w", err)
		}

		out, _, err := rule.prg.Eval(vars)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", rule.expr, err.Error()))

			continue
		}

		if out.Type() != types.BoolType {
			return false, nil, nil, fmt.Errorf("CEL expression %q returned %s, want bool", rule.expr, out.Type())
		}

		valid, ok := out.Value().(bool)
		if !ok {
			return false, nil, nil, fmt.Errorf("CEL expression %q returned %T, want bool", rule.expr, out.Value())
		}

		if !valid {
			failures = append(failures, fmt.Sprintf("CEL expression evaluated to false: %s", rule.expr))
		}
	}

	if len(failures) > 0 {
		return false, failures, nil, nil
	}

	return true, nil, nil, nil
}

// recordActivation binds a RecordState the way the evaluation environment
// declares it. An unset meta reads as every fact unspecified. An unset
// indexed_at is absent rather than a zero time.
func recordActivation(state *policyv1.RecordState) map[string]any {
	meta := state.GetMeta()
	if meta == nil {
		meta = &policyv1.RecordMeta{}
	}

	return map[string]any{
		celVarRecord:    state.GetRecord().AsMap(),
		celVarRecordCID: state.GetRecordCid(),
		celVarIndexedAt: optionalTime(state.GetIndexedAt()),
		celVarMeta:      meta,
	}
}

func optionalTime(ts *timestamppb.Timestamp) any {
	if ts == nil {
		return types.OptionalNone
	}

	return types.OptionalOf(types.Timestamp{Time: ts.AsTime()})
}
