// Package users provides the Tink Users and Credentials API client.
//
// Manage Tink users and their bank connections (credentials).
// Mutating operations automatically invalidate the response cache.
//
// https://docs.tink.com/api#users
package users

import (
	"context"
	"fmt"
	"net/url"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Users operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Users service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// CreateUser creates a new Tink user with an external user ID.
// Store the returned UserID in your database.
// Required scope: user:create
func (s *Service) CreateUser(ctx context.Context, params types.CreateUserParams) (*types.TinkUser, error) {
	var out types.TinkUser
	if err := s.http.Post(ctx, "/api/v1/user/create", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteUser permanently deletes a user and all their bank data.
// This action is irreversible. Required scope: user:delete
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	body := map[string]string{"user_id": userID}
	if err := s.http.Post(ctx, "/api/v1/user/delete", body, nil); err != nil {
		return err
	}
	s.http.InvalidateUser(userID)
	return nil
}

// ListCredentials lists all bank connections for the authenticated user.
// Cached for 30 seconds. Required scope: credentials:read
func (s *Service) ListCredentials(ctx context.Context) (*types.CredentialsResponse, error) {
	var out types.CredentialsResponse
	if err := s.http.Get(ctx, "/api/v1/credentials/list", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCredential fetches a single credential by ID. Cached for 30 seconds.
// Required scope: credentials:read
func (s *Service) GetCredential(ctx context.Context, credentialID string) (*types.Credential, error) {
	var out types.Credential
	if err := s.http.Get(ctx, fmt.Sprintf("/api/v1/credentials/%s", credentialID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteCredential permanently removes a bank connection.
// Required scope: credentials:write
func (s *Service) DeleteCredential(ctx context.Context, credentialID string) error {
	return s.http.Delete(ctx, fmt.Sprintf("/api/v1/credentials/%s", credentialID))
}

// RefreshCredential triggers a data refresh from the bank.
// Cache is fully invalidated after refresh. Required scope: credentials:refresh
func (s *Service) RefreshCredential(ctx context.Context, credentialID string) (*types.Credential, error) {
	var out types.Credential
	if err := s.http.Post(ctx, fmt.Sprintf("/api/v1/credentials/%s/refresh", credentialID), struct{}{}, &out); err != nil {
		return nil, err
	}
	// Extra explicit invalidation — all user data was refreshed at the bank.
	s.http.InvalidateUser()
	return &out, nil
}

// CreateAuthorization creates an authorization grant for a user.
// Returns a short-lived code to exchange for a user access token.
// Required scope: authorization:grant
func (s *Service) CreateAuthorization(ctx context.Context, userID, scope string) (*types.AuthorizationCode, error) {
	form := url.Values{"user_id": {userID}, "scope": {scope}}
	var out types.AuthorizationCode
	if err := s.http.PostForm(ctx, "/api/v1/oauth/authorization-grant", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUserAccessToken exchanges an authorization code for a user access token.
func (s *Service) GetUserAccessToken(ctx context.Context, clientID, clientSecret, code string) (*types.TokenResponse, error) {
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
