// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestExtractNameVersion(t *testing.T) {
	tests := []struct {
		name      string
		record    *corev1.Record
		want      string
		wantErr   bool
		errString string
	}{
		{
			name: "valid record",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name":    structpb.NewStringValue("test-server"),
						"version": structpb.NewStringValue("1.0.0"),
					},
				},
			},
			want:    "test-server@1.0.0",
			wantErr: false,
		},
		{
			name: "record with additional fields",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name":        structpb.NewStringValue("my-server"),
						"version":     structpb.NewStringValue("2.5.3"),
						"description": structpb.NewStringValue("A test server"),
						"skills": structpb.NewListValue(&structpb.ListValue{
							Values: []*structpb.Value{},
						}),
					},
				},
			},
			want:    "my-server@2.5.3",
			wantErr: false,
		},
		{
			name:      "nil record",
			record:    nil,
			wantErr:   true,
			errString: "record or record data is nil",
		},
		{
			name: "record with nil data",
			record: &corev1.Record{
				Data: nil,
			},
			wantErr:   true,
			errString: "record or record data is nil",
		},
		{
			name: "record with nil fields",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: nil,
				},
			},
			wantErr:   true,
			errString: "record data fields are nil",
		},
		{
			name: "missing name field",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"version": structpb.NewStringValue("1.0.0"),
					},
				},
			},
			wantErr:   true,
			errString: "record missing 'name' field",
		},
		{
			name: "missing version field",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name": structpb.NewStringValue("test-server"),
					},
				},
			},
			wantErr:   true,
			errString: "record missing 'version' field",
		},
		{
			name: "empty name field",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name":    structpb.NewStringValue(""),
						"version": structpb.NewStringValue("1.0.0"),
					},
				},
			},
			wantErr:   true,
			errString: "record 'name' field is empty",
		},
		{
			name: "empty version field",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name":    structpb.NewStringValue("test-server"),
						"version": structpb.NewStringValue(""),
					},
				},
			},
			wantErr:   true,
			errString: "record 'version' field is empty",
		},
		{
			name: "name field is not a string",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name":    structpb.NewNumberValue(123),
						"version": structpb.NewStringValue("1.0.0"),
					},
				},
			},
			want:      "", // GetStringValue() returns empty string for non-string values, which triggers error
			wantErr:   true,
			errString: "record 'name' field is empty",
		},
		{
			name: "version field is not a string",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name":    structpb.NewStringValue("test-server"),
						"version": structpb.NewNumberValue(1),
					},
				},
			},
			want:      "", // GetStringValue() returns empty string for non-string values, which triggers error
			wantErr:   true,
			errString: "record 'version' field is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractNameVersion(tt.record)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")

					return
				}

				if tt.errString != "" && !contains(err.Error(), tt.errString) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errString)
				}

				// For error cases, result might still be set (partial extraction)
				if tt.want != "" && result != tt.want {
					t.Errorf("result = %q, want %q", result, tt.want)
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if result != tt.want {
				t.Errorf("result = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON bool // Whether output should be valid JSON
	}{
		{
			name:     "valid JSON object",
			input:    `{"name":"test","version":"1.0.0"}`,
			wantJSON: true,
		},
		{
			name:     "valid JSON array",
			input:    `[{"name":"item1"},{"name":"item2"}]`,
			wantJSON: true,
		},
		{
			name:     "already formatted JSON",
			input:    "{\n  \"name\": \"test\",\n  \"version\": \"1.0.0\"\n}",
			wantJSON: true,
		},
		{
			name:     "invalid JSON",
			input:    `not json at all`,
			wantJSON: false, // Should return original string
		},
		{
			name:     "empty string",
			input:    ``,
			wantJSON: false,
		},
		{
			name:     "malformed JSON",
			input:    `{"name": "test",}`,
			wantJSON: false, // Should return original string
		},
		{
			name:     "JSON with nested structures",
			input:    `{"server":{"name":"test","config":{"port":8080}}}`,
			wantJSON: true,
		},
		{
			name:     "JSON string value",
			input:    `"just a string"`,
			wantJSON: true,
		},
		{
			name:     "JSON number",
			input:    `123`,
			wantJSON: true,
		},
		{
			name:     "JSON boolean",
			input:    `true`,
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatJSON(tt.input)

			if tt.wantJSON {
				verifyValidJSON(t, result, tt.input)
			} else if result != tt.input {
				t.Errorf("formatJSON() = %q, want original input %q for invalid JSON", result, tt.input)
			}
		})
	}
}

func TestNewClientPusher(t *testing.T) {
	mockClient := &mockClientInterface{}
	signFunc := func(ctx context.Context, cid string) error {
		return nil
	}

	pusher := NewClientPusher(mockClient, true, signFunc)

	if pusher == nil {
		t.Fatal("NewClientPusher() returned nil")
	}

	if pusher.client != mockClient {
		t.Error("client was not set correctly")
	}

	if !pusher.debug {
		t.Error("debug flag was not set correctly")
	}

	if pusher.signFunc == nil {
		t.Error("signFunc was not set correctly")
	}

	// Test with nil signFunc
	pusher2 := NewClientPusher(mockClient, false, nil)
	if pusher2.signFunc != nil {
		t.Error("signFunc should be nil when not provided")
	}
}

func verifyValidJSON(t *testing.T, result, input string) {
	t.Helper()

	var obj any
	if err := json.Unmarshal([]byte(result), &obj); err != nil {
		t.Errorf("formatJSON() returned invalid JSON: %v, input was: %q", err, input)
	}
}

// Helper functions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// mockClientInterface is a minimal mock for testing.
type mockClientInterface struct {
	pushErr error
}

func (m *mockClientInterface) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	if m.pushErr != nil {
		return nil, m.pushErr
	}
	return &corev1.RecordRef{Cid: "test-cid"}, nil
}

func (m *mockClientInterface) PullBatch(ctx context.Context, recordRefs []*corev1.RecordRef) ([]*corev1.Record, error) {
	return []*corev1.Record{}, nil
}

func (m *mockClientInterface) SearchCIDs(ctx context.Context, req *searchv1.SearchCIDsRequest) (streaming.StreamResult[searchv1.SearchCIDsResponse], error) {
	return nil, errors.New("not implemented")
}

func TestClientPusher_Push_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockClientInterface{}
	pusher := NewClientPusher(mockClient, false, nil)

	record := &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"name":    structpb.NewStringValue("test-server"),
				"version": structpb.NewStringValue("1.0.0"),
			},
		},
	}
	ch := make(chan *corev1.Record, 1)
	ch <- record
	close(ch)

	refCh, errCh := pusher.Push(ctx, ch)

	refCount := 0
	for ref := range refCh {
		if ref != nil && ref.GetCid() != "" {
			refCount++
		}
	}
	for range errCh {
		t.Error("expected no push errors")
	}
	if refCount != 1 {
		t.Errorf("expected 1 ref, got %d", refCount)
	}
}

func TestClientPusher_Push_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockClientInterface{pushErr: errors.New("push failed")}
	pusher := NewClientPusher(mockClient, false, nil)

	record := &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"name":    structpb.NewStringValue("test-server"),
				"version": structpb.NewStringValue("1.0.0"),
			},
		},
	}
	ch := make(chan *corev1.Record, 1)
	ch <- record
	close(ch)

	refCh, errCh := pusher.Push(ctx, ch)

	// Drain both channels in parallel so pusher goroutine can finish (it sends to errCh, not refCh)
	var refCount, errCount int
	done := make(chan struct{})
	go func() {
		for range refCh {
			refCount++
		}
		done <- struct{}{}
	}()
	go func() {
		for range errCh {
			errCount++
		}
		done <- struct{}{}
	}()
	<-done
	<-done

	if refCount != 0 {
		t.Errorf("expected 0 refs on error, got %d", refCount)
	}
	if errCount != 1 {
		t.Errorf("expected 1 error, got %d", errCount)
	}
}

func TestClientPusher_Push_EmptyChannel(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockClientInterface{}
	pusher := NewClientPusher(mockClient, false, nil)

	ch := make(chan *corev1.Record)
	close(ch)

	refCh, errCh := pusher.Push(ctx, ch)

	refCount := 0
	for range refCh {
		refCount++
	}
	for range errCh {
	}
	if refCount != 0 {
		t.Errorf("expected 0 refs for empty channel, got %d", refCount)
	}
}
