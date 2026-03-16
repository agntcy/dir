// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
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

// extractSourceInfo decodes the record into its typed OASF representation
// and extracts the source-code repository URL and optional subfolder.
func extractSourceInfo(record *corev1.Record) (string, string) {
	if record == nil {
		return "", ""
	}

	decoded, err := record.Decode()
	if err != nil {
		logger.Debug("could not decode record, skipping source extraction", "error", err)

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

// extractSubfolder walks modules[*].data.mcp_data.repository.subfolder.
// Module.Data is still an untyped *structpb.Struct in the OASF schema.
func extractSubfolder(modules []*typesv1.Module) string {
	for _, mod := range modules {
		sf := getNestedString(mod.GetData(), "mcp_data", "repository", "subfolder")
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
