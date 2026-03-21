// Package accountcheck provides the Tink Account Check API client.
//
// Verify bank account ownership by matching user information against bank
// records. Supports two verification flows:
//
//  1. One-time verification (user-match): create a session, redirect user,
//     receive a report ID, fetch the report.
//
//  2. Continuous access: create a persistent user with ongoing access for
//     repeated verification and transaction monitoring.
//
// https://docs.tink.com/resources/account-check
package accountcheck

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

const linkBase = "https://link.tink.com/1.0"

// Service provides Tink Account Check operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs an AccountCheck service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ── One-time verification ─────────────────────────────────────────────────

// CreateSession creates a Tink Link session for one-time account verification.
// Required scope: link-session:write
//
// Example:
//
//	session, err := client.AccountCheck.CreateSession(ctx, types.CreateSessionParams{
//	    User:   types.AccountCheckUser{FirstName: "Jane", LastName: "Smith"},
//	    Market: "GB",
//	})
//	url := client.AccountCheck.BuildLinkURL(session, types.BuildLinkURLOptions{ClientID: "...", Market: "GB"})
func (s *Service) CreateSession(ctx context.Context, params types.CreateSessionParams) (*types.AccountCheckSession, error) {
	var out types.AccountCheckSession
	if err := s.http.Post(ctx, "/link/v1/session", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BuildLinkURLOptions are the parameters for building an account check Tink Link URL.
type BuildLinkURLOptions struct {
	ClientID    string
	Market      string // defaults to "GB"
	RedirectURI string // defaults to https://console.tink.com/callback
}

// BuildLinkURL constructs the Tink Link URL for one-time account check.
// Redirect the user to this URL. After authentication, Tink redirects to
// your RedirectURI with an account_verification_report_id parameter.
func (s *Service) BuildLinkURL(session *types.AccountCheckSession, opts BuildLinkURLOptions) string {
	market := opts.Market
	if market == "" {
		market = "GB"
	}
	redirectURI := opts.RedirectURI
	if redirectURI == "" {
		redirectURI = "https://console.tink.com/callback"
	}
	q := url.Values{
		"client_id":    {opts.ClientID},
		"redirect_uri": {redirectURI},
		"market":       {market},
		"session_id":   {session.SessionID},
	}
	return fmt.Sprintf("%s/account-check?%s", linkBase, q.Encode())
}

// GetReport retrieves an account ownership verification report by ID.
// Report verification.status values: MATCH, NO_MATCH, INDETERMINATE
// Required scope: account-verification-reports:read
func (s *Service) GetReport(ctx context.Context, reportID string) (*types.AccountCheckReport, error) {
	var out types.AccountCheckReport
	path := fmt.Sprintf("/api/v1/account-verification-reports/%s", reportID)
	if err := s.http.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetReportPDF downloads a verification report as a PDF binary.
// template defaults to "standard-1.0" when empty.
// Required scope: account-verification-reports:read
func (s *Service) GetReportPDF(ctx context.Context, reportID, template string) ([]byte, error) {
	if template == "" {
		template = "standard-1.0"
	}
	q := url.Values{"template": {template}}
	path := fmt.Sprintf("/api/v1/account-verification-reports/%s/pdf", reportID)
	return s.http.GetRaw(ctx, path, q)
}

// ListReports lists all account verification reports.
// Required scope: account-verification-reports:read
func (s *Service) ListReports(ctx context.Context, opts *types.PaginationOptions) (*types.AccountCheckReportsResponse, error) {
	q := paginationQuery(opts)
	var out types.AccountCheckReportsResponse
	if err := s.http.Get(ctx, "/api/v1/account-verification-reports", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Continuous access ─────────────────────────────────────────────────────

// CreateUserParams are the parameters for creating a continuous access user.
type CreateUserParams struct {
	ExternalUserID string `json:"external_user_id"`
	Market         string `json:"market"`
	Locale         string `json:"locale"`
}

// CreateUser creates a permanent Tink user for continuous account access.
// Required scope: user:create
func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (*types.TinkUser, error) {
	var out types.TinkUser
	if err := s.http.Post(ctx, "/api/v1/user/create", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GrantUserAccess delegates authorization for Tink Link access.
// Returns a code used to build the Tink Link URL.
// Required scope: authorization:grant
func (s *Service) GrantUserAccess(ctx context.Context, params types.GrantUserAccessParams, defaultClientID string) (*types.AuthorizationCode, error) {
	actorID := params.ActorClientID
	if actorID == "" {
		actorID = defaultClientID
	}
	form := url.Values{
		"user_id":         {params.UserID},
		"id_hint":         {params.IDHint},
		"scope":           {params.Scope},
		"actor_client_id": {actorID},
	}
	var out types.AuthorizationCode
	if err := s.http.PostForm(ctx, "/api/v1/oauth/authorization-grant/delegate", form, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// BuildContinuousAccessLink builds the Tink Link URL for the continuous access flow.
func (s *Service) BuildContinuousAccessLink(authorizationCode string, opts types.ContinuousAccessLinkOptions) string {
	products := opts.Products
	if products == "" {
		products = "ACCOUNT_CHECK,TRANSACTIONS"
	}
	q := url.Values{
		"client_id":          {opts.ClientID},
		"products":           {products},
		"redirect_uri":       {opts.RedirectURI},
		"authorization_code": {authorizationCode},
		"market":             {opts.Market},
		"locale":             {opts.Locale},
	}
	return fmt.Sprintf("%s/products/connect-accounts?%s", linkBase, q.Encode())
}

// CreateAuthorization creates a data authorization grant for an existing user.
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

// ListAccounts lists accounts for the connected user.
// Required scope: accounts:read
func (s *Service) ListAccounts(ctx context.Context, opts *types.PaginationOptions) (*types.AccountsResponse, error) {
	var out types.AccountsResponse
	if err := s.http.Get(ctx, "/data/v2/accounts", paginationQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAccountParties returns account owners and co-owners.
// Required scope: accounts.parties:readonly
func (s *Service) GetAccountParties(ctx context.Context, accountID string) (*types.AccountPartiesResponse, error) {
	var out types.AccountPartiesResponse
	path := fmt.Sprintf("/data/v2/accounts/%s/parties", accountID)
	if err := s.http.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListIdentities returns identity records (name, address, national ID) from connected accounts.
// Required scope: identities:readonly
func (s *Service) ListIdentities(ctx context.Context) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := s.http.Get(ctx, "/data/v2/identities", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListTransactions lists transactions for the connected user.
// Required scope: transactions:read
func (s *Service) ListTransactions(ctx context.Context, opts *types.TransactionsListOptions) (*types.TransactionsResponse, error) {
	var out types.TransactionsResponse
	if err := s.http.Get(ctx, "/data/v2/transactions", txQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteUser permanently deletes a Tink user and all their data.
// Required scope: user:delete
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	return s.http.Delete(ctx, fmt.Sprintf("/api/v1/user/%s", userID))
}

// ── private helpers ───────────────────────────────────────────────────────

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
	return q
}
