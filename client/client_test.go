package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iamkanishka/tink-client-go/accountcheck"
	"github.com/iamkanishka/tink-client-go/client"
	tinkErrors "github.com/iamkanishka/tink-client-go/errors"
	"github.com/iamkanishka/tink-client-go/types"
)

// mockServer returns an httptest.Server that responds to all requests
// with the given status code and JSON body.
func mockServer(t *testing.T, status int, body interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	}))
}

// routedServer returns a server that dispatches to different handlers by path prefix.
func routedServer(t *testing.T, routes map[string]func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for prefix, handler := range routes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				handler(w, r)
				return
			}
		}
		// Default: 404
		http.Error(w, `{"errorMessage":"not found"}`, http.StatusNotFound)
	}))
}

func jsonHandler(t *testing.T, status int, body interface{}) func(http.ResponseWriter, *http.Request) {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			json.NewEncoder(w).Encode(body)
		}
	}
}

// newTestClient creates a Client pointed at a test server with caching disabled.
func newTestClient(t *testing.T, srv *httptest.Server, opts ...client.Option) *client.Client {
	t.Helper()
	baseOpts := []client.Option{
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
		client.WithDisableCache(),
		client.WithMaxRetries(1),
		client.WithTimeout(5 * time.Second),
		client.WithAccessToken("test-bearer-token"),
	}
	c, err := client.NewWithOptions(append(baseOpts, opts...)...)
	if err != nil {
		t.Fatalf("client.NewWithOptions: %v", err)
	}
	return c
}

// ── Construction ──────────────────────────────────────────────────────────

func TestNew_Defaults(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]string{})
	defer srv.Close()
	c := newTestClient(t, srv)
	info := c.Info()
	if info.Version == "" {
		t.Error("Info.Version should not be empty")
	}
	if info.BaseURL != srv.URL {
		t.Errorf("Info.BaseURL = %q, want %q", info.BaseURL, srv.URL)
	}
}

func TestNew_ReadsEnvVars(t *testing.T) {
	os.Setenv("TINK_CLIENT_ID", "env-client-id")
	os.Setenv("TINK_CLIENT_SECRET", "env-client-secret")
	defer os.Unsetenv("TINK_CLIENT_ID")
	defer os.Unsetenv("TINK_CLIENT_SECRET")
	c, err := client.New(client.Config{})
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	if c.ClientID() != "env-client-id" {
		t.Errorf("ClientID = %q, want env-client-id", c.ClientID())
	}
}

func TestNewWithOptions_FunctionalOptions(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c, err := client.NewWithOptions(
		client.WithCredentials("cid", "csec"),
		client.WithBaseURL(srv.URL),
		client.WithTimeout(10*time.Second),
		client.WithMaxRetries(2),
		client.WithHeader("X-Custom", "header-value"),
		client.WithDisableCache(),
		client.WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	if c.ClientID() != "cid" {
		t.Errorf("ClientID = %q, want cid", c.ClientID())
	}
}

func TestNew_AllNamespacesPresent(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c := newTestClient(t, srv)

	// Verify all 24 service namespaces are non-nil
	checks := []struct {
		name string
		val  interface{}
	}{
		{"Auth", c.Auth},
		{"Accounts", c.Accounts},
		{"Transactions", c.Transactions},
		{"TransactionsOneTimeAccess", c.TransactionsOneTimeAccess},
		{"TransactionsContinuousAccess", c.TransactionsContinuousAccess},
		{"Providers", c.Providers},
		{"Categories", c.Categories},
		{"Statistics", c.Statistics},
		{"Users", c.Users},
		{"Investments", c.Investments},
		{"Loans", c.Loans},
		{"Budgets", c.Budgets},
		{"CashFlow", c.CashFlow},
		{"FinancialCalendar", c.FinancialCalendar},
		{"AccountCheck", c.AccountCheck},
		{"BalanceCheck", c.BalanceCheck},
		{"BusinessAccountCheck", c.BusinessAccountCheck},
		{"IncomeCheck", c.IncomeCheck},
		{"ExpenseCheck", c.ExpenseCheck},
		{"RiskInsights", c.RiskInsights},
		{"RiskCategorisation", c.RiskCategorisation},
		{"Connector", c.Connector},
		{"Link", c.Link},
		{"Connectivity", c.Connectivity},
	}
	for _, tc := range checks {
		if tc.val == nil {
			t.Errorf("client.%s is nil — all namespaces must be initialized", tc.name)
		}
	}
}

// ── Token management ──────────────────────────────────────────────────────

func TestSetAccessToken(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c := newTestClient(t, srv)

	c.SetAccessToken("new-bearer-token")
	if c.AccessToken() != "new-bearer-token" {
		t.Errorf("AccessToken() = %q, want new-bearer-token", c.AccessToken())
	}
}

func TestInfo_HasToken(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c, _ := client.NewWithOptions(
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
	)
	if c.Info().HasToken {
		t.Error("HasToken should be false with no token")
	}
	c.SetAccessToken("tok123")
	if !c.Info().HasToken {
		t.Error("HasToken should be true after SetAccessToken")
	}
}

func TestAuthenticate_SetsToken(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"access_token": "acquired-token",
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        "accounts:read",
	})
	defer srv.Close()

	c, _ := client.NewWithOptions(
		client.WithCredentials("cid", "csec"),
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
		client.WithDisableCache(),
		client.WithMaxRetries(1),
	)
	if err := c.Authenticate(context.Background(), "accounts:read"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if c.AccessToken() != "acquired-token" {
		t.Errorf("AccessToken after Authenticate = %q, want acquired-token", c.AccessToken())
	}
}

func TestAuthenticate_ErrorWithoutCredentials(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c, _ := client.NewWithOptions(
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
	)
	err := c.Authenticate(context.Background(), "accounts:read")
	if err == nil {
		t.Fatal("Authenticate should fail without credentials")
	}
	var te *tinkErrors.TinkError
	if isTE := func() bool {
		if e, ok := err.(*tinkErrors.TinkError); ok {
			te = e
			return true
		}
		return false
	}(); !isTE {
		t.Fatalf("expected *tinkErrors.TinkError, got %T", err)
	}
	if te.Type != types.ErrorTypeValidation {
		t.Errorf("Type = %q, want validation_error", te.Type)
	}
}

// ── Accounts API ──────────────────────────────────────────────────────────

func TestAccounts_ListAccounts_Success(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"accounts": []map[string]interface{}{
			{"id": "acc_1", "name": "Checking", "type": "CHECKING"},
			{"id": "acc_2", "name": "Savings", "type": "SAVINGS"},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Accounts.ListAccounts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(resp.Accounts) != 2 {
		t.Errorf("got %d accounts, want 2", len(resp.Accounts))
	}
	if resp.Accounts[0].ID != "acc_1" {
		t.Errorf("accounts[0].ID = %q, want acc_1", resp.Accounts[0].ID)
	}
	if resp.Accounts[1].Type != "SAVINGS" {
		t.Errorf("accounts[1].Type = %q, want SAVINGS", resp.Accounts[1].Type)
	}
}

func TestAccounts_ListAccounts_WithFilter(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"accounts": []interface{}{}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	c.Accounts.ListAccounts(context.Background(), &types.AccountsListOptions{ //nolint
		TypeIn:            []string{"CHECKING"},
		PaginationOptions: types.PaginationOptions{PageSize: 50},
	})
	if !strings.Contains(capturedURL, "pageSize=50") {
		t.Errorf("URL missing pageSize: %s", capturedURL)
	}
}

func TestAccounts_GetAccount(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"id": "acc_123", "name": "Main Account", "type": "CHECKING",
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	acc, err := c.Accounts.GetAccount(context.Background(), "acc_123")
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acc.ID != "acc_123" {
		t.Errorf("ID = %q, want acc_123", acc.ID)
	}
}

func TestAccounts_GetBalances(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"booked":    map[string]interface{}{"amount": map[string]interface{}{"value": "1500.00", "currencyCode": "GBP"}},
		"available": map[string]interface{}{"amount": map[string]interface{}{"value": "1200.00", "currencyCode": "GBP"}},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	b, err := c.Accounts.GetBalances(context.Background(), "acc_1")
	if err != nil {
		t.Fatalf("GetBalances: %v", err)
	}
	if b.Booked == nil {
		t.Fatal("Booked balance should not be nil")
	}
	if b.Booked.Amount.Value != "1500.00" {
		t.Errorf("Booked.Amount.Value = %q, want 1500.00", b.Booked.Amount.Value)
	}
}

// ── Transactions API ──────────────────────────────────────────────────────

func TestTransactions_ListTransactions(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"transactions": []map[string]interface{}{
			{"id": "txn_1", "status": "BOOKED", "amount": map[string]interface{}{"value": "-15.00", "currencyCode": "GBP"}},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Transactions.ListTransactions(context.Background(), &types.TransactionsListOptions{
		BookedDateGte: "2024-01-01",
		BookedDateLte: "2024-12-31",
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if len(resp.Transactions) != 1 {
		t.Errorf("got %d transactions, want 1", len(resp.Transactions))
	}
	if resp.Transactions[0].ID != "txn_1" {
		t.Errorf("transactions[0].ID = %q, want txn_1", resp.Transactions[0].ID)
	}
}

func TestTransactions_QueryParamsForwarded(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"transactions": []interface{}{}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	c.Transactions.ListTransactions(context.Background(), &types.TransactionsListOptions{ //nolint
		BookedDateGte: "2024-01-01",
		BookedDateLte: "2024-03-31",
		StatusIn:      []string{"BOOKED"},
	})
	for _, expected := range []string{"bookedDateGte=2024-01-01", "bookedDateLte=2024-03-31", "statusIn=BOOKED"} {
		if !strings.Contains(capturedURL, expected) {
			t.Errorf("URL missing param %q: %s", expected, capturedURL)
		}
	}
}

// ── Providers API ─────────────────────────────────────────────────────────

func TestProviders_ListProviders(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"providers": []map[string]interface{}{
			{"name": "uk-ob-barclays", "displayName": "Barclays", "status": "ENABLED", "market": "GB"},
			{"name": "uk-ob-hsbc", "displayName": "HSBC", "status": "ENABLED", "market": "GB"},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Providers.ListProviders(context.Background(), &types.ProvidersListOptions{Market: "GB"})
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(resp.Providers) != 2 {
		t.Errorf("got %d providers, want 2", len(resp.Providers))
	}
	if resp.Providers[0].Name != "uk-ob-barclays" {
		t.Errorf("providers[0].Name = %q", resp.Providers[0].Name)
	}
}

func TestProviders_GetProvider(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"name": "uk-ob-barclays", "displayName": "Barclays", "status": "ENABLED", "market": "GB",
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	p, err := c.Providers.GetProvider(context.Background(), "uk-ob-barclays")
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if p.Name != "uk-ob-barclays" {
		t.Errorf("Name = %q, want uk-ob-barclays", p.Name)
	}
}

// ── Categories API ────────────────────────────────────────────────────────

func TestCategories_ListCategories(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"categories": []map[string]interface{}{
			{"id": "cat_1", "code": "EXPENSES:FOOD", "displayName": "Food"},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Categories.ListCategories(context.Background(), "en_US")
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(resp.Categories) != 1 {
		t.Errorf("got %d categories, want 1", len(resp.Categories))
	}
}

func TestCategories_DefaultLocale(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"categories": []interface{}{}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	c.Categories.ListCategories(context.Background(), "") //nolint
	if !strings.Contains(capturedURL, "locale=en_US") {
		t.Errorf("default locale not set: %s", capturedURL)
	}
}

// ── Statistics API ────────────────────────────────────────────────────────

func TestStatistics_GetStatistics(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"periods": []map[string]interface{}{
			{"period": "2024-01", "income": map[string]interface{}{"amount": map[string]interface{}{"value": "3000.00", "currencyCode": "GBP"}}},
			{"period": "2024-02", "income": map[string]interface{}{"amount": map[string]interface{}{"value": "3000.00", "currencyCode": "GBP"}}},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Statistics.GetStatistics(context.Background(), types.StatisticsOptions{
		PeriodGte: "2024-01-01", PeriodLte: "2024-03-31", Resolution: "MONTHLY",
	})
	if err != nil {
		t.Fatalf("GetStatistics: %v", err)
	}
	if len(resp.Periods) != 2 {
		t.Errorf("got %d periods, want 2", len(resp.Periods))
	}
}

func TestStatistics_DefaultResolution(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"periods": []interface{}{}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	c.Statistics.GetStatistics(context.Background(), types.StatisticsOptions{ //nolint
		PeriodGte: "2024-01-01", PeriodLte: "2024-12-31",
	})
	if !strings.Contains(capturedURL, "resolution=MONTHLY") {
		t.Errorf("default resolution not set: %s", capturedURL)
	}
}

// ── Users API ─────────────────────────────────────────────────────────────

func TestUsers_CreateUser(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"userId": "u_new_123", "externalUserId": "ext_001",
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	u, err := c.Users.CreateUser(context.Background(), types.CreateUserParams{
		ExternalUserID: "ext_001", Locale: "en_US", Market: "GB",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.UserID != "u_new_123" {
		t.Errorf("UserID = %q, want u_new_123", u.UserID)
	}
}

func TestUsers_ListCredentials(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"credentials": []map[string]interface{}{
			{"id": "cred_1", "providerName": "uk-ob-barclays", "status": "UPDATED"},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Users.ListCredentials(context.Background())
	if err != nil {
		t.Fatalf("ListCredentials: %v", err)
	}
	if len(resp.Credentials) != 1 {
		t.Errorf("got %d credentials, want 1", len(resp.Credentials))
	}
	if resp.Credentials[0].ProviderName != "uk-ob-barclays" {
		t.Errorf("ProviderName = %q", resp.Credentials[0].ProviderName)
	}
}

// ── Investments API ───────────────────────────────────────────────────────

func TestInvestments_ListAccounts(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"accounts": []map[string]interface{}{
			{"id": "inv_1", "name": "ISA", "type": "ISA"},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Investments.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(resp.Accounts) != 1 {
		t.Errorf("got %d accounts, want 1", len(resp.Accounts))
	}
}

func TestInvestments_GetHoldings(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"holdings": []map[string]interface{}{
			{"id": "h_1", "instrument": map[string]interface{}{"type": "STOCK", "symbol": "AAPL"}},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.Investments.GetHoldings(context.Background(), "inv_1")
	if err != nil {
		t.Fatalf("GetHoldings: %v", err)
	}
	if len(resp.Holdings) != 1 {
		t.Errorf("got %d holdings, want 1", len(resp.Holdings))
	}
}

// ── Balance Check API ─────────────────────────────────────────────────────

func TestBalanceCheck_RefreshBalance(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"balanceRefreshId": "refresh_001",
			"status":           "INITIATED",
		})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.BalanceCheck.RefreshBalance(context.Background(), "acc_123")
	if err != nil {
		t.Fatalf("RefreshBalance: %v", err)
	}
	if resp.BalanceRefreshID != "refresh_001" {
		t.Errorf("BalanceRefreshID = %q, want refresh_001", resp.BalanceRefreshID)
	}
	if capturedBody["accountId"] != "acc_123" {
		t.Errorf("body accountId = %v, want acc_123", capturedBody["accountId"])
	}
}

func TestBalanceCheck_GetRefreshStatus(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"balanceRefreshId": "refresh_001",
		"status":           "COMPLETED",
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	resp, err := c.BalanceCheck.GetRefreshStatus(context.Background(), "refresh_001")
	if err != nil {
		t.Fatalf("GetRefreshStatus: %v", err)
	}
	if resp.Status != types.BalanceRefreshCompleted {
		t.Errorf("Status = %q, want COMPLETED", resp.Status)
	}
}

func TestBalanceCheck_BuildAccountCheckLink(t *testing.T) {
	c := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	u := c.BalanceCheck.BuildAccountCheckLink("grant_code_xyz", types.BuildAccountCheckLinkOptions{
		ClientID:    "client_id",
		Market:      "SE",
		RedirectURI: "https://example.com/callback",
		Test:        false,
		State:       "csrf_abc",
	})
	if !strings.Contains(u, "link.tink.com/1.0/account-check/connect") {
		t.Errorf("link URL missing expected path: %s", u)
	}
	if !strings.Contains(u, "authorization_code=grant_code_xyz") {
		t.Errorf("link URL missing authorization_code: %s", u)
	}
	if !strings.Contains(u, "state=csrf_abc") {
		t.Errorf("link URL missing state: %s", u)
	}
	if !strings.Contains(u, "test=false") {
		t.Errorf("link URL missing test=false: %s", u)
	}
}

func TestBalanceCheck_BuildConsentUpdateLink(t *testing.T) {
	c := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	u := c.BalanceCheck.BuildConsentUpdateLink("grant_code", types.ConsentUpdateLinkOptions{
		ClientID:      "cid",
		CredentialsID: "cred_1",
		Market:        "SE",
		RedirectURI:   "https://example.com/callback",
	})
	if !strings.Contains(u, "update-consent") {
		t.Errorf("consent link missing update-consent: %s", u)
	}
	if !strings.Contains(u, "credentials_id=cred_1") {
		t.Errorf("consent link missing credentials_id: %s", u)
	}
}

// ── Error propagation ─────────────────────────────────────────────────────

func TestError_401_ReturnsAuthError(t *testing.T) {
	srv := mockServer(t, http.StatusUnauthorized, map[string]interface{}{
		"errorMessage": "Invalid token",
		"errorCode":    "TOKEN_INVALID",
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	_, err := c.Accounts.ListAccounts(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	te, ok := err.(*tinkErrors.TinkError)
	if !ok {
		t.Fatalf("expected *TinkError, got %T", err)
	}
	if te.Type != types.ErrorTypeAuthentication {
		t.Errorf("Type = %q, want authentication_error", te.Type)
	}
	if te.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", te.StatusCode)
	}
	if te.ErrorCode != "TOKEN_INVALID" {
		t.Errorf("ErrorCode = %q, want TOKEN_INVALID", te.ErrorCode)
	}
}

func TestError_429_ReturnsRateLimitError(t *testing.T) {
	srv := mockServer(t, http.StatusTooManyRequests, map[string]interface{}{"errorMessage": "Rate limit exceeded"})
	defer srv.Close()
	c := newTestClient(t, srv)

	_, err := c.Providers.ListProviders(context.Background(), nil)
	te, ok := err.(*tinkErrors.TinkError)
	if !ok {
		t.Fatalf("expected *TinkError, got %T", err)
	}
	if te.Type != types.ErrorTypeRateLimit {
		t.Errorf("Type = %q, want rate_limit_error", te.Type)
	}
	if !te.Retryable() {
		t.Error("429 should be retryable")
	}
}

func TestError_500_ReturnsAPIError(t *testing.T) {
	srv := mockServer(t, http.StatusInternalServerError, map[string]interface{}{"errorMessage": "Internal error"})
	defer srv.Close()
	c := newTestClient(t, srv)

	_, err := c.Accounts.ListAccounts(context.Background(), nil)
	te, ok := err.(*tinkErrors.TinkError)
	if !ok {
		t.Fatalf("expected *TinkError, got %T", err)
	}
	if te.Type != types.ErrorTypeAPI {
		t.Errorf("Type = %q, want api_error", te.Type)
	}
	if !te.Retryable() {
		t.Error("500 should be retryable")
	}
}

func TestError_400_ValidationError(t *testing.T) {
	srv := mockServer(t, http.StatusBadRequest, map[string]interface{}{"errorMessage": "Invalid params"})
	defer srv.Close()
	c := newTestClient(t, srv)

	_, err := c.Accounts.ListAccounts(context.Background(), nil)
	te, ok := err.(*tinkErrors.TinkError)
	if !ok {
		t.Fatalf("expected *TinkError, got %T", err)
	}
	if te.Retryable() {
		t.Error("400 should NOT be retryable")
	}
}

// ── Authorization header ──────────────────────────────────────────────────

func TestAuthorizationHeader(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"accounts": []interface{}{}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv) // sets token = "test-bearer-token"

	c.Accounts.ListAccounts(context.Background(), nil) //nolint
	if capturedAuth != "Bearer test-bearer-token" {
		t.Errorf("Authorization = %q, want 'Bearer test-bearer-token'", capturedAuth)
	}
}

func TestUserAgentHeader(t *testing.T) {
	var capturedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"accounts": []interface{}{}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	c.Accounts.ListAccounts(context.Background(), nil) //nolint
	if !strings.HasPrefix(capturedUA, "tink-client-go/") {
		t.Errorf("User-Agent = %q, want prefix 'tink-client-go/'", capturedUA)
	}
}

func TestCustomHeader(t *testing.T) {
	var capturedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("X-Request-ID")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"accounts": []interface{}{}})
	}))
	defer srv.Close()
	c, _ := client.NewWithOptions(
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
		client.WithDisableCache(),
		client.WithMaxRetries(1),
		client.WithAccessToken("tok"),
		client.WithHeader("X-Request-ID", "req-123-abc"),
	)
	c.Accounts.ListAccounts(context.Background(), nil) //nolint
	if capturedHeader != "req-123-abc" {
		t.Errorf("X-Request-ID = %q, want req-123-abc", capturedHeader)
	}
}

// ── Cache management ──────────────────────────────────────────────────────

func TestClearCache_DoesNotPanic(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{"accounts": []interface{}{}})
	defer srv.Close()
	c, _ := client.NewWithOptions(
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
		client.WithAccessToken("tok"),
	)
	c.ClearCache() // should not panic
}

func TestInvalidateCache_DoesNotPanic(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c := newTestClient(t, srv)
	c.InvalidateCache("/api/v1/providers") // should not panic
}

func TestCaching_SecondRequestHitsCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"providers": []interface{}{}})
	}))
	defer srv.Close()

	// Enable cache for this test
	c, _ := client.NewWithOptions(
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
		client.WithAccessToken("tok"),
		client.WithMaxRetries(1),
		// No WithDisableCache — cache is ON
	)
	ctx := context.Background()
	c.Providers.ListProviders(ctx, nil) //nolint
	c.Providers.ListProviders(ctx, nil) //nolint
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call (2nd hit cache), got %d", callCount)
	}
}

// ── Webhook factories ─────────────────────────────────────────────────────

func TestNewWebhookHandler_NotNil(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c := newTestClient(t, srv)
	wh := c.NewWebhookHandler("secret123")
	if wh == nil {
		t.Error("NewWebhookHandler should not return nil")
	}
}

func TestNewWebhookVerifier_NotNil(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c := newTestClient(t, srv)
	v := c.NewWebhookVerifier("secret123")
	if v == nil {
		t.Error("NewWebhookVerifier should not return nil")
	}
}

// ── Token helpers ─────────────────────────────────────────────────────────

func TestParseToken(t *testing.T) {
	tok := &types.TokenResponse{
		AccessToken:  "tok123",
		ExpiresIn:    3600,
		Scope:        "accounts:read",
		RefreshToken: "refresh_xyz",
	}
	info := client.ParseToken(tok)
	if info.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", info.ExpiresIn)
	}
	if info.Scope != "accounts:read" {
		t.Errorf("Scope = %q, want accounts:read", info.Scope)
	}
	if info.RefreshToken != "refresh_xyz" {
		t.Errorf("RefreshToken = %q", info.RefreshToken)
	}
	// ExpiresAt should be roughly 1 hour in the future
	if time.Until(info.ExpiresAt) < 59*time.Minute {
		t.Errorf("ExpiresAt should be ~1 hour from now, got %v", time.Until(info.ExpiresAt))
	}
}

func TestIsExpired_Fresh(t *testing.T) {
	expiresAt := time.Now().Add(1 * time.Hour)
	if client.IsExpired(expiresAt) {
		t.Error("a token expiring in 1 hour should not be considered expired")
	}
}

func TestIsExpired_ExpiredWithBuffer(t *testing.T) {
	// Token expiring in 2 minutes is within the 5-minute buffer → expired
	expiresAt := time.Now().Add(2 * time.Minute)
	if !client.IsExpired(expiresAt) {
		t.Error("token expiring in 2 minutes should be considered expired (within 5min buffer)")
	}
}

func TestIsExpired_AlreadyExpired(t *testing.T) {
	expiresAt := time.Now().Add(-10 * time.Minute)
	if !client.IsExpired(expiresAt) {
		t.Error("token that expired 10 minutes ago should be considered expired")
	}
}

// ── Connectivity ──────────────────────────────────────────────────────────

func TestConnectivity_CheckAPIHealth_Healthy(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{"providers": []interface{}{}})
	defer srv.Close()
	c := newTestClient(t, srv)

	if err := c.Connectivity.CheckAPIHealth(context.Background()); err != nil {
		t.Errorf("CheckAPIHealth: %v", err)
	}
}

func TestConnectivity_CheckAPIHealth_Unhealthy(t *testing.T) {
	srv := mockServer(t, http.StatusServiceUnavailable, map[string]interface{}{"errorMessage": "Service down"})
	defer srv.Close()
	c, _ := client.NewWithOptions(
		client.WithBaseURL(srv.URL),
		client.WithHTTPClient(srv.Client()),
		client.WithDisableCache(),
		client.WithMaxRetries(1), // only 1 retry for test speed
		client.WithAccessToken("tok"),
	)
	if err := c.Connectivity.CheckAPIHealth(context.Background()); err == nil {
		t.Error("CheckAPIHealth should fail when API returns 503")
	}
}

func TestConnectivity_CheckProviderStatus(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"name": "uk-ob-barclays", "status": "ENABLED", "market": "GB",
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	result, err := c.Connectivity.CheckProviderStatus(context.Background(), "uk-ob-barclays", "GB")
	if err != nil {
		t.Fatalf("CheckProviderStatus: %v", err)
	}
	if !result.Active {
		t.Error("provider should be active")
	}
}

func TestConnectivity_CheckProviderStatus_MarketMismatch(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"name": "se-ob-swedbank", "status": "ENABLED", "market": "SE",
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	result, _ := c.Connectivity.CheckProviderStatus(context.Background(), "se-ob-swedbank", "GB")
	if result.Active {
		t.Error("should not be active when market doesn't match")
	}
}

func TestConnectivity_CheckCredentialConnectivity(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{
		"credentials": []map[string]interface{}{
			{"id": "c1", "providerName": "bank1", "status": "UPDATED"},
			{"id": "c2", "providerName": "bank2", "status": "SESSION_EXPIRED", "statusPayload": "Reconnect needed"},
		},
	})
	defer srv.Close()
	c := newTestClient(t, srv)

	summary, err := c.Connectivity.CheckCredentialConnectivity(context.Background(), nil)
	if err != nil {
		t.Fatalf("CheckCredentialConnectivity: %v", err)
	}
	if summary.Total != 2 {
		t.Errorf("Total = %d, want 2", summary.Total)
	}
	if summary.Healthy != 1 {
		t.Errorf("Healthy = %d, want 1", summary.Healthy)
	}
	if summary.Unhealthy != 1 {
		t.Errorf("Unhealthy = %d, want 1", summary.Unhealthy)
	}
	if summary.Credentials[1].ErrorMessage != "Reconnect needed" {
		t.Errorf("ErrorMessage = %q, want 'Reconnect needed'", summary.Credentials[1].ErrorMessage)
	}
}

// ── Reports ───────────────────────────────────────────────────────────────

func TestIncomeCheck_GetReport(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{"id": "rep_1", "created": "2024-01-01"})
	defer srv.Close()
	c := newTestClient(t, srv)

	rep, err := c.IncomeCheck.GetReport(context.Background(), "rep_1")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if rep.ID != "rep_1" {
		t.Errorf("ID = %q, want rep_1", rep.ID)
	}
}

func TestExpenseCheck_GetReport(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{"id": "rep_2"})
	defer srv.Close()
	c := newTestClient(t, srv)

	rep, err := c.ExpenseCheck.GetReport(context.Background(), "rep_2")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if rep.ID != "rep_2" {
		t.Errorf("ID = %q, want rep_2", rep.ID)
	}
}

func TestBusinessAccountCheck_GetReport(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]interface{}{"id": "biz_rep_1", "status": "COMPLETED"})
	defer srv.Close()
	c := newTestClient(t, srv)

	rep, err := c.BusinessAccountCheck.GetReport(context.Background(), "biz_rep_1")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if rep.Status != "COMPLETED" {
		t.Errorf("Status = %q, want COMPLETED", rep.Status)
	}
}

// ── Account Check ─────────────────────────────────────────────────────────

func TestAccountCheck_BuildLinkURL(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c := newTestClient(t, srv)

	session := &types.AccountCheckSession{SessionID: "sess_abc123"}
	u := c.AccountCheck.BuildLinkURL(session, accountcheck.BuildLinkURLOptions{
		ClientID: "cid",
		Market:   "GB",
	})
	if !strings.Contains(u, "account-check?") {
		t.Errorf("link URL missing account-check: %s", u)
	}
	if !strings.Contains(u, "session_id=sess_abc123") {
		t.Errorf("link URL missing session_id: %s", u)
	}
}

func TestAccountCheck_BuildContinuousAccessLink(t *testing.T) {
	srv := mockServer(t, http.StatusOK, nil)
	defer srv.Close()
	c := newTestClient(t, srv)

	u := c.AccountCheck.BuildContinuousAccessLink("grant_code_abc", types.ContinuousAccessLinkOptions{
		ClientID:    "cid",
		Market:      "GB",
		Locale:      "en_US",
		RedirectURI: "https://example.com/callback",
	})
	if !strings.Contains(u, "products/connect-accounts") {
		t.Errorf("link URL missing products/connect-accounts: %s", u)
	}
	if !strings.Contains(u, "authorization_code=grant_code_abc") {
		t.Errorf("link URL missing authorization_code: %s", u)
	}
	if !strings.Contains(u, "ACCOUNT_CHECK") {
		t.Errorf("link URL missing default products: %s", u)
	}
}
