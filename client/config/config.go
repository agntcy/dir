// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config loads reusable Directory client context configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	dirclient "github.com/agntcy/dir/client"
	"go.yaml.in/yaml/v3"
)

const (
	// DefaultEnvPrefix is the default environment variable prefix for client config.
	DefaultEnvPrefix = dirclient.DefaultEnvPrefix

	// ClientContextEnv is the DIRECTORY_CLIENT-prefixed context selection environment variable.
	ClientContextEnv = "DIRECTORY_CLIENT_CONTEXT"

	configDirName  = "dirctl"
	configFileName = "config.yaml"
	configDirPerm  = 0o700
	configFilePerm = 0o600
)

// File is the top-level reusable client context configuration file.
type File struct {
	CurrentContext string             `yaml:"current_context"`
	Contexts       map[string]Context `yaml:"contexts"`
	Extractor      *Extractor         `yaml:"extractor,omitempty"`
}

// Extractor is the machine-wide OASF taxonomy extractor provisioning record,
// written by `dirctl init` so import/search consumers can load the provisioned
// assets in-process without re-choosing the endpoint or asset location.
type Extractor struct {
	OASFURL  string `yaml:"oasf_url"`
	AssetDir string `yaml:"asset_dir"`
}

// Context is a named client configuration block.
type Context struct {
	ServerAddress    string `yaml:"server_address"`
	TlsSkipVerify    bool   `yaml:"tls_skip_verify"`
	TlsCertFile      string `yaml:"tls_cert_file"`
	TlsKeyFile       string `yaml:"tls_key_file"`
	TlsCAFile        string `yaml:"tls_ca_file"`
	SpiffeSocketPath string `yaml:"spiffe_socket_path"`
	SpiffeToken      string `yaml:"spiffe_token"`
	AuthMode         string `yaml:"auth_mode"`
	JWTAudience      string `yaml:"jwt_audience"`
	OIDCIssuer       string `yaml:"oidc_issuer"`
	OIDCClientID     string `yaml:"oidc_client_id"`
	AuthToken        string `yaml:"auth_token"`
	Doctor           Doctor `yaml:"doctor"`
}

// Doctor holds diagnostic-only settings for dirctl doctor.
type Doctor struct {
	BootstrapPeers []string `yaml:"bootstrap_peers"`
}

// ResolveOptions controls context and client configuration resolution.
type ResolveOptions struct {
	// Path is the config file path. If empty, DefaultPath is used.
	Path string

	// Context is an explicit context name override.
	Context string

	// EnvPrefix is the client environment variable prefix. If empty,
	// DefaultEnvPrefix is used.
	EnvPrefix string

	// Overrides contains explicit values, typically from CLI flags.
	Overrides *dirclient.Config

	// OverrideFields lists schema field names from Overrides that should be
	// applied, including zero values. If empty, non-zero override values are
	// applied.
	OverrideFields []string

	// SkipValidation skips required-field validation for callers that only need
	// a subset of client config, such as auth token cache commands.
	SkipValidation bool

	// AllowUnknownFields permits forward-compatible parsing for callers that only
	// need known client fields from a config that may include command extensions.
	AllowUnknownFields bool
}

// LoadOptions controls how a reusable client context config file is loaded.
type LoadOptions struct {
	// AllowUnknownFields permits forward-compatible parsing for callers that only
	// need known client fields from a config that may include command extensions.
	AllowUnknownFields bool
}

// ResolvedContext describes which context was selected during resolution.
type ResolvedContext struct {
	Name   string
	Source string
	Path   string
}

// ContextSummary is a list entry for a configured context.
type ContextSummary struct {
	Name    string
	Current bool
}

// ContextValidation describes the validation result for a configured context.
type ContextValidation struct {
	Name  string
	Error error
}

// LoadFile loads a reusable client context config file from path.
func LoadFile(path string) (*File, error) {
	return loadFile(path, false)
}

// LoadFileWithOptions loads a reusable client context config file from path with parsing options.
func LoadFileWithOptions(path string, opts LoadOptions) (*File, error) {
	return loadFile(path, opts.AllowUnknownFields)
}

func loadFile(path string, allowUnknownFields bool) (*File, error) {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return nil, err
		}

		path = defaultPath
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open client config file %s: %w", path, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(!allowUnknownFields)

	file := &File{}
	if err := decoder.Decode(file); err != nil {
		return nil, fmt.Errorf("failed to parse client config file %s: %w", path, err)
	}

	if file.Contexts == nil {
		file.Contexts = map[string]Context{}
	}

	if err := validateContextNames(file); err != nil {
		return nil, fmt.Errorf("invalid client config file %s: %w", path, err)
	}

	return file, nil
}

// DefaultPath returns the default reusable client config file path.
func DefaultPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine user home directory: %w", err)
		}

		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, configDirName, configFileName), nil
}

// Resolve resolves the effective Directory client config.
func Resolve(opts ResolveOptions) (*dirclient.Config, *ResolvedContext, error) {
	path, explicitPath, err := resolvePath(opts.Path)
	if err != nil {
		return nil, nil, err
	}

	file, err := loadOptionalFile(path, explicitPath, opts.AllowUnknownFields)
	if err != nil {
		return nil, nil, err
	}

	contextName, source := selectedContextName(opts, file)
	cfg := &dirclient.Config{}

	if contextName != "" {
		contextConfig, ok := file.Contexts[contextName]
		if !ok {
			return nil, nil, fmt.Errorf("unknown client context %q in %s", contextName, path)
		}

		*cfg = contextConfig.toClientConfig()
	}

	if err := applyEnv(cfg, envPrefix(opts.EnvPrefix)); err != nil {
		return nil, nil, err
	}

	if err := applyOverrides(cfg, opts.Overrides, opts.OverrideFields); err != nil {
		return nil, nil, err
	}

	if !opts.SkipValidation {
		if err := validateClientConfig(cfg); err != nil {
			return nil, nil, err
		}
	}

	return cfg, &ResolvedContext{
		Name:   contextName,
		Source: source,
		Path:   path,
	}, nil
}

// ResolveDoctor resolves diagnostic-only settings for the selected context.
func ResolveDoctor(opts ResolveOptions) (*Doctor, *ResolvedContext, error) {
	path, explicitPath, err := resolvePath(opts.Path)
	if err != nil {
		return nil, nil, err
	}

	file, err := loadOptionalFile(path, explicitPath, opts.AllowUnknownFields)
	if err != nil {
		return nil, nil, err
	}

	contextName, source := selectedContextName(opts, file)
	cfg := &Doctor{}

	if contextName != "" {
		contextConfig, ok := file.Contexts[contextName]
		if !ok {
			return nil, nil, fmt.Errorf("unknown client context %q in %s", contextName, path)
		}

		cfg.BootstrapPeers = append([]string(nil), contextConfig.Doctor.BootstrapPeers...)
	}

	return cfg, &ResolvedContext{
		Name:   contextName,
		Source: source,
		Path:   path,
	}, nil
}

// ListContexts lists configured contexts in name order.
func ListContexts(path string) ([]ContextSummary, error) {
	resolvedPath, explicitPath, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := loadOptionalFile(resolvedPath, explicitPath, true)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(file.Contexts))
	for name := range file.Contexts {
		names = append(names, name)
	}

	sort.Strings(names)

	summaries := make([]ContextSummary, 0, len(names))
	for _, name := range names {
		summaries = append(summaries, ContextSummary{
			Name:    name,
			Current: name == file.CurrentContext,
		})
	}

	return summaries, nil
}

// CurrentContext returns the persisted current_context without resolving client settings.
func CurrentContext(path string) (*ResolvedContext, error) {
	resolvedPath, explicitPath, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := loadOptionalFile(resolvedPath, explicitPath, true)
	if err != nil {
		return nil, err
	}

	if file.CurrentContext == "" {
		return &ResolvedContext{
			Source: "none",
			Path:   resolvedPath,
		}, nil
	}

	if _, ok := file.Contexts[file.CurrentContext]; !ok {
		return nil, fmt.Errorf("unknown client context %q in %s", file.CurrentContext, resolvedPath)
	}

	return &ResolvedContext{
		Name:   file.CurrentContext,
		Source: "current_context",
		Path:   resolvedPath,
	}, nil
}

// SetCurrentContext persists name as the active context.
func SetCurrentContext(path string, name string) (*ResolvedContext, error) {
	resolvedPath, explicitPath, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := loadOptionalFile(resolvedPath, explicitPath, true)
	if err != nil {
		return nil, err
	}

	if _, ok := file.Contexts[name]; !ok {
		return nil, fmt.Errorf("unknown client context %q in %s", name, resolvedPath)
	}

	file.CurrentContext = name
	if err := SaveFile(resolvedPath, file); err != nil {
		return nil, err
	}

	return &ResolvedContext{
		Name:   name,
		Source: "current_context",
		Path:   resolvedPath,
	}, nil
}

// SaveFile writes a reusable client context config file.
func SaveFile(path string, file *File) error {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return err
		}

		path = defaultPath
	}

	if file.Contexts == nil {
		file.Contexts = map[string]Context{}
	}

	if err := validateContextNames(file); err != nil {
		return fmt.Errorf("invalid client config file %s: %w", path, err)
	}

	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("failed to encode client config file %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), configDirPerm); err != nil {
		return fmt.Errorf("failed to create client config directory %s: %w", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, data, configFilePerm); err != nil {
		return fmt.Errorf("failed to write client config file %s: %w", path, err)
	}

	return nil
}

// LoadExtractor returns the persisted machine-wide extractor section, or nil
// when it is unset or the config file does not exist.
func LoadExtractor(path string) (*Extractor, error) {
	resolvedPath, _, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := loadOptionalFile(resolvedPath, false, true)
	if err != nil {
		return nil, err
	}

	return file.Extractor, nil
}

// SaveExtractor persists the machine-wide extractor section, preserving all
// other config (contexts, current_context).
func SaveExtractor(path string, e *Extractor) error {
	resolvedPath, _, err := resolvePath(path)
	if err != nil {
		return err
	}

	file, err := loadOptionalFile(resolvedPath, false, true)
	if err != nil {
		return err
	}

	file.Extractor = e

	return SaveFile(resolvedPath, file)
}

// ClearExtractor removes the machine-wide extractor section, preserving all
// other config. It is a no-op when the section is already absent.
func ClearExtractor(path string) error {
	return SaveExtractor(path, nil)
}

// ValidateContexts validates stored context definitions without applying environment overrides.
func ValidateContexts(path string, name string) ([]ContextValidation, error) {
	resolvedPath, explicitPath, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := loadOptionalFile(resolvedPath, explicitPath, true)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(file.Contexts))
	if name != "" {
		if _, ok := file.Contexts[name]; !ok {
			return nil, fmt.Errorf("unknown client context %q in %s", name, resolvedPath)
		}

		names = append(names, name)
	} else {
		for contextName := range file.Contexts {
			names = append(names, contextName)
		}

		sort.Strings(names)
	}

	results := make([]ContextValidation, 0, len(names))
	for _, contextName := range names {
		cfg := file.Contexts[contextName].toClientConfig()
		results = append(results, ContextValidation{
			Name:  contextName,
			Error: validateClientConfig(&cfg),
		})
	}

	return results, nil
}

func resolvePath(path string) (string, bool, error) {
	if path != "" {
		return path, true, nil
	}

	defaultPath, err := DefaultPath()
	if err != nil {
		return "", false, err
	}

	return defaultPath, false, nil
}

func loadOptionalFile(path string, explicitPath bool, allowUnknownFields bool) (*File, error) {
	file, err := loadFile(path, allowUnknownFields)
	if err == nil {
		return file, nil
	}

	if explicitPath || !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return &File{Contexts: map[string]Context{}}, nil
}

func selectedContextName(opts ResolveOptions, file *File) (string, string) {
	if opts.Context != "" {
		return opts.Context, "option"
	}

	if value, ok := os.LookupEnv(ClientContextEnv); ok {
		return value, "env"
	}

	if file.CurrentContext != "" {
		return file.CurrentContext, "current_context"
	}

	return "", "none"
}

func envPrefix(prefix string) string {
	if prefix == "" {
		return DefaultEnvPrefix
	}

	return prefix
}

func applyEnv(cfg *dirclient.Config, prefix string) error {
	stringEnv := map[string]*string{
		"server_address":     &cfg.ServerAddress,
		"tls_cert_file":      &cfg.TlsCertFile,
		"tls_key_file":       &cfg.TlsKeyFile,
		"tls_ca_file":        &cfg.TlsCAFile,
		"spiffe_socket_path": &cfg.SpiffeSocketPath,
		"spiffe_token":       &cfg.SpiffeToken,
		"auth_mode":          &cfg.AuthMode,
		"jwt_audience":       &cfg.JWTAudience,
		"oidc_issuer":        &cfg.OIDCIssuer,
		"oidc_client_id":     &cfg.OIDCClientID,
		"auth_token":         &cfg.AuthToken,
	}

	for key, target := range stringEnv {
		if value, ok := os.LookupEnv(envVarName(prefix, key)); ok {
			*target = value
		}
	}

	if value, ok := os.LookupEnv(envVarName(prefix, "tls_skip_verify")); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid %s value %q: %w", envVarName(prefix, "tls_skip_verify"), value, err)
		}

		cfg.TlsSkipVerify = parsed
	}

	return nil
}

func applyOverrides(cfg *dirclient.Config, overrides *dirclient.Config, fields []string) error {
	if overrides == nil {
		return nil
	}

	if len(fields) == 0 {
		applyNonZeroOverrides(cfg, overrides)

		return nil
	}

	for _, field := range fields {
		if err := applyOverrideField(cfg, overrides, field); err != nil {
			return err
		}
	}

	return nil
}

func applyNonZeroOverrides(cfg *dirclient.Config, overrides *dirclient.Config) {
	if overrides.ServerAddress != "" {
		cfg.ServerAddress = overrides.ServerAddress
	}

	if overrides.TlsSkipVerify {
		cfg.TlsSkipVerify = overrides.TlsSkipVerify
	}

	if overrides.TlsCertFile != "" {
		cfg.TlsCertFile = overrides.TlsCertFile
	}

	if overrides.TlsKeyFile != "" {
		cfg.TlsKeyFile = overrides.TlsKeyFile
	}

	if overrides.TlsCAFile != "" {
		cfg.TlsCAFile = overrides.TlsCAFile
	}

	if overrides.SpiffeSocketPath != "" {
		cfg.SpiffeSocketPath = overrides.SpiffeSocketPath
	}

	if overrides.SpiffeToken != "" {
		cfg.SpiffeToken = overrides.SpiffeToken
	}

	if overrides.AuthMode != "" {
		cfg.AuthMode = overrides.AuthMode
	}

	if overrides.JWTAudience != "" {
		cfg.JWTAudience = overrides.JWTAudience
	}

	if overrides.OIDCIssuer != "" {
		cfg.OIDCIssuer = overrides.OIDCIssuer
	}

	if overrides.OIDCClientID != "" {
		cfg.OIDCClientID = overrides.OIDCClientID
	}

	if overrides.AuthToken != "" {
		cfg.AuthToken = overrides.AuthToken
	}
}

func applyOverrideField(cfg *dirclient.Config, overrides *dirclient.Config, field string) error {
	switch field {
	case "server_address":
		cfg.ServerAddress = overrides.ServerAddress
	case "tls_skip_verify":
		cfg.TlsSkipVerify = overrides.TlsSkipVerify
	case "tls_cert_file":
		cfg.TlsCertFile = overrides.TlsCertFile
	case "tls_key_file":
		cfg.TlsKeyFile = overrides.TlsKeyFile
	case "tls_ca_file":
		cfg.TlsCAFile = overrides.TlsCAFile
	case "spiffe_socket_path":
		cfg.SpiffeSocketPath = overrides.SpiffeSocketPath
	case "spiffe_token":
		cfg.SpiffeToken = overrides.SpiffeToken
	case "auth_mode":
		cfg.AuthMode = overrides.AuthMode
	case "jwt_audience":
		cfg.JWTAudience = overrides.JWTAudience
	case "oidc_issuer":
		cfg.OIDCIssuer = overrides.OIDCIssuer
	case "oidc_client_id":
		cfg.OIDCClientID = overrides.OIDCClientID
	case "auth_token":
		cfg.AuthToken = overrides.AuthToken
	default:
		return fmt.Errorf("unknown override field %q", field)
	}

	return nil
}

//nolint:cyclop // Keep auth-mode validation in one place for clearer, mode-specific errors.
func validateClientConfig(cfg *dirclient.Config) error {
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

func validateContextNames(file *File) error {
	for name := range file.Contexts {
		if strings.TrimSpace(name) == "" {
			return errors.New("context name must not be empty")
		}

		if strings.ContainsAny(name, `/\`) {
			return fmt.Errorf("context name %q must not contain path separators", name)
		}
	}

	if file.CurrentContext != "" {
		if _, ok := file.Contexts[file.CurrentContext]; !ok {
			return fmt.Errorf("current_context %q does not match a configured context", file.CurrentContext)
		}
	}

	return nil
}

func envVarName(prefix string, key string) string {
	replaced := strings.NewReplacer(".", "_", "-", "_").Replace(key)

	return prefix + "_" + strings.ToUpper(replaced)
}

func (c Context) toClientConfig() dirclient.Config {
	return dirclient.Config{
		ServerAddress:    c.ServerAddress,
		TlsSkipVerify:    c.TlsSkipVerify,
		TlsCertFile:      c.TlsCertFile,
		TlsKeyFile:       c.TlsKeyFile,
		TlsCAFile:        c.TlsCAFile,
		SpiffeSocketPath: c.SpiffeSocketPath,
		SpiffeToken:      c.SpiffeToken,
		AuthMode:         c.AuthMode,
		JWTAudience:      c.JWTAudience,
		OIDCIssuer:       c.OIDCIssuer,
		OIDCClientID:     c.OIDCClientID,
		AuthToken:        c.AuthToken,
	}
}
