// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pusher

import (
	"context"
	"errors"
	"sync"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
	"google.golang.org/protobuf/types/known/structpb"
)

type stubClient struct {
	pushFn func(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error)
}

func (s *stubClient) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	if s.pushFn != nil {
		return s.pushFn(ctx, record)
	}

	return &corev1.RecordRef{Cid: "defaultcid"}, nil
}

func (s *stubClient) SearchCIDs(ctx context.Context, req *searchv1.SearchCIDsRequest) (streaming.StreamResult[searchv1.SearchCIDsResponse], error) {
	return nil, errors.New("not implemented")
}

func (s *stubClient) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
	return nil, errors.New("not implemented")
}

func recordWithNameVer(name, version string) *corev1.Record {
	return &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"name":    structpb.NewStringValue(name),
				"version": structpb.NewStringValue(version),
			},
		},
	}
}

func TestClientPusher_Push_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := &stubClient{
		pushFn: func(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
			return &corev1.RecordRef{Cid: "bafygood"}, nil
		},
	}

	p := NewClientPusher(client, false, nil)

	in := make(chan *corev1.Record, 1)
	in <- recordWithNameVer("a", "1")

	close(in)

	refCh, errCh := p.Push(ctx, in)

	var cids []string

	for r := range refCh {
		if r != nil {
			cids = append(cids, r.GetCid())
		}
	}

	for e := range errCh {
		if e != nil {
			t.Fatalf("unexpected err: %v", e)
		}
	}

	if len(cids) != 1 || cids[0] != "bafygood" {
		t.Errorf("refs = %v", cids)
	}
}

func TestClientPusher_Push_ErrorContinues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	calls := 0

	client := &stubClient{
		pushFn: func(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("first push failed")
			}

			return &corev1.RecordRef{Cid: "ok"}, nil
		},
	}

	p := NewClientPusher(client, false, nil)

	in := make(chan *corev1.Record, 2)
	in <- recordWithNameVer("a", "1")

	in <- recordWithNameVer("b", "1")

	close(in)

	refCh, errCh := p.Push(ctx, in)

	var (
		refs int
		errs int
		wg   sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()

		for r := range refCh {
			if r != nil && r.GetCid() != "" {
				refs++
			}
		}
	}()

	go func() {
		defer wg.Done()

		for e := range errCh {
			if e != nil {
				errs++
			}
		}
	}()

	wg.Wait()

	if errs != 1 {
		t.Fatalf("errors = %d, want 1", errs)
	}

	if refs != 1 {
		t.Fatalf("refs = %d, want 1", refs)
	}

	if calls != 2 {
		t.Errorf("Push calls = %d, want 2", calls)
	}
}

func TestClientPusher_Push_SignErrorNonFatal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	client := &stubClient{
		pushFn: func(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
			return &corev1.RecordRef{Cid: "cid1"}, nil
		},
	}

	signErr := errors.New("sign failed")

	p := NewClientPusher(client, false, func(ctx context.Context, cid string) error {
		return signErr
	})

	in := make(chan *corev1.Record, 1)
	in <- recordWithNameVer("a", "1")

	close(in)

	refCh, errCh := p.Push(ctx, in)

	var (
		refs       int
		signErrors int
		wg         sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		defer wg.Done()

		for r := range refCh {
			if r != nil && r.GetCid() != "" {
				refs++
			}
		}
	}()

	go func() {
		defer wg.Done()

		for e := range errCh {
			if e != nil {
				signErrors++
			}
		}
	}()

	wg.Wait()

	if refs != 1 {
		t.Errorf("refs = %d, want 1", refs)
	}

	if signErrors != 1 {
		t.Errorf("sign errors = %d, want 1", signErrors)
	}
}

func TestExtractNameVersion(t *testing.T) {
	t.Parallel()

	nv, err := ExtractNameVersion(recordWithNameVer("x", "y"))
	if err != nil {
		t.Fatal(err)
	}

	if nv != "x@y" {
		t.Errorf("got %q", nv)
	}

	_, err = ExtractNameVersion(nil)
	if err == nil {
		t.Fatal("expected error for nil record")
	}
}
