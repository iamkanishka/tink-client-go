// Package investments provides the Tink Investments API client.
//
// Access investment portfolio data including accounts and individual holdings.
//
// Required scopes: accounts:read, investment-accounts:readonly
// https://docs.tink.com/api#investments
package investments

import (
	"context"
	"fmt"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Investments operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs an Investments service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ListAccounts lists all investment accounts (brokerage, ISA, pension, etc.).
func (s *Service) ListAccounts(ctx context.Context) (*types.InvestmentAccountsResponse, error) {
	var out types.InvestmentAccountsResponse
	if err := s.http.Get(ctx, "/data/v2/investment-accounts", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAccount fetches a single investment account by ID.
func (s *Service) GetAccount(ctx context.Context, accountID string) (*types.InvestmentAccount, error) {
	var out types.InvestmentAccount
	if err := s.http.Get(ctx, fmt.Sprintf("/data/v2/investment-accounts/%s", accountID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetHoldings returns all positions (stocks, bonds, funds, ETFs) for an account.
func (s *Service) GetHoldings(ctx context.Context, accountID string) (*types.HoldingsResponse, error) {
	var out types.HoldingsResponse
	if err := s.http.Get(ctx, fmt.Sprintf("/data/v2/investment-accounts/%s/holdings", accountID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
