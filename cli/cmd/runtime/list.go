// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

// agentRecordLabel is the label or annotation that references the OASF record a workload
// was started from.
const agentRecordLabel = "org.agntcy/agent-record"

var listOpts struct {
	Names            []string
	Skills           []string
	Domains          []string
	Authors          []string
	SchemaVersions   []string
	LocatorProtocols []string
	RuntimeTypes     []string
	ReachableFrom    string
	Labels           []string
	SchemaURLs       []string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running workloads",
	Long: `List running workloads discovered by runtime discovery.

The agent filters (--name, --skill, --domain, --author, --schema-version) read what a
workload says about the agent it runs: its A2A card's name and OASF discovery and
identity extensions, or else the OASF record it references. Values match as in
dirctl search: exactly (case-insensitive), or with * and ? wildcards. Labels are
regular expressions. Different filters combine with AND; a repeated flag matches any
of its values.

The default output summarizes each workload. --output json and jsonl print the
workloads exactly as runtime discovery returns them, A2A card included; --output raw
prints their IDs.

Workloads are listed as announced: nothing here is verified.

Usage examples:

1. List every running workload:

	dirctl runtime list

2. Agents with a debugging skill, reachable over SLIM:

	dirctl runtime list --skill "software_engineering/code_debugging/*" --locator-protocol slim

3. Agents vouched for by a given human, as JSON:

	dirctl runtime list --author did:key:z6Mk... --output json

4. Workloads that can reach a given workload:

	dirctl runtime list --reachable-from <workload-id>

5. With a schema server running locally, accept its extensions too:

	dirctl runtime list --oasf-schema-url https://schema.oasf.outshift.com --oasf-schema-url http://localhost:8080
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runList(cmd)
	},
}

func init() {
	flags := listCmd.Flags()
	flags.StringArrayVar(&listOpts.Names, "name", nil, "Agent name (repeatable)")
	flags.StringArrayVar(&listOpts.Skills, "skill", nil, "OASF skill name (repeatable)")
	flags.StringArrayVar(&listOpts.Domains, "domain", nil, "OASF domain name (repeatable)")
	flags.StringArrayVar(&listOpts.Authors, "author", nil, "Author, e.g. the DID of the human vouching for the agent (repeatable)")
	flags.StringArrayVar(&listOpts.SchemaVersions, "schema-version", nil, "OASF schema version (repeatable)")
	flags.StringArrayVar(&listOpts.LocatorProtocols, "locator-protocol", nil, "Locator protocol, e.g. slim, grpc, http, tcp (repeatable)")
	flags.StringArrayVar(&listOpts.RuntimeTypes, "runtime", nil, "Runtime type: docker, kubernetes, process (repeatable)")
	flags.StringVar(&listOpts.ReachableFrom, "reachable-from", "", "Only workloads reachable from the workload with this ID")
	flags.StringArrayVar(&listOpts.Labels, "label", nil, "Label filter key=regex (repeatable)")
	flags.StringArrayVar(&listOpts.SchemaURLs, "oasf-schema-url", []string{runtimev1.DefaultOASFSchemaURL},
		"OASF schema server whose card extensions are read (repeatable); should match the server's")

	presenter.AddOutputFlags(listCmd)
}

func runList(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	req, err := buildRequest()
	if err != nil {
		return err
	}

	workloads, err := c.ListWorkloadsWithRequest(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to list workloads: %w", err)
	}

	// Filter again on the client: a server older than these filters ignores them and
	// returns every workload.
	filterOpts := runtimev1.FilterOptions{AcceptedSchemaURLs: listOpts.SchemaURLs}

	if req.ReachableFromWorkloadId != nil {
		// Without the reference workload nothing would match, which would hide the failure
		// behind an empty result.
		from, err := c.GetWorkload(cmd.Context(), req.GetReachableFromWorkloadId())
		if err != nil {
			return fmt.Errorf("failed to look up --reachable-from workload %q: %w", req.GetReachableFromWorkloadId(), err)
		}

		filterOpts.ReachableFrom = from
	}

	matched := make([]*runtimev1.Workload, 0, len(workloads))

	for _, w := range workloads {
		if w.MatchesFilters(req, filterOpts) {
			matched = append(matched, w)
		}
	}

	switch presenter.GetOutputOptions(cmd).Format {
	case presenter.FormatRaw:
		// Raw output is one workload ID per line, for piping into other commands.
		ids := make([]string, 0, len(matched))
		for _, w := range matched {
			ids = append(ids, w.GetId())
		}

		return presenter.PrintMessage(cmd, "workloads", "Workloads", ids)
	case presenter.FormatJSON, presenter.FormatJSONL:
		docs, err := workloadsJSON(matched)
		if err != nil {
			return err
		}

		return presenter.PrintMessage(cmd, "workloads", "Workloads", docs)
	case presenter.FormatHuman:
	}

	summaries := make([]summary, 0, len(matched))
	for _, w := range matched {
		summaries = append(summaries, summarize(w, listOpts.SchemaURLs))
	}

	printTable(cmd, summaries)

	return nil
}

// workloadsJSON renders each workload with protojson, so the output uses the API's
// field names and carries every field once.
func workloadsJSON(workloads []*runtimev1.Workload) ([]json.RawMessage, error) {
	docs := make([]json.RawMessage, 0, len(workloads))

	for _, w := range workloads {
		raw, err := protojson.Marshal(w)
		if err != nil {
			return nil, fmt.Errorf("failed to encode workload %q: %w", w.GetId(), err)
		}

		docs = append(docs, raw)
	}

	return docs, nil
}

func buildRequest() (*runtimev1.ListWorkloadsRequest, error) {
	// Repeated values for one key match any of them: they are joined into one
	// alternation, since the request holds a single regular expression per label.
	alternatives := make(map[string][]string, len(listOpts.Labels))

	var keys []string

	for _, label := range listOpts.Labels {
		key, value, ok := strings.Cut(label, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --label %q: want key=regex", label)
		}

		if _, seen := alternatives[key]; !seen {
			keys = append(keys, key)
		}

		alternatives[key] = append(alternatives[key], value)
	}

	labels := make(map[string]string, len(keys))

	for _, key := range keys {
		values := alternatives[key]
		if len(values) == 1 {
			labels[key] = values[0]

			continue
		}

		parts := make([]string, 0, len(values))
		for _, v := range values {
			parts = append(parts, "(?:"+v+")")
		}

		labels[key] = strings.Join(parts, "|")
	}

	req := &runtimev1.ListWorkloadsRequest{
		Labels:           labels,
		RuntimeTypes:     listOpts.RuntimeTypes,
		Skills:           listOpts.Skills,
		Domains:          listOpts.Domains,
		LocatorProtocols: listOpts.LocatorProtocols,
		Names:            listOpts.Names,
		Authors:          listOpts.Authors,
		SchemaVersions:   listOpts.SchemaVersions,
	}

	if listOpts.ReachableFrom != "" {
		req.ReachableFromWorkloadId = &listOpts.ReachableFrom
	}

	return req, nil
}

// summary is what the default output shows of a workload: what it is, where to reach
// it, and what its agent says about itself.
type summary struct {
	ID           string
	Runtime      string
	Type         string
	Hostname     string
	Locators     []string
	Agent        agent
	SourceRecord string
}

type agent struct {
	Name    string
	Version string
	Skills  []string
	Domains []string
	Authors []string
}

func summarize(w *runtimev1.Workload, schemaURLs []string) summary {
	info := w.AgentInfo(schemaURLs)

	out := summary{
		ID:       w.GetId(),
		Runtime:  w.GetRuntime(),
		Type:     w.GetType(),
		Hostname: w.GetHostname(),
		Agent: agent{
			Name:    info.Name,
			Version: agentVersion(w),
			Skills:  classNames(info.Skills),
			Domains: classNames(info.Domains),
			Authors: info.Authors,
		},
		SourceRecord: w.GetLabels()[agentRecordLabel],
	}

	if out.SourceRecord == "" {
		out.SourceRecord = w.GetAnnotations()[agentRecordLabel]
	}

	for _, l := range w.GetLocators() {
		out.Locators = append(out.Locators, l.GetUrl())
	}

	return out
}

// agentVersion is the version in the workload's A2A card, or else in the OASF record
// it references.
func agentVersion(w *runtimev1.Workload) string {
	if v := w.GetServices().GetA2A().GetFields()["version"].GetStringValue(); v != "" {
		return v
	}

	return w.GetServices().GetOasf().GetFields()["record"].GetStructValue().GetFields()["version"].GetStringValue()
}

func classNames(classes []runtimev1.OASFClass) []string {
	out := make([]string, 0, len(classes))
	for _, c := range classes {
		out = append(out, c.Name)
	}

	return out
}

func printTable(cmd *cobra.Command, listed []summary) {
	if len(listed) == 0 {
		presenter.Println(cmd, "No workloads found")

		return
	}

	for i, w := range listed {
		if i > 0 {
			presenter.Println(cmd, "")
		}

		presenter.Println(cmd, w.ID)

		printField(cmd, "runtime", runtimeLabel(w))
		printField(cmd, "host", w.Hostname)

		name := w.Agent.Name
		if name != "" && w.Agent.Version != "" {
			name += " " + w.Agent.Version
		}

		printField(cmd, "agent", name)
		printField(cmd, "locators", w.Locators...)
		printField(cmd, "skills", w.Agent.Skills...)
		printField(cmd, "domains", w.Agent.Domains...)
		printField(cmd, "authors", w.Agent.Authors...)
		printField(cmd, "record", w.SourceRecord)
	}

	presenter.Println(cmd, "")
	presenter.Println(cmd, "Workloads are listed as announced; nothing here is verified.")
}

// runtimeLabel is the runtime, with the workload type when it says more, e.g.
// "docker (container)" but just "process".
func runtimeLabel(w summary) string {
	if w.Type == "" || w.Type == w.Runtime {
		return w.Runtime
	}

	return w.Runtime + " (" + w.Type + ")"
}

// printField prints one labelled field, its values one per line under each other.
// Empty values are skipped, and so is a field without any.
func printField(cmd *cobra.Command, label string, values ...string) {
	const width = len("locators: ")

	prefix := fmt.Sprintf("  %-*s", width, label+":")

	for _, v := range values {
		if v == "" {
			continue
		}

		presenter.Println(cmd, prefix+v)

		prefix = strings.Repeat(" ", len(prefix))
	}
}
