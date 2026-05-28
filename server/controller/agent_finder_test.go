// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

// fakeAgentFinderStore is a hand-rolled types.StoreAPI test double. It
// implements only the methods exercised by the AgentFinder controller's
// ExportAgent path; everything else panics so accidental coupling is
// loud.
type fakeAgentFinderStore struct {
	// pullFn lets each test case stub Pull independently (return a
	// record, a gRPC error, or nil/nil for the "missing data"
	// scenario).
	pullFn func(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error)
}

func (f *fakeAgentFinderStore) Push(context.Context, *corev1.Record) (*corev1.RecordRef, error) {
	panic("fakeAgentFinderStore.Push: not implemented")
}

func (f *fakeAgentFinderStore) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	return f.pullFn(ctx, ref)
}

func (f *fakeAgentFinderStore) Lookup(context.Context, *corev1.RecordRef) (*corev1.RecordMeta, error) {
	panic("fakeAgentFinderStore.Lookup: not implemented")
}

func (f *fakeAgentFinderStore) Delete(context.Context, *corev1.RecordRef) error {
	panic("fakeAgentFinderStore.Delete: not implemented")
}

func (f *fakeAgentFinderStore) IsReady(context.Context) bool { return true }

// fakeAgentFinderCatalogDB is a hand-rolled types.CatalogDatabaseAPI
// test double. It captures the FilterOption list the controller passes
// (so tests can assert on filter shape) and dispatches the body of
// GetCatalogEntries to a per-test getFn.
type fakeAgentFinderCatalogDB struct {
	getFn        func(filters *types.RecordFilters) ([]*catalogv1.CatalogEntry, bool, error)
	lastFilters  *types.RecordFilters
	callsObserve int
}

func (f *fakeAgentFinderCatalogDB) GetCatalogEntries(opts ...types.FilterOption) ([]*catalogv1.CatalogEntry, bool, error) {
	cfg := &types.RecordFilters{}
	for _, opt := range opts {
		opt(cfg)
	}

	f.lastFilters = cfg
	f.callsObserve++

	if f.getFn == nil {
		return nil, false, nil
	}

	return f.getFn(cfg)
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const testValidCID = "baeareihtah57men2xt7sgcekhimvdw7oywycmdckey7xhmjzuw5jh6owya"

// minimalOASFRecord builds a non-empty *corev1.Record with just enough
// content for the oasf formatter to round-trip without errors. It
// deliberately omits module-specific fields (A2A, Skills, MCP) so this
// fixture is reusable across the GetAgent and ExportAgent suites.
func minimalOASFRecord(t *testing.T, name string) *corev1.Record {
	t.Helper()

	data, err := structpb.NewStruct(map[string]any{
		"name":           name,
		"schema_version": "1.0.0",
		"version":        "1.0.0",
		"description":    "test fixture",
	})
	require.NoError(t, err)

	return &corev1.Record{Data: data}
}

// minimalCatalogEntry builds the projected CatalogEntry shape callers
// will see from GetAgent. It mirrors what Record.ToCatalog would emit
// for a leaf entry without exercising the full GORM projection.
func minimalCatalogEntry(cid string) *catalogv1.CatalogEntry {
	return &catalogv1.CatalogEntry{
		Identifier:  "urn:ai:agntcy.org:cid:" + cid,
		DisplayName: "burger_seller_agent",
		MediaType:   "application/a2a-agent-card+json",
		Artifact:    &catalogv1.CatalogEntry_Url{Url: "https://example.test/agent-card"},
	}
}

// ---------------------------------------------------------------------------
// GetAgent — REST-symmetric singular endpoint
// ---------------------------------------------------------------------------

func TestAgentFinderGetAgent(t *testing.T) {
	t.Run("returns CatalogEntry for a known CID", func(t *testing.T) {
		db := &fakeAgentFinderCatalogDB{
			getFn: func(_ *types.RecordFilters) ([]*catalogv1.CatalogEntry, bool, error) {
				return []*catalogv1.CatalogEntry{minimalCatalogEntry(testValidCID)}, false, nil
			},
		}
		ctlr := NewAgentFinderController(db, &fakeAgentFinderStore{}, "")

		resp, err := ctlr.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: testValidCID})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.GetEntry())
		assert.Equal(t, "burger_seller_agent", resp.GetEntry().GetDisplayName())

		// REST-symmetry check: the controller must pass the CID
		// through as a WithCIDs filter so the DB layer can do a
		// primary-key lookup. Drift here breaks the "GET /agents/{cid}
		// returns one of the things GET /agents returns" invariant.
		require.NotNil(t, db.lastFilters)
		assert.Equal(t, []string{testValidCID}, db.lastFilters.RecordCIDs)
		assert.Equal(t, 1, db.lastFilters.Limit, "controller should cap the lookup at one row")
	})

	t.Run("returns NotFound for an unknown CID", func(t *testing.T) {
		db := &fakeAgentFinderCatalogDB{
			getFn: func(_ *types.RecordFilters) ([]*catalogv1.CatalogEntry, bool, error) {
				return nil, false, nil
			},
		}
		ctlr := NewAgentFinderController(db, &fakeAgentFinderStore{}, "")

		_, err := ctlr.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: testValidCID})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("rejects a whitespace-only CID with InvalidArgument", func(t *testing.T) {
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, &fakeAgentFinderStore{}, "")

		_, err := ctlr.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: "   "})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("rejects a nil request with InvalidArgument", func(t *testing.T) {
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, &fakeAgentFinderStore{}, "")

		_, err := ctlr.GetAgent(context.Background(), nil)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("maps DB errors to Internal", func(t *testing.T) {
		db := &fakeAgentFinderCatalogDB{
			getFn: func(_ *types.RecordFilters) ([]*catalogv1.CatalogEntry, bool, error) {
				return nil, false, errors.New("database on fire")
			},
		}
		ctlr := NewAgentFinderController(db, &fakeAgentFinderStore{}, "")

		_, err := ctlr.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: testValidCID})
		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

// ---------------------------------------------------------------------------
// ExportAgent — multi-format export sub-resource
// ---------------------------------------------------------------------------

func TestAgentFinderExportAgent(t *testing.T) {
	t.Run("defaults to oasf when format is empty and returns JSON Content-Type", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return minimalOASFRecord(t, "burger_seller_agent"), nil
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		resp, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: testValidCID})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "application/json", resp.GetContentType())

		// Round-trip the body bytes to prove the controller forwards
		// the formatter's output unchanged (same indentation, same
		// trailing newline, identical to dirctl export).
		var parsed map[string]any
		require.NoError(t, json.Unmarshal(resp.GetData(), &parsed))
		assert.Equal(t, "burger_seller_agent", parsed["name"])
	})

	t.Run("accepts explicit format=oasf", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return minimalOASFRecord(t, "agent-x"), nil
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		resp, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{
			Cid:    testValidCID,
			Format: exportfmt.FormatOASF,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "application/json", resp.GetContentType())
	})

	t.Run("rejects unknown formats with InvalidArgument", func(t *testing.T) {
		store := &fakeAgentFinderStore{}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{
			Cid:    testValidCID,
			Format: "json5",
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		// Error message must enumerate the known formats so HTTP
		// clients can self-correct.
		st, _ := status.FromError(err)
		assert.Contains(t, st.Message(), exportfmt.FormatOASF)
	})

	t.Run("rejects a whitespace-only CID with InvalidArgument", func(t *testing.T) {
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, &fakeAgentFinderStore{}, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "   "})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("rejects a nil request with InvalidArgument", func(t *testing.T) {
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, &fakeAgentFinderStore{}, "")

		_, err := ctlr.ExportAgent(context.Background(), nil)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("returns Unimplemented when no store is wired", func(t *testing.T) {
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, nil, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: testValidCID})
		require.Error(t, err)
		assert.Equal(t, codes.Unimplemented, status.Code(err))
	})

	t.Run("propagates NotFound from the store", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return nil, status.Error(codes.NotFound, "record not found")
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: testValidCID})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("maps opaque store errors to Internal", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return nil, errors.New("disk on fire")
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: testValidCID})
		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("returns Internal when the record has no data", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return &corev1.Record{}, nil
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: testValidCID})
		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("passes the request CID through to the store unchanged", func(t *testing.T) {
		var observed string

		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
				observed = ref.GetCid()

				return minimalOASFRecord(t, "ok"), nil
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: testValidCID})
		require.NoError(t, err)
		assert.Equal(t, testValidCID, observed)
	})

	// FailedPrecondition (HTTP 400) is the right shape for "this
	// record exists and the format is real, but the data doesn't
	// carry what the format reads" — e.g. asking for agent-skill on
	// a record without core/language_model/agentskills. Previously
	// this fell into the Internal bucket and would page operators.
	//
	// The minimal OASF record we use elsewhere lacks all module-
	// specific data, so it's a perfect victim for the a2a, skill,
	// and mcp-ghcopilot formatters.
	t.Run("returns FailedPrecondition when record cannot be projected to a2a", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return minimalOASFRecord(t, "no-a2a-here"), nil
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{
			Cid:    testValidCID,
			Format: exportfmt.FormatA2A,
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))

		// The status message must mention both the CID and the
		// format so clients can surface a useful error without
		// reading server logs.
		st, _ := status.FromError(err)
		assert.Contains(t, st.Message(), testValidCID)
		assert.Contains(t, st.Message(), exportfmt.FormatA2A)
	})

	t.Run("returns FailedPrecondition when record cannot be projected to agent-skill", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return minimalOASFRecord(t, "no-skill-here"), nil
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{
			Cid:    testValidCID,
			Format: exportfmt.FormatAgentSkill,
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})

	t.Run("returns FailedPrecondition when record cannot be projected to mcp-ghcopilot", func(t *testing.T) {
		store := &fakeAgentFinderStore{
			pullFn: func(_ context.Context, _ *corev1.RecordRef) (*corev1.Record, error) {
				return minimalOASFRecord(t, "no-mcp-here"), nil
			},
		}
		ctlr := NewAgentFinderController(&fakeAgentFinderCatalogDB{}, store, "")

		_, err := ctlr.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{
			Cid:    testValidCID,
			Format: exportfmt.FormatMCPGHCopilot,
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// ---------------------------------------------------------------------------
// Cross-cutting: ContentType mapping is shared with api/exportfmt and
// is exercised through ExportAgent above. The standalone test below
// pins the matrix so a future formatter that returns a new extension
// (e.g. text/yaml) immediately fails this assertion rather than
// silently shipping a bad Content-Type.
// ---------------------------------------------------------------------------

func TestExportContentTypeMatrix(t *testing.T) {
	cases := []struct {
		ext  string
		want string
	}{
		{exportfmt.ExtJSON, "application/json"},
		{exportfmt.ExtMarkdown, "text/markdown; charset=utf-8"},
		{".unknown", "application/octet-stream"},
	}

	for _, tc := range cases {
		t.Run(tc.ext, func(t *testing.T) {
			assert.Equal(t, tc.want, exportfmt.ContentTypeForExtension(tc.ext))
		})
	}
}
