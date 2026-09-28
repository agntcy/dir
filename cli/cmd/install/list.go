// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/internal/pkgstate"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

// ListCommand is exported so root.go can skip client setup for it: it reads
// only local state.
//
// Up to v1.7.0 this name showed detected agents; that view is now
// `dirctl install agents`.
var ListCommand = &cobra.Command{
	Use:   "list [name]",
	Short: "Show installed packages, or one package's installed artifacts",
	Long: `List what install has put on this machine, from the install manifest at
$XDG_CONFIG_HOME/dirctl/installed.json. Reads only local state and does not
contact the Directory.

  dirctl install list                  every installed package, one row per agent
  dirctl install list <name>           that package's skill files and MCP entries
  dirctl install list -o json          the same rows as JSON

Installed artifacts carry no provenance of their own, so the manifest is the
only record of what was written — and the only thing that can be checked
against disk. Every row is stat'd, and one whose artifacts are gone is marked
"missing".

Up to v1.7.0 this command showed detected agents; that view is now
` + "`dirctl install agents`" + `.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		structured, err := structuredOutput(cmd)
		if err != nil {
			return err
		}

		manifest, err := readManifest()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			return listOnePackage(cmd, manifest, args[0], structured)
		}

		return listPackages(cmd, manifest, structured)
	},
}

func init() {
	addOutputFlag(ListCommand)
}

// listedPackage is one manifest row as `install list` reports it.
type listedPackage struct {
	Name    string          `json:"name"`
	Version string          `json:"version,omitempty"`
	CID     string          `json:"cid,omitempty"`
	Agent   string          `json:"agent"`
	Kind    string          `json:"kind"`
	Scope   pkgstate.Scope  `json:"scope"`
	Origin  pkgstate.Origin `json:"origin"`
	Pinned  bool            `json:"pinned"`
	Missing bool            `json:"missing"`
}

// listedArtifacts is one row's artifacts, as `install list <name>` reports them.
type listedArtifacts struct {
	listedPackage

	Artifacts []agentinstall.Installed `json:"artifacts"`
}

func listPackages(cmd *cobra.Command, manifest *pkgstate.Manifest, structured bool) error {
	env := agentcfg.ResolveEnv()

	rows := make([]listedPackage, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		rows = append(rows, describe(entry, env))
	}

	if structured {
		return printJSONRows(cmd, rows)
	}

	if len(rows) == 0 {
		presenter.Printf(cmd, "No packages installed.\n")

		return nil
	}

	// The first four columns match `install outdated`, which then continues with
	// the installed version, so both tables identify a row the same way.
	table := newTable("NAME", "AGENT", "KIND", "SCOPE", "VERSION", "FLAGS")
	for _, row := range rows {
		table.row(row.Name, row.Agent, row.Kind, shortScope(row.Scope), orDash(row.Version), flags(row))
	}

	presenter.Printf(cmd, "%s", table.String())

	return nil
}

func listOnePackage(cmd *cobra.Command, manifest *pkgstate.Manifest, name string, structured bool) error {
	entries := manifest.ByName(name)
	if len(entries) == 0 {
		return fmt.Errorf("%q is not installed", name)
	}

	env := agentcfg.ResolveEnv()

	rows := make([]listedArtifacts, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, listedArtifacts{
			listedPackage: describe(entry, env),
			Artifacts:     agentinstall.Inspect(entry, env),
		})
	}

	if structured {
		return printJSONRows(cmd, rows)
	}

	for i, row := range rows {
		if i > 0 {
			presenter.Printf(cmd, "\n")
		}

		printArtifacts(cmd, row)
	}

	return nil
}

// printArtifacts renders one agent's installed artifacts: the skill folder with
// its files indented under it, then each MCP server key with the config file it
// lives in.
func printArtifacts(cmd *cobra.Command, row listedArtifacts) {
	presenter.Printf(cmd, "%s %s [%s, %s]%s\n",
		row.Name, orDash(row.Version), row.Agent, shortScope(row.Scope), flagSuffix(row.listedPackage))

	if len(row.Artifacts) == 0 {
		presenter.Printf(cmd, "  (no artifacts recorded)\n")

		return
	}

	for _, artifact := range row.Artifacts {
		switch artifact.Kind {
		case agentcfg.ArtifactSkill:
			presenter.Printf(cmd, "  skill  %s%s\n", artifact.Path, presenceMark(artifact))

			for _, file := range artifact.Files {
				presenter.Printf(cmd, "           %s%s\n", file.Name, absentMark(file.Present))
			}
		case agentcfg.ArtifactMCP:
			presenter.Printf(cmd, "  mcp    %s in %s%s\n",
				artifact.Server, orDash(artifact.Path), presenceMark(artifact))
		}
	}
}

// describe turns a manifest row into its reported form, stat'ing the artifacts
// it names rather than trusting the row.
func describe(entry pkgstate.Entry, env agentcfg.Env) listedPackage {
	return listedPackage{
		Name:    entry.Name,
		Version: entry.Version,
		CID:     entry.CID,
		Agent:   entry.Agent,
		Kind:    entry.Kind(),
		Scope:   entry.Scope,
		Origin:  entry.Origin,
		Pinned:  entry.Pinned,
		Missing: !agentinstall.Present(entry, env),
	}
}

// flags renders the row's notable states for the FLAGS column: whether it is
// held, whether dirctl built it rather than pulled it, and whether what it
// names is still there.
func flags(row listedPackage) string {
	var set []string

	if row.Pinned {
		set = append(set, "pinned")
	}

	if row.Origin == pkgstate.OriginBuiltin {
		set = append(set, "builtin")
	}

	if row.Missing {
		set = append(set, "missing")
	}

	if len(set) == 0 {
		return "-"
	}

	return strings.Join(set, ",")
}

func flagSuffix(row listedPackage) string {
	if f := flags(row); f != "-" {
		return " " + f
	}

	return ""
}

// presenceMark annotates an artifact the row names but disk does not have,
// keeping "we looked and it is gone" apart from "we could not look" — an
// unreadable config or an agent this binary does not know.
func presenceMark(artifact agentinstall.Installed) string {
	if artifact.Unchecked {
		return "  (could not check)"
	}

	return absentMark(artifact.Present)
}

// absentMark annotates a file the row names but disk does not have.
func absentMark(present bool) string {
	if present {
		return ""
	}

	return "  (missing)"
}
