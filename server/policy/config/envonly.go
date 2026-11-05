// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"strings"
)

// envOnlyKeys are the policy settings no variable can carry, relative to the
// block. They are lists of policies, and the block that holds them.
//
// A variable for one is refused rather than ignored for the same reason the
// retired keys are. Given a value, the decoder refuses it loudly, because a
// string is not a list of policies. Given none — exported empty, as a
// template with a missing substitution leaves it — it decodes to an empty
// list and takes the configured policies with it, and the node starts clean,
// reports healthy, and enforces nothing it was told to.
var envOnlyKeys = []string{"policy", "policy.operations", "policy.evaluations"}

// CheckEnvOnly refuses a variable set for a policy setting that only the
// configuration file can carry.
func CheckEnvOnly(at ...KeyLocation) error {
	var found []string

	for _, location := range at {
		for _, key := range envOnlyKeys {
			variable := location.envName(key)
			if _, inEnv := os.LookupEnv(variable); !inEnv {
				continue
			}

			found = append(found, fmt.Sprintf("  %s is set: write %s in the configuration file",
				variable, location.KeyPrefix+key))
		}
	}

	if len(found) == 0 {
		return nil
	}

	return fmt.Errorf(
		"policy configuration cannot be set from the environment:\n%s\n"+
			"A list of policies has no environment form, and an empty variable would leave the node with none",
		strings.Join(found, "\n"))
}
