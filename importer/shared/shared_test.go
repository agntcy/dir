// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestExtractNameVersion(t *testing.T) {
	t.Parallel()

	r := &corev1.Record{
		Data: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"name":    structpb.NewStringValue("srv"),
				"version": structpb.NewStringValue("1.2.3"),
			},
		},
	}

	s, err := ExtractNameVersion(r)
	if err != nil {
		t.Fatal(err)
	}

	if s != "srv@1.2.3" {
		t.Errorf("got %q", s)
	}

	_, err = ExtractNameVersion(&corev1.Record{})
	if err == nil {
		t.Fatal("expected error")
	}
}
