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
	// key is relative to the policy block, so one entry serves every loader
	// that has one, wherever each keeps it.
	key string

	// hadEnv is whether a variable used to set it. Only leaves did: the
	// block as a whole was never one variable.
	hadEnv bool

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
		hadEnv:      true,
		replacement: `a policy at op: ["search"] carrying that mode`,
	},
	{
		key:         "enforcement.fetch",
		hadEnv:      true,
		replacement: `a policy at op: ["pull"] carrying that mode`,
	},
	{
		key:         "enforcement.policies",
		hadEnv:      true,
		replacement: `verdicts["<id>"].compliant_and_current in that policy's expressions`,
	},
	{
		key:         "enforcement.refresh_interval",
		hadEnv:      true,
		replacement: "policy.refresh_interval, one level up",
	},
	{
		key:         "enforcement",
		replacement: `policies at op: ["pull"] and op: ["search"]`,
	},
}

// migrationGuide is where the replacement for each retired key is written out.
const migrationGuide = "https://docs.agntcy.org/dir/dir-content-policy-enforcement/#migrating-from-policyenforcement"

// KeyLocation is where a loader keeps this block. The three that read it
// agree on everything below policy., and on nothing above it.
type KeyLocation struct {
	// KeyPrefix is what precedes "policy." in the configuration file: empty
	// for the server and the reconciler, "server." for the daemon.
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

// CheckRetired refuses a configuration that still carries a removed policy
// setting, naming each one it found, where it found it, and what replaces it.
func CheckRetired(v isSetter, at KeyLocation) error {
	var found []string

	for _, retired := range retiredKeys {
		key := "policy." + retired.key

		// The environment is checked by name rather than through viper,
		// which binds variables only for keys it still knows: a retired one
		// is invisible to IsSet however it was set. Presence is the test,
		// not value, because every loader here allows an empty variable to
		// count.
		variable := at.envName(key)
		_, inEnv := os.LookupEnv(variable)

		switch {
		case retired.hadEnv && inEnv:
			found = append(found, fmt.Sprintf("  %s is set: write %s", variable, retired.replacement))
		case v.IsSet(at.KeyPrefix + key):
			found = append(found, fmt.Sprintf("  %s is set: write %s", at.KeyPrefix+key, retired.replacement))
		default:
			continue
		}

		// The parent stands for the whole block, so reporting it as well as
		// the children it contains would say the same thing twice.
		if retired.key == "enforcement" && len(found) > 1 {
			found = found[:len(found)-1]
		}
	}

	if len(found) == 0 {
		return nil
	}

	return fmt.Errorf(
		"policy configuration uses settings that were removed:\n%s\nSee %s",
		strings.Join(found, "\n"), migrationGuide)
}
