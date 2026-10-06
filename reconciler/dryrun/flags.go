// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dryrun

import "github.com/spf13/pflag"

// DefaultSamples is how many excluded records are listed unless asked
// otherwise.
const DefaultSamples = 10

// BindFlags defines the flags of a dry run on fs, filling opts. When
// --policy-dir is not given opts.PolicyDir is empty, and the caller fills it
// with the node's own policy directory.
func BindFlags(fs *pflag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Candidate, "candidate", "", "file that defines the policy to try: a validators list whose entries have op \"evaluate\" (required)")
	fs.StringVar(&opts.PolicyDir, "policy-dir", "", "directory the candidate's OPA (.rego) files are read from (default: the node's policy directory)")
	fs.StringVar(&opts.Policy, "policy", "", "try only the policy with this ID, such as opa:require-license (default: every policy in the file)")
	fs.IntVar(&opts.Samples, "samples", DefaultSamples, "how many of the records a policy would exclude to list")
	fs.StringVarP(&opts.Output, "output", "o", OutputHuman, "format of the report: human or json")
}
