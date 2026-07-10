// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"sort"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/dir/cli/util/records"
	recordutil "github.com/agntcy/oasf-sdk/pkg/record"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/types/known/structpb"
)

// mcpServer is one MCP server entry to place: the key it is stored under and the
// {command, args, env} value (any-typed for round-trip idempotency).
type mcpServer struct {
	name  string
	entry map[string]any
}

// artifacts is the set of installable artifacts derived from a record's modules.
type artifacts struct {
	slug        string // sanitized record name; skill folder/file + block marker id
	skill       string // canonical SKILL.md content for single-file skill targets
	skillBundle []byte // gzip skill bundle when the record stores application/agent-skills+gzip
	mcpServers  []mcpServer
}

func (a artifacts) hasSkill() bool {
	return a.skill != "" || a.hasSkillBundle()
}

func (a artifacts) hasSkillBundle() bool {
	return len(a.skillBundle) > 0
}

// deriveArtifacts inspects the record's OASF modules and builds the installable
// artifact set. A record with only integration/a2a, or with no installable
// module, returns an error.
func deriveArtifacts(record *corev1.Record) (artifacts, error) {
	data := record.GetData()
	if data == nil {
		return artifacts{}, fmt.Errorf("record contains no data")
	}

	arts := artifacts{slug: sanitizeSlug(record.GetName())}

	if ok, _ := recordutil.GetModule(data, translator.AgentSkillsModuleName); ok {
		if err := deriveSkillArtifacts(record, &arts); err != nil {
			return artifacts{}, err
		}
	}

	if ok, _ := recordutil.GetModule(data, translator.MCPModuleName); ok {
		cfg, err := translator.RecordToGHCopilot(data)
		if err != nil {
			return artifacts{}, fmt.Errorf("translate MCP module: %w", err)
		}

		// Sort server names so the derived order (and thus plan/summary output) is
		// deterministic across runs; map iteration order is otherwise random.
		names := make([]string, 0, len(cfg.Servers))
		for name := range cfg.Servers {
			names = append(names, name)
		}

		sort.Strings(names)

		for _, name := range names {
			arts.mcpServers = append(arts.mcpServers, mcpServer{
				name:  name,
				entry: mcpEntry(cfg.Servers[name]),
			})
		}
	}

	if !arts.hasSkill() && len(arts.mcpServers) == 0 {
		if ok, _ := recordutil.GetModule(data, translator.A2AModuleName); ok {
			return artifacts{}, fmt.Errorf(
				"record carries an A2A AgentCard (%s), which cannot be installed into agent configs; use `dirctl export` instead",
				translator.A2AModuleName)
		}

		return artifacts{}, fmt.Errorf(
			"record has no installable module (found: %s); installable modules are %s and %s",
			strings.Join(moduleNames(data), ", "),
			translator.AgentSkillsModuleName, translator.MCPModuleName)
	}

	if arts.slug == "" {
		return artifacts{}, fmt.Errorf("record has no name; cannot derive an install identity")
	}

	return arts, nil
}

func deriveSkillArtifacts(record *corev1.Record, arts *artifacts) error {
	data := record.GetData()
	mediaType := agentSkillsArtifactMediaType(data)

	formatName := exportfmt.FormatAgentSkill
	if mediaType == catalogv1.ProtocolAgentSkillsBundleMediaType {
		formatName = exportfmt.FormatAgentSkillBundle
	}

	f, err := exportfmt.GetFormatter(formatName)
	if err != nil {
		return fmt.Errorf("get %s formatter: %w", formatName, err)
	}

	out, err := f.Format(record)
	if err != nil {
		return fmt.Errorf("render skill: %w", err)
	}

	if formatName == exportfmt.FormatAgentSkillBundle {
		arts.skillBundle = out

		md, err := exportfmt.SkillMarkdownFromArchive(out)
		if err != nil {
			return fmt.Errorf("read SKILL.md from bundle: %w", err)
		}

		arts.skill = md

		return nil
	}

	arts.skill = string(out)

	return nil
}

func agentSkillsArtifactMediaType(data *structpb.Struct) string {
	ok, module := recordutil.GetModule(data, translator.AgentSkillsModuleName)
	if !ok || module == nil {
		return ""
	}

	return module.GetFields()["artifact"].GetStructValue().GetFields()["media_type"].GetStringValue()
}

// mcpEntry converts a translator MCP server into a config entry using any-typed
// collections so InstallMCP's DeepEqual idempotency holds after a JSON/YAML/TOML
// round-trip.
func mcpEntry(srv translator.MCPServer) map[string]any {
	args := make([]any, 0, len(srv.Args))
	for _, a := range srv.Args {
		args = append(args, a)
	}

	env := map[string]any{}
	for k, v := range srv.Env {
		env[k] = v
	}

	return map[string]any{
		"command": srv.Command,
		"args":    args,
		"env":     env,
	}
}

// moduleNames lists the module names present in the record data, for error text.
func moduleNames(data *structpb.Struct) []string {
	mods, ok := data.GetFields()["modules"]
	if !ok {
		return []string{"none"}
	}

	var names []string

	for _, m := range mods.GetListValue().GetValues() {
		if n := m.GetStructValue().GetFields()["name"]; n != nil {
			names = append(names, n.GetStringValue())
		}
	}

	if len(names) == 0 {
		return []string{"none"}
	}

	return names
}

// sanitizeSlug turns a record name into a filesystem/marker-safe slug.
func sanitizeSlug(name string) string {
	return records.SanitizeSlug(name)
}
