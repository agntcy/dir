// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

var opts = &options{}

type options struct {
	AgentDigest string
	JSON        bool
}

func init() {
	flags := Command.Flags()
	flags.StringVar(&opts.AgentDigest, "digest", "", "Digest of the agent to pull")
	flags.BoolVar(&opts.JSON, "json", false, "Output in JSON format")
}
