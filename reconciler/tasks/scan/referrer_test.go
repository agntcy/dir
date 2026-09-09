// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"context"
	"errors"
	"slices"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	scanv1 "github.com/agntcy/dir/api/security/v1"
)

const testRecordCID = "baeareitestrecordcid"

// fakeReferrerStore records the referrer writes a task makes and serves a fixed
// set of existing referrers back to it.
type fakeReferrerStore struct {
	pushedCID string
	pushErr   error

	existing  []*corev1.RecordReferrer
	walkErr   error
	walkCalls int

	deleteErr   error
	deleted     []string
	deleteCalls int
}

func (f *fakeReferrerStore) PushReferrer(_ context.Context, _ string, _ *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	if f.pushErr != nil {
		return nil, f.pushErr
	}

	return &corev1.ReferrerRef{Cid: f.pushedCID}, nil
}

func (f *fakeReferrerStore) WalkReferrers(_ context.Context, _ string, referrerType string, walkFn func(*corev1.RecordReferrer) error) error {
	f.walkCalls++

	if f.walkErr != nil {
		return f.walkErr
	}

	for _, ref := range f.existing {
		if !sharesArtifactType(referrerType, ref.GetType()) {
			continue
		}

		if err := walkFn(ref); err != nil {
			return err
		}
	}

	return nil
}

// sharesArtifactType reproduces the OCI store's type filter, which is coarser than it looks:
// apiToOCIType maps every referrer type except signatures and public keys onto one shared media
// type, so a walk filtered on scan reports also yields every other custom referrer. The fake
// matches that, otherwise it would hide the task's need to check the type itself.
func sharesArtifactType(want, got string) bool {
	if want == "" || want == got {
		return true
	}

	distinct := func(t string) bool {
		return t == corev1.SignatureReferrerType || t == corev1.PublicKeyReferrerType
	}

	return !distinct(want) && !distinct(got)
}

func (f *fakeReferrerStore) DeleteReferrer(_ context.Context, _ string, referrerCID string, _ string) ([]string, error) {
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}

	f.deleteCalls++
	f.deleted = append(f.deleted, referrerCID)

	return []string{referrerCID}, nil
}

func (f *fakeReferrerStore) DeleteReferrers(_ context.Context, _ string, referrerCIDs []string, _ string) ([]string, error) {
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}

	f.deleteCalls++
	f.deleted = append(f.deleted, referrerCIDs...)

	return referrerCIDs, nil
}

// scanReferrer builds a stored scan report referrer as WalkReferrers would
// return it.
func scanReferrer(t *testing.T, cid string, scannerType scanv1.ScannerType) *corev1.RecordReferrer {
	t.Helper()

	ref, err := (&scanv1.ScanReport{
		ScannerType: scannerType,
		ScannedAt:   "2026-01-01T00:00:00Z",
		IsSafe:      true,
	}).MarshalReferrer()
	if err != nil {
		t.Fatalf("MarshalReferrer: %v", err)
	}

	ref.ReferrerRef = &corev1.ReferrerRef{Cid: cid}

	return ref
}

// mcpReport is the report being pushed in these tests. MCP throughout, so that
// the reports the other scanners left on the same record stay visible as
// records the prune must not touch.
func mcpReport() *scanv1.ScanReport {
	return &scanv1.ScanReport{
		ScannerType: scanv1.ScannerType_SCANNER_TYPE_MCP,
		ScannedAt:   "2026-06-01T00:00:00Z",
		IsSafe:      true,
	}
}

// Reports from earlier runs of the same scanner go; the other scanners' reports
// for the same record stay. Without the scanner-type scoping the MCP runner
// would delete the A2A verdict every time it ran.
func TestPushReferrer_PrunesOnlySameScannerReports(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{
		pushedCID: "new-mcp",
		existing: []*corev1.RecordReferrer{
			scanReferrer(t, "old-mcp-1", scanv1.ScannerType_SCANNER_TYPE_MCP),
			scanReferrer(t, "old-a2a", scanv1.ScannerType_SCANNER_TYPE_A2A),
			scanReferrer(t, "old-mcp-2", scanv1.ScannerType_SCANNER_TYPE_MCP),
			scanReferrer(t, "old-skill", scanv1.ScannerType_SCANNER_TYPE_SKILL),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	want := []string{"old-mcp-1", "old-mcp-2"}

	slices.Sort(store.deleted)

	if !slices.Equal(store.deleted, want) {
		t.Errorf("deleted = %v, want %v", store.deleted, want)
	}
}

// Pruning a backlog must cost one delete call, not one per report. A delete needs the referrer's
// manifest descriptor, which only a walk supplies, so a loop of single deletes repeats the walk
// once per report and the cost grows with the square of the backlog.
func TestPushReferrer_PrunesInOneCall(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{pushedCID: "new-mcp"}

	for _, cid := range []string{"old-1", "old-2", "old-3", "old-4", "old-5", "old-6", "old-7", "old-8"} {
		store.existing = append(store.existing, scanReferrer(t, cid, scanv1.ScannerType_SCANNER_TYPE_MCP))
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	if len(store.deleted) != len(store.existing) {
		t.Errorf("deleted %d referrers, want %d", len(store.deleted), len(store.existing))
	}

	if store.walkCalls != 1 {
		t.Errorf("walkCalls = %d, want 1", store.walkCalls)
	}

	if store.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", store.deleteCalls)
	}
}

// The store's type filter passes every custom referrer type, so a walk for scan reports also
// yields unrelated artifacts. Here one carries a payload that does parse as an MCP report, which
// leaves the task's own type check as the only thing keeping it.
func TestPushReferrer_LeavesOtherReferrerTypesAlone(t *testing.T) {
	t.Parallel()

	foreign := scanReferrer(t, "not-a-scan-report", scanv1.ScannerType_SCANNER_TYPE_MCP)
	foreign.Type = "agntcy.dir.other.v1.Artifact"

	store := &fakeReferrerStore{
		pushedCID: "new-mcp",
		existing: []*corev1.RecordReferrer{
			foreign,
			scanReferrer(t, "old-mcp", scanv1.ScannerType_SCANNER_TYPE_MCP),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	if !slices.Equal(store.deleted, []string{"old-mcp"}) {
		t.Errorf("deleted = %v, want [old-mcp]", store.deleted)
	}
}

// Nothing superseded means no delete call at all. Reaching the store with an empty set would
// depend on it reading that as "delete none" rather than "delete all of this type".
func TestPushReferrer_NothingSupersededIssuesNoDelete(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{
		pushedCID: "new-mcp",
		existing: []*corev1.RecordReferrer{
			scanReferrer(t, "old-a2a", scanv1.ScannerType_SCANNER_TYPE_A2A),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	if store.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", store.deleteCalls)
	}
}

// A rescan producing a byte-identical report resolves to the CID already
// stored, so the prune must not delete what the push just re-created.
func TestPushReferrer_KeepsJustPushedReferrer(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{
		pushedCID: "same-cid",
		existing: []*corev1.RecordReferrer{
			scanReferrer(t, "same-cid", scanv1.ScannerType_SCANNER_TYPE_MCP),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	if len(store.deleted) != 0 {
		t.Errorf("deleted %v, want nothing", store.deleted)
	}
}

// Pruning is housekeeping. A scan that reached a verdict and stored it must not
// be reported as failed because the cleanup behind it did not work.
func TestPushReferrer_PruneFailureDoesNotFailPush(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		store *fakeReferrerStore
	}{
		{
			name: "walk fails",
			store: &fakeReferrerStore{
				pushedCID: "new-mcp",
				walkErr:   errors.New("registry unavailable"),
			},
		},
		{
			name: "delete fails",
			store: &fakeReferrerStore{
				pushedCID: "new-mcp",
				existing: []*corev1.RecordReferrer{
					scanReferrer(t, "old-mcp", scanv1.ScannerType_SCANNER_TYPE_MCP),
				},
				deleteErr: errors.New("manifest delete rejected"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			task := &Task{refStore: tc.store}

			if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
				t.Errorf("pushReferrer: %v, want nil", err)
			}
		})
	}
}

// A failing push is the caller's problem, and nothing should be deleted when
// the replacement never landed.
func TestPushReferrer_PushFailureSkipsPrune(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{
		pushErr: errors.New("registry unavailable"),
		existing: []*corev1.RecordReferrer{
			scanReferrer(t, "old-mcp", scanv1.ScannerType_SCANNER_TYPE_MCP),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err == nil {
		t.Error("pushReferrer: nil error, want failure")
	}

	if len(store.deleted) != 0 {
		t.Errorf("deleted %v, want nothing", store.deleted)
	}
}

// Referrers whose payload will not unmarshal are left alone rather than
// treated as belonging to the running scanner.
func TestPushReferrer_SkipsUnparsableReferrers(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{
		pushedCID: "new-mcp",
		existing: []*corev1.RecordReferrer{
			{Type: corev1.ScanReportReferrerType, ReferrerRef: &corev1.ReferrerRef{Cid: "garbage"}},
			scanReferrer(t, "old-mcp", scanv1.ScannerType_SCANNER_TYPE_MCP),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	if !slices.Equal(store.deleted, []string{"old-mcp"}) {
		t.Errorf("deleted = %v, want [old-mcp]", store.deleted)
	}
}

// UNSPECIFIED is the bucket shared by every scanner this enum does not name, so
// it cannot say which reports an earlier run of the same producer left. Pruning
// on it would have each third party delete the others' reports.
func TestPushReferrer_NeverPrunesUnspecifiedReports(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{
		pushedCID: "new-vendor-a",
		existing: []*corev1.RecordReferrer{
			scanReferrer(t, "vendor-a", scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED),
			scanReferrer(t, "vendor-b", scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, &scanv1.ScanReport{
		ScannerType: scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED,
		ScannedAt:   "2026-06-01T00:00:00Z",
		IsSafe:      true,
	}); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	if len(store.deleted) != 0 {
		t.Errorf("deleted %v, want nothing", store.deleted)
	}
}

// The reverse direction: a third-party report is not an earlier run of a named
// scanner, so a named scanner must leave it in place.
func TestPushReferrer_LeavesThirdPartyReportsAlone(t *testing.T) {
	t.Parallel()

	store := &fakeReferrerStore{
		pushedCID: "new-mcp",
		existing: []*corev1.RecordReferrer{
			scanReferrer(t, "old-mcp", scanv1.ScannerType_SCANNER_TYPE_MCP),
			scanReferrer(t, "vendor-a", scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED),
		},
	}

	task := &Task{refStore: store}

	if err := task.pushReferrer(t.Context(), testRecordCID, mcpReport()); err != nil {
		t.Fatalf("pushReferrer: %v", err)
	}

	if !slices.Equal(store.deleted, []string{"old-mcp"}) {
		t.Errorf("deleted = %v, want [old-mcp]", store.deleted)
	}
}

// A runner name absent from the switch falls into the UNSPECIFIED bucket, where
// its reports would be indistinguishable from a third party's. NewTask refuses
// to start with such a runner; this pins the fallthrough that check relies on.
func TestToProtoScannerType_UnknownRunnerIsUnspecified(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]scanv1.ScannerType{
		"mcp":     scanv1.ScannerType_SCANNER_TYPE_MCP,
		"skill":   scanv1.ScannerType_SCANNER_TYPE_SKILL,
		"a2a":     scanv1.ScannerType_SCANNER_TYPE_A2A,
		"llm":     scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED,
		"MCP-v2":  scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED,
		"unknown": scanv1.ScannerType_SCANNER_TYPE_UNSPECIFIED,
	} {
		if got := toProtoScannerType(name); got != want {
			t.Errorf("toProtoScannerType(%q) = %v, want %v", name, got, want)
		}
	}
}
