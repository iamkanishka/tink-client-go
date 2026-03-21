// Package loans provides the Tink Loans API client.
//
// Access loan and mortgage account information including balances,
// interest rates, payment schedules, and maturity dates.
//
// Required scopes: accounts:read, loan-accounts:readonly
// https://docs.tink.com/api#loans
package loans

import (
	"context"
	"fmt"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Loans operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Loans service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ListAccounts lists all loan accounts (mortgages, personal loans, auto loans).
func (s *Service) ListAccounts(ctx context.Context) (*types.LoanAccountsResponse, error) {
	var out types.LoanAccountsResponse
	if err := s.http.Get(ctx, "/data/v2/loan-accounts", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAccount returns detailed information for a single loan account.
// Includes interest rate, monthly payment, remaining term, and payment history.
func (s *Service) GetAccount(ctx context.Context, accountID string) (*types.LoanAccount, error) {
	var out types.LoanAccount
	if err := s.http.Get(ctx, fmt.Sprintf("/data/v2/loan-accounts/%s", accountID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
