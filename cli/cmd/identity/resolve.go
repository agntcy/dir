// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package identity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve <name[:version]>",
	Short: "Resolve a record name to the CIDs of its versions",
	Long: `Resolve a record name to the CIDs of its versions.

Without a version, all versions of the name are listed, newest first. With a
version, only that version is.

Usage examples:

1. List all versions of a name:
   dirctl identity resolve cisco.com/agent

2. Resolve one version:
   dirctl identity resolve cisco.com/agent:v1.0.0

3. Print only the CIDs:
   dirctl identity resolve cisco.com/agent --output raw
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runResolve(cmd, args[0])
	},
}

// resolvedRecord is one resolved record as printed.
type resolvedRecord struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	CID     string `json:"cid"`
}

func runResolve(cmd *cobra.Command, input string) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	ref := reference.Parse(input)
	if ref.IsCID() {
		return fmt.Errorf("%q is a CID: resolve takes a record name, optionally with a version", input)
	}

	resp, err := c.ResolveIdentity(cmd.Context(), ref.Name, ref.Version)
	if err != nil {
		return err
	}

	records := make([]resolvedRecord, 0, len(resp.GetRecords()))
	for _, r := range resp.GetRecords() {
		records = append(records, resolvedRecord{Name: r.GetName(), Version: r.GetVersion(), CID: r.GetCid()})
	}

	switch presenter.GetOutputOptions(cmd).Format {
	case presenter.FormatRaw:
		cids := make([]string, 0, len(records))
		for _, r := range records {
			cids = append(cids, r.CID)
		}

		return presenter.PrintMessage(cmd, "Records", "Records", cids)
	case presenter.FormatHuman:
		var b strings.Builder
		for _, r := range records {
			fmt.Fprintf(&b, "%s  %s  %s\n", r.CID, r.Name, r.Version)
		}

		presenter.Printf(cmd, "%s", b.String())

		return nil
	case presenter.FormatJSON, presenter.FormatJSONL:
		return presenter.PrintMessage(cmd, "Records", "Records", records)
	}

	return nil
}
