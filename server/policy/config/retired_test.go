// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"slices"
	"testing"

	"github.com/agntcy/dir/server/policy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keysSet is a configuration carrying exactly these keys.
type keysSet map[string]bool

func (k keysSet) IsSet(key string) bool { return k[key] }

var (
	serverKeys     = config.KeyLocation{EnvPrefix: "DIRECTORY_SERVER"}
	daemonKeys     = config.KeyLocation{KeyPrefix: "server.", EnvPrefix: "DIRECTORY_DAEMON"}
	reconcilerKeys = config.KeyLocation{EnvPrefix: "RECONCILER"}

	// The reconciler's copy of the block as the daemon holds it, alongside
	// the server's.
	daemonReconcilerKeys = config.KeyLocation{KeyPrefix: "reconciler.", EnvPrefix: "DIRECTORY_DAEMON"}
)

// Every loader of this block keeps it somewhere different, and each must
// look where it keeps it: a check that reads the server's key would miss the
// daemon's file entirely.
func TestCheckRetired_LooksWhereTheLoaderKeepsTheBlock(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		at   config.KeyLocation
		key  string
	}{
		{"server", serverKeys, "policy.enforcement.fetch"},
		{"daemon", daemonKeys, "server.policy.enforcement.fetch"},
		{"reconciler", reconcilerKeys, "policy.enforcement.fetch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := config.CheckRetired(keysSet{tt.key: true}, tt.at)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.key)
			assert.Contains(t, err.Error(), `op: ["pull"]`)
		})
	}
}

// A loader must not be stopped by another's key. The daemon's block is under
// server., so the server's bare policy.enforcement is not the daemon's, and
// refusing to start over it would blame a setting the daemon never read.
func TestCheckRetired_IgnoresAnotherLoadersKey(t *testing.T) {
	t.Parallel()

	require.NoError(t, config.CheckRetired(keysSet{"policy.enforcement.fetch": true}, daemonKeys))
}

// The variable names are derived rather than listed, because the three
// prefixes are the only thing that differs and listing them once per loader
// is how they come to be named after each other.
func TestCheckRetired_NamesEachLoadersOwnEnvironmentVariable(t *testing.T) {
	for _, tt := range []struct {
		at   config.KeyLocation
		env  string
		want string
	}{
		{serverKeys, "DIRECTORY_SERVER_POLICY_ENFORCEMENT_FETCH", `op: ["pull"]`},
		{serverKeys, "DIRECTORY_SERVER_POLICY_ENFORCEMENT_SEARCH", `op: ["search"]`},
		{serverKeys, "DIRECTORY_SERVER_POLICY_ENFORCEMENT_POLICIES", "compliant_and_current"},
		{serverKeys, "DIRECTORY_SERVER_POLICY_ENFORCEMENT_REFRESH_INTERVAL", "policy.refresh_interval"},
		{daemonKeys, "DIRECTORY_DAEMON_SERVER_POLICY_ENFORCEMENT_FETCH", `op: ["pull"]`},
		{reconcilerKeys, "RECONCILER_POLICY_ENFORCEMENT_FETCH", `op: ["pull"]`},
	} {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv(tt.env, "enforce")

			err := config.CheckRetired(keysSet{}, tt.at)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.env)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// Presence is the test, not value: every loader here calls AllowEmptyEnv, so
// an exported-but-empty variable was a setting too.
func TestCheckRetired_CatchesAnEmptyVariable(t *testing.T) {
	t.Setenv("DIRECTORY_SERVER_POLICY_ENFORCEMENT_FETCH", "")

	require.Error(t, config.CheckRetired(keysSet{}, serverKeys))
}

// The block stands for its children, so a configuration setting both is told
// once rather than twice about the same thing. An empty block is still
// reported, since there is nothing else to report it by.
func TestCheckRetired_ReportsTheBlockOnlyWhenNothingUnderItWas(t *testing.T) {
	t.Parallel()

	withChild := config.CheckRetired(keysSet{
		"policy.enforcement":       true,
		"policy.enforcement.fetch": true,
	}, serverKeys)
	require.Error(t, withChild)
	assert.Contains(t, withChild.Error(), "policy.enforcement.fetch")
	assert.NotContains(t, withChild.Error(), "policy.enforcement is set")

	bare := config.CheckRetired(keysSet{"policy.enforcement": true}, serverKeys)
	require.Error(t, bare)
	assert.Contains(t, bare.Error(), "policy.enforcement is set")
}

// A configuration carrying none of them loads, and one carrying only the
// settings that replaced them is not mistaken for one that carries the old.
func TestCheckRetired_AcceptsWhatReplacedThem(t *testing.T) {
	t.Parallel()

	require.NoError(t, config.CheckRetired(keysSet{
		"policy.refresh_interval": true,
		"policy.operations":       true,
		"policy.evaluations":      true,
		"policy.dir":              true,
	}, serverKeys))
}

// A loader holding the block twice is told about both copies at once. The
// daemon configures a server and a reconciler, each with one, and an
// operator who fixes what the first start reported should not be stopped by
// the second on the next.
func TestCheckRetired_ChecksEveryCopyOfTheBlock(t *testing.T) {
	t.Parallel()

	err := config.CheckRetired(keysSet{
		"server.policy.enforcement.fetch":      true,
		"reconciler.policy.enforcement.search": true,
	}, daemonKeys, daemonReconcilerKeys)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "server.policy.enforcement.fetch")
	assert.Contains(t, err.Error(), "reconciler.policy.enforcement.search")
}

// Each copy is read on its own, so a block found whole in one of them is
// still reported when the other supplied the children. Sharing the count
// would drop the line that is the only evidence of the second copy.
func TestCheckRetired_ReportsTheBlockFoundInASecondCopy(t *testing.T) {
	t.Parallel()

	err := config.CheckRetired(keysSet{
		"server.policy.enforcement.fetch": true,
		"reconciler.policy.enforcement":   true,
	}, daemonKeys, daemonReconcilerKeys)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reconciler.policy.enforcement is set")
}

// Whether the file carries one decides whether a caller can name the file,
// and the environment is not the file's doing.
func TestCheckRetired_SaysWhetherTheFileCarriesOne(t *testing.T) {
	var refused *config.RetiredError

	t.Run("from the file", func(t *testing.T) {
		err := config.CheckRetired(keysSet{"policy.enforcement.fetch": true}, serverKeys)
		require.ErrorAs(t, err, &refused)
		assert.True(t, refused.InFile)
	})

	t.Run("from the environment", func(t *testing.T) {
		t.Setenv("DIRECTORY_SERVER_POLICY_ENFORCEMENT_FETCH", "enforce")

		err := config.CheckRetired(keysSet{}, serverKeys)
		require.ErrorAs(t, err, &refused)
		assert.False(t, refused.InFile)
	})

	// No variable ever set the block itself, but viper answers IsSet from
	// the environment for any key at all. Read through viper, a variable
	// left over the block would look like configuration, and the caller
	// would name a file that does not contain it.
	t.Run("the block itself, from the environment", func(t *testing.T) {
		t.Setenv("DIRECTORY_SERVER_POLICY_ENFORCEMENT", "")

		err := config.CheckRetired(viperLike{"policy.enforcement"}, serverKeys)
		require.ErrorAs(t, err, &refused)
		assert.False(t, refused.InFile)
		assert.Contains(t, err.Error(), "DIRECTORY_SERVER_POLICY_ENFORCEMENT is set")
	})
}

// viperLike answers IsSet for these keys the way viper does with
// AutomaticEnv: from the environment, whether or not the key is in the file.
type viperLike []string

func (k viperLike) IsSet(key string) bool { return slices.Contains(k, key) }
