// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package pull

var opts = &options{}

type options struct {
	AgentDigest string
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.AgentDigest, "digest", "", "Digest of the agent to pull")
}
