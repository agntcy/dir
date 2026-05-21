// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config is the canonical configuration package for every dir
// process: the standalone apiserver, the standalone reconciler, and the
// in-process daemon (dirctl daemon). All three populate the same Config
// struct, read the same dir.config.yml file, and honour the same
// DIRECTORY_* environment variables.
//
// Per-feature config sub-packages (registry, database, authn, ...) live
// underneath this module so that the Config struct can compose them
// without dragging the rest of the dir codebase as a dependency.
package config
