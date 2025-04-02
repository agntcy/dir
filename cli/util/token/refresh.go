package token

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/secretstore"
)

func RefreshTokenIfExpired(idpClient idp.Client, token *secretstore.TokenSecret, clientId string) (*secretstore.TokenSecret, bool, error) {
	if isTokenExpired(token.AccessToken) {
		if token.RefreshToken == "" {
			return nil, false, fmt.Errorf("access token is expired and refresh token is empty")
		}

		resp, err := idpClient.RefreshToken(&idp.RefreshTokenRequest{
			RefreshToken: token.RefreshToken,
			ClientId:     clientId,
		})
		if err != nil {
			return nil, false, fmt.Errorf("failed to refresh token: %w", err)
		}
		if resp.Response.StatusCode != http.StatusOK {
			return nil, false, fmt.Errorf("failed to refresh token: %s", string(resp.Body))
		}

		return &secretstore.TokenSecret{
			IdToken:      resp.Token.IdToken,
			RefreshToken: resp.Token.RefreshToken,
			AccessToken:  resp.Token.AccessToken,
		}, true, nil
	}

	return token, false, nil
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
