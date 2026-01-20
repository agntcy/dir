// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validate

var opts = &options{}

type options struct {
	SchemaURL string
}

func init() {
	flags := Command.Flags()

	flags.StringVar(&opts.SchemaURL, "url", "",
		"OASF schema URL for API-based validation (required)")
}
