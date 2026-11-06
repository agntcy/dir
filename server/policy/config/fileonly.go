// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"strings"
)

// fileOnlyKeys are the policy settings only the configuration file can carry,
// relative to the block: the two lists of policies, and the block holding them.
//
// Presence is the test rather than value, because no value is worth accepting.
// A list of policies has no environment form, so a variable can only subtract.
// Exported empty, as a template with a missing substitution leaves it, it
// decodes to an empty list that replaces whatever the file configured. The
// block is worse than its leaves: set to anything at all, even a value plainly
// not a policy block, it empties both lists along with policy.dir and
// policy.refresh_interval, so there is no value the decoder objects to.
//
// They are refused rather than ignored for the reason the retired keys are.
// What they configure is the read gate, and a node that loses it starts clean,
// reports healthy, and serves every record it was withholding.
var fileOnlyKeys = []string{"policy", "policy.operations", "policy.evaluations"}

// CheckFileOnly refuses a variable set for a policy setting that only the
// configuration file can carry.
//
// It reads the environment directly rather than through viper, for the reason
// findRetired does: AutomaticEnv answers IsSet from the environment for any key
// at all, so a variable over the block is indistinguishable from configuration
// unless the variable itself is looked for.
//
// A loader passes every copy of the block it holds, so that one start reports
// everything there is to fix rather than one copy at a time.
func CheckFileOnly(at ...KeyLocation) error {
	var found []string

	for _, location := range at {
		for _, key := range fileOnlyKeys {
			variable := location.envName(key)

			if _, inEnv := os.LookupEnv(variable); !inEnv {
				continue
			}

			// The block and a list under it are separate variables, each of
			// which has to be unset, so both are named when both are present.
			found = append(found, fmt.Sprintf("  %s is set: write %s in the configuration file",
				variable, location.KeyPrefix+key))
		}
	}

	if len(found) == 0 {
		return nil
	}

	return fmt.Errorf(
		"policy configuration cannot be set from the environment:\n%s\n"+
			"These have no environment form, so a variable for one can only subtract: set but empty, "+
			"as a missing template substitution leaves it, it takes the configured policies with it",
		strings.Join(found, "\n"))
}
