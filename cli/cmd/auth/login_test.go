// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	"github.com/zitadel/oidc/v3/pkg/client/rp/cli"
	"github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

/*
CLIENT_ID=Ov23li6yijNJ4SvklsFU CLIENT_SECRET="3a2900ad086e40d7751889ad4eac138f943911bd" SCOPES="openid email profile" PORT=5556 \
	go run github.com/zitadel/oidc/v3/example/client/github
*/

var (
	clientID           = getEnv("CLIENT_ID", "Ov23li6yijNJ4SvklsFU")
	clientSecret       = getEnv("CLIENT_SECRET", "3a2900ad086e40d7751889ad4eac138f943911bd")
	clientPort         = getEnv("PORT", "5556")
	clientCallbackPath = "/orbctl/github/callback"
	clientScopes       = []string{oidc.ScopeOpenID, "profile", "email", "user:email"}
	clientEndpoint     = githubOAuth.Endpoint
	key                = genRandomKey()
)

func TestLogin(t *testing.T) {
	rpConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       clientScopes,
		Endpoint:     clientEndpoint,
		RedirectURL:  fmt.Sprintf("http://localhost:%v%v", clientPort, clientCallbackPath),
	}

	ctx := t.Context()
	cookieHandler := http.NewCookieHandler(key, key, http.WithUnsecure())
	relyingParty, err := rp.NewRelyingPartyOAuth(rpConfig, rp.WithCookieHandler(cookieHandler))
	if err != nil {
		fmt.Printf("error creating relaying party: %v", err)
		return
	}
	state := func() string {
		return uuid.New().String()
	}
	token := cli.CodeFlow[*oidc.IDTokenClaims](ctx, relyingParty, clientCallbackPath, clientPort, state)

	tokenRaw, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		fmt.Printf("error marshaling token: %v", err)
		return
	}

	fmt.Printf("Access Token: %s\n", token.AccessToken)
	fmt.Printf("ID Token Claims: %v\n", token.IDTokenClaims)
	fmt.Printf("Token: %s\n", tokenRaw)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func genRandomKey() []byte {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(fmt.Sprintf("failed to generate random key: %v", err))
	}
	return key
}
