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
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/verify"
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

func TestTask_Run_WithOneRecord_NoSignatures_NoUpsert(t *testing.T) {
	const testCID = "test-record-cid"

	upsertCount := 0

	db := &fakeSignatureDB{
		getRecordsNeedingSignatureVerification: func(time.Duration) ([]types.Record, error) {
			return []types.Record{&fakeRecord{cid: testCID}}, nil
		},
		upsertSignatureVerification: func(types.SignatureVerificationObject) { upsertCount++ },
	}
	// Fetcher returns no signatures → empty perSig → no upserts
	fakeFetcher := &fakeFetcher{
		pullSignatures: func(_ context.Context, _ *corev1.RecordRef) ([]*signv1.Signature, error) {
			return nil, nil
		},
		pullPublicKeys: func(_ context.Context, _ *corev1.RecordRef) ([]string, error) {
			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true, RecordTimeout: time.Second}, db, fakeFetcher)
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, upsertCount)
}

func TestTask_Run_WithOneRecord_VerifyError_Continues(t *testing.T) {
	const testCID = "test-record-cid"

	db := &fakeSignatureDB{
		getRecordsNeedingSignatureVerification: func(time.Duration) ([]types.Record, error) {
			return []types.Record{&fakeRecord{cid: testCID}}, nil
		},
	}
	fakeFetcher := &fakeFetcher{
		pullSignatures: func(_ context.Context, _ *corev1.RecordRef) ([]*signv1.Signature, error) {
			return nil, errors.New("pull unavailable")
		},
		pullPublicKeys: func(_ context.Context, _ *corev1.RecordRef) ([]string, error) {
			return nil, nil
		},
	}
	task, err := NewTask(Config{Enabled: true, RecordTimeout: time.Second}, db, fakeFetcher)
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.NoError(t, err) // Run logs and continues on verify error
}

// fakeRecord implements types.Record for tests (signature task only uses GetCid).
type fakeRecord struct {
	cid string
}

func (r *fakeRecord) GetCid() string { return r.cid }

func (r *fakeRecord) GetRecordData() (types.RecordData, error) { return nil, nil }

// Ensure fakeFetcher implements verify.Fetcher.
var _ verify.Fetcher = (*fakeFetcher)(nil)

// fakeFetcher implements verify.Fetcher for tests.
type fakeFetcher struct {
	pullSignatures func(context.Context, *corev1.RecordRef) ([]*signv1.Signature, error)
	pullPublicKeys func(context.Context, *corev1.RecordRef) ([]string, error)
}

func (f *fakeFetcher) PullSignatures(ctx context.Context, recordRef *corev1.RecordRef) ([]*signv1.Signature, error) {
	if f.pullSignatures != nil {
		return f.pullSignatures(ctx, recordRef)
	}

	return nil, nil
}

func (f *fakeFetcher) PullPublicKeys(ctx context.Context, recordRef *corev1.RecordRef) ([]string, error) {
	if f.pullPublicKeys != nil {
		return f.pullPublicKeys(ctx, recordRef)
	}

	return nil, nil
}

// fakeSignatureDB implements types.DatabaseAPI for tests.
// GetRecordsNeedingSignatureVerification and UpsertSignatureVerification are configurable.
type fakeSignatureDB struct {
	getRecordsNeedingSignatureVerification func(time.Duration) ([]types.Record, error)
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
func (f *fakeSignatureDB) CreateSync(remoteURL string, cids []string, remoteRegistryURL string, repositoryName string) (string, error) {
	return "", nil
}
func (f *fakeSignatureDB) GetSyncByID(syncID string) (types.SyncObject, error)    { return nil, nil }
func (f *fakeSignatureDB) GetSyncs(offset, limit int) ([]types.SyncObject, error) { return nil, nil }
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

func (f *fakeSignatureDB) InvalidateSignatureVerificationsForRecord(recordCID string) error {
	return nil
}

func (f *fakeSignatureDB) Close() error                 { return nil }
func (f *fakeSignatureDB) IsReady(context.Context) bool { return true }
