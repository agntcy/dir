// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"errors"
	"fmt"
	"strings"

	cliconfig "github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

func resolveConfig(cmd *cobra.Command) (*client.Config, *clientconfig.ResolvedContext, checkResult, []string) {
	fields := cliconfig.ChangedClientConfigFields(cmd)

	var overrides *client.Config
	if len(fields) > 0 {
		overrides = cliconfig.Client
	}

	resolveOpts := clientconfig.ResolveOptions{
		Context:            cliconfig.Context,
		Overrides:          overrides,
		OverrideFields:     fields,
		SkipValidation:     true,
		AllowUnknownFields: true,
	}

	cfg, resolved, err := clientconfig.Resolve(resolveOpts)
	if err != nil {
		return nil, nil, checkResult{
			Name:    "context_config",
			Status:  statusFail,
			Message: "Failed to resolve dirctl client context",
			Details: map[string]string{
				"error": err.Error(),
			},
		}, append([]string(nil), opts.BootstrapPeers...)
	}

	doctorCfg, _, doctorErr := clientconfig.ResolveDoctor(resolveOpts)

	bootstrapPeers := append([]string(nil), opts.BootstrapPeers...)
	if len(bootstrapPeers) == 0 && doctorCfg != nil {
		bootstrapPeers = append(bootstrapPeers, doctorCfg.BootstrapPeers...)
	}

	details := map[string]string{
		"context":        resolved.Name,
		"context_source": resolved.Source,
		"config_path":    resolved.Path,
		"server_address": cfg.ServerAddress,
		"auth_mode":      cfg.AuthMode,
	}
	if doctorErr != nil {
		details["doctor_config_error"] = doctorErr.Error()
	}

	if validationErr := validateClientConfig(cfg); validationErr != nil {
		details["error"] = validationErr.Error()

		return cfg, resolved, checkResult{
			Name:    "context_config",
			Status:  statusFail,
			Message: "Resolved dirctl client context is invalid",
			Details: details,
		}, bootstrapPeers
	}

	if doctorErr != nil {
		return cfg, resolved, checkResult{
			Name:    "context_config",
			Status:  statusWarn,
			Message: "Resolved dirctl client context, but doctor settings could not be resolved",
			Details: details,
		}, bootstrapPeers
	}

	return cfg, resolved, checkResult{
		Name:    "context_config",
		Status:  statusPass,
		Message: "Resolved dirctl client context",
		Details: details,
	}, bootstrapPeers
}

//nolint:cyclop // Keep auth-mode validation in one place for clearer, mode-specific doctor results.
func validateClientConfig(cfg *client.Config) error {
	if cfg == nil {
		return errors.New("client config is empty")
	}

	if strings.TrimSpace(cfg.ServerAddress) == "" {
		return errors.New("server_address is required; set a context, --server-addr, or DIRECTORY_CLIENT_SERVER_ADDRESS")
	}

	switch cfg.AuthMode {
	case "", "insecure", "none":
		return nil
	case "x509":
		if cfg.SpiffeSocketPath == "" {
			return errors.New("spiffe_socket_path is required for x509 authentication")
		}
	case "jwt":
		if cfg.SpiffeSocketPath == "" {
			return errors.New("spiffe_socket_path is required for jwt authentication")
		}

		if cfg.JWTAudience == "" {
			return errors.New("jwt_audience is required for jwt authentication")
		}
	case "token":
		if cfg.SpiffeToken == "" {
			return errors.New("spiffe_token is required for token authentication")
		}
	case "tls":
		if cfg.TlsCAFile == "" || cfg.TlsCertFile == "" || cfg.TlsKeyFile == "" {
			return errors.New("tls_ca_file, tls_cert_file, and tls_key_file are required for tls authentication")
		}
	case "oidc":
		if cfg.AuthToken == "" && cfg.OIDCIssuer == "" {
			return errors.New("oidc_issuer is required for oidc authentication unless auth_token is set")
		}
	default:
		return fmt.Errorf("unsupported auth_mode %q", cfg.AuthMode)
	}

	return nil
}
