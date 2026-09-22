// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

type Config struct {
	// ServerAddress is the address the server binds to.
	ServerAddress string `json:"server_address,omitempty" mapstructure:"server_address"`

	// MetricsAddress is the address the metric endpoint binds to.
	MetricsAddress string `json:"metrics_address,omitempty" mapstructure:"metrics_address"`

	// GatewayAddress is the base URL of the HTTP gateway (e.g.
	// "http://localhost:19892"). Empty when the gateway is not deployed in
	// this environment, in which case AI Finder HTTP tests are skipped.
	GatewayAddress string `json:"gateway_address,omitempty" mapstructure:"gateway_address"`

	// ExtractorMode names the OASF extractor backend the deployed gateway
	// resolves in this environment: "local" for in-process assets provisioned
	// by `dirctl init`, "remote" for a gRPC OASF-SDK server. Empty means the
	// environment declares no extractor, so extractor-backed specs skip.
	//
	// The extractor-backed specs read this rather than probing the asset
	// directory. Probing cannot tell the two apart: a developer machine that has
	// run `dirctl init` has local assets on disk even when the gateway under
	// test is wired to a remote server.
	//
	// It also decides whether the search parity spec can run. Comparing
	// `dirctl search` against POST /v1/search only means something when both
	// reach the same extractor, which is true in "local" mode and not under
	// kind, where the OASF-SDK Service is ClusterIP-only and the CLI cannot
	// reach the backend the gateway uses.
	//
	// This replaces the earlier remote_extractor_enabled boolean, which said
	// only whether the backend was remote. Two fields for one fact let an
	// environment set one and not the other, and the specs disagree about
	// whether to skip; the mode also has to distinguish "local" from "no
	// extractor", which a boolean cannot.
	ExtractorMode string `json:"extractor_mode,omitempty" mapstructure:"extractor_mode"`

	// CliPath is the path to the CLI binary.
	CliPath string `json:"cli_path,omitempty" mapstructure:"cli_path"`

	// CliExtraArgs are extra arguments to pass to the CLI.
	CliExtraArgs []string `json:"cli_extra_args,omitempty" mapstructure:"cli_extra_args"`

	// NameVerificationHost is the host:port the dir-daemon uses to fetch the JWKS
	// well-known file from the dns-validation infrastructure in this environment.
	NameVerificationHost string `json:"name_verification_host,omitempty" mapstructure:"name_verification_host"`
}
