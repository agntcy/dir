// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package dirpkg builds the built-in DIR package — the record dirctl compiles
// into itself.
//
// It is a package rather than a function inside `dirctl init` because two
// commands need the same derivation, and they must not drift apart.
// `init` installs the DIR skill as the last step of onboarding;
// `install upgrade` re-derives it locally instead of pulling the
// Directory-published record of the same name, which also removes any
// possibility of offering a downgrade when this binary leads the server's
// published build.
package dirpkg

import (
	"fmt"
	"time"

	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/server/skill"
)

// Name is the built-in record's name, which is also the key its manifest rows
// are stored under.
func Name() string { return skill.RecordName }

// Version is the built-in record's version, which for a built-in row is the
// upstream a version check compares against: the upstream is this binary.
func Version() string { return skill.RecordVersion() }

// Artifacts builds the built-in DIR record locally and derives its installable
// artifacts — the skill. No Directory round-trip.
func Artifacts() (agentinstall.Artifacts, error) {
	rec, err := skill.BuildRecord(time.Now().UTC())
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("build DIR record: %w", err)
	}

	arts, err := agentinstall.DeriveArtifacts(rec)
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("derive DIR artifacts: %w", err)
	}

	return arts, nil
}
