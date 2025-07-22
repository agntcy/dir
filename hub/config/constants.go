// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration management for the Agent Hub CLI and related applications.
package config

const (
	// LocalWebserverPort is the default port for the local OAuth webserver.
	LocalWebserverPort = 48043

	// DefaultHubAddress is the default URL for the Agent Hub service.
	DefaultHubAddress = "https://agent-directory.outshift.com"
	// DefaultHubBackendGRPCPort is the default gRPC port for the Agent Hub backend.
	DefaultHubBackendGRPCPort = 443
)
