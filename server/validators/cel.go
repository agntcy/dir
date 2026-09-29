// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"google.golang.org/protobuf/types/known/structpb"
)

type celRule struct {
	expr string
	prg  cel.Program
}

// celValidator evaluates inline CEL expressions against record data.
// Each expression is compiled once at construction and must return bool.
// The record is bound as the `record` variable (a map of the OASF fields).
// Validation fails if any expression is false or errors.
type celValidator struct {
	rules []celRule
}

func newCELValidator(entry validatorsconfig.Validator) (corev1.Validator, error) {
	exprs, err := entry.ConfigStrings(validatorsconfig.ConfigKeyExpressions)
	if err != nil {
		return nil, fmt.Errorf("config.expressions: %w", err)
	}

	env, err := cel.NewEnv(
		cel.Variable("record", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("create CEL environment: %w", err)
	}

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

	return &celValidator{rules: rules}, nil
}

func (v *celValidator) ValidateRecord(ctx context.Context, data *structpb.Struct) (bool, []string, []string, error) {
	if err := ctx.Err(); err != nil {
		return false, nil, nil, fmt.Errorf("CEL validation canceled: %w", err)
	}

	if data == nil {
		return false, []string{"record data is nil"}, nil, nil
	}

	vars := map[string]any{"record": data.AsMap()}

	var failures []string

	for _, rule := range v.rules {
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
