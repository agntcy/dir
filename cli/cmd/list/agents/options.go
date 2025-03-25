// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agents

var opts = &options{}

type options struct {
	Query string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Query, "query", "", "Query to search for agents")

	Command.MarkFlagRequired("query") //nolint:errcheck
}
