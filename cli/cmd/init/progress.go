// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/term"
)

// Provisioning phase labels shown next to the spinner. The underlying oasf-sdk
// call is a single blocking operation with no progress callbacks, so phases are
// inferred from the (otherwise hidden) log stream via phaseFor.
const (
	phaseDownload = "Downloading model (~89 MB)…"
	phaseConvert  = "Converting model…"
	phaseEmbed    = "Fetching OASF taxonomy & embedding…"

	spinnerInterval = 100 * time.Millisecond
	scanBufferMax   = 1024 * 1024
)

// provisionAction is the work whose noisy stdout/stderr/zerolog output should be
// captured and hidden behind the progress UI. Isolated as a parameter so the
// orchestration can be exercised without the real (network + 89 MB) SDK call.
type provisionAction func(ctx context.Context) error

// phaseFor maps a captured provisioning log line to a user-facing phase label,
// returning ok=false when the line implies no phase change. The oasf-sdk /
// cybertron log stream is the only progress signal available, so this reads a
// bit heuristically; it only drives the spinner label, never correctness.
func phaseFor(line string) (string, bool) {
	l := strings.ToLower(line)

	switch {
	case strings.Contains(l, "skipping conversion"),
		strings.Contains(l, "spago_model"),
		strings.Contains(l, "done."):
		return phaseEmbed, true
	case strings.Contains(l, "serializing model"),
		strings.Contains(l, "conversion"),
		strings.Contains(l, "converting"):
		return phaseConvert, true
	case strings.Contains(l, "downloading"),
		strings.Contains(l, "downloaded"),
		strings.Contains(l, "mib"),
		strings.Contains(l, "kib"):
		return phaseDownload, true
	default:
		return "", false
	}
}

// runWithSpinner runs action while (1) redirecting process stdout/stderr and
// cybertron's global zerolog logger into a capture buffer so their raw noise is
// hidden, and (2) showing an animated spinner (on a TTY) or a static phase line
// (otherwise) on the real terminal. It returns everything it captured — so the
// caller can surface it only on failure — and the action's error.
//
// initialPhase is the starting spinner label; when phaseFn is non-nil, each
// captured line is passed through it to advance the label.
func runWithSpinner(
	ctx context.Context,
	uiOut *os.File,
	initialPhase string,
	phaseFn func(string) (string, bool),
	action provisionAction,
) (string, error) {
	pr, pw, pipeErr := os.Pipe()
	if pipeErr != nil {
		// Capture unavailable — run the action directly rather than fail the
		// whole command over a progress nicety.
		return "", action(ctx)
	}

	//nolint:gosec // G115: a file descriptor comfortably fits in an int.
	isTTY := term.IsTerminal(int(uiOut.Fd()))

	origStdout, origStderr := os.Stdout, os.Stderr
	prevLogger := zlog.Logger

	os.Stdout, os.Stderr = pw, pw
	zlog.Logger = zerolog.New(pw)

	sp := spinner.New(spinner.CharSets[14], spinnerInterval, spinner.WithWriterFile(uiOut))
	sp.Suffix = " " + initialPhase

	if isTTY {
		sp.Start()
	} else {
		fmt.Fprintf(uiOut, "  %s\n", initialPhase)
	}

	var (
		buf  bytes.Buffer
		wg   sync.WaitGroup
		once sync.Once
	)

	// cleanup drains the pipe, joins the reader goroutine, and restores the real
	// stdout/stderr/logger. It runs exactly once — on the normal path below, or
	// via the recover handler if action panics — so the process is never left
	// with redirected output or a leaked goroutine.
	cleanup := func() {
		once.Do(func() {
			_ = pw.Close()

			wg.Wait()

			_ = pr.Close()

			os.Stdout, os.Stderr = origStdout, origStderr
			zlog.Logger = prevLogger

			if isTTY {
				sp.Stop()
			}
		})
	}

	defer func() {
		if r := recover(); r != nil {
			cleanup()
			panic(r)
		}
	}()

	wg.Go(func() {
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), scanBufferMax)

		last := initialPhase

		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line)
			buf.WriteByte('\n')

			if phaseFn == nil {
				continue
			}

			p, ok := phaseFn(line)
			if !ok || p == last {
				continue
			}

			last = p

			if isTTY {
				sp.Suffix = " " + p
			} else {
				//nolint:gosec // G705: p is one of our own phase constants, not user input.
				fmt.Fprintf(uiOut, "  %s\n", p)
			}
		}
	})

	actionErr := action(ctx)

	cleanup()

	return buf.String(), actionErr
}

// indentLines prefixes every non-empty line of s with prefix, for readable
// error-detail blocks.
func indentLines(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}

	return strings.Join(lines, "\n")
}
