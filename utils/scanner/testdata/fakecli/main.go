// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Command fakecli is a stand-in for the mcp-scanner binary, used only by
// utils/scanner tests to exercise the real exec.Command invocation paths
// (RemoteRunner.runOne / runMCPScannerRemote, MCPRunner.Run / runMCPScanner)
// without depending on a real mcp-scanner install being present in the test
// environment.
//
// Behavior is selected by looking for a marker substring among the process's
// argv (which includes the --server-url value for RemoteRunner, or the scan
// directory for MCPRunner), so each test can steer the fake CLI simply by
// choosing what URL/path it feeds into the runner under test:
//
//	fail-exec  -> exits non-zero with a stderr message (simulates an
//	              unreachable server / mcp-scanner invocation failure)
//	bad-json   -> exits 0 but writes non-JSON stdout (simulates a corrupt
//	              or unparsable mcp-scanner response)
//	empty-safe -> exits 0 and writes "[]" (simulates a scan that found
//	              nothing to report)
//	(default)  -> exits 0 and writes a single well-formed, unsafe finding
//	              so callers can assert on parsed output.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := strings.Join(os.Args[1:], " ")

	switch {
	case strings.Contains(args, "fail-exec"):
		fmt.Fprintln(os.Stderr, "simulated exec failure: connection refused")
		os.Exit(1)
	case strings.Contains(args, "bad-json"):
		fmt.Fprintln(os.Stdout, "not valid json output {{{")
	case strings.Contains(args, "empty-safe"):
		fmt.Fprintln(os.Stdout, "[]")
	default:
		tool := "default"

		for _, a := range os.Args[1:] {
			switch a {
			case "remote", "prompts", "resources", "instructions", "behavioral":
				tool = a
			}
		}

		fmt.Fprintf(os.Stdout, `[{"tool_name":%q,"status":"done","is_safe":false,"findings":{"test_analyzer":{"severity":"HIGH","threat_summary":"synthetic finding","threat_names":["synthetic"],"total_findings":1}}}]`, tool)
	}
}
