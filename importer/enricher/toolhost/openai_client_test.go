// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package toolhost

import (
	"testing"
)

func TestAzureOpenAIBaseURL(t *testing.T) {
	// Subtests mutate process env; do not run in parallel with each other.
	t.Run("prefers AZURE_OPENAI_BASE_URL", func(t *testing.T) {
		t.Setenv("AZURE_OPENAI_BASE_URL", "https://a.example.com/")
		t.Setenv("AZURE_OPENAI_ENDPOINT", "https://b.example.com/")

		if got := azureOpenAIBaseURL(); got != "https://a.example.com/" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("falls back to AZURE_OPENAI_ENDPOINT", func(t *testing.T) {
		t.Setenv("AZURE_OPENAI_BASE_URL", "")
		t.Setenv("AZURE_OPENAI_ENDPOINT", "https://legacy.example.com/")

		if got := azureOpenAIBaseURL(); got != "https://legacy.example.com/" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("empty when neither set", func(t *testing.T) {
		t.Setenv("AZURE_OPENAI_BASE_URL", "")
		t.Setenv("AZURE_OPENAI_ENDPOINT", "")

		if got := azureOpenAIBaseURL(); got != "" {
			t.Fatalf("got %q want empty", got)
		}
	})
}
