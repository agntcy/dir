// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
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

// celValidator evaluates a CEL policy.
//
// Its layer is fixed when it is built, because that is when its expressions
// were compiled: the one method matching that layer is the only one it can
// answer.
type celValidator struct {
	*celProgram
	layer layer
}

func newCELValidator(exprs []string, l layer) (*celValidator, error) {
	env, err := celEnv(l)
	if err != nil {
		return nil, err
	}

	prog, err := compileCEL(exprs, env)
	if err != nil {
		return nil, err
	}

	return &celValidator{celProgram: prog, layer: l}, nil
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
	if v.layer != layerEvaluation {
		return false, nil, nil, fmt.Errorf("CEL policy compiled for the %s layer cannot produce a verdict", v.layer)
	}

	if err := ctx.Err(); err != nil {
		return false, nil, nil, fmt.Errorf("CEL validation canceled: %w", err)
	}

	if state.GetRecord() == nil {
		return false, []string{"record data is nil"}, nil, nil
	}

	return v.eval(ctx, stateActivation(state))
}

func (v *celValidator) ValidateOperation(ctx context.Context, input *policyv1.OperationInput) (bool, []string, []string, error) {
	if v.layer != layerOperation {
		return false, nil, nil, fmt.Errorf("CEL policy compiled for the %s layer cannot decide an operation", v.layer)
	}

	if err := ctx.Err(); err != nil {
		return false, nil, nil, fmt.Errorf("CEL validation canceled: %w", err)
	}

	if input.GetRecord() == nil {
		return false, []string{"record data is nil"}, nil, nil
	}

	vars, err := operationActivation(input)
	if err != nil {
		return false, nil, nil, err
	}

	return v.eval(ctx, vars)
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

// stateActivation binds a RecordState the way the evaluation environment
// declares it. An unset meta reads as every fact unspecified. An unset
// indexed_at is absent rather than a zero time.
func stateActivation(state *policyv1.RecordState) map[string]any {
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

// operationActivation binds an OperationInput the way the operation
// environment declares it.
func operationActivation(input *policyv1.OperationInput) (map[string]any, error) {
	if err := checkOperationInput(input); err != nil {
		return nil, err
	}

	return map[string]any{
		celVarRecord:    input.GetRecord().AsMap(),
		celVarRecordCID: input.GetRecordCid(),
		celVarContext:   input.GetContext(),
		celVarNow:       types.Timestamp{Time: input.GetNow().AsTime()},
	}, nil
}

func optionalTime(ts *timestamppb.Timestamp) any {
	if ts == nil {
		return types.OptionalNone
	}

	return types.OptionalOf(types.Timestamp{Time: ts.AsTime()})
}
