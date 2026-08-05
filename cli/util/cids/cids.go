// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package cids holds helpers shared by the commands that operate on a batch of
// record CIDs supplied as variadic arguments or on stdin (delete, routing
// publish, routing unpublish), so that the output of `dirctl search` can be
// piped into any of them.
package cids

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// StdinFlagUsage is the help text for the --stdin flag of every command that
// accepts a batch of CIDs.
const StdinFlagUsage = "Read CIDs from standard input. Accepts a JSON array of CIDs, as produced by 'dirctl search --format cid --output json', or line-delimited CIDs."

// Args validates positional arguments for a command that accepts CIDs either as
// arguments or, when fromStdin points at a set --stdin flag, on stdin.
func Args(fromStdin *bool) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if *fromStdin {
			//nolint:wrapcheck
			return cobra.MaximumNArgs(0)(cmd, args)
		}

		//nolint:wrapcheck
		return cobra.MinimumNArgs(1)(cmd, args)
	}
}

// Collect gathers the CIDs to operate on from the positional arguments and,
// when fromStdin is set, from reader. The result is deduplicated and guaranteed
// to be non-empty.
func Collect(args []string, reader io.Reader, fromStdin bool) ([]string, error) {
	cids := append([]string{}, args...)

	if fromStdin {
		stdinCIDs, err := ReadFromStdin(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read CIDs from stdin: %w", err)
		}

		cids = append(cids, stdinCIDs...)
	}

	cids = Deduplicate(cids)
	if len(cids) == 0 {
		return nil, errors.New("at least one CID is required (pass arguments or use --stdin)")
	}

	return cids, nil
}

// ReadFromStdin reads CIDs from reader, accepting either a JSON array as
// produced by `dirctl search --format cid --output json` or line-delimited CIDs.
func ReadFromStdin(reader io.Reader) ([]string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	input := strings.TrimSpace(string(data))
	if input == "" {
		return nil, nil
	}

	if strings.HasPrefix(input, "[") {
		var cids []string
		if err := json.Unmarshal([]byte(input), &cids); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array of CIDs: %w", err)
		}

		return cids, nil
	}

	cids := make([]string, 0)

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		cid := strings.TrimSpace(scanner.Text())
		if cid == "" {
			continue
		}

		cids = append(cids, cid)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan stdin: %w", err)
	}

	return cids, nil
}

// Deduplicate trims each CID and drops empty and repeated entries, preserving
// first-seen order.
func Deduplicate(cids []string) []string {
	seen := make(map[string]struct{}, len(cids))
	out := make([]string, 0, len(cids))

	for _, cid := range cids {
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}

		if _, exists := seen[cid]; exists {
			continue
		}

		seen[cid] = struct{}{}
		out = append(out, cid)
	}

	return out
}
