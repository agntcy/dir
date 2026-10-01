// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"strconv"
)

// sigstoreEnvKey prefixes the Sigstore keys in environment variable names, so
// fulcio_url reads from DIRECTORY_CLIENT_SIGSTORE_FULCIO_URL.
const sigstoreEnvKey = "sigstore_"

// Sigstore holds the keyless (OIDC) signing and verification settings used by
// dirctl sign and dirctl verify. Empty fields leave the built-in public-good
// Sigstore defaults in place.
//
// Secrets (the OIDC client secret and ID token) are deliberately absent: they
// are accepted only as flags so they never land in the config file.
type Sigstore struct {
	// Signing.
	FulcioURL       string `yaml:"fulcio_url,omitempty"`
	RekorURL        string `yaml:"rekor_url,omitempty"`
	TimestampURL    string `yaml:"timestamp_url,omitempty"`
	SkipTlog        bool   `yaml:"skip_tlog,omitempty"`
	OIDCProviderURL string `yaml:"oidc_provider_url,omitempty"`
	// OIDCClientID is the client used to obtain a Fulcio certificate. It is a
	// different client from the context's top-level oidc_client_id, which
	// dirctl auth login uses.
	OIDCClientID string `yaml:"oidc_client_id,omitempty"`

	// Verification.
	TufMirrorURL    string `yaml:"tuf_mirror_url,omitempty"`
	TrustedRootPath string `yaml:"trusted_root_path,omitempty"`
	IgnoreTlog      bool   `yaml:"ignore_tlog,omitempty"`
	IgnoreTsa       bool   `yaml:"ignore_tsa,omitempty"`
	IgnoreSct       bool   `yaml:"ignore_sct,omitempty"`
}

// ResolveSigstore resolves the Sigstore settings for the selected context, with
// DIRECTORY_CLIENT_SIGSTORE_* environment variables applied over the context's
// sigstore section. Flag precedence is left to the caller, which knows which
// flags were set explicitly.
func ResolveSigstore(opts ResolveOptions) (*Sigstore, *ResolvedContext, error) {
	path, explicitPath, err := resolvePath(opts.Path)
	if err != nil {
		return nil, nil, err
	}

	file, err := loadOptionalFile(path, explicitPath, opts.AllowUnknownFields)
	if err != nil {
		return nil, nil, err
	}

	contextName, source := selectedContextName(opts, file)
	cfg := &Sigstore{}

	if contextName != "" {
		contextConfig, ok := file.Contexts[contextName]
		if !ok {
			return nil, nil, fmt.Errorf("unknown client context %q in %s", contextName, path)
		}

		*cfg = contextConfig.Sigstore
	}

	if err := applySigstoreEnv(cfg, envPrefix(opts.EnvPrefix)); err != nil {
		return nil, nil, err
	}

	return cfg, &ResolvedContext{
		Name:   contextName,
		Source: source,
		Path:   path,
	}, nil
}

func applySigstoreEnv(cfg *Sigstore, prefix string) error {
	stringEnv := map[string]*string{
		"fulcio_url":        &cfg.FulcioURL,
		"rekor_url":         &cfg.RekorURL,
		"timestamp_url":     &cfg.TimestampURL,
		"oidc_provider_url": &cfg.OIDCProviderURL,
		"oidc_client_id":    &cfg.OIDCClientID,
		"tuf_mirror_url":    &cfg.TufMirrorURL,
		"trusted_root_path": &cfg.TrustedRootPath,
	}

	for key, target := range stringEnv {
		if value, ok := os.LookupEnv(envVarName(prefix, sigstoreEnvKey+key)); ok {
			*target = value
		}
	}

	boolEnv := map[string]*bool{
		"skip_tlog":   &cfg.SkipTlog,
		"ignore_tlog": &cfg.IgnoreTlog,
		"ignore_tsa":  &cfg.IgnoreTsa,
		"ignore_sct":  &cfg.IgnoreSct,
	}

	for key, target := range boolEnv {
		name := envVarName(prefix, sigstoreEnvKey+key)

		value, ok := os.LookupEnv(name)
		if !ok {
			continue
		}

		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid %s value %q: %w", name, value, err)
		}

		*target = parsed
	}

	return nil
}
