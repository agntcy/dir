// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/agntcy/dir/cli/hub/idp"
	secretstore2 "github.com/agntcy/dir/cli/hub/secretstore"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
)

func RefreshTokenIfExpired(cmd *cobra.Command, addr string, secret *secretstore2.HubSecret, secretStore secretstore2.SecretStore, idpClient idp.Client) error {
	if secret.AccessToken != "" && isTokenExpired(secret.AccessToken) {
		if secret.RefreshToken == "" {
			return errors.New("access token is expired and refresh token is empty")
		}

		resp, err := idpClient.RefreshToken(&idp.RefreshTokenRequest{
			RefreshToken: secret.RefreshToken,
			ClientID:     secret.ClientID,
		})
		if err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}

		if resp.Response.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to refresh token: %s", string(resp.Body))
		}

		newTokenSecret := &secretstore2.TokenSecret{
			AccessToken:  resp.Token.AccessToken,
			RefreshToken: resp.Token.RefreshToken,
			IDToken:      resp.Token.IDToken,
		}
		secret.TokenSecret = newTokenSecret

		// Update context with new token
		newCtx := ctxUtils.SetCurrentHubSecretForContext(cmd.Context(), secret)
		cmd.SetContext(newCtx)

		// Update secret store with new token
		if err = secretStore.SaveHubSecret(addr, secret); err != nil {
			return fmt.Errorf("failed to save hub secret: %w", err)
		}

		return nil
	}

	return nil
}

func isTokenExpired(token string) bool {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, &claims); err != nil {
		return true
	}

	expTime, err := claims.GetExpirationTime()
	if err != nil || expTime == nil || expTime.Before(time.Now()) {
		return true
	}

	return false
}
