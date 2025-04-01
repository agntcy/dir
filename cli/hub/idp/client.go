package idp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	paramClientId            = "client_id"
	paramCodeChallenge       = "code_challenge"
	paramCodeChallengeMethod = "code_challenge_method"
	paramNonce               = "nonce"
	paramRedirectUri         = "redirect_uri"
	paramResponseType        = "response_type"
	paramState               = "state"
	paramScope               = "scope"
	paramGrantType           = "grant_type"
	paramCode                = "code"
	paramCodeVerifier        = "code_verifier"

	headerAccept       = "Accept"
	headerContentType  = "Content-Type"
	headerCacheControl = "Cache-Control"
)

var (
	ErrRequestCreation = fmt.Errorf("failed to create request")
	ErrRequestSending  = fmt.Errorf("failed to send request")
	ErrParsingResponse = fmt.Errorf("failed to parse response")
)

type RequestTokenRequest struct {
	ClientId    string
	RedirectUri string
	Verifier    string
	Code        string
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}
type RequestTokenResponse struct {
	Token    *Token
	Body     []byte
	Response *http.Response
}

type ChallengeMethod string

const (
	ChallengeMethodS256 ChallengeMethod = "S256"
)

type AuthorizeRequest struct {
	ClientId      string
	S256Challenge string
	Nonce         string
	RedirectUri   string
	RequestId     string
}

type LogoutRequest struct {
	IdToken string
}

type LogoutResponse struct {
	Body     []byte
	Response *http.Response
}

type IdpClient interface {
	AuthorizeUrl(r *AuthorizeRequest) string

	RequestToken(request *RequestTokenRequest) (*RequestTokenResponse, error)
	Logout(request *LogoutRequest) (*LogoutResponse, error)
}

type idpClient struct {
	BaseUrl string
}

func NewClient(baseUrl string) *idpClient {
	return &idpClient{BaseUrl: baseUrl}
}

func (i *idpClient) RequestToken(request *RequestTokenRequest) (*RequestTokenResponse, error) {
	data := url.Values{}
	data.Set(paramGrantType, "authorization_code")
	data.Set(paramClientId, request.ClientId)
	data.Set(paramRedirectUri, request.RedirectUri)
	data.Set(paramCode, request.Code)
	data.Set(paramCodeVerifier, request.Verifier)

	tokenReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/token", i.BaseUrl), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: token request", ErrRequestCreation)
	}

	tokenReq.Header.Add(headerAccept, "application/json")
	tokenReq.Header.Add(headerContentType, "application/x-www-form-urlencoded")
	tokenReq.Header.Add(headerCacheControl, "no-cache")

	resp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("%w: token request", ErrRequestSending)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrParsingResponse
	}

	var t *Token
	err = json.Unmarshal(body, &t)
	if err != nil {
		return nil, ErrParsingResponse
	}

	return &RequestTokenResponse{
		Token:    t,
		Body:     body,
		Response: resp,
	}, nil
}

func (i *idpClient) AuthorizeUrl(r *AuthorizeRequest) string {
	params := url.Values{}
	params.Add(paramClientId, r.ClientId)
	params.Add(paramCodeChallenge, r.S256Challenge)
	params.Add(paramCodeChallengeMethod, string(ChallengeMethodS256))
	params.Add(paramNonce, r.Nonce)
	params.Add(paramRedirectUri, r.RedirectUri)
	params.Add(paramResponseType, "code")
	params.Add(paramState, fmt.Sprintf(`{"sessionRequest":"%s"}`, r.RequestId))
	params.Add(paramScope, "openid offline_access")

	return fmt.Sprintf("%s/v1/authorize?%s", i.BaseUrl, params.Encode())
}

func (i *idpClient) Logout(request *LogoutRequest) (*LogoutResponse, error) {
	data := url.Values{}
	data.Set("id_token_hint", request.IdToken)

	logoutReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/logout", i.BaseUrl), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: logout request", ErrRequestCreation)
	}

	logoutReq.Header.Add(headerAccept, "application/json")
	logoutReq.Header.Add(headerContentType, "application/x-www-form-urlencoded")
	logoutReq.Header.Add(headerCacheControl, "no-cache")

	resp, err := http.DefaultClient.Do(logoutReq)
	if err != nil {
		return nil, fmt.Errorf("%w: logout request", ErrRequestSending)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrParsingResponse
	}

	return &LogoutResponse{
		Body:     body,
		Response: resp,
	}, nil
}
