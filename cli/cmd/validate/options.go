// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"github.com/agntcy/dir/cli/presenter"
)

var opts = &options{}

type options struct {
	DisableAPI    bool
	DisableStrict bool
	SchemaURL     string
	ValidateAll   bool
}

func init() {
	flags := Command.Flags()

	flags.BoolVar(&opts.DisableAPI, "disable-api", false,
		"Disable API-based validation (use embedded schemas instead, required if --url is not specified)")
	flags.BoolVar(&opts.DisableStrict, "disable-strict", false,
		"Disable strict validation mode (more permissive validation, only works with --url)")
	flags.StringVar(&opts.SchemaURL, "url", "",
		"OASF schema URL for API-based validation (required if --disable-api is not specified)")
	flags.BoolVar(&opts.ValidateAll, "all", false,
		"Validate all records in the directory instance (requires connection to Directory server)")

	// Add output format flags
	presenter.AddOutputFlags(Command)
}
