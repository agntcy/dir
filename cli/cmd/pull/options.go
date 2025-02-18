// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package pull

var opts = &options{}

type options struct {
	AgentDigest string
	OutputFormat      string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.AgentDigest, "digest", "", "Digest of the agent to pull")
	Command.Flags().StringVarP(&opts.OutputFormat, "output", "o", "json", "Output format (json|yaml)")
}
