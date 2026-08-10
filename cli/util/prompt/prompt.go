// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompt

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// Confirm prompts for a yes/no answer, defaulting to no.
func Confirm(cmd *cobra.Command, text string) (bool, error) {
	presenter.PrintSmartf(cmd, "%s [y/N]: ", text)

	reader := bufio.NewReader(cmd.InOrStdin())

	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))

	return answer == "y" || answer == "yes", nil
}
