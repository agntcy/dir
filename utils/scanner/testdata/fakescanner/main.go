// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command fakescanner is a test-only stand-in for the a2a-scanner binary. The
// scanner tests build it into a temp dir and point A2ARunner.CLIPath at it, so
// the runner's exec path (Run + runA2AScanner) can be exercised without the
// real CLI installed. Behaviour is driven by env vars set by the test:
//
//	FAKE_A2A_FAIL=1     exit non-zero (simulate a scanner error)
//	FAKE_A2A_NO_FILE=1  do not write the --output file (simulate missing output)
//	FAKE_A2A_OUTPUT=... raw bytes to write to the --output file
//
// It lives under testdata/ so it is excluded from the module build, `go test`,
// `go vet`, and golangci-lint (all of which skip testdata directories).
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 && args[0] == "--version" {
		fmt.Fprintln(os.Stdout, "a2a-scanner 9.9.9-test")
		os.Exit(0)
	}

	if os.Getenv("FAKE_A2A_FAIL") == "1" {
		fmt.Fprintln(os.Stderr, "fake a2a-scanner: forced failure")
		os.Exit(3)
	}

	outputPath := ""

	for i, a := range args {
		if a == "--output" && i+1 < len(args) {
			outputPath = args[i+1]
		}
	}

	if outputPath != "" && os.Getenv("FAKE_A2A_NO_FILE") != "1" {
		body := os.Getenv("FAKE_A2A_OUTPUT")
		if body == "" {
			body = `{"is_safe": true, "findings": []}`
		}

		if err := os.WriteFile(outputPath, []byte(body), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(4)
		}
	}

	os.Exit(0)
}
