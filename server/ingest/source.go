// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ingest

import "context"

// Ingest sources. Push and import are client writes evaluated by ingest
// policies. Autosync and sync are not gated here yet.
const (
	SourcePush     = "push"
	SourceImport   = "import"
	SourceAutosync = "autosync"
	SourceSync     = "sync"
)

type sourceKey struct{}

// WithSource records how a record is being ingested so policy can distinguish
// a client push from later paths (autosync, sync).
func WithSource(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, sourceKey{}, source)
}

// SourceFrom returns the ingest source, or empty if unset.
func SourceFrom(ctx context.Context) string {
	source, _ := ctx.Value(sourceKey{}).(string)

	return source
}
