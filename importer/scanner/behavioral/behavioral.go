// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
	"github.com/agntcy/dir/importer/scanner/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/protobuf/types/known/structpb"
)

var logger = logging.Logger("importer/scanner/behavioral")

// Scanner scans MCP server source code using mcp-scanner's behavioral mode.
// It clones the source repository, runs `mcp-scanner behavioral --raw`,
// and maps the findings to the scanner result format.
type Scanner struct {
	cfg scannerconfig.Config
}

// New creates a behavioral Scanner with the given config.
func New(cfg scannerconfig.Config) *Scanner {
	return &Scanner{cfg: cfg}
}

// Name returns the scanner name.
func (s *Scanner) Name() string { return "behavioral" }

// Scan extracts the source-code URL from the record, clones it,
// runs mcp-scanner behavioral --raw, and returns mapped findings.
func (s *Scanner) Scan(ctx context.Context, record *corev1.Record) (*types.ScanResult, error) {
	repoURL, subfolder := extractSourceInfo(record)
	if repoURL == "" {
		return &types.ScanResult{
			Skipped:       true,
			SkippedReason: "no source-code locator found",
		}, nil
	}

	if isPlaceholderURL(repoURL) {
		return &types.ScanResult{
			Skipped:       true,
			SkippedReason: "placeholder source-code URL",
		}, nil
	}

	tmpDir, err := os.MkdirTemp("", "behavioral-scan-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	if err := gitClone(ctx, repoURL, tmpDir); err != nil {
		logger.Warn("repository not cloneable, skipping scan", "url", repoURL, "error", err)

		return &types.ScanResult{
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

	rawOutput, err := runMCPScanner(ctx, s.cfg.CLIPath, absDir)
	if err != nil {
		return nil, fmt.Errorf("mcp-scanner: %w", err)
	}

	return parseOutput(rawOutput)
}

// isPlaceholderURL returns true for URLs that are not real repositories
// (e.g. example.com placeholders injected by the transformer).
func isPlaceholderURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.TrimPrefix(parsed.Hostname(), "www.")

	return host == "example.com" || host == "example.org" || host == "example.net"
}

// extractSourceInfo extracts the source-code repository URL and optional
// subfolder from a record's locators and module data.
func extractSourceInfo(record *corev1.Record) (string, string) {
	if record == nil || record.GetData() == nil {
		return "", ""
	}

	fields := record.GetData().GetFields()

	return extractSourceCodeURL(fields), extractSubfolder(fields)
}

func extractSourceCodeURL(fields map[string]*structpb.Value) string {
	locatorsVal, ok := fields["locators"]
	if !ok || locatorsVal == nil {
		return ""
	}

	listVal := locatorsVal.GetListValue()
	if listVal == nil {
		return ""
	}

	for _, v := range listVal.GetValues() {
		s := v.GetStructValue()
		if s == nil {
			continue
		}

		f := s.GetFields()

		typeVal := f["type"]
		if typeVal == nil {
			continue
		}

		if typeVal.GetStringValue() != "source_code" {
			continue
		}

		if urlsVal := f["urls"]; urlsVal != nil && urlsVal.GetListValue() != nil {
			for _, u := range urlsVal.GetListValue().GetValues() {
				if u.GetStringValue() != "" {
					return u.GetStringValue()
				}
			}
		}
	}

	return ""
}

// extractSubfolder walks modules[*].data.mcp_data.repository.subfolder.
func extractSubfolder(fields map[string]*structpb.Value) string {
	modulesVal, ok := fields["modules"]
	if !ok || modulesVal == nil {
		return ""
	}

	listVal := modulesVal.GetListValue()
	if listVal == nil {
		return ""
	}

	for _, modVal := range listVal.GetValues() {
		sf := getNestedString(modVal, "data", "mcp_data", "repository", "subfolder")
		if sf != "" {
			return sf
		}
	}

	return ""
}

// getNestedString traverses nested protobuf Structs by the given keys
// and returns the final value as a string, or "" if any step is missing.
func getNestedString(v *structpb.Value, keys ...string) string {
	for i, k := range keys {
		if v == nil {
			return ""
		}

		s := v.GetStructValue()
		if s == nil {
			return ""
		}

		v = s.GetFields()[k]

		if i == len(keys)-1 {
			if v != nil {
				return v.GetStringValue()
			}

			return ""
		}
	}

	return ""
}

func gitClone(ctx context.Context, repoURL, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", repoURL, dest)
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

func runMCPScanner(ctx context.Context, cliPath, scanDir string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, cliPath, "behavioral", "--raw", scanDir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = buildScannerEnv()

	if err := cmd.Run(); err != nil {
		logger.Warn("mcp-scanner stderr", "output", strings.TrimSpace(stderr.String()))

		return nil, fmt.Errorf("mcp-scanner exited with error: %w", err)
	}

	return stdout.Bytes(), nil
}

// buildScannerEnv returns the parent env with MCP_SCANNER_LLM_* vars
// derived from the AZURE_* equivalents used by the enricher,
// so CI config doesn't need to duplicate them.
func buildScannerEnv() []string {
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
