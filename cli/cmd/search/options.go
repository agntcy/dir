// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package search

var opts = &options{}

type options struct {
	Query string
	Limit uint32
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.Query, "query", "", "Query text to search agents")
	flags.Uint32Var(&opts.Limit, "limit", 5, "Maximum number to return")
}
