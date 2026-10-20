// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package dryrun evaluates a candidate content policy against the records a
// node holds, before the policy is deployed, and reports how many it would
// exclude and why. It stores nothing: no verdict and no policy version, so it
// changes nothing a read returns.
package dryrun

import (
	"errors"
	"fmt"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/spf13/viper"
)

// LoadCandidate reads the policies in a candidate file: the same
// policy.evaluations list the server and the reconciler take, so that a
// policy that is good can be pasted into their configuration as it is.
// Only evaluation policies produce a verdict, so an operations block in the
// file is ignored.
func LoadCandidate(path string) (policyconfig.Evaluations, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// JSON is YAML, and a candidate may come on standard input, which has no
	// extension to tell the format by.
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read candidate file %s: %w", path, err)
	}

	var policies policyconfig.Evaluations
	if err := v.UnmarshalKey("policy.evaluations", &policies); err != nil {
		return nil, fmt.Errorf("read the evaluations in %s: %w", path, err)
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("%s defines no policy: it needs a policy.evaluations entry", path)
	}

	if err := policies.Validate(); err != nil {
		return nil, fmt.Errorf("invalid candidate in %s: %w", path, err)
	}

	return policies, nil
}

// errNoPolicyNamed is returned when the policy asked for is not in the file.
var errNoPolicyNamed = errors.New("no such policy in the candidate file")
