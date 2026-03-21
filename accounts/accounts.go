// Package accounts provides the Tink Accounts API client.
//
// Access bank account data and real-time balances for authenticated users.
// Responses for /data/v2/accounts are cached for 5 minutes.
//
// Required scopes: accounts:read, balances:read
// https://docs.tink.com/api#accounts
package accounts

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Accounts operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs an Accounts service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ListAccounts returns all bank accounts for the authenticated user.
// Pass nil opts for defaults (no filter, no pagination).
func (s *Service) ListAccounts(ctx context.Context, opts *types.AccountsListOptions) (*types.AccountsResponse, error) {
	var out types.AccountsResponse
	if err := s.http.Get(ctx, "/data/v2/accounts", accountsQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAccount fetches a single account by ID.
func (s *Service) GetAccount(ctx context.Context, accountID string) (*types.Account, error) {
	var out types.Account
	if err := s.http.Get(ctx, fmt.Sprintf("/data/v2/accounts/%s", accountID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBalances returns the current balances for a specific account.
func (s *Service) GetBalances(ctx context.Context, accountID string) (*types.AccountBalances, error) {
	var out types.AccountBalances
	if err := s.http.Get(ctx, fmt.Sprintf("/data/v2/accounts/%s/balances", accountID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func accountsQuery(opts *types.AccountsListOptions) url.Values {
	if opts == nil {
		return nil
	}
	q := url.Values{}
	if opts.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(opts.PageSize))
	}
	if opts.PageToken != "" {
		q.Set("pageToken", opts.PageToken)
	}
	if len(opts.TypeIn) > 0 {
		q.Set("typeIn", strings.Join(opts.TypeIn, ","))
	}
	return q
}
