// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package env holds the one convention every component shares for reading
// configuration from the environment: which variable stands for which key.
package env

import "strings"

// KeyReplacer turns a configuration key into the body of its variable. Every
// loader hands it to viper, so that routing.key_path is set by
// <PREFIX>_ROUTING_KEY_PATH.
var KeyReplacer = strings.NewReplacer(".", "_", "-", "_")

// Name is the variable viper reads for key under prefix.
//
// Code that needs a name viper will not supply — for a key viper no longer
// binds, say — should derive it here rather than write it out, since the
// prefix is all that differs between components and a name copied from one
// is a name the others never read.
func Name(prefix, key string) string {
	return strings.ToUpper(prefix + "_" + KeyReplacer.Replace(key))
}
