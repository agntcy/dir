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

	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/spf13/viper"
)

// LoadCandidate reads the policies in a candidate file: the same validators
// list the server and the reconciler take, so that a policy that is good can be
// pasted into their configuration as it is. Only the entries with op evaluate
// define a policy; any others in the file are ignored.
func LoadCandidate(path string) (validatorsconfig.Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// JSON is YAML, and a candidate may come on standard input, which has no
	// extension to tell the format by.
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read candidate file %s: %w", path, err)
	}

	var all validatorsconfig.Config
	if err := v.UnmarshalKey("validators", &all); err != nil {
		return nil, fmt.Errorf("read the validators in %s: %w", path, err)
	}

	var policies validatorsconfig.Config

	for _, entry := range all {
		if entry.HasOp(validatorsconfig.OpEvaluate) {
			policies = append(policies, entry)
		}
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("%s defines no policy: a validators entry needs op: [%q]", path, validatorsconfig.OpEvaluate)
	}

	if err := policies.Validate(); err != nil {
		return nil, fmt.Errorf("invalid candidate in %s: %w", path, err)
	}

	return policies, nil
}

// errNoPolicyNamed is returned when the policy asked for is not in the file.
var errNoPolicyNamed = errors.New("no such policy in the candidate file")
