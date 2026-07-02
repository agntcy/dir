// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
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
	slug       string // sanitized record name; skill folder/file + block marker id
	skill      string // canonical SKILL.md content, empty if the record has no skill
	mcpServers []mcpServer
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
		f, err := exportfmt.GetFormatter(exportfmt.FormatAgentSkill)
		if err != nil {
			return artifacts{}, fmt.Errorf("get agent-skill formatter: %w", err)
		}

		out, err := f.Format(record)
		if err != nil {
			return artifacts{}, fmt.Errorf("render skill: %w", err)
		}

		arts.skill = string(out)
	}

	if ok, _ := recordutil.GetModule(data, translator.MCPModuleName); ok {
		cfg, err := translator.RecordToGHCopilot(data)
		if err != nil {
			return artifacts{}, fmt.Errorf("translate MCP module: %w", err)
		}

		for name, srv := range cfg.Servers {
			arts.mcpServers = append(arts.mcpServers, mcpServer{
				name:  name,
				entry: mcpEntry(srv),
			})
		}
	}

	if arts.skill == "" && len(arts.mcpServers) == 0 {
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
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")

	// Trim leading/trailing "." as well as "-" so a name of "." or ".." (and
	// "../../etc" after the separator replacement above) cannot escape a parent
	// directory when the slug is used as a filesystem path component. Embedded
	// dots (e.g. "io.example") are preserved as a single filename component.
	return strings.Trim(r.Replace(name), "-.")
}
