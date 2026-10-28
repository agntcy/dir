// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/authn"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLog swaps the package logger for one writing JSON at debug level, and
// returns the lines written while it is in place. JSON rather than the default
// text format so a test asserts on an attribute rather than on a rendering of
// it.
//
// The tests using this cannot run in parallel with each other: the logger they
// replace is one variable for the package.
func captureLog(t *testing.T) func() []map[string]any {
	t.Helper()

	var buf bytes.Buffer

	restore := logger
	logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Cleanup(func() { logger = restore })

	return func() []map[string]any {
		t.Helper()

		var lines []map[string]any

		for raw := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
			if raw == "" {
				continue
			}

			var line map[string]any

			require.NoError(t, json.Unmarshal([]byte(raw), &line))

			lines = append(lines, line)
		}

		return lines
	}
}

// requireOneLine asserts exactly one decision was logged, and returns it.
func requireOneLine(t *testing.T, lines []map[string]any) map[string]any {
	t.Helper()

	require.Len(t, lines, 1, "exactly one line per decision")

	return lines[0]
}

// A refusal is the case the log exists for: an operation policy persists
// nothing, so without this line the only trace of one is the error the client
// got.
func TestLogDecision_RefusalNamesWhatDecided(t *testing.T) {
	logged := captureLog(t)

	v := &countingValidator{}
	reg := &Registry{byOp: map[string][]operationPolicy{
		policyconfig.OpPush: {{id: "cel:strict", mode: policyconfig.ModeEnforce, validator: v}},
	}}

	record := corev1.New(&typesv1alpha1.Record{Name: "test"})

	ok, _, err := reg.Decide(context.Background(), policyconfig.OpPush, record)
	require.NoError(t, err)
	require.False(t, ok)

	line := requireOneLine(t, logged())
	assert.Equal(t, "OPERATION_PUSH", line["operation"])
	assert.Equal(t, record.GetCid(), line["cid"])
	assert.Equal(t, "cel:strict", line["policy_id"])
	assert.Equal(t, string(policyconfig.ModeEnforce), line["mode"])
	assert.Equal(t, decisionRefuse, line["decision"])
	assert.Equal(t, []any{"ERROR: rejected"}, line["reason"])
}

// Shadow mode reports what it would have done and nothing else, so the log is
// the whole of it. The line therefore has to carry the real decision, with the
// mode beside it saying it was not acted on.
func TestLogDecision_ShadowReportsTheRealDecision(t *testing.T) {
	logged := captureLog(t)

	reg := &Registry{byOp: map[string][]operationPolicy{
		policyconfig.OpPush: {{id: "cel:trial", mode: policyconfig.ModeShadow, validator: &countingValidator{}}},
	}}

	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPush,
		corev1.New(&typesv1alpha1.Record{Name: "test"}))
	require.NoError(t, err)
	require.True(t, ok, "a shadow policy must not refuse the record")
	require.Empty(t, msgs)

	line := requireOneLine(t, logged())
	assert.Equal(t, decisionRefuse, line["decision"])
	assert.Equal(t, string(policyconfig.ModeShadow), line["mode"])
	assert.Contains(t, line["msg"], "would have refused")
}

// An off policy is not run, so it decides nothing and logs nothing.
func TestLogDecision_OffLogsNothing(t *testing.T) {
	logged := captureLog(t)

	reg := &Registry{byOp: map[string][]operationPolicy{
		policyconfig.OpPush: {{id: "cel:disabled", mode: policyconfig.ModeOff, validator: &countingValidator{}}},
	}}

	_, _, err := reg.Decide(context.Background(), policyconfig.OpPush,
		corev1.New(&typesv1alpha1.Record{Name: "test"}))
	require.NoError(t, err)
	assert.Empty(t, logged())
}

// Every push, autosync and index of every record produces an allowed decision,
// so those are logged at debug and carry no reason to report.
func TestLogDecision_AllowedIsDebug(t *testing.T) {
	logged := captureLog(t)

	reg := With(policyconfig.OpPush, fakeValidator{valid: true})

	ok, _, err := reg.Decide(context.Background(), policyconfig.OpPush,
		corev1.New(&typesv1alpha1.Record{Name: "test"}))
	require.NoError(t, err)
	require.True(t, ok)

	line := requireOneLine(t, logged())
	assert.Equal(t, decisionAllow, line["decision"])
	assert.Equal(t, slog.LevelDebug.String(), line["level"])
	assert.NotContains(t, line, "reason")
}

// The peer and the caller are the attribution, and which one is known depends
// on the operation. Neither is reported as empty: a line naming a peer of ""
// claims a peer that does not exist.
func TestLogDecision_AttributesToPeerOrCaller(t *testing.T) {
	id, err := spiffeid.FromString("spiffe://example.org/agent")
	require.NoError(t, err)

	tests := []struct {
		name       string
		ctx        context.Context //nolint:containedctx // the context under test
		opts       []DecideOption
		wantPeer   any
		wantCaller any
	}{
		{
			name: "push has neither",
			ctx:  context.Background(),
		},
		{
			name:     "autosync names the peer",
			ctx:      context.Background(),
			opts:     []DecideOption{FromPeer("12D3KooWPeer")},
			wantPeer: "12D3KooWPeer",
		},
		{
			name:       "an authenticated push names the caller",
			ctx:        context.WithValue(context.Background(), authn.SpiffeIDContextKey, id),
			wantCaller: "spiffe://example.org/agent",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logged := captureLog(t)

			reg := With(policyconfig.OpPush, fakeValidator{valid: true})

			_, _, err := reg.Decide(tc.ctx, policyconfig.OpPush,
				corev1.New(&typesv1alpha1.Record{Name: "test"}), tc.opts...)
			require.NoError(t, err)

			line := requireOneLine(t, logged())
			assert.Equal(t, tc.wantPeer, line["peer"])
			assert.Equal(t, tc.wantCaller, line["caller"])
		})
	}
}
