// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/agntcy/dir/utils/env"
)

// retiredKey is a setting that no longer exists, and what to write instead.
type retiredKey struct {
	// key is relative to the policy block, so one entry serves every copy of
	// the block, wherever a loader keeps it.
	key string

	// replacement completes "write ... instead".
	replacement string
}

// retiredKeys is the policy.enforcement block, removed in #2348.
//
// They are refused rather than ignored because of what they used to do.
// v1.8.0 shipped the enforcement block, and shipped no operation at pull or
// search, so it was not one way for a node to enforce reads but the only
// way. Dropped silently, an upgraded node starts clean, reports healthy, and
// serves every record it was withholding — the one failure a read gate must
// not have.
var retiredKeys = []retiredKey{
	{
		key:         "enforcement.search",
		replacement: `a policy at op: ["search"] carrying that mode`,
	},
	{
		key:         "enforcement.fetch",
		replacement: `a policy at op: ["pull"] carrying that mode`,
	},
	{
		key:         "enforcement.policies",
		replacement: `verdicts["<id>"].compliant_and_current in that policy's expressions`,
	},
	{
		key:         "enforcement.refresh_interval",
		replacement: "policy.refresh_interval, one level up",
	},
	{
		key:         "enforcement",
		replacement: `policies at op: ["pull"] and op: ["search"]`,
	},
}

// migrationGuide is where the replacement for each retired key is written out.
const migrationGuide = "https://docs.agntcy.org/dir/dir-content-policy-enforcement/#migrating-from-policyenforcement"

// KeyLocation is where a copy of this block lives. Every copy agrees on
// everything below policy., and on nothing above it.
type KeyLocation struct {
	// KeyPrefix is what precedes "policy." in the configuration file: empty
	// for the server and the reconciler read on their own, and "server." or
	// "reconciler." for the daemon, which holds both.
	KeyPrefix string

	// EnvPrefix is the loader's viper environment prefix, which the three do
	// not share: DIRECTORY_SERVER, DIRECTORY_DAEMON, RECONCILER. A variable
	// carrying another loader's prefix was never read by this one, so
	// refusing to start over it would blame a setting that did nothing.
	EnvPrefix string
}

// envName is the variable this loader would read for key, were the key
// still one viper binds.
func (l KeyLocation) envName(key string) string {
	return env.Name(l.EnvPrefix, l.KeyPrefix+key)
}

// isSetter is the one thing CheckRetired needs of a loaded configuration.
// *viper.Viper satisfies it.
type isSetter interface {
	IsSet(key string) bool
}

// RetiredError is what CheckRetired returns, and says where the settings it
// refused came from.
//
// A caller that read a configuration file can name it, but only the settings
// in the file are the file's doing: a file named for a variable's mistake
// sends the operator to edit a config that does not carry the key.
type RetiredError struct {
	// InFile is whether any refused setting came from the configuration
	// rather than from the environment.
	InFile bool

	message string
}

func (e *RetiredError) Error() string { return e.message }

// CheckRetired refuses a configuration that still carries a removed policy
// setting, naming each one it found, where it found it, and what replaces it.
//
// A loader passes every copy of the block it holds, so that one start reports
// everything there is to fix rather than one copy at a time.
func CheckRetired(v isSetter, at ...KeyLocation) error {
	var (
		found  []string
		inFile bool
	)

	for _, location := range at {
		lines, fromFile := findRetired(v, location)

		found = append(found, lines...)
		inFile = inFile || fromFile
	}

	if len(found) == 0 {
		return nil
	}

	return &RetiredError{
		InFile: inFile,
		message: fmt.Sprintf(
			"policy configuration uses settings that were removed:\n%s\nSee %s",
			strings.Join(found, "\n"), migrationGuide),
	}
}

// findRetired is what one copy of the block still carries, and whether the
// configuration rather than the environment is what carries it.
func findRetired(v isSetter, at KeyLocation) ([]string, bool) {
	var (
		found  []string
		inFile bool
	)

	for _, retired := range retiredKeys {
		key := "policy." + retired.key

		// The environment is checked by name rather than through viper,
		// which binds variables only for keys it still knows: a retired one
		// is invisible to IsSet however it was set. Presence is the test,
		// not value, because every loader here allows an empty variable to
		// count.
		//
		// It is also checked first, and for the block as well as its
		// leaves. No variable ever set the block, but viper's AutomaticEnv
		// answers IsSet from the environment for any key at all, so a
		// variable left over the block reads as configuration unless the
		// variable is looked for — and the caller then names a file that
		// does not contain it.
		variable := at.envName(key)
		_, inEnv := os.LookupEnv(variable)

		switch {
		case inEnv:
			found = append(found, fmt.Sprintf("  %s is set: write %s", variable, retired.replacement))
		case v.IsSet(at.KeyPrefix + key):
			found = append(found, fmt.Sprintf("  %s is set: write %s", at.KeyPrefix+key, retired.replacement))
			inFile = true
		default:
			continue
		}

		// The parent stands for the whole block, so reporting it as well as
		// the children it contains would say the same thing twice.
		if retired.key == "enforcement" && len(found) > 1 {
			found = found[:len(found)-1]
		}
	}

	return found, inFile
}
