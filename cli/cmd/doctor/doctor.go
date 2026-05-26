// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"errors"
	"time"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
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

type doctorDeps struct {
	directoryAPI      func(context.Context, string, time.Duration) checkResult
	directoryClient   func(context.Context, *client.Config) (closer, checkResult)
	routingList       func(context.Context, closer, string, time.Duration) checkResult
	bootstrapPeers    func(context.Context, bootstrapPeerValidation, time.Duration) []checkResult
	dhtBootstrap      func(context.Context, bootstrapPeerValidation, time.Duration) checkResult
	validateBootstrap func([]string) bootstrapPeerValidation
}

type doctorRunner struct {
	deps doctorDeps
}

func newDoctorRunner(deps doctorDeps) doctorRunner {
	if deps.directoryAPI == nil {
		deps.directoryAPI = directoryAPI
	}

	if deps.directoryClient == nil {
		deps.directoryClient = func(ctx context.Context, cfg *client.Config) (closer, checkResult) {
			return directoryClient(ctx, cfg)
		}
	}

	if deps.routingList == nil {
		deps.routingList = func(ctx context.Context, dirClient closer, addr string, timeout time.Duration) checkResult {
			typedClient, ok := dirClient.(*client.Client)
			if !ok {
				return skipped("routing_list", "Skipped because client setup returned a non-routing client", nil)
			}

			return routingList(ctx, typedClient, addr, timeout)
		}
	}

	if deps.bootstrapPeers == nil {
		deps.bootstrapPeers = bootstrapPeerChecks
	}

	if deps.dhtBootstrap == nil {
		deps.dhtBootstrap = dhtBootstrap
	}

	if deps.validateBootstrap == nil {
		deps.validateBootstrap = validateBootstrapPeers
	}

	return doctorRunner{deps: deps}
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
	return newDoctorRunner(doctorDeps{}).run(ctx, cmd)
}

func (r doctorRunner) run(ctx context.Context, cmd *cobra.Command) doctorOutput {
	cfg, resolved, contextResult, bootstrapPeers := resolveConfig(cmd)
	results := []checkResult{contextResult}

	if !canRunCoreChecks(contextResult) {
		results = append(results, skipped("directory_api_tcp", "Skipped because context configuration failed", nil))
		results = append(results, skipped("grpc_client_setup", "Skipped because context configuration failed", nil))
		results = append(results, skipped("routing_list", "Skipped because client setup failed", nil))
		bootstrapValidation := r.deps.validateBootstrap(bootstrapPeers)
		results = append(results, r.deps.bootstrapPeers(ctx, bootstrapValidation, opts.Timeout)...)
		results = append(results, r.deps.dhtBootstrap(ctx, bootstrapValidation, opts.Timeout))

		return newOutput(cfg, resolved, bootstrapPeers, results)
	}

	apiCheck := r.deps.directoryAPI(ctx, cfg.ServerAddress, opts.Timeout)
	results = append(results, apiCheck)

	dirClient, clientCheck := r.deps.directoryClient(ctx, cfg)

	results = append(results, clientCheck)
	if clientCheck.Status == statusPass {
		results = append(results, r.deps.routingList(ctx, dirClient, cfg.ServerAddress, opts.Timeout))
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

	bootstrapValidation := r.deps.validateBootstrap(bootstrapPeers)
	results = append(results, r.deps.bootstrapPeers(ctx, bootstrapValidation, opts.Timeout)...)
	results = append(results, r.deps.dhtBootstrap(ctx, bootstrapValidation, opts.Timeout))

	return newOutput(cfg, resolved, bootstrapPeers, results)
}

func canRunCoreChecks(contextResult checkResult) bool {
	return contextResult.Status != statusFail
}
