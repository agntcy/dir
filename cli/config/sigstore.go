// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/pflag"
)

// ResolveSigstore returns the Sigstore settings of the context this invocation
// names — `--context`, or current_context when the flag is unset — with
// DIRECTORY_CLIENT_SIGSTORE_* environment variables applied on top. Commands
// layer their explicitly set flags over the result with ApplyUnlessChanged.
func ResolveSigstore() (*clientconfig.ResolvedSigstore, error) {
	cfg, _, err := clientconfig.ResolveSigstore(clientconfig.ResolveOptions{
		Context:            Context,
		AllowUnknownFields: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sigstore config: %w", err)
	}

	return cfg, nil
}

// ApplyUnlessChanged sets target to value when value is non-zero and the named
// flag was not set explicitly, so a configured value replaces the flag's
// built-in default but never a value the user passed. It reports whether it
// applied value.
//
// The zero value stands for "unset" (see clientconfig.Sigstore): config can turn
// a boolean on but never force it to false, and cannot set a string to empty.
// Both are safe while every boolean default is false; a default that flips to
// true needs a pointer-typed field here instead.
func ApplyUnlessChanged[T comparable](flags *pflag.FlagSet, name string, target *T, value T) bool {
	var zero T
	if value == zero || flags.Changed(name) {
		return false
	}

	*target = value

	return true
}
