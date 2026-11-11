// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// CheckBlocks refuses a variable set for a key that is a block rather than a
// setting.
//
// keys are the loader's leaf keys, as viper's AllKeys reports them after the
// configuration is read: every default, every bound variable and every
// setting in the file. A block is any proper prefix of one of them. No
// variable can usefully carry a block, yet with AutomaticEnv and
// AllowEmptyEnv(true) viper answers IsSet for a non-leaf key too, so a
// variable over a block, even an empty one, replaces everything the file and
// the defaults put under it.
//
// The blocks are derived from the keys the loader already knows rather than
// from a hand-maintained list, so a block added later is covered without
// touching this check. The environment is read directly, since viper cannot
// tell a variable over a block from configuration.
func CheckBlocks(prefix string, keys []string) error {
	leaves := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		leaves[Name(prefix, key)] = struct{}{}
	}

	blocks := map[string]struct{}{}

	for _, key := range keys {
		parts := strings.Split(key, ".")
		for i := 1; i < len(parts); i++ {
			blocks[strings.Join(parts[:i], ".")] = struct{}{}
		}
	}

	var found []string

	for block := range blocks {
		variable := Name(prefix, block)

		// A setting whose variable happens to share the block's name is the
		// setting's, and it is read as one.
		if _, isLeaf := leaves[variable]; isLeaf {
			continue
		}

		if _, inEnv := os.LookupEnv(variable); inEnv {
			found = append(found, fmt.Sprintf("  %s is set: %s is a block of settings, not a setting", variable, block))
		}
	}

	if len(found) == 0 {
		return nil
	}

	slices.Sort(found)

	return fmt.Errorf(
		"configuration blocks cannot be set from the environment:\n%s\n"+
			"A variable over a block replaces every setting under it, even when empty; "+
			"unset it and set the individual settings instead",
		strings.Join(found, "\n"))
}
