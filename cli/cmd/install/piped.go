// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// maxPipedRefs caps how many references one run will install, so a pipe from
// an unbounded search cannot rewrite every agent config on the machine before
// anyone notices. `dirctl search` has its own limit; this is the backstop.
const maxPipedRefs = 1000

// errPipedNeedsYes is returned when a piped run would have to prompt. stdin is
// the reference list, so the prompt would read a CID as the answer.
var errPipedNeedsYes = errors.New(
	"a piped install cannot prompt, because stdin carries the references: pass --yes, or --dry-run to preview")

// hasPipedInput reports whether references are waiting on stdin.
//
// A terminal means the user is typing at us and meant to pass a reference as
// an argument, so an empty invocation prints help as it always has. Anything
// else — a pipe, a file redirect, /dev/null — is treated as a list, and an
// empty one installs nothing.
func hasPipedInput(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		// Not a real file, so it is a test buffer or similar: readable, and
		// certainly not a terminal.
		return true
	}

	return !isTerminal(f)
}

func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd())) //nolint:gosec // G115: a file descriptor fits in an int.
}

// readPipedRefs reads one reference per line, skipping blanks and `#` comments
// so a hand-written list can be annotated.
//
// The contract is `dirctl search -o raw`, which is one CID per line. `-o
// jsonl` works too, since its quotes are stripped. Names are accepted as well:
// `dirctl install list` can produce them, and resolving one costs the same
// call install already makes.
func readPipedRefs(cmd *cobra.Command) ([]string, error) {
	var refs []string

	scanner := bufio.NewScanner(cmd.InOrStdin())
	for scanner.Scan() {
		// `-o jsonl` quotes each CID, so a line may arrive as "bafy...".
		ref := strings.Trim(strings.TrimSpace(scanner.Text()), `"`)
		if ref == "" || strings.HasPrefix(ref, "#") {
			continue
		}

		if len(refs) == maxPipedRefs {
			return nil, fmt.Errorf("refusing to install more than %d references in one run", maxPipedRefs)
		}

		refs = append(refs, ref)
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read references from stdin: %w", err)
	}

	return refs, nil
}
