// Package balancecheck provides the Tink Balance Check API client.
//
// Real-time balance verification with continuous access.
// Supports on-demand balance refresh, refresh status polling,
// and consent lifecycle management.
//
// Flow:
//  1. CreateUser + GrantUserAccess — set up the user (once per customer)
//  2. BuildAccountCheckLink — redirect user to connect their bank
//  3. GetAccountCheckReport — retrieve the initial account check report
//  4. CreateAuthorization + GetUserAccessToken — get user token
//  5. RefreshBalance — trigger on-demand real-time balance fetch
//  6. GetRefreshStatus — poll until status is Completed
//  7. GetAccountBalance — read the updated balance
//
// https://docs.tink.com/resources/account-check
package balancecheck

import (
	"context"
	"fmt"
	"net/url"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

const linkBase = "https://link.tink.com/1.0"

// Service provides Tink Balance Check operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a BalanceCheck service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ── User setup ────────────────────────────────────────────────────────────

// CreateUserParams are the parameters for creating a balance check user.
type CreateUserParams struct {
	ExternalUserID string `json:"external_user_id"`
	Market         string `json:"market"`
	Locale         string `json:"locale"`
}

// CreateUser creates a permanent Tink user for balance checking.
// Required scope: user:create
func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (*types.TinkUser, error) {
	var out types.TinkUser
	if err := s.http.Post(ctx, "/api/v1/user/create", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GrantUserAccess generates an authorization code for building the Tink Link URL.
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

// BuildAccountCheckLink builds the Tink Link URL for Account Check with balance verification.
// After authentication, Tink redirects to RedirectURI with an account_verification_report_id.
func (s *Service) BuildAccountCheckLink(authorizationCode string, opts types.BuildAccountCheckLinkOptions) string {
	state := opts.State
	if state == "" {
		state = "OPTIONAL"
	}
	test := "false"
	if opts.Test {
		test = "true"
	}
	q := url.Values{
		"client_id":          {opts.ClientID},
		"state":              {state},
		"redirect_uri":       {opts.RedirectURI},
		"authorization_code": {authorizationCode},
		"market":             {opts.Market},
		"test":               {test},
	}
	return fmt.Sprintf("%s/account-check/connect?%s", linkBase, q.Encode())
}

// ── Report retrieval ──────────────────────────────────────────────────────

// GetAccountCheckReport retrieves an Account Check report.
// Required scope: account-verification-reports:read
func (s *Service) GetAccountCheckReport(ctx context.Context, reportID string) (*types.AccountCheckReport, error) {
	var out types.AccountCheckReport
	path := fmt.Sprintf("/api/v1/account-verification-reports/%s", reportID)
	if err := s.http.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Balance operations ────────────────────────────────────────────────────

// CreateAuthorization creates a data authorization grant for balance operations.
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

// RefreshBalance triggers an asynchronous real-time balance refresh.
// Poll GetRefreshStatus until status is Completed or Failed.
// Required scope: balance-refresh
func (s *Service) RefreshBalance(ctx context.Context, accountID string) (*types.BalanceRefreshResponse, error) {
	body := map[string]string{"accountId": accountID}
	var out types.BalanceRefreshResponse
	if err := s.http.Post(ctx, "/api/v1/balance-refresh", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRefreshStatus returns the current status of a balance refresh operation.
//
// Status values: INITIATED, IN_PROGRESS, COMPLETED, FAILED
// Required scope: balance-refresh:readonly
func (s *Service) GetRefreshStatus(ctx context.Context, refreshID string) (*types.BalanceRefreshStatusResponse, error) {
	var out types.BalanceRefreshStatusResponse
	if err := s.http.Get(ctx, fmt.Sprintf("/api/v1/balance-refresh/%s", refreshID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAccountBalance returns the current balance for a specific account.
// Call this after a refresh completes.
// Required scope: accounts.balances:readonly
func (s *Service) GetAccountBalance(ctx context.Context, accountID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("/data/v2/accounts/%s/balances", accountID)
	if err := s.http.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ── Consent management ────────────────────────────────────────────────────

// GrantConsentUpdate generates an authorization code for renewing user consent.
// Required scopes: credentials:write, authorization:grant
func (s *Service) GrantConsentUpdate(ctx context.Context, params types.GrantUserAccessParams, defaultClientID string) (*types.AuthorizationCode, error) {
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

// BuildConsentUpdateLink builds the Tink Link URL for consent renewal.
// Note: the URL does not include the https:// prefix — this matches the Tink API spec.
func (s *Service) BuildConsentUpdateLink(authorizationCode string, opts types.ConsentUpdateLinkOptions) string {
	q := url.Values{
		"client_id":          {opts.ClientID},
		"redirect_uri":       {opts.RedirectURI},
		"credentials_id":     {opts.CredentialsID},
		"authorization_code": {authorizationCode},
		"market":             {opts.Market},
	}
	return fmt.Sprintf("link.tink.com/1.0/account-check/update-consent?%s", q.Encode())
}
