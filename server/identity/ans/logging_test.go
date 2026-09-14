// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"bytes"
	"log/slog"
	"testing"
)

// captureLogs routes the package logger into a buffer at DEBUG level for the
// rest of the test. Tests that capture logs must not run in parallel.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer

	previous := logger
	logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Cleanup(func() { logger = previous })

	return &buf
}
