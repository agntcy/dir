// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "runtime",
	Short: "Running workloads discovered by runtime discovery",
	Long: `Running workloads discovered by runtime discovery.

Runtime discovery (dir-runtime, embedded in dirctl daemon) lists the agents running in
Docker, Kubernetes and as host processes. These are not records: they are kept apart
from dirctl search, since they show where agents run.

- list: List running workloads, filtered by agent, runtime or reachability
`,
}

func init() {
	Command.AddCommand(listCmd)
}
