// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"fmt"

	"github.com/agntcy/dir/cli/presenter"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

const (
	localContextName   = "local"
	localServerAddress = "localhost:8888"
	localAuthMode      = "insecure"
)

// defaultLocalContext is the context seeded for a fresh environment: the local
// daemon reached over an insecure connection.
func defaultLocalContext() clientconfig.Context {
	return clientconfig.Context{
		ServerAddress: localServerAddress,
		AuthMode:      localAuthMode,
	}
}

// runContextSetup seeds a default `local` client context (and makes it current)
// when no context is configured yet, so later commands reach a local node
// without --server-addr. It is opt-out on a TTY and skips non-interactively
// without --yes. It never touches an existing context or current_context: if any
// context is already configured, the step is a no-op.
func runContextSetup(cmd *cobra.Command, opts *options) error {
	presenter.Printf(cmd, "\nStep 1 — Client context\n")

	summaries, err := clientconfig.ListContexts("")
	if err != nil {
		return fmt.Errorf("read client config: %w", err)
	}

	if len(summaries) > 0 {
		presenter.Printf(cmd, "A client context is already configured; leaving it untouched.\n")

		return nil
	}

	presenter.Printf(cmd, "Point dirctl at a Directory node. This adds a %q context (server %s, insecure\nauth) and makes it the default, so push/search/pull reach a local daemon\nwithout --server-addr.\n",
		localContextName, localServerAddress)

	if !opts.yes {
		if !isInteractive(cmd) {
			presenter.Printf(cmd, "Skipping context setup (non-interactive). Pass --yes to configure.\n")

			return nil
		}

		ok, err := confirm(cmd, fmt.Sprintf("Configure a %q context → %s?", localContextName, localServerAddress), true)
		if err != nil {
			return err
		}

		if !ok {
			presenter.Printf(cmd, "Skipped context setup.\n")

			return nil
		}
	}

	if err := clientconfig.SaveContext("", localContextName, defaultLocalContext(), true); err != nil {
		return fmt.Errorf("save local context: %w", err)
	}

	presenter.Printf(cmd, "✔ Context %q configured and set as current (server %s).\n", localContextName, localServerAddress)

	return nil
}
