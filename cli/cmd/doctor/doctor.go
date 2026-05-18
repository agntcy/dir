// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"errors"
	"time"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

const defaultTimeout = 2 * time.Second

// ErrChecksFailed is returned when at least one doctor check fails.
var ErrChecksFailed = errors.New("one or more doctor checks failed")

var opts = options{
	Timeout: defaultTimeout,
}

type options struct {
	Timeout        time.Duration
	BootstrapPeers []string
}

// Command runs read-only diagnostics for the effective Directory endpoint.
var Command = &cobra.Command{
	Use:   "doctor",
	Short: "Run read-only diagnostics for Directory endpoint health",
	Long: `Run read-only diagnostics for the effective Directory endpoint.

The command resolves the same client context and authentication settings as
other dirctl API commands, then reports configuration, TCP, gRPC, routing, and
optional bootstrap/DHT reachability checks.`,
	Args: cobra.NoArgs,
	// Override root PersistentPreRunE so config and client setup failures can be
	// emitted as structured doctor results instead of aborting before RunE.
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		return nil
	},
	RunE: runCommand,
}

func init() {
	Command.Flags().DurationVar(&opts.Timeout, "timeout", defaultTimeout, "Per-check timeout")
	Command.Flags().StringSliceVar(&opts.BootstrapPeers, "bootstrap-peer", nil, "Bootstrap peer multiaddr; may be repeated")
	presenter.AddOutputFlags(Command)
}

func runCommand(cmd *cobra.Command, _ []string) error {
	output := runDoctor(cmd.Context(), cmd)
	if err := printOutput(cmd, output); err != nil {
		return err
	}

	if hasFailure(output.Results) {
		return ErrChecksFailed
	}

	return nil
}

func runDoctor(ctx context.Context, cmd *cobra.Command) doctorOutput {
	cfg, resolved, contextResult, bootstrapPeers := resolveConfig(cmd)
	results := []checkResult{contextResult}

	if !canRunCoreChecks(contextResult) {
		results = append(results, skipped("directory_api_tcp", "Skipped because context configuration failed", nil))
		results = append(results, skipped("grpc_client_setup", "Skipped because context configuration failed", nil))
		results = append(results, skipped("routing_list", "Skipped because client setup failed", nil))
		bootstrapValidation := validateBootstrapPeers(bootstrapPeers)
		results = append(results, bootstrapPeerChecks(ctx, bootstrapValidation, opts.Timeout)...)
		results = append(results, dhtBootstrap(ctx, bootstrapValidation, opts.Timeout))

		return newOutput(cfg, resolved, bootstrapPeers, results)
	}

	apiCheck := directoryAPI(ctx, cfg.ServerAddress, opts.Timeout)
	results = append(results, apiCheck)

	dirClient, clientCheck := directoryClient(ctx, cfg)

	results = append(results, clientCheck)
	if clientCheck.Status == statusPass {
		results = append(results, routingList(ctx, dirClient, cfg.ServerAddress, opts.Timeout))
		if closeErr := dirClient.Close(); closeErr != nil {
			results = append(results, checkResult{
				Name:    "grpc_client_close",
				Status:  statusWarn,
				Message: "Directory client cleanup reported an error",
				Details: map[string]string{
					"error": closeErr.Error(),
				},
			})
		}
	} else {
		results = append(results, skipped("routing_list", "Skipped because client setup failed", nil))
	}

	bootstrapValidation := validateBootstrapPeers(bootstrapPeers)
	results = append(results, bootstrapPeerChecks(ctx, bootstrapValidation, opts.Timeout)...)
	results = append(results, dhtBootstrap(ctx, bootstrapValidation, opts.Timeout))

	return newOutput(cfg, resolved, bootstrapPeers, results)
}

func canRunCoreChecks(contextResult checkResult) bool {
	return contextResult.Status != statusFail
}
