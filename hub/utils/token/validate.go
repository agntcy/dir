// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"errors"

	"github.com/agntcy/dir/hub/sessionstore"
	ctxUtils "github.com/agntcy/dir/hub/utils/context"
	"github.com/spf13/cobra"
)

func ValidateAccessTokenFromContext(cmd *cobra.Command) error {
	session, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd)
	if !ok {
		return errors.New("could not get current hub session from context")
	}

	return ValidateAccessToken(session)
}

func ValidateAccessToken(session *sessionstore.HubSession) error {
	if session == nil || session.CurrentTenant == "" || session.Tokens == nil {
		return errors.New("invalid session token")
	}

	if _, ok := session.Tokens[session.CurrentTenant]; !ok ||
		session.Tokens[session.CurrentTenant] == nil ||
		session.Tokens[session.CurrentTenant].AccessToken == "" {
		return errors.New("invalid session token")
	}

	return nil
}
