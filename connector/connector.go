// Package connector provides the Tink Connector API client.
//
// Ingest your own financial data into the Tink platform. Use this when you
// already have account and transaction data and want to leverage Tink's
// enrichment and analytics on top of it.
//
// Flow:
//  1. CreateUser — create a Tink user to own the data
//  2. IngestAccounts — push account data for the user
//  3. IngestTransactions — push transaction data per account
//
// https://docs.tink.com/api#connector
package connector

import (
	"context"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Connector operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Connector service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// CreateUserParams are the parameters for creating a connector user.
type CreateUserParams struct {
	ExternalUserID string `json:"external_user_id"`
	Market         string `json:"market"`
	Locale         string `json:"locale"`
}

// CreateUser creates a Tink user to own the ingested data.
func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (*types.TinkUser, error) {
	var out types.TinkUser
	if err := s.http.Post(ctx, "/api/v1/user/create", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IngestAccounts pushes account data for a user into the Tink platform.
// externalUserID is your internal identifier for the user.
func (s *Service) IngestAccounts(ctx context.Context, externalUserID string, params types.IngestAccountsParams) (map[string]any, error) {
	var out map[string]any
	path := "/connector/users/" + externalUserID + "/accounts"
	if err := s.http.Post(ctx, path, params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// IngestTransactions pushes transaction data for a user into the Tink platform.
// Use IngestTypeRealTime for live feeds; IngestTypeBatch for historical imports.
func (s *Service) IngestTransactions(ctx context.Context, externalUserID string, params types.IngestTransactionsParams) (map[string]any, error) {
	var out map[string]any
	path := "/connector/users/" + externalUserID + "/transactions"
	if err := s.http.Post(ctx, path, params, &out); err != nil {
		return nil, err
	}
	return out, nil
}
