// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunScannerInstall(t *testing.T) {
	origInstall, origLook := installTool, lookPath

	t.Cleanup(func() { installTool, lookPath = origInstall, origLook })

	newCmd := func() *cobra.Command {
		c := &cobra.Command{}
		c.SetOut(&bytes.Buffer{})
		c.SetErr(&bytes.Buffer{})
		c.SetContext(context.Background())

		return c
	}

	t.Run("installs all scanners", func(t *testing.T) {
		var got []string

		lookPath = func(string) (string, error) { return "/bin/uv", nil }
		installTool = func(_ context.Context, sp scannerPackage) ([]byte, error) {
			got = append(got, sp.binary)

			return nil, nil
		}

		if err := runScannerInstall(newCmd()); err != nil {
			t.Fatal(err)
		}

		if len(got) != len(scannerPackages) {
			t.Fatalf("installed %v", got)
		}
	})

	t.Run("fails without uv", func(t *testing.T) {
		lookPath = func(string) (string, error) { return "", errors.New("missing") }

		if err := runScannerInstall(newCmd()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("surfaces install failure", func(t *testing.T) {
		lookPath = func(string) (string, error) { return "/bin/uv", nil }
		installTool = func(context.Context, scannerPackage) ([]byte, error) { return []byte("boom"), errors.New("exit 1") }

		if err := runScannerInstall(newCmd()); err == nil {
			t.Fatal("expected error")
		}
	})
}
