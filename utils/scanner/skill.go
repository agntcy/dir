// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/dir/utils/logging"
)

const (
	// DefaultSkillCLIPath is the default binary name resolved via PATH.
	DefaultSkillCLIPath = "skill-scanner"

	agentSkillsModuleName = "core/language_model/agentskills"
)

var skillLogger = logging.Logger("utils/scanner/skill")

// SkillConfig holds configuration for the Skill runner.
type SkillConfig struct {
	// CLIPath is the path to the skill-scanner binary. Defaults to DefaultSkillCLIPath.
	CLIPath string
}

// SkillRunner invokes skill-scanner to scan agent skill directories.
// It reads the skill bundle stored in the record artifact, extracts it to a
// temporary directory, and runs `skill-scanner scan`.
type SkillRunner struct {
	cfg SkillConfig
}

// NewSkillRunner creates a SkillRunner. If cfg.CLIPath is empty, DefaultSkillCLIPath is used.
func NewSkillRunner(cfg SkillConfig) *SkillRunner {
	if cfg.CLIPath == "" {
		cfg.CLIPath = DefaultSkillCLIPath
	}

	return &SkillRunner{cfg: cfg}
}

// Name returns the runner name.
func (r *SkillRunner) Name() string { return "skill" }

// Run extracts the agentskills bundle from the record, writes it to a temporary
// directory, and runs `skill-scanner scan` against it.
func (r *SkillRunner) Run(ctx context.Context, record *corev1.Record) (*ScanResult, error) {
	raw := skillArtifactBytes(record)
	if raw == nil {
		return &ScanResult{Skipped: true, SkippedReason: "no agentskills module found"}, nil
	}

	tmpDir, err := os.MkdirTemp("", "skill-scan-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	if err := exportfmt.ExtractSkillBundleArchive(raw, tmpDir); err != nil {
		return nil, fmt.Errorf("extract skill content: %w", err)
	}

	rawOutput, err := runSkillScanner(ctx, r.cfg.CLIPath, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("skill-scanner: %w", err)
	}

	result, err := parseSkillOutput(rawOutput)
	if err != nil {
		return nil, err
	}

	result.Version = getVersion(r.cfg.CLIPath, "--version")
	// static, bytecode, and pipeline run by default; behavioral and trigger are
	// explicitly enabled via --use-behavioral and --use-trigger.
	result.Analyzers = []string{"behavioral", "bytecode", "pipeline", "static", "trigger"}

	return result, nil
}

// skillArtifactBytes finds the agentskills module in the record and returns the
// raw artifact bytes (base64-decoded from modules[*].artifact.data).
// Returns nil if the module is absent or the artifact is unreadable.
func skillArtifactBytes(record *corev1.Record) []byte {
	data := record.GetData()
	if data == nil {
		return nil
	}

	for _, modVal := range data.GetFields()["modules"].GetListValue().GetValues() {
		mod := modVal.GetStructValue()
		if mod == nil {
			continue
		}

		if mod.GetFields()["name"].GetStringValue() != agentSkillsModuleName {
			continue
		}

		encoded := mod.GetFields()["artifact"].GetStructValue().GetFields()["data"].GetStringValue()
		if encoded == "" {
			return nil
		}

		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			skillLogger.Warn("failed to base64-decode skill artifact", "error", err)

			return nil
		}

		return raw
	}

	return nil
}

func runSkillScanner(ctx context.Context, cliPath, scanDir string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, cliPath, "scan", scanDir,
		"--use-behavioral",
		"--use-trigger",
		"--detailed",
		"--format", "json",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = buildSkillScannerEnv()

	if err := cmd.Run(); err != nil {
		skillLogger.Warn("skill-scanner stderr", "output", strings.TrimSpace(stderr.String()))

		return nil, fmt.Errorf("skill-scanner exited with error: %w", err)
	}

	return stdout.Bytes(), nil
}

// buildSkillScannerEnv returns the parent env with SKILL_SCANNER_LLM_* vars derived
// from the AZURE_* equivalents so CI config does not need to duplicate them.
func buildSkillScannerEnv() []string {
	env := os.Environ()
	env = appendEnvIfMissing(env, "SKILL_SCANNER_LLM_API_KEY", os.Getenv("AZURE_OPENAI_API_KEY"))
	env = appendEnvIfMissing(env, "SKILL_SCANNER_LLM_BASE_URL", os.Getenv("AZURE_OPENAI_BASE_URL"))
	env = appendEnvIfMissing(env, "SKILL_SCANNER_LLM_MODEL", "azure/"+os.Getenv("AZURE_OPENAI_DEPLOYMENT"))
	env = appendEnvIfMissing(env, "SKILL_SCANNER_LLM_API_VERSION", os.Getenv("AZURE_OPENAI_API_VERSION"))
	env = appendEnvIfMissing(env, "SKILL_SCANNER_LLM_PROVIDER", "openai-compatible")

	return env
}

// --- output parsing ---

type skillScanResult struct {
	IsSafe   bool           `json:"is_safe"`
	Findings []skillFinding `json:"findings"`
}

type skillFinding struct {
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

func parseSkillOutput(raw []byte) (*ScanResult, error) {
	raw = trimToSkillJSON(raw)

	// skill-scanner scan outputs a single object; scan-all outputs an array.
	// Normalise to a slice so we can handle both.
	results, err := parseSkillJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("parse skill-scanner output: %w", err)
	}

	if len(results) == 0 {
		return &ScanResult{Safe: true}, nil
	}

	var findings []Finding

	for _, r := range results {
		for _, f := range r.Findings {
			findings = append(findings, Finding{
				Severity: mapSkillSeverity(f.Severity),
				Message:  fmt.Sprintf("[%s] %s: %s", f.RuleID, f.Category, f.Description),
			})
		}
	}

	allSafe := true

	for _, r := range results {
		if !r.IsSafe {
			allSafe = false

			break
		}
	}

	return &ScanResult{Safe: allSafe, Findings: findings}, nil
}

// trimToSkillJSON strips leading non-JSON bytes by finding the first '{' or '['.
// Unlike the MCP trimToJSON (which only looks for '['), skill-scanner emits a
// top-level object so we must also accept '{'.
func trimToSkillJSON(raw []byte) []byte {
	for i, b := range raw {
		if b == '{' || b == '[' {
			return raw[i:]
		}
	}

	return raw
}

// parseSkillJSON decodes either a single skillScanResult object or an array of them.
func parseSkillJSON(raw []byte) ([]skillScanResult, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}

	if raw[0] == '[' {
		var results []skillScanResult
		if err := json.Unmarshal(raw, &results); err != nil {
			return nil, fmt.Errorf("unmarshal skill-scanner array output: %w", err)
		}

		return results, nil
	}

	var result skillScanResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal skill-scanner object output: %w", err)
	}

	return []skillScanResult{result}, nil
}

func mapSkillSeverity(s string) FindingSeverity {
	switch strings.ToUpper(s) {
	case "CRITICAL", "HIGH":
		return SeverityError
	case "MEDIUM":
		return SeverityWarning
	default:
		return SeverityInfo
	}
}
