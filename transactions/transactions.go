// Package transactions provides the Tink Transactions API client.
//
// Supports all three access patterns:
//   - Transactions: standard one-time and recurring transaction access
//   - OneTimeAccess: single-authorization flow, no persistent user
//   - ContinuousAccess: persistent user with recurring bank sync
//
// Required scopes: accounts:read, transactions:read
// https://docs.tink.com/api#transactions
package transactions

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides standard transaction listing.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Transactions service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ListAccounts returns accounts for the authenticated user.
func (s *Service) ListAccounts(ctx context.Context, opts *types.PaginationOptions) (*types.AccountsResponse, error) {
	var out types.AccountsResponse
	if err := s.http.Get(ctx, "/data/v2/accounts", paginationQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTransactions returns transactions with optional filtering.
func (s *Service) ListTransactions(ctx context.Context, opts *types.TransactionsListOptions) (*types.TransactionsResponse, error) {
	var out types.TransactionsResponse
	if err := s.http.Get(ctx, "/data/v2/transactions", txQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// OneTimeAccess
// ─────────────────────────────────────────────────────────────────────────────

// OneTimeAccessService provides single-authorization transaction access.
type OneTimeAccessService struct {
	http *httpclient.HTTPClient
}

// NewOneTimeAccess constructs a OneTimeAccessService.
func NewOneTimeAccess(h *httpclient.HTTPClient) *OneTimeAccessService {
	return &OneTimeAccessService{http: h}
}

// ListAccounts lists accounts for the one-time authorized user.
func (s *OneTimeAccessService) ListAccounts(ctx context.Context, opts *types.PaginationOptions) (*types.AccountsResponse, error) {
	var out types.AccountsResponse
	if err := s.http.Get(ctx, "/data/v2/accounts", paginationQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTransactions lists transactions for the one-time authorized user.
func (s *OneTimeAccessService) ListTransactions(ctx context.Context, opts *types.TransactionsListOptions) (*types.TransactionsResponse, error) {
	var out types.TransactionsResponse
	if err := s.http.Get(ctx, "/data/v2/transactions", txQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ContinuousAccess
// ─────────────────────────────────────────────────────────────────────────────

// ContinuousAccessService manages persistent users with ongoing bank sync.
//
// Flow:
//  1. CreateUser — create a permanent Tink user (once per customer)
//  2. GrantUserAccess — delegate Tink Link access, get authorization code
//  3. BuildTinkLink — redirect user to connect their bank
//  4. CreateAuthorization — create data access grant
//  5. GetUserAccessToken — exchange code for user access token
//  6. ListAccounts / ListTransactions — fetch data on demand
type ContinuousAccessService struct {
	http          *httpclient.HTTPClient
	actorClientID string
}

// NewContinuousAccess constructs a ContinuousAccessService.
func NewContinuousAccess(h *httpclient.HTTPClient, actorClientID string) *ContinuousAccessService {
	return &ContinuousAccessService{http: h, actorClientID: actorClientID}
}

// CreateUserParams are the parameters for creating a continuous access user.
type CreateUserParams struct {
	ExternalUserID string `json:"external_user_id"`
	Locale         string `json:"locale"`
	Market         string `json:"market"`
}

// CreateUserResponse holds the new user ID.
type CreateUserResponse struct {
	UserID string `json:"user_id"`
}

// CreateUser creates a permanent Tink user for ongoing data access.
// Store the returned UserID — you will need it for all subsequent calls.
func (s *ContinuousAccessService) CreateUser(ctx context.Context, params CreateUserParams) (*CreateUserResponse, error) {
	var out CreateUserResponse
	if err := s.http.Post(ctx, "/api/v1/user/create", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GrantUserAccessParams are the parameters for granting user access.
type GrantUserAccessParams struct {
	UserID        string
	IDHint        string
	Scope         string
	ActorClientID string // overrides the default actor client ID
}

// GrantUserAccess generates an authorization code for building a Tink Link URL.
func (s *ContinuousAccessService) GrantUserAccess(ctx context.Context, params GrantUserAccessParams) (*types.AuthorizationCode, error) {
	actorID := params.ActorClientID
	if actorID == "" {
		actorID = s.actorClientID
	}
	form := url.Values{
		"user_id":         {params.UserID},
		"id_hint":         {params.IDHint},
		"actor_client_id": {actorID},
		"scope":           {params.Scope},
	}
	var out types.AuthorizationCode
	if err := s.http.PostForm(ctx, "/api/v1/oauth/authorization-grant/delegate", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BuildTinkLinkOptions are the options for building a Tink Link transactions URL.
type BuildTinkLinkOptions struct {
	ClientID    string
	RedirectURI string
	Market      string
	Locale      string
}

// BuildTinkLink returns the Tink Link URL to redirect the user to for bank connection.
func (s *ContinuousAccessService) BuildTinkLink(authorizationCode string, opts BuildTinkLinkOptions) string {
	q := url.Values{
		"client_id":          {opts.ClientID},
		"redirect_uri":       {opts.RedirectURI},
		"authorization_code": {authorizationCode},
		"market":             {opts.Market},
		"locale":             {opts.Locale},
	}
	return fmt.Sprintf("https://link.tink.com/1.0/transactions/connect-accounts?%s", q.Encode())
}

// CreateAuthorization creates a data authorization grant for an existing user.
func (s *ContinuousAccessService) CreateAuthorization(ctx context.Context, userID, scope string) (*types.AuthorizationCode, error) {
	form := url.Values{"user_id": {userID}, "scope": {scope}}
	var out types.AuthorizationCode
	if err := s.http.PostForm(ctx, "/api/v1/oauth/authorization-grant", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUserAccessToken exchanges an authorization code for a user access token.
func (s *ContinuousAccessService) GetUserAccessToken(ctx context.Context, clientID, clientSecret, code string) (*types.TokenResponse, error) {
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

// ListAccounts lists accounts for the continuous access user.
func (s *ContinuousAccessService) ListAccounts(ctx context.Context, opts *types.PaginationOptions) (*types.AccountsResponse, error) {
	var out types.AccountsResponse
	if err := s.http.Get(ctx, "/data/v2/accounts", paginationQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTransactions lists transactions for the continuous access user.
func (s *ContinuousAccessService) ListTransactions(ctx context.Context, opts *types.TransactionsListOptions) (*types.TransactionsResponse, error) {
	var out types.TransactionsResponse
	if err := s.http.Get(ctx, "/data/v2/transactions", txQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Private helpers
// ─────────────────────────────────────────────────────────────────────────────

func paginationQuery(opts *types.PaginationOptions) url.Values {
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
	return q
}

func txQuery(opts *types.TransactionsListOptions) url.Values {
	if opts == nil {
		return nil
	}
	q := paginationQuery(&opts.PaginationOptions)
	if q == nil {
		q = url.Values{}
	}
	for _, id := range opts.AccountIDIn {
		q.Add("accountIdIn", id)
	}
	if opts.BookedDateGte != "" {
		q.Set("bookedDateGte", opts.BookedDateGte)
	}
	if opts.BookedDateLte != "" {
		q.Set("bookedDateLte", opts.BookedDateLte)
	}
	for _, s := range opts.StatusIn {
		q.Add("statusIn", s)
	}
	for _, id := range opts.CategoryIDIn {
		q.Add("categoryIdIn", id)
	}
	return q
}
