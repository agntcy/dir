// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"github.com/agntcy/dir/cli/cmd/search"
	"github.com/agntcy/dir/cli/presenter"
)

var opts = &options{}

type options struct {
	PublicKey  bool
	Signature  bool
	ScanReport bool

	// File output (single record) and batch output (search → directory).
	OutputFile  string
	OutputDir   string
	Limit       uint32
	AllVersions bool
	Filters     search.Filters
}

func init() {
	flags := Command.Flags()
	flags.BoolVar(&opts.PublicKey, "public-key", false, "Pull the public key for the record.")
	flags.BoolVar(&opts.Signature, "signature", false, "Pull the signature for the record.")
	flags.BoolVar(&opts.ScanReport, "scan-report", false, "Pull security scan reports for the record.")

	// Output destinations.
	flags.StringVar(&opts.OutputFile, "output-file", "", "Write the record JSON to a file instead of stdout (single record)")
	flags.StringVar(&opts.OutputDir, "output-dir", "", "Directory for batch pull from search results (one JSON file per record)")
	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of records to pull in batch mode") //nolint:mnd
	flags.BoolVar(&opts.AllVersions, "all-versions", false, "Keep all versions in batch pull (default: latest per name wins)")

	// Batch selection reuses the standard search filters.
	search.RegisterFilterFlags(Command, &opts.Filters)

	// Records are primarily machine-consumed, so default to JSON on stdout.
	presenter.AddOutputFlagsWithDefault(Command, string(presenter.FormatJSON))
}
