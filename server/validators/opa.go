// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"google.golang.org/protobuf/types/known/structpb"
)

const allowQuerySfx = ".allow"

type opaValidator struct {
	name    string
	version string
	query   rego.PreparedEvalQuery
}

func newOPAValidator(ctx context.Context, policyDir, filename string) (*opaValidator, error) {
	if err := validatorsconfig.ValidatePolicyFile(filename); err != nil {
		return nil, fmt.Errorf("invalid policy file: %w", err)
	}

	if strings.TrimSpace(policyDir) == "" {
		return nil, fmt.Errorf("policy directory is required for provider %q", validatorsconfig.ProviderOPA)
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

// contentVersion is a hash of the policy file as it was loaded.
func (v *opaValidator) contentVersion() string { return v.version }

func (v *opaValidator) ValidateRecord(ctx context.Context, data *structpb.Struct) (bool, []string, []string, error) {
	if data == nil {
		return false, []string{"record data is required"}, nil, nil
	}

	rs, err := v.query.Eval(ctx, rego.EvalInput(data.AsMap()))
	if err != nil {
		return false, nil, nil, fmt.Errorf("evaluate policy %q: %w", v.name, err)
	}

	if rs.Allowed() {
		return true, nil, nil, nil
	}

	return false, []string{fmt.Sprintf("record rejected by policy %q", v.name)}, nil, nil
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
