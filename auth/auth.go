// Package auth implements OAuth 2.0 flows for the Tink API.
//
// Supports all Tink OAuth flows:
//   - Client credentials (server-to-server)
//   - Authorization code exchange (after user redirect)
//   - Token refresh
//   - Authorization grant creation and delegation (for Tink Link flows)
//
// Example:
//
//	token, err := client.Auth.GetAccessToken(ctx, "accounts:read transactions:read")
//	if err != nil { log.Fatal(err) }
//	client.SetAccessToken(token.AccessToken)
package auth

import (
	"context"
	"fmt"
	"net/url"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink OAuth 2.0 operations.
type Service struct {
	http    *httpclient.HTTPClient
	baseURL string
}

// New constructs an Auth service.
func New(h *httpclient.HTTPClient, baseURL string) *Service {
	return &Service{http: h, baseURL: baseURL}
}

// GetAccessToken acquires a client credentials bearer token.
// Sets the token on the underlying HTTP client.
func (s *Service) GetAccessToken(ctx context.Context, clientID, clientSecret, scope string) (*types.TokenResponse, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
		"scope":         {scope},
	}
	var out types.TokenResponse
	if err := s.http.PostForm(ctx, "/api/v1/oauth/token", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ExchangeCode exchanges an authorization code (received after user redirect) for a token.
func (s *Service) ExchangeCode(ctx context.Context, clientID, clientSecret, code string) (*types.TokenResponse, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
	}
	var out types.TokenResponse
	if err := s.http.PostForm(ctx, "/api/v1/oauth/token", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RefreshAccessToken exchanges a refresh token for a new access token.
func (s *Service) RefreshAccessToken(ctx context.Context, clientID, clientSecret, refreshToken string) (*types.TokenResponse, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	var out types.TokenResponse
	if err := s.http.PostForm(ctx, "/api/v1/oauth/token", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BuildAuthorizationURL constructs a Tink OAuth authorization URL for user redirect.
// After the user grants access, Tink redirects to RedirectURI with a code parameter.
func (s *Service) BuildAuthorizationURL(opts types.AuthorizationURLOptions) string {
	q := url.Values{
		"client_id":    {opts.ClientID},
		"redirect_uri": {opts.RedirectURI},
		"scope":        {opts.Scope},
	}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Market != "" {
		q.Set("market", opts.Market)
	}
	if opts.Locale != "" {
		q.Set("locale", opts.Locale)
	}
	return fmt.Sprintf("%s/api/v1/oauth/authorization-grant?%s", s.baseURL, q.Encode())
}

// CreateAuthorization creates an authorization grant for a user.
// Returns a short-lived code to exchange for a user access token.
func (s *Service) CreateAuthorization(ctx context.Context, params types.CreateAuthorizationParams) (*types.AuthorizationCode, error) {
	form := url.Values{
		"user_id": {params.UserID},
		"scope":   {params.Scope},
	}
	var out types.AuthorizationCode
	if err := s.http.PostForm(ctx, "/api/v1/oauth/authorization-grant", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DelegateAuthorization delegates an authorization grant to an actor client.
// The returned code is used to build Tink Link URLs.
func (s *Service) DelegateAuthorization(ctx context.Context, params types.DelegateAuthorizationParams) (*types.AuthorizationCode, error) {
	form := url.Values{
		"user_id":         {params.UserID},
		"id_hint":         {params.IDHint},
		"scope":           {params.Scope},
		"actor_client_id": {params.ActorClientID},
	}
	var out types.AuthorizationCode
	if err := s.http.PostForm(ctx, "/api/v1/oauth/authorization-grant/delegate", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateToken checks whether the current access token is valid.
// Returns true if valid, false if expired or invalid.
func (s *Service) ValidateToken(ctx context.Context) bool {
	var out any
	err := s.http.Get(ctx, "/api/v1/user", nil, &out)
	return err == nil
}
