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

	// DisableEndpointScan turns off the live-endpoint phase, leaving only the
	// source-code scan. The zero value keeps the phase on. It is worth turning
	// off where the reconciler must not make outbound connections at all: the
	// phase dials third-party servers named in publisher-controlled record
	// data, and costs one mcp-scanner invocation per (endpoint, subcommand).
	DisableEndpointScan bool

	// AllowPrivateEndpoints permits endpoints on loopback, link-local,
	// private, and other reserved ranges. The zero value rejects them, so the
	// safe posture is the default one. Self-hosted deployments where the
	// directory and the MCP servers share a private network need this on.
	AllowPrivateEndpoints bool

	// AllowInsecureTransport permits plain http endpoints. It is separate from
	// AllowPrivateEndpoints on purpose: an operator who needs to reach a
	// private-range endpoint should not thereby also opt into cleartext scans
	// of arbitrary public hosts.
	AllowInsecureTransport bool

	// MaxEndpointsPerRecord bounds how many endpoints a single record can make
	// the reconciler dial. Zero or negative means
	// DefaultMaxEndpointsPerRecord. Validating where a scan may connect does
	// not bound how many connections one record can cause, and the list is
	// publisher-controlled.
	MaxEndpointsPerRecord int
}

// MCPRunner invokes mcp-scanner against an MCP server in two phases: it clones
// and scans the source repository, then scans any live endpoints the record
// declares, and merges the findings.
//
// The two phases fail differently on purpose. The source scan fails hard,
// because a scanner that cannot run at all is a scan we cannot vouch for. The
// endpoint phase is skip-with-warning, because endpoints are third-party
// servers that may be down, moved, or behind auth, and one unreachable
// endpoint must not fail an otherwise-clean source scan.
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

// Run scans the record's MCP server in two phases and merges the results:
// the source repository is cloned and scanned, then any live endpoints the
// record declares are scanned. Either phase may skip without the other doing
// so, and a record with only one of the two still produces a usable result.
func (r *MCPRunner) Run(ctx context.Context, record *corev1.Record) (*ScanResult, error) {
	sourceResult, err := r.runSourceScan(ctx, record)
	if err != nil {
		// Hard failure: the scanner could not run, so we cannot vouch for the
		// record. Returning the error discards any endpoint findings, which is
		// the intended trade - a partial scan reported as a whole one is worse
		// than no scan.
		return nil, err
	}

	if r.cfg.DisableEndpointScan {
		return sourceResult, nil
	}

	endpointResult := r.runEndpointScan(ctx, record)

	return merge([]*ScanResult{sourceResult, endpointResult}), nil
}

// runSourceScan clones the record's source repository and runs the
// source-code analyzers over it.
func (r *MCPRunner) runSourceScan(ctx context.Context, record *corev1.Record) (*ScanResult, error) {
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

	// The behavioral subparser declares its own --raw, whose default
	// overwrites a global one, so --raw must follow the subcommand to get
	// JSON rather than a human-readable summary.
	cmd := exec.CommandContext(ctx, cliPath, "--analyzers", "yara,readiness", "behavioral", scanDir, "--raw")
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

// mcpScannerResult represents a single result from `mcp-scanner --raw`.
// Findings stays raw because its shape varies; see analyzerFindings.
type mcpScannerResult struct {
	ToolName   string          `json:"tool_name"`
	ServerName string          `json:"server_name"`
	Status     string          `json:"status"`
	IsSafe     bool            `json:"is_safe"`
	Findings   json.RawMessage `json:"findings"`
}

// mcpAnalyzerResult represents the output of a single analyzer within mcp-scanner.
type mcpAnalyzerResult struct {
	Severity      string   `json:"severity"`
	ThreatSummary string   `json:"threat_summary"`
	ThreatNames   []string `json:"threat_names"`
	TotalFindings int      `json:"total_findings"`
}

// mcpListedFinding is one entry of the array form of findings.
type mcpListedFinding struct {
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Analyzer string `json:"analyzer"`
}

// mcpFinding is the shape-independent view of one analyzer's verdict.
type mcpFinding struct {
	analyzer string
	severity string
	summary  string
	names    []string
}

// analyzerFindings normalizes the two shapes mcp-scanner uses for findings:
// most subcommands return an object keyed by analyzer name, while
// `instructions` returns an array of entries that name their own analyzer.
func (r mcpScannerResult) analyzerFindings() ([]mcpFinding, error) {
	if len(r.Findings) == 0 {
		return nil, nil
	}

	var byAnalyzer map[string]mcpAnalyzerResult
	if err := json.Unmarshal(r.Findings, &byAnalyzer); err == nil {
		findings := make([]mcpFinding, 0, len(byAnalyzer))
		for name, a := range byAnalyzer {
			findings = append(findings, mcpFinding{
				analyzer: name,
				severity: a.Severity,
				summary:  a.ThreatSummary,
				names:    a.ThreatNames,
			})
		}

		return findings, nil
	}

	var listed []mcpListedFinding
	if err := json.Unmarshal(r.Findings, &listed); err != nil {
		return nil, fmt.Errorf("parse mcp-scanner findings: %w", err)
	}

	findings := make([]mcpFinding, 0, len(listed))
	for _, f := range listed {
		findings = append(findings, mcpFinding{
			analyzer: f.Analyzer,
			severity: f.Severity,
			summary:  f.Summary,
		})
	}

	return findings, nil
}

// subject names what a result is about. `instructions` results describe a
// server rather than a tool and so carry no tool_name.
func (r mcpScannerResult) subject() string {
	if r.ToolName != "" {
		return r.ToolName
	}

	return r.ServerName
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

		analyzerFindings, err := r.analyzerFindings()
		if err != nil {
			return nil, err
		}

		for _, af := range analyzerFindings {
			msg := fmt.Sprintf("[%s] %s: %s", af.analyzer, r.subject(), af.summary)

			if len(af.names) > 0 {
				msg += " (" + strings.Join(af.names, ", ") + ")"
			}

			findings = append(findings, Finding{Severity: mapScannerSeverity(af.severity), Message: msg})
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
