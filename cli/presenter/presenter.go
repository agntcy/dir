// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Print(cmd *cobra.Command, args ...interface{}) {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), args...)
}

func Println(cmd *cobra.Command, args ...interface{}) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), args...)
}

func Printf(cmd *cobra.Command, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), format, args...)
}

func Error(cmd *cobra.Command, args ...interface{}) {
	_, _ = fmt.Fprint(cmd.ErrOrStderr(), args...)
}

func Errorf(cmd *cobra.Command, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
}
