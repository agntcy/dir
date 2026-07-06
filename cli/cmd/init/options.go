// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	extractor "github.com/agntcy/dir/cli/internal/extractor"
	"github.com/spf13/cobra"
)

// options holds the parsed flags for `dirctl init`.
type options struct {
	oasfURL  string
	assetDir string
	yes      bool
	remove   bool
}

// addFlags registers the `dirctl init` flags on cmd.
func addFlags(cmd *cobra.Command, opts *options) {
	flags := cmd.Flags()
	flags.StringVar(&opts.oasfURL, "oasf-url", extractor.DefaultOASFURL,
		"OASF schema endpoint to pull the taxonomy from")
	flags.StringVar(&opts.assetDir, "asset-dir", "",
		"Local directory for provisioned extractor assets (default ~/.agntcy/oasf-sdk/extractor)")
	flags.BoolVarP(&opts.yes, "yes", "y", false,
		"Skip prompts and proceed non-interactively (provisions ~89 MB unattended)")
	flags.BoolVar(&opts.remove, "remove", false,
		"Remove the provisioned extractor assets and clear the saved config")
}
