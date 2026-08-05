// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cids

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ReadFrom reads a JSON array or newline-delimited CIDs from reader.
func ReadFrom(reader io.Reader) ([]string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	input := strings.TrimSpace(string(data))
	if input == "" {
		return nil, nil
	}

	if strings.HasPrefix(input, "[") {
		var values []string
		if err := json.Unmarshal([]byte(input), &values); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array of CIDs: %w", err)
		}
		return values, nil
	}

	values := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		if cid := strings.TrimSpace(scanner.Text()); cid != "" {
			values = append(values, cid)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan stdin: %w", err)
	}
	return values, nil
}

// Deduplicate removes blank and repeated CIDs while preserving input order.
func Deduplicate(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
