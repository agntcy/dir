// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// scannerPackage describes a security scanner CLI installable as a Python tool.
// Versions match the ones the Directory images and Taskfile pin.
type scannerPackage struct {
	binary  string
	pkg     string
	version string
}

// Versions are bumped by Renovate (see customManagers in renovate.json); keep
// the "renovate:" comment directly above each entry's version field.
var scannerPackages = []scannerPackage{
	{
		binary: "mcp-scanner",
		pkg:    "cisco-ai-mcp-scanner",
		// renovate: datasource=pypi depName=cisco-ai-mcp-scanner versioning=pep440
		version: "4.8.6",
	},
	{
		binary: "skill-scanner",
		pkg:    "cisco-ai-skill-scanner",
		// renovate: datasource=pypi depName=cisco-ai-skill-scanner versioning=pep440
		version: "2.0.12",
	},
	{
		binary: "a2a-scanner",
		pkg:    "cisco-ai-a2a-scanner",
		// renovate: datasource=pypi depName=cisco-ai-a2a-scanner versioning=pep440
		version: "1.0.1",
	},
}

// installTool runs `uv tool install` for one scanner; replaced in tests.
var installTool = func(ctx context.Context, sp scannerPackage) ([]byte, error) {
	//nolint:gosec // G204: package and version come from the pinned scannerPackages table, not user input.
	return exec.CommandContext(ctx, "uv", "tool", "install", "--python", "3.13",
		fmt.Sprintf("%s==%s", sp.pkg, sp.version)).CombinedOutput()
}

// lookPath reports whether uv is available; replaced in tests.
var lookPath = exec.LookPath

// runScannerInstall installs the mcp, skill and a2a scanner CLIs with uv so the
// daemon's scan reconciler can find them on $PATH.
func runScannerInstall(cmd *cobra.Command) error {
	presenter.Printf(cmd, `
Security scanners
Installing mcp-scanner, skill-scanner and a2a-scanner with uv. Enable them by
setting scan.enabled: true in the daemon config.
`)

	if _, err := lookPath("uv"); err != nil {
		return errors.New("uv is required to install the scanners (https://docs.astral.sh/uv/getting-started/installation/)")
	}

	for _, sp := range scannerPackages {
		out, err := installTool(cmd.Context(), sp)
		if err != nil {
			printDetails(cmd, string(out))

			return fmt.Errorf("install %s: %w", sp.binary, err)
		}

		presenter.Printf(cmd, "✔ %s %s installed.\n", sp.binary, sp.version)
	}

	presenter.Printf(cmd, "Make sure the uv tool bin directory is on your PATH (`uv tool update-shell`).\n")

	return nil
}
