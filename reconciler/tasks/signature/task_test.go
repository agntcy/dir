// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:nilnil
package signature

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Name_Interval_IsEnabled(t *testing.T) {
	task, err := NewTask(
		Config{Enabled: true, Interval: 2 * time.Minute},
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, "signature", task.Name())
	assert.Equal(t, 2*time.Minute, task.Interval())
	assert.True(t, task.IsEnabled())
}

func TestTask_IsEnabled_False(t *testing.T) {
	task, err := NewTask(
		Config{Enabled: false, Interval: time.Minute},
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.False(t, task.IsEnabled())
}

func TestTask_Interval_Zero_UsesDefault(t *testing.T) {
	task, err := NewTask(
		Config{Enabled: true, Interval: 0},
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, DefaultInterval, task.Interval())
}

func TestTask_GetRecordTimeout_Zero_UsesDefault(t *testing.T) {
	task, err := NewTask(
		Config{RecordTimeout: 0},
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, DefaultRecordTimeout, task.config.GetRecordTimeout())
}

func TestDigestReferrer(t *testing.T) {
	ref := &corev1.RecordReferrer{Type: "test-type"}
	digest := digestReferrer(ref)
	assert.NotEmpty(t, digest)
	assert.Len(t, digest, 64) // SHA256 hex = 64 chars
	// Same input must produce same digest
	assert.Equal(t, digest, digestReferrer(ref))
}

func TestDigestReferrer_DifferentInputs_DifferentDigests(t *testing.T) {
	ref1 := &corev1.RecordReferrer{Type: "type-a"}
	ref2 := &corev1.RecordReferrer{Type: "type-b"}
	assert.NotEqual(t, digestReferrer(ref1), digestReferrer(ref2))
}

func TestTask_Run_NoRecords_ReturnsNil(t *testing.T) {
	db := &fakeSignatureDB{
		getRecordsNeedingSignatureVerification: func(time.Duration) ([]types.Record, error) {
			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true}, db, nil)
	require.NoError(t, err)

	err = task.Run(context.Background())
	assert.NoError(t, err)
}

func TestTask_Run_DBError_ReturnsError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	db := &fakeSignatureDB{
		getRecordsNeedingSignatureVerification: func(time.Duration) ([]types.Record, error) {
			return nil, wantErr
		},
	}
	task, err := NewTask(Config{Enabled: true}, db, nil)
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get records needing signature verification")
	assert.ErrorIs(t, err, wantErr)
}

func TestTask_Run_WithOneRecord_NoSignatures_DoesNotCallSetRecordTrustedOrUpsert(t *testing.T) {
	const testCID = "test-record-cid"

	setRecordTrustedCalled := false
	upsertCount := 0

	db := &fakeSignatureDB{
		getRecordsNeedingSignatureVerification: func(time.Duration) ([]types.Record, error) {
			return []types.Record{&fakeRecord{cid: testCID}}, nil
		},
		setRecordTrusted:            func(string, bool) { setRecordTrustedCalled = true },
		upsertSignatureVerification: func(types.SignatureVerificationObject) { upsertCount++ },
	}
	store := &fakeReferrerStore{} // no referrers → 0 signatures → early return
	task, err := NewTask(Config{Enabled: true, RecordTimeout: time.Second}, db, store)
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.NoError(t, err)
	assert.False(t, setRecordTrustedCalled, "SetRecordTrusted should not be called when there are no signatures")
	assert.Equal(t, 0, upsertCount)
}

func TestTask_Run_WithOneRecord_CollectSignaturesError_DoesNotCallSetRecordTrusted(t *testing.T) {
	const testCID = "test-record-cid"

	setRecordTrustedCalled := false

	db := &fakeSignatureDB{
		getRecordsNeedingSignatureVerification: func(time.Duration) ([]types.Record, error) {
			return []types.Record{&fakeRecord{cid: testCID}}, nil
		},
		setRecordTrusted: func(string, bool) {
			setRecordTrustedCalled = true
		},
	}
	store := &fakeReferrerStore{walkErr: errors.New("store unavailable")}
	task, err := NewTask(Config{Enabled: true, RecordTimeout: time.Second}, db, store)
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.NoError(t, err)
	assert.False(t, setRecordTrustedCalled, "SetRecordTrusted should not be called when collect signatures fails")
}

// fakeRecord implements types.Record for tests (signature task only uses GetCid).
type fakeRecord struct {
	cid string
}

func (r *fakeRecord) GetCid() string { return r.cid }

func (r *fakeRecord) GetRecordData() (types.RecordData, error) { return nil, nil }

// fakeReferrerStore implements types.ReferrerStoreAPI for tests.
type fakeReferrerStore struct {
	walkErr error // if set, WalkReferrers returns this error
}

func (s *fakeReferrerStore) PushReferrer(context.Context, string, *corev1.RecordReferrer) error {
	return nil
}

func (s *fakeReferrerStore) WalkReferrers(ctx context.Context, recordCID string, referrerType string, walkFn func(*corev1.RecordReferrer) error) error {
	return s.walkErr
}

// fakeSignatureDB implements types.DatabaseAPI for tests.
// GetRecordsNeedingSignatureVerification is configurable; optional hooks record SetRecordTrusted and UpsertSignatureVerification calls.
type fakeSignatureDB struct {
	getRecordsNeedingSignatureVerification func(time.Duration) ([]types.Record, error)
	setRecordTrusted                       func(recordCID string, trusted bool)
	upsertSignatureVerification            func(types.SignatureVerificationObject)
}

func (f *fakeSignatureDB) GetRecordsNeedingSignatureVerification(ttl time.Duration) ([]types.Record, error) {
	if f.getRecordsNeedingSignatureVerification != nil {
		return f.getRecordsNeedingSignatureVerification(ttl)
	}

	return nil, nil
}

func (f *fakeSignatureDB) AddRecord(record types.Record) error { return nil }
func (f *fakeSignatureDB) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
	return nil, nil
}

func (f *fakeSignatureDB) GetRecords(opts ...types.FilterOption) ([]types.Record, error) {
	return nil, nil
}
func (f *fakeSignatureDB) RemoveRecord(cid string) error          { return nil }
func (f *fakeSignatureDB) SetRecordSigned(recordCID string) error { return nil }
func (f *fakeSignatureDB) SetRecordTrusted(recordCID string, trusted bool) error {
	if f.setRecordTrusted != nil {
		f.setRecordTrusted(recordCID, trusted)
	}

	return nil
}
func (f *fakeSignatureDB) CreateSync(remoteURL string, cids []string) (string, error) { return "", nil }
func (f *fakeSignatureDB) GetSyncByID(syncID string) (types.SyncObject, error)        { return nil, nil }
func (f *fakeSignatureDB) GetSyncs(offset, limit int) ([]types.SyncObject, error)     { return nil, nil }
func (f *fakeSignatureDB) GetSyncsByStatus(status storev1.SyncStatus) ([]types.SyncObject, error) {
	return nil, nil
}

func (f *fakeSignatureDB) UpdateSyncStatus(syncID string, status storev1.SyncStatus) error {
	return nil
}
func (f *fakeSignatureDB) DeleteSync(syncID string) error { return nil }
func (f *fakeSignatureDB) CreatePublication(request *routingv1.PublishRequest) (string, error) {
	return "", nil
}

func (f *fakeSignatureDB) GetPublicationByID(publicationID string) (types.PublicationObject, error) {
	return nil, nil
}

func (f *fakeSignatureDB) GetPublications(offset, limit int) ([]types.PublicationObject, error) {
	return nil, nil
}

func (f *fakeSignatureDB) GetPublicationsByStatus(status routingv1.PublicationStatus) ([]types.PublicationObject, error) {
	return nil, nil
}

func (f *fakeSignatureDB) UpdatePublicationStatus(publicationID string, status routingv1.PublicationStatus) error {
	return nil
}
func (f *fakeSignatureDB) DeletePublication(publicationID string) error { return nil }
func (f *fakeSignatureDB) CreateNameVerification(verification types.NameVerificationObject) error {
	return nil
}

func (f *fakeSignatureDB) UpdateNameVerification(verification types.NameVerificationObject) error {
	return nil
}

func (f *fakeSignatureDB) GetVerificationByCID(cid string) (types.NameVerificationObject, error) {
	return nil, nil
}

func (f *fakeSignatureDB) GetRecordsNeedingVerification(ttl time.Duration) ([]types.Record, error) {
	return nil, nil
}

func (f *fakeSignatureDB) CreateSignatureVerification(verification types.SignatureVerificationObject) error {
	return nil
}

func (f *fakeSignatureDB) UpdateSignatureVerification(verification types.SignatureVerificationObject) error {
	return nil
}

func (f *fakeSignatureDB) UpsertSignatureVerification(verification types.SignatureVerificationObject) error {
	if f.upsertSignatureVerification != nil {
		f.upsertSignatureVerification(verification)
	}

	return nil
}

func (f *fakeSignatureDB) GetSignatureVerificationsByRecordCID(recordCID string) ([]types.SignatureVerificationObject, error) {
	return nil, nil
}
func (f *fakeSignatureDB) Close() error                 { return nil }
func (f *fakeSignatureDB) IsReady(context.Context) bool { return true }
