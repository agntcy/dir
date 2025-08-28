// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package user provides the CLI commands for getting user information.
package user

import (
	"errors"
	"fmt"
	"io"
	"time"

	saasv1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	auth "github.com/agntcy/dir/hub/auth"
	authUtils "github.com/agntcy/dir/hub/auth/utils"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

// NewCommand creates the "user" subcommand for the info command.
// It gets information about the current authenticated user.
// Returns the configured *cobra.Command.
func NewCommand(_ *hubOptions.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Get information about the current user",
		Long:  "Get detailed information about the currently authenticated user",
		Args:  cobra.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)

		if !ok || !auth.HasLoginCreds(currentSession) {
			return errors.New("no current session found. please login first")
		}

		hc, err := hubClient.New(currentSession.HubBackendAddress)
		if err != nil {
			return fmt.Errorf("failed to create hub client: %w", err)
		}

		ctx := authUtils.AddAuthToContext(cmd.Context(), currentSession)

		req := &saasv1alpha1.GetUserRequest{}

		user, err := hc.GetUser(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		renderUser(cmd.OutOrStdout(), user)

		return nil
	}

	return cmd
}

// renderUser renders user information.
func renderUser(stream io.Writer, user *saasv1alpha1.User) {
	const (
		gapSize    = 4
		labelWidth = 10
	)

	fields := []struct {
		label string
		value string
	}{
		{"ID:", user.GetId()},
		{"Username:", user.GetUsername()},
	}

	if createdAt := user.GetCreatedAt(); createdAt != nil {
		createdTime := time.Unix(createdAt.GetSeconds(), int64(createdAt.GetNanos()))
		fields = append(fields, struct {
			label string
			value string
		}{"Created:", createdTime.Format(time.RFC3339)})
	}

	if updatedAt := user.GetUpdatedAt(); updatedAt != nil {
		updatedTime := time.Unix(updatedAt.GetSeconds(), int64(updatedAt.GetNanos()))
		fields = append(fields, struct {
			label string
			value string
		}{"Updated:", updatedTime.Format(time.RFC3339)})
	}

	for _, field := range fields {
		labelCol := text.AlignLeft.Apply(field.label, labelWidth+gapSize)
		fmt.Fprintf(stream, "  %s%s\n", labelCol, field.value)
	}
}
