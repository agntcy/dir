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
// dirctl sign and dirctl verify.
//
// A zero value means "unset" and leaves the built-in public-good default in
// place. Two consequences follow. A boolean can only be turned on from config,
// never forced to false; that is safe because every boolean default is false
// today, but flipping a default to true would make a configured false a silent
// no-op. And a string cannot be set to empty from config or the environment
// (for example to drop the timestamp authority); only an explicitly passed flag
// such as --timestamp-url "" can do that.
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

// ResolvedSigstore is the effective Sigstore settings with the origin of each
// setting that is set.
type ResolvedSigstore struct {
	Sigstore

	// Sources maps the key of every set setting (for example "ignore_tlog") to
	// where its value came from: `context "corp"`, or the name of the
	// environment variable. Unset settings have no entry.
	Sources map[string]string
}

type sigstoreStringField struct {
	key    string
	target *string
}

type sigstoreBoolField struct {
	key    string
	target *bool
}

// The field lists are ordered so that resolution, and with it the error
// reported for an invalid value, is deterministic.
func (s *Sigstore) stringFields() []sigstoreStringField {
	return []sigstoreStringField{
		{"fulcio_url", &s.FulcioURL},
		{"rekor_url", &s.RekorURL},
		{"timestamp_url", &s.TimestampURL},
		{"oidc_provider_url", &s.OIDCProviderURL},
		{"oidc_client_id", &s.OIDCClientID},
		{"tuf_mirror_url", &s.TufMirrorURL},
		{"trusted_root_path", &s.TrustedRootPath},
	}
}

func (s *Sigstore) boolFields() []sigstoreBoolField {
	return []sigstoreBoolField{
		{"skip_tlog", &s.SkipTlog},
		{"ignore_tlog", &s.IgnoreTlog},
		{"ignore_tsa", &s.IgnoreTsa},
		{"ignore_sct", &s.IgnoreSct},
	}
}

// ResolveSigstore resolves the Sigstore settings for the selected context, with
// DIRECTORY_CLIENT_SIGSTORE_* environment variables applied over the context's
// sigstore section. An environment variable set to an empty string counts as
// unset. Flag precedence is left to the caller, which knows which flags were
// set explicitly.
func ResolveSigstore(opts ResolveOptions) (*ResolvedSigstore, *ResolvedContext, error) {
	path, explicitPath, err := resolvePath(opts.Path)
	if err != nil {
		return nil, nil, err
	}

	file, err := loadOptionalFile(path, explicitPath, opts.AllowUnknownFields)
	if err != nil {
		return nil, nil, err
	}

	contextName, source := selectedContextName(opts, file)
	cfg := &ResolvedSigstore{Sources: map[string]string{}}

	if contextName != "" {
		contextConfig, ok := file.Contexts[contextName]
		if !ok {
			return nil, nil, fmt.Errorf("unknown client context %q in %s", contextName, path)
		}

		cfg.Sigstore = contextConfig.Sigstore
		cfg.recordContextSources(fmt.Sprintf("context %q", contextName))
	}

	if err := cfg.applyEnv(envPrefix(opts.EnvPrefix)); err != nil {
		return nil, nil, err
	}

	return cfg, &ResolvedContext{
		Name:   contextName,
		Source: source,
		Path:   path,
	}, nil
}

func (r *ResolvedSigstore) recordContextSources(source string) {
	for _, field := range r.stringFields() {
		if *field.target != "" {
			r.Sources[field.key] = source
		}
	}

	for _, field := range r.boolFields() {
		if *field.target {
			r.Sources[field.key] = source
		}
	}
}

func (r *ResolvedSigstore) applyEnv(prefix string) error {
	for _, field := range r.stringFields() {
		name := envVarName(prefix, sigstoreEnvKey+field.key)
		if value := os.Getenv(name); value != "" {
			*field.target = value
			r.Sources[field.key] = name
		}
	}

	for _, field := range r.boolFields() {
		name := envVarName(prefix, sigstoreEnvKey+field.key)

		value := os.Getenv(name)
		if value == "" {
			continue
		}

		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid %s value %q: %w", name, value, err)
		}

		*field.target = parsed

		if parsed {
			r.Sources[field.key] = name
		} else {
			delete(r.Sources, field.key)
		}
	}

	return nil
}
