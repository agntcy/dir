// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

const prioritizedDetailKeys = 2

func newOutput(cfg *client.Config, resolved *clientconfig.ResolvedContext, bootstrapPeers []string, results []checkResult) doctorOutput {
	info := configInfo{
		BootstrapPeers: len(bootstrapPeers),
	}
	if cfg != nil {
		info.ServerAddress = cfg.ServerAddress
		info.AuthMode = cfg.AuthMode
		info.TLSSkipVerify = cfg.TlsSkipVerify
	}

	if resolved != nil {
		info.Context = resolved.Name
		info.ContextSource = resolved.Source
		info.ConfigPath = resolved.Path
	}

	return doctorOutput{
		Config:  info,
		Summary: summarize(results),
		Results: results,
	}
}

func printOutput(cmd *cobra.Command, output doctorOutput) error {
	switch presenter.GetOutputOptions(cmd).Format {
	case presenter.FormatHuman:
		printHuman(cmd, output)

		return nil
	case presenter.FormatJSON:
		return printJSON(cmd, output, true)
	case presenter.FormatJSONL:
		return printJSON(cmd, output, false)
	case presenter.FormatRaw:
		presenter.Printf(cmd, "%d passed, %d failed, %d warned, %d skipped\n",
			output.Summary.Passed,
			output.Summary.Failed,
			output.Summary.Warned,
			output.Summary.Skipped,
		)

		return nil
	default:
		return fmt.Errorf("unsupported output format %q", presenter.GetOutputOptions(cmd).Format)
	}
}

func printJSON(cmd *cobra.Command, output doctorOutput, pretty bool) error {
	var (
		data []byte
		err  error
	)
	if pretty {
		data, err = json.MarshalIndent(output, "", "  ")
	} else {
		data, err = json.Marshal(output)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal doctor output: %w", err)
	}

	presenter.Printf(cmd, "%s\n", string(data))

	return nil
}

func printHuman(cmd *cobra.Command, output doctorOutput) {
	presenter.Println(cmd, "dirctl doctor")
	printConfig(cmd, output.Config)

	for _, result := range output.Results {
		if result.Elapsed == "" {
			presenter.Printf(cmd, "[%s] %s: %s\n", result.Status, result.Name, result.Message)
		} else {
			presenter.Printf(cmd, "[%s] %s: %s (%s)\n", result.Status, result.Name, result.Message, result.Elapsed)
		}

		printDetails(cmd, result)
	}

	presenter.Printf(cmd, "\nSummary: %d passed, %d failed, %d warned, %d skipped\n",
		output.Summary.Passed,
		output.Summary.Failed,
		output.Summary.Warned,
		output.Summary.Skipped,
	)
}

func printConfig(cmd *cobra.Command, info configInfo) {
	if info.Context == "" {
		presenter.Printf(cmd, "context: none (%s)\n", info.ContextSource)
	} else {
		presenter.Printf(cmd, "context: %s (%s)\n", info.Context, info.ContextSource)
	}

	if info.ConfigPath == "" {
		presenter.Println(cmd, "config: not found")
	} else {
		presenter.Printf(cmd, "config: %s\n", info.ConfigPath)
	}

	if info.ServerAddress == "" {
		presenter.Println(cmd, "server: not configured")
	} else {
		presenter.Printf(cmd, "server: %s\n", info.ServerAddress)
	}

	if info.AuthMode != "" {
		presenter.Printf(cmd, "auth: %s\n", info.AuthMode)
	}

	if info.TLSSkipVerify {
		presenter.Println(cmd, "tls_skip_verify: true")
	}

	presenter.Printf(cmd, "bootstrap peers: %d\n\n", info.BootstrapPeers)
}

func printDetails(cmd *cobra.Command, result checkResult) {
	if result.Status != statusFail && result.Status != statusWarn {
		return
	}

	for _, key := range detailKeys(result.Details) {
		presenter.Printf(cmd, "  %s: %s\n", key, result.Details[key])
	}
}

func detailKeys(details map[string]string) []string {
	if len(details) == 0 {
		return nil
	}

	keys := make([]string, 0, len(details))
	if _, ok := details["error"]; ok {
		keys = append(keys, "error")
	}

	if _, ok := details["close_error"]; ok {
		keys = append(keys, "close_error")
	}

	for key := range details {
		if key != "error" && key != "close_error" {
			keys = append(keys, key)
		}
	}

	if len(keys) > prioritizedDetailKeys {
		sort.Strings(keys[prioritizedDetailKeys:])
	}

	return keys
}
