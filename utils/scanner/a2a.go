// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/protobuf/types/known/structpb"
)

// DefaultA2ACLIPath is the default binary name resolved via PATH.
const DefaultA2ACLIPath = "a2a-scanner"

var a2aLogger = logging.Logger("utils/scanner/a2a")

// A2AConfig holds configuration for the A2A runner.
type A2AConfig struct {
	// CLIPath is the path to the a2a-scanner binary. Defaults to DefaultA2ACLIPath.
	CLIPath string
}

// A2ARunner invokes a2a-scanner to scan an Agent-to-Agent (A2A) protocol
// AgentCard. It reads the a2a module's card_data stored on the record, writes
// the card to a temporary file, and runs `a2a-scanner scan-card`.
type A2ARunner struct {
	cfg A2AConfig
}

// NewA2ARunner creates an A2ARunner. If cfg.CLIPath is empty, DefaultA2ACLIPath is used.
func NewA2ARunner(cfg A2AConfig) *A2ARunner {
	if cfg.CLIPath == "" {
		cfg.CLIPath = DefaultA2ACLIPath
	}

	return &A2ARunner{cfg: cfg}
}

// Name returns the runner name.
func (r *A2ARunner) Name() string { return "a2a" }

// Run extracts the A2A AgentCard from the record, writes it to a temporary
// file, and runs `a2a-scanner scan-card` against it.
func (r *A2ARunner) Run(ctx context.Context, record *corev1.Record) (*ScanResult, error) {
	card, ok := extractA2ACard(record)
	if !ok {
		return &ScanResult{Skipped: true, SkippedReason: "no A2A agent card found"}, nil
	}

	cardJSON, err := json.Marshal(card)
	if err != nil {
		return nil, fmt.Errorf("marshal A2A agent card: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "a2a-scan-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	cardPath := filepath.Join(tmpDir, "agent-card.json")
	if err := os.WriteFile(cardPath, cardJSON, 0o600); err != nil { //nolint:mnd
		return nil, fmt.Errorf("write agent card: %w", err)
	}

	outputPath := filepath.Join(tmpDir, "scan-result.json")

	if err := runA2AScanner(ctx, r.cfg.CLIPath, cardPath, outputPath); err != nil {
		return nil, fmt.Errorf("a2a-scanner: %w", err)
	}

	rawOutput, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read a2a-scanner output: %w", err)
	}

	result, err := parseA2AOutput(rawOutput)
	if err != nil {
		return nil, err
	}

	result.Version = getVersion(r.cfg.CLIPath, "--version")
	// heuristic, spec, and yara are zero-dependency analyzers (no third-party
	// credentials or live endpoint required) so they are always run. llm requires
	// third-party credentials and endpoint requires a reachable A2A server; both are
	// opt-in only and wiring them up is left as follow-up work (mirroring the mcp
	// runner's llm/api analyzers and the separate RemoteRunner for live endpoints).
	result.Analyzers = []string{"heuristic", "spec", "yara"}

	return result, nil
}

// extractA2ACard decodes the record and returns the A2A AgentCard stored in
// the a2a module's card_data field, if present. Per the OASF a2a_data schema
// (https://schema.oasf.outshift.com/1.0.0/objects/a2a_data) the module's data
// object holds card_data directly, so the lookup is data.card_data — not a
// nested data.a2a_data.card_data. The bool return indicates whether a card was
// found.
func extractA2ACard(record *corev1.Record) (map[string]any, bool) {
	if record == nil {
		return nil, false
	}

	decoded, err := record.Decode()
	if err != nil {
		return nil, false
	}

	if !decoded.HasV1() {
		return nil, false
	}

	for _, mod := range decoded.GetV1().GetModules() {
		if card := getNestedStructValue(mod.GetData(), "card_data"); card != nil {
			return card.AsMap(), true
		}
	}

	return nil, false
}

// getNestedStructValue traverses nested protobuf Structs by the given keys
// and returns the struct value at the final key, or nil if any step is
// missing or not itself a struct.
func getNestedStructValue(s *structpb.Struct, keys ...string) *structpb.Struct {
	cur := s

	for _, k := range keys {
		if cur == nil {
			return nil
		}

		v := cur.GetFields()[k]
		if v == nil {
			return nil
		}

		cur = v.GetStructValue()
	}

	return cur
}

func runA2AScanner(ctx context.Context, cliPath, cardPath, outputPath string) error {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, cliPath, "scan-card", cardPath,
		"--analyzers", "heuristic,spec,yara",
		"--output", outputPath,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = buildA2AScannerEnv()

	if err := cmd.Run(); err != nil {
		a2aLogger.Warn("a2a-scanner stderr", "output", strings.TrimSpace(stderr.String()))

		return fmt.Errorf("a2a-scanner exited with error: %w", err)
	}

	return nil
}

// buildA2AScannerEnv returns the parent env with A2A_SCANNER_LLM_* vars derived
// from the AZURE_* equivalents so CI config does not need to duplicate them.
// These are only consumed if the llm analyzer is enabled in the future; they
// are harmless no-ops while llm stays out of the default analyzer set.
func buildA2AScannerEnv() []string {
	env := os.Environ()
	env = appendEnvIfMissing(env, "A2A_SCANNER_LLM_API_KEY", os.Getenv("AZURE_OPENAI_API_KEY"))
	env = appendEnvIfMissing(env, "A2A_SCANNER_LLM_BASE_URL", os.Getenv("AZURE_OPENAI_BASE_URL"))
	env = appendEnvIfMissing(env, "A2A_SCANNER_LLM_MODEL", "azure/"+os.Getenv("AZURE_OPENAI_DEPLOYMENT"))
	env = appendEnvIfMissing(env, "A2A_SCANNER_LLM_API_VERSION", os.Getenv("AZURE_OPENAI_API_VERSION"))
	env = appendEnvIfMissing(env, "A2A_SCANNER_LLM_PROVIDER", "openai-compatible")

	return env
}

// --- output parsing ---

// a2aScanResult represents the single-object result of `a2a-scanner scan-card`.
//
// NOTE: a2a-scanner's JSON output schema is not fully documented upstream at
// the time of writing (see https://cisco-ai-defense.github.io/docs/a2a-scanner).
// The field names below follow the same is_safe/findings[severity,category,
// rule_id,description,remediation] contract already exposed by skill-scanner,
// the other single-artifact scanner in this family. If the installed
// a2a-scanner binary uses different field names, update this struct and
// parseA2AOutput to match — the CLI invocation and record-extraction logic
// above are unaffected by this assumption.
type a2aScanResult struct {
	IsSafe   bool         `json:"is_safe"`
	Findings []a2aFinding `json:"findings"`
}

type a2aFinding struct {
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

func parseA2AOutput(raw []byte) (*ScanResult, error) {
	raw = bytes.TrimSpace(trimToA2AJSON(raw))

	if len(raw) == 0 {
		return &ScanResult{Safe: true}, nil
	}

	var result a2aScanResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse a2a-scanner output: %w", err)
	}

	var findings []Finding

	for _, f := range result.Findings {
		findings = append(findings, Finding{
			Severity: mapA2ASeverity(f.Severity),
			Message:  fmt.Sprintf("[%s] %s: %s", f.RuleID, f.Category, f.Description),
		})
	}

	return &ScanResult{Safe: result.IsSafe, Findings: findings}, nil
}

// trimToA2AJSON strips leading non-JSON bytes by finding the first '{' or '['.
func trimToA2AJSON(raw []byte) []byte {
	if idx := bytes.IndexAny(raw, "{["); idx > 0 {
		return raw[idx:]
	}

	return raw
}

func mapA2ASeverity(s string) FindingSeverity {
	switch strings.ToUpper(s) {
	case "CRITICAL", "HIGH":
		return SeverityError
	case "MEDIUM":
		return SeverityWarning
	default:
		return SeverityInfo
	}
}
