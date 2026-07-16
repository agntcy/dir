// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/dir/cli/internal/agentcfg/fsutil"
)

const skillFilePerm = 0o644

// InstallSkill renders and writes the DIR skill/rules for an agent using its
// strategy. It is idempotent: identical content reports ActionUnchanged. When
// the global path is unavailable it falls back to the project path and notes
// this in the outcome reason.
func InstallSkill(target *SkillTarget, env Env, slug, canonical string, dryRun bool) (Outcome, error) {
	path, usedProject, err := resolveSkillTargetPath(target, env, slug)
	if err != nil {
		return Outcome{Artifact: "skill", Action: ActionFailed, Err: err}, err
	}

	outcome := Outcome{Artifact: "skill", Path: path}
	if usedProject {
		outcome.Reason = "no global rules path for this agent — wrote a project rule in the current repo"
	}

	desired, err := renderForTarget(target, slug, canonical, path)
	if err != nil {
		outcome.Action = ActionFailed
		outcome.Err = err

		return outcome, err
	}

	existing, existed, err := readFileIfExists(path)
	if err != nil {
		return failOutcome(outcome, err)
	}

	switch {
	case target.Strategy == ManagedBlock:
		hadBlock := hasBlock(slug, string(existing))
		if bytes.Equal(existing, desired) {
			outcome.Action = ActionUnchanged

			return outcome, nil
		}

		if existed && hadBlock {
			outcome.Action = ActionUpdated
		} else {
			outcome.Action = ActionAdded
		}
	case !existed:
		outcome.Action = ActionAdded
	case bytes.Equal(existing, desired):
		outcome.Action = ActionUnchanged

		return outcome, nil
	default:
		outcome.Action = ActionUpdated
	}

	if dryRun {
		return outcome, nil
	}

	if err := fsutil.WriteAtomic(path, desired, skillFilePerm); err != nil {
		return failOutcome(outcome, fmt.Errorf("write skill %s: %w", path, err))
	}

	return outcome, nil
}

// InstallSkillBundle extracts a skill bundle archive into an agent's skill folder.
// It is idempotent: identical on-disk content reports ActionUnchanged.
func InstallSkillBundle(target *SkillTarget, env Env, slug string, archive []byte, dryRun bool) (Outcome, error) {
	path, usedProject, err := resolveSkillTargetPath(target, env, slug)
	if err != nil {
		return Outcome{Artifact: "skill", Action: ActionFailed, Err: err}, err
	}

	destDir := filepath.Dir(path)

	outcome := Outcome{Artifact: "skill", Path: destDir}
	if usedProject {
		outcome.Reason = "no global rules path for this agent — wrote a project rule in the current repo"
	}

	matches, err := exportfmt.SkillBundleMatchesDir(archive, destDir)
	if err != nil {
		return failOutcome(outcome, fmt.Errorf("compare skill bundle %s: %w", destDir, err))
	}

	if matches {
		outcome.Action = ActionUnchanged

		return outcome, nil
	}

	if _, err := os.Stat(path); err == nil {
		outcome.Action = ActionUpdated
	} else if os.IsNotExist(err) {
		outcome.Action = ActionAdded
	} else {
		return failOutcome(outcome, fmt.Errorf("stat skill %s: %w", path, err))
	}

	if dryRun {
		return outcome, nil
	}

	if err := exportfmt.ExtractSkillBundleArchive(archive, destDir); err != nil {
		return failOutcome(outcome, fmt.Errorf("extract skill bundle %s: %w", destDir, err))
	}

	return outcome, nil
}

// RemoveSkill removes the DIR skill/rules we installed for an agent, preserving
// any surrounding user content for managed-block files. Absent artifacts report
// ActionUnchanged so uninstall is idempotent.
func RemoveSkill(target *SkillTarget, env Env, slug string, dryRun bool) (Outcome, error) {
	path, usedProject, err := resolveSkillTargetPath(target, env, slug)
	if err != nil {
		return Outcome{Artifact: "skill", Action: ActionFailed, Err: err}, err
	}

	outcome := Outcome{Artifact: "skill", Path: path}
	_ = usedProject

	switch target.Strategy {
	case SkillFolder:
		return removeSkillFolder(outcome, path, dryRun)
	case ManagedBlock:
		return removeSkillBlock(outcome, path, slug, dryRun)
	case DedicatedFile:
		fallthrough
	default:
		return removeSkillFile(outcome, path, dryRun)
	}
}

// renderForTarget produces the on-disk bytes for the target strategy.
func renderForTarget(target *SkillTarget, slug, canonical, path string) ([]byte, error) {
	switch target.Strategy {
	case SkillFolder:
		return []byte(canonical), nil
	case ManagedBlock:
		if target.Render == nil {
			return nil, fmt.Errorf("managed-block target %s has no renderer", path)
		}

		inner, err := target.Render(canonical)
		if err != nil {
			return nil, err
		}

		existing, _, err := readFileIfExists(path)
		if err != nil {
			return nil, err
		}

		return []byte(upsertBlock(slug, string(existing), string(inner))), nil
	case DedicatedFile:
		fallthrough
	default:
		if target.Render == nil {
			return nil, fmt.Errorf("dedicated-file target %s has no renderer", path)
		}

		return target.Render(canonical)
	}
}

func removeSkillFolder(outcome Outcome, path string, dryRun bool) (Outcome, error) {
	dir := filepath.Dir(path) // the slug/ folder we own
	outcome.Path = dir

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		outcome.Action = ActionUnchanged

		return outcome, nil
	}

	outcome.Action = ActionRemoved
	if dryRun {
		return outcome, nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return failOutcome(outcome, fmt.Errorf("remove skill folder %s: %w", dir, err))
	}

	return outcome, nil
}

func removeSkillFile(outcome Outcome, path string, dryRun bool) (Outcome, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		outcome.Action = ActionUnchanged

		return outcome, nil
	}

	outcome.Action = ActionRemoved
	if dryRun {
		return outcome, nil
	}

	if err := os.Remove(path); err != nil {
		return failOutcome(outcome, fmt.Errorf("remove skill file %s: %w", path, err))
	}

	return outcome, nil
}

func removeSkillBlock(outcome Outcome, path, slug string, dryRun bool) (Outcome, error) {
	existing, existed, err := readFileIfExists(path)
	if err != nil {
		return failOutcome(outcome, err)
	}

	if !existed {
		outcome.Action = ActionUnchanged

		return outcome, nil
	}

	stripped, removed := removeBlock(slug, string(existing))
	if !removed {
		outcome.Action = ActionUnchanged

		return outcome, nil
	}

	outcome.Action = ActionRemoved
	if dryRun {
		return outcome, nil
	}

	// If our block was the only content, remove the now-empty file.
	if strings.TrimSpace(stripped) == "" {
		if err := os.Remove(path); err != nil {
			return failOutcome(outcome, fmt.Errorf("remove emptied skill file %s: %w", path, err))
		}

		return outcome, nil
	}

	if err := fsutil.WriteAtomic(path, []byte(stripped), skillFilePerm); err != nil {
		return failOutcome(outcome, fmt.Errorf("write skill %s: %w", path, err))
	}

	return outcome, nil
}

// resolveSkillTargetPath resolves the global path, falling back to the project
// path when the global resolver reports ErrNoGlobalPath. It reports whether the
// project fallback was used.
func resolveSkillTargetPath(target *SkillTarget, env Env, slug string) (string, bool, error) {
	if target.Path != nil {
		path, err := target.Path(env, slug)
		if err == nil {
			return path, false, nil
		}

		if !errors.Is(err, ErrNoGlobalPath) {
			return "", false, err
		}
	}

	if target.ProjectPath != nil {
		path, err := target.ProjectPath(env, slug)
		if err != nil {
			return "", false, err
		}

		return path, true, nil
	}

	return "", false, ErrNoGlobalPath
}

// ResolveSkillTargetPath is the exported wrapper around resolveSkillTargetPath,
// used by the command layer for dedupe and display.
func ResolveSkillTargetPath(target *SkillTarget, env Env, slug string) (string, bool, error) {
	return resolveSkillTargetPath(target, env, slug)
}

// readFileIfExists reads path and reports whether it exists. A missing file is
// (nil, false, nil); a real read error (e.g. permissions) is (nil, false, err)
// so callers can surface it rather than silently treating the file as absent.
func readFileIfExists(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}

	return data, true, nil
}
