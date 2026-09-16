// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// TriggerIngest is the metadata trigger for admission checks on record persist.
	TriggerIngest = "ingest"

	regoExt        = ".rego"
	triggersKey    = "triggers"
	allowQuerySfx  = ".allow"
	scopePackage   = "package"
	defaultRegoVer = ast.RegoV1
)

var (
	logger = logging.Logger("policy")

	_ Evaluator = (*Engine)(nil)
	_ Evaluator = nop{}
)

// Evaluator decides whether a record may be ingested.
type Evaluator interface {
	EvaluateIngest(ctx context.Context, source string, record *corev1.Record) error
}

// Engine is a compiled set of file-based OPA policies.
type Engine struct {
	policies []compiledPolicy
}

type compiledPolicy struct {
	name     string
	filename string
	query    rego.PreparedEvalQuery
}

type nop struct{}

func (nop) EvaluateIngest(context.Context, string, *corev1.Record) error { return nil }

// Nop returns an evaluator that admits every record.
func Nop() Evaluator {
	return nop{}
}

// Load compiles .rego files from cfg.Dir. Disabled config yields a no-op evaluator.
// Startup fails if the directory cannot be read or a policy file is invalid.
func Load(ctx context.Context, cfg config.Config) (Evaluator, error) {
	if !cfg.Enabled {
		return Nop(), nil
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	entries, err := os.ReadDir(cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("read policy directory %s: %w", cfg.Dir, err)
	}

	engine := &Engine{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), regoExt) {
			continue
		}

		path := filepath.Join(cfg.Dir, entry.Name())

		compiled, skip, err := compileFile(ctx, path, entry.Name())
		if err != nil {
			return nil, err
		}

		if skip {
			continue
		}

		engine.policies = append(engine.policies, compiled)
	}

	slices.SortFunc(engine.policies, func(a, b compiledPolicy) int {
		return strings.Compare(a.filename, b.filename)
	})

	logger.Info("Content policies loaded", "dir", cfg.Dir, "ingest_policies", len(engine.policies))

	return engine, nil
}

// EvaluateIngest runs every ingest-triggered policy against the record.
// The first deny wins. Evaluation errors fail closed.
func (e *Engine) EvaluateIngest(ctx context.Context, source string, record *corev1.Record) error {
	if e == nil || len(e.policies) == 0 {
		return nil
	}

	input, err := ingestInput(source, record)
	if err != nil {
		logger.Error("Failed to build policy input", "error", err, "source", source)

		return status.Errorf(codes.FailedPrecondition, "content policy evaluation failed: %v", err)
	}

	for _, p := range e.policies {
		if err := p.eval(ctx, input); err != nil {
			return err
		}
	}

	return nil
}

func (p compiledPolicy) eval(ctx context.Context, input map[string]any) error {
	rs, err := p.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		logger.Error("Content policy evaluation error", "policy", p.name, "file", p.filename, "error", err)

		return status.Errorf(codes.FailedPrecondition, "content policy %q evaluation failed: %v", p.name, err)
	}

	if rs.Allowed() {
		return nil
	}

	logger.Warn("Record rejected by content policy", "policy", p.name, "file", p.filename)

	return status.Errorf(codes.FailedPrecondition, "record rejected by policy %q", p.name)
}

func compileFile(ctx context.Context, path, filename string) (compiledPolicy, bool, error) {
	src, err := os.ReadFile(path) //nolint:gosec // Policy path is operator-configured, not user input.
	if err != nil {
		return compiledPolicy{}, false, fmt.Errorf("read policy file %s: %w", path, err)
	}

	mod, err := ast.ParseModuleWithOpts(filename, string(src), ast.ParserOptions{
		ProcessAnnotation: true,
		RegoVersion:       defaultRegoVer,
	})
	if err != nil {
		return compiledPolicy{}, false, fmt.Errorf("parse policy file %s: %w", filename, err)
	}

	triggers, hasTriggers := packageTriggers(mod)
	if !hasTriggers {
		return compiledPolicy{}, false, fmt.Errorf("policy file %s: package metadata custom.triggers is required", filename)
	}

	if !slices.Contains(triggers, TriggerIngest) {
		logger.Info("Skipping policy without ingest trigger", "file", filename, "triggers", triggers)

		return compiledPolicy{}, true, nil
	}

	query := mod.Package.Path.String() + allowQuerySfx

	prepared, err := rego.New(
		rego.Query(query),
		rego.ParsedModule(mod),
		rego.StrictBuiltinErrors(true),
	).PrepareForEval(ctx)
	if err != nil {
		return compiledPolicy{}, false, fmt.Errorf("compile policy file %s: %w", filename, err)
	}

	return compiledPolicy{
		name:     policyName(mod, filename),
		filename: filename,
		query:    prepared,
	}, false, nil
}

func ingestInput(source string, record *corev1.Record) (map[string]any, error) {
	if record == nil || record.GetData() == nil {
		return nil, errors.New("record data is required")
	}

	return map[string]any{
		"trigger": TriggerIngest,
		"source":  source,
		"record":  record.GetData().AsMap(),
	}, nil
}

func packageTriggers(mod *ast.Module) ([]string, bool) {
	if mod == nil {
		return nil, false
	}

	for _, annot := range mod.Annotations {
		if annot == nil || annot.Scope != scopePackage {
			continue
		}

		raw, ok := annot.Custom[triggersKey]
		if !ok {
			continue
		}

		return asStrings(raw), true
	}

	return nil, false
}

func policyName(mod *ast.Module, filename string) string {
	for _, annot := range mod.Annotations {
		if annot == nil || annot.Scope != scopePackage {
			continue
		}

		if title := strings.TrimSpace(annot.Title); title != "" {
			return title
		}
	}

	return strings.TrimSuffix(filename, regoExt)
}

func asStrings(raw any) []string {
	switch v := raw.(type) {
	case string:
		if v == "" {
			return nil
		}

		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok || s == "" {
				continue
			}

			out = append(out, s)
		}

		return out
	case []string:
		return slices.Clone(v)
	default:
		return nil
	}
}
