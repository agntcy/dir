// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// buildFakeCLIOnce compiles testdata/fakecli exactly once per test binary run
// and returns the resulting executable's path. All tests that need to stand
// in for the real mcp-scanner/skill-scanner CLI share this single binary.
var buildFakeCLIOnce = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "fakecli-bin-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir for fake CLI: %w", err)
	}

	bin := filepath.Join(dir, "fakecli")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	//nolint:noctx // test helper, no request-scoped context available
	cmd := exec.Command("go", "build", "-o", bin, "./testdata/fakecli")

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build fake CLI test helper: %w: %s", err, out)
	}

	return bin, nil
})

// fakeCLIPath returns the path to the compiled fake CLI binary described in
// testdata/fakecli/main.go, building it on first use. It fails the test if
// the build fails (e.g. no Go toolchain available in the test environment).
func fakeCLIPath(t *testing.T) string {
	t.Helper()

	bin, err := buildFakeCLIOnce()
	if err != nil {
		t.Fatalf("failed to build fake CLI test helper: %v", err)
	}

	return bin
}
