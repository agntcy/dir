// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

// BehavioralRunner runs the MCP scanner in behavioral/supplychain mode: clone repo and scan source path.
type BehavioralRunner struct {
	cfg scannerconfig.Config
}

// NewBehavioralRunner returns a Runner that clones the record's source repo and runs mcp-scanner.
func NewBehavioralRunner(cfg scannerconfig.Config) *BehavioralRunner {
	return &BehavioralRunner{cfg: cfg}
}

// Run implements Runner. It extracts repo URL from the record, clones to a temp dir,
// runs mcp-scanner --analyzers yara behavioral <path>, and returns ScanResult.
func (r *BehavioralRunner) Run(ctx context.Context, record *corev1.Record) (*ScanResult, error) {
	repoURL, subfolder, ok := getSourceLocator(record)
	if !ok || repoURL == "" {
		return &ScanResult{
			Skipped:       true,
			SkippedReason: "no source locator",
		}, nil
	}

	timeout := r.cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dir, err := os.MkdirTemp("", "dir-scanner-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	// Clone (shallow, single branch to save time)
	cloneDir := filepath.Join(dir, "repo")
	if err := r.gitClone(runCtx, repoURL, cloneDir); err != nil {
		return nil, fmt.Errorf("git clone %s: %w", repoURL, err)
	}

	scanPath := cloneDir
	if subfolder != "" {
		scanPath = filepath.Join(cloneDir, subfolder)
		if _, err := os.Stat(scanPath); err != nil {
			return nil, fmt.Errorf("subfolder %q not found in repo: %w", subfolder, err)
		}
	}

	// Run mcp-scanner --analyzers yara behavioral <path>
	analyzers := r.cfg.Analyzers
	if len(analyzers) == 0 {
		analyzers = []string{"yara"}
	}
	cliPath := r.cfg.CLIPath
	if cliPath == "" {
		cliPath = "mcp-scanner"
	}

	args := []string{"--analyzers", strings.Join(analyzers, ","), "behavioral", scanPath}
	cmd := exec.CommandContext(runCtx, cliPath, args...)
	cmd.Dir = scanPath
	out, cmdErr := cmd.CombinedOutput()

	if runCtx.Err() == context.DeadlineExceeded {
		return &ScanResult{
			Safe:     false,
			Findings: []Finding{{Severity: SeverityError, Message: "scan timed out"}},
		}, nil
	}

	if cmdErr != nil {
		// Non-zero exit: treat as findings (error severity so fail-on-error and fail-on-warning both apply)
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = cmdErr.Error()
		}
		return &ScanResult{
			Safe: false,
			Findings: []Finding{
				{Severity: SeverityError, Message: msg},
			},
		}, nil
	}

	return &ScanResult{Safe: true}, nil
}

func (r *BehavioralRunner) gitClone(ctx context.Context, repoURL, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--single-branch", repoURL, dir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// getSourceLocator extracts a git repository URL and optional subfolder from an OASF record
// for behavioral/supplychain scanning. It checks:
// - data.modules[] for name == "integration/mcp", then mcp_data.repository.url and subfolder
// - data.locators[] for type == "source_code" and urls[] or url
// Returns repoURL, subfolder (may be ""), and true if a source was found.
func getSourceLocator(record *corev1.Record) (repoURL, subfolder string, ok bool) {
	if record == nil || record.GetData() == nil {
		return "", "", false
	}
	fields := record.GetData().GetFields()
	if fields == nil {
		return "", "", false
	}

	// Try modules[].name == "integration/mcp" and mcp_data.repository.url
	if modules := fields["modules"]; modules != nil {
		if list := modules.GetListValue(); list != nil {
			for _, v := range list.GetValues() {
				s := v.GetStructValue()
				if s == nil || s.GetFields() == nil {
					continue
				}
				if n := s.GetFields()["name"]; n != nil && n.GetStringValue() == "integration/mcp" {
					data := s.GetFields()["data"]
					if data == nil {
						continue
					}
					dataStruct := data.GetStructValue()
					if dataStruct == nil || dataStruct.GetFields() == nil {
						continue
					}
					mcpData := dataStruct.GetFields()["mcp_data"]
					if mcpData == nil {
						continue
					}
					mcpStruct := mcpData.GetStructValue()
					if mcpStruct == nil || mcpStruct.GetFields() == nil {
						continue
					}
					repo := mcpStruct.GetFields()["repository"]
					if repo != nil {
						repoStruct := repo.GetStructValue()
						if repoStruct != nil && repoStruct.GetFields() != nil {
							if u := repoStruct.GetFields()["url"]; u != nil {
								repoURL = strings.TrimSpace(u.GetStringValue())
							}
							if sub := repoStruct.GetFields()["subfolder"]; sub != nil {
								subfolder = strings.TrimSpace(sub.GetStringValue())
							}
							if repoURL != "" {
								return repoURL, subfolder, true
							}
						}
					}
					break
				}
			}
		}
	}

	// Try top-level mcp_data.repository (some records may have it at data level)
	if mcpData := fields["mcp_data"]; mcpData != nil {
		mcpStruct := mcpData.GetStructValue()
		if mcpStruct != nil && mcpStruct.GetFields() != nil {
			repo := mcpStruct.GetFields()["repository"]
			if repo != nil {
				repoStruct := repo.GetStructValue()
				if repoStruct != nil && repoStruct.GetFields() != nil {
					if u := repoStruct.GetFields()["url"]; u != nil {
						repoURL = strings.TrimSpace(u.GetStringValue())
					}
					if sub := repoStruct.GetFields()["subfolder"]; sub != nil {
						subfolder = strings.TrimSpace(sub.GetStringValue())
					}
					if repoURL != "" {
						return repoURL, subfolder, true
					}
				}
			}
		}
	}

	// Try locators[] with type == "source_code"
	if locators := fields["locators"]; locators != nil {
		list := locators.GetListValue()
		if list == nil {
			return "", "", false
		}
		for _, v := range list.GetValues() {
			s := v.GetStructValue()
			if s == nil || s.GetFields() == nil {
				continue
			}
			t := s.GetFields()["type"]
			if t == nil || t.GetStringValue() != "source_code" {
				continue
			}
			// OASF v1 uses "urls" (plural), older may use "url"
			if urls := s.GetFields()["urls"]; urls != nil {
				urlList := urls.GetListValue()
				if urlList != nil && len(urlList.GetValues()) > 0 {
					first := urlList.GetValues()[0]
					if first != nil {
						repoURL = strings.TrimSpace(first.GetStringValue())
					}
				}
			}
			if repoURL == "" {
				if u := s.GetFields()["url"]; u != nil {
					repoURL = strings.TrimSpace(u.GetStringValue())
				}
			}
			if sub := s.GetFields()["subfolder"]; sub != nil {
				subfolder = strings.TrimSpace(sub.GetStringValue())
			}
			if repoURL != "" {
				return repoURL, subfolder, true
			}
		}
	}

	return "", "", false
}
