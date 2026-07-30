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

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/protobuf/types/known/structpb"
)

// DefaultMCPCLIPath is the default binary name resolved via PATH.
const DefaultMCPCLIPath = "mcp-scanner"

var mcpLogger = logging.Logger("utils/scanner/mcp")

// MCPConfig holds configuration for the MCP runner.
type MCPConfig struct {
	// CLIPath is the path to the mcp-scanner binary. Defaults to DefaultMCPCLIPath.
	CLIPath string
}

// MCPRunner invokes mcp-scanner to scan MCP server source code.
// It clones the source repository, runs `mcp-scanner behavioral --raw`, and maps the output.
type MCPRunner struct {
	cfg MCPConfig
}

// NewMCPRunner creates an MCPRunner. If cfg.CLIPath is empty, DefaultMCPCLIPath is used.
func NewMCPRunner(cfg MCPConfig) *MCPRunner {
	if cfg.CLIPath == "" {
		cfg.CLIPath = DefaultMCPCLIPath
	}

	return &MCPRunner{cfg: cfg}
}

// Name returns the runner name.
func (r *MCPRunner) Name() string { return "mcp" }

// Run extracts the source-code URL from the record, clones the repository,
// runs `mcp-scanner behavioral --raw`, and returns mapped findings.
func (r *MCPRunner) Run(ctx context.Context, record *corev1.Record) (*ScanResult, error) {
	repoURL, subfolder := extractSourceInfo(record)
	if repoURL == "" {
		return &ScanResult{Skipped: true, SkippedReason: "no source-code locator found"}, nil
	}

	tmpDir, err := os.MkdirTemp("", "mcp-scan-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	if err := gitClone(ctx, repoURL, tmpDir); err != nil {
		mcpLogger.Warn("repository not cloneable, skipping scan", "url", repoURL, "error", err)

		return &ScanResult{
			Skipped:       true,
			SkippedReason: fmt.Sprintf("git clone failed: %s", repoURL),
		}, nil
	}

	scanDir := tmpDir
	if subfolder != "" {
		scanDir = filepath.Join(tmpDir, subfolder)
	}

	absDir, err := filepath.Abs(scanDir)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	rawOutput, err := runMCPScanner(ctx, r.cfg.CLIPath, absDir)
	if err != nil {
		return nil, fmt.Errorf("mcp-scanner: %w", err)
	}

	result, err := parseMCPOutput(rawOutput)
	if err != nil {
		return nil, err
	}

	// yara and readiness are zero-dependency analyzers (no third-party credentials
	// required) so they are always run. llm and api analyzers require third-party
	// credentials and are opt-in only; wiring them up is left as follow-up work.
	result.Analyzers = []string{"yara", "readiness"}

	return result, nil
}

// extractSourceInfo decodes the record and extracts the source-code repository URL
// and optional subfolder.
func extractSourceInfo(record *corev1.Record) (string, string) {
	if record == nil {
		return "", ""
	}

	decoded, err := record.Decode()
	if err != nil {
		return "", ""
	}

	if !decoded.HasV1() {
		return "", ""
	}

	v1 := decoded.GetV1()

	return extractSourceCodeURL(v1.GetLocators()), extractSubfolder(v1.GetModules())
}

func extractSourceCodeURL(locators []*typesv1.Locator) string {
	for _, loc := range locators {
		if loc.GetType() == "source_code" && len(loc.GetUrls()) > 0 {
			return loc.GetUrls()[0]
		}
	}

	return ""
}

// extractSubfolder walks modules[*].data.repository.subfolder. A module's
// data is itself the OASF mcp_data object, so repository sits directly under it.
func extractSubfolder(modules []*typesv1.Module) string {
	for _, mod := range modules {
		sf := getNestedString(mod.GetData(), "repository", "subfolder")
		if sf != "" {
			return sf
		}
	}

	return ""
}

// getNestedString traverses nested protobuf Structs by the given keys
// and returns the final value as a string, or "" if any step is missing.
func getNestedString(s *structpb.Struct, keys ...string) string {
	if s == nil || len(keys) == 0 {
		return ""
	}

	for i, k := range keys {
		v := s.GetFields()[k]
		if v == nil {
			return ""
		}

		if i == len(keys)-1 {
			return v.GetStringValue()
		}

		s = v.GetStructValue()
		if s == nil {
			return ""
		}
	}

	return ""
}

func gitClone(ctx context.Context, repoURL, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", repoURL, dest)
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	return nil
}

func runMCPScanner(ctx context.Context, cliPath, scanDir string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	// mcp-scanner requires global flags (--analyzers, --raw) to precede the
	// subcommand; placing them after `behavioral` is rejected by the CLI.
	cmd := exec.CommandContext(ctx, cliPath, "--analyzers", "yara,readiness", "--raw", "behavioral", scanDir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = buildMCPScannerEnv()

	if err := cmd.Run(); err != nil {
		mcpLogger.Warn("mcp-scanner stderr", "output", strings.TrimSpace(stderr.String()))

		return nil, fmt.Errorf("mcp-scanner exited with error: %w", err)
	}

	return stdout.Bytes(), nil
}

// buildMCPScannerEnv returns the parent env with MCP_SCANNER_LLM_* vars derived from
// the AZURE_* equivalents so CI config does not need to duplicate them.
func buildMCPScannerEnv() []string {
	env := os.Environ()
	env = appendEnvIfMissing(env, "MCP_SCANNER_LLM_API_KEY", os.Getenv("AZURE_OPENAI_API_KEY"))
	env = appendEnvIfMissing(env, "MCP_SCANNER_LLM_BASE_URL", os.Getenv("AZURE_OPENAI_BASE_URL"))
	env = appendEnvIfMissing(env, "MCP_SCANNER_LLM_MODEL", "azure/"+os.Getenv("AZURE_OPENAI_DEPLOYMENT"))
	env = appendEnvIfMissing(env, "MCP_SCANNER_LLM_API_VERSION", os.Getenv("AZURE_OPENAI_API_VERSION"))

	return env
}

func appendEnvIfMissing(env []string, key, fallback string) []string {
	if os.Getenv(key) != "" || fallback == "" {
		return env
	}

	return append(env, key+"="+fallback)
}

// --- output parsing ---

// mcpScannerResult represents a single tool result from `mcp-scanner behavioral --raw`.
type mcpScannerResult struct {
	ToolName string                       `json:"tool_name"`
	Status   string                       `json:"status"`
	IsSafe   bool                         `json:"is_safe"`
	Findings map[string]mcpAnalyzerResult `json:"findings"`
}

// mcpAnalyzerResult represents the output of a single analyzer within mcp-scanner.
type mcpAnalyzerResult struct {
	Severity      string   `json:"severity"`
	ThreatSummary string   `json:"threat_summary"`
	ThreatNames   []string `json:"threat_names"`
	TotalFindings int      `json:"total_findings"`
}

func parseMCPOutput(raw []byte) (*ScanResult, error) {
	raw = trimToJSON(raw)

	var results []mcpScannerResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("parse mcp-scanner output: %w", err)
	}

	if len(results) == 0 {
		return &ScanResult{Safe: true}, nil
	}

	var findings []Finding

	for _, r := range results {
		if r.IsSafe {
			continue
		}

		for analyzerName, ar := range r.Findings {
			severity := mapScannerSeverity(ar.Severity)
			msg := fmt.Sprintf("[%s] %s: %s", analyzerName, r.ToolName, ar.ThreatSummary)

			if len(ar.ThreatNames) > 0 {
				msg += " (" + strings.Join(ar.ThreatNames, ", ") + ")"
			}

			findings = append(findings, Finding{Severity: severity, Message: msg})
		}
	}

	return &ScanResult{Safe: len(findings) == 0, Findings: findings}, nil
}

// trimToJSON strips any leading non-JSON content by finding the first '['.
func trimToJSON(raw []byte) []byte {
	idx := bytes.IndexByte(raw, '[')
	if idx > 0 {
		return raw[idx:]
	}

	return raw
}
