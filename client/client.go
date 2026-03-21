// Package client provides the main TinkClient entry point.
//
// Create a single TinkClient per application and reuse it across goroutines.
// It is fully safe for concurrent use.
//
// Quick start:
//
//	client, err := client.New(client.Config{
//	    ClientID:     os.Getenv("TINK_CLIENT_ID"),
//	    ClientSecret: os.Getenv("TINK_CLIENT_SECRET"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Acquire a client credentials token (sets it automatically)
//	if err := client.Authenticate(context.Background(), "accounts:read transactions:read"); err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := client.Accounts.ListAccounts(ctx, nil)
package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/iamkanishka/tink-client-go/accountcheck"
	"github.com/iamkanishka/tink-client-go/accounts"
	"github.com/iamkanishka/tink-client-go/auth"
	"github.com/iamkanishka/tink-client-go/balancecheck"
	"github.com/iamkanishka/tink-client-go/budgets"
	"github.com/iamkanishka/tink-client-go/calendar"
	"github.com/iamkanishka/tink-client-go/cashflow"
	"github.com/iamkanishka/tink-client-go/categories"
	"github.com/iamkanishka/tink-client-go/connectivity"
	"github.com/iamkanishka/tink-client-go/connector"
	tinkErrors "github.com/iamkanishka/tink-client-go/errors"
	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/investments"
	"github.com/iamkanishka/tink-client-go/link"
	"github.com/iamkanishka/tink-client-go/loans"
	"github.com/iamkanishka/tink-client-go/providers"
	"github.com/iamkanishka/tink-client-go/reports"
	"github.com/iamkanishka/tink-client-go/statistics"
	"github.com/iamkanishka/tink-client-go/transactions"
	"github.com/iamkanishka/tink-client-go/types"
	"github.com/iamkanishka/tink-client-go/users"
	"github.com/iamkanishka/tink-client-go/webhooks"
)

const (
	version        = "1.0.0"
	defaultBaseURL = "https://api.tink.com"
)

// Config holds all options for constructing a Client.
// Credentials are read from environment variables when the corresponding field is empty:
//   - ClientID     → TINK_CLIENT_ID
//   - ClientSecret → TINK_CLIENT_SECRET
//   - AccessToken  → TINK_ACCESS_TOKEN
//   - BaseURL      → TINK_BASE_URL
type Config struct {
	// DefaultHeaders are extra HTTP headers sent on every request.
	DefaultHeaders map[string]string
	// HTTPClient overrides the default net/http.Client (e.g. for testing).
	HTTPClient *http.Client
	// ClientID is your Tink application client ID.
	ClientID string
	// ClientSecret is your Tink application client secret.
	ClientSecret string
	// AccessToken is a pre-existing bearer token (skips client credentials flow).
	AccessToken string
	// UserID is used to scope cache invalidation per Tink user.
	UserID string
	// BaseURL overrides https://api.tink.com.
	BaseURL string
	// Timeout is the per-request timeout. Defaults to 30 seconds.
	Timeout time.Duration
	// MaxRetries is the maximum number of retry attempts. Defaults to 3.
	MaxRetries int
	// CacheMaxSize is the maximum number of LRU cache entries. Defaults to 512.
	CacheMaxSize int
	// DisableCache disables the in-memory LRU response cache.
	DisableCache bool
}

// Client is the main Tink Open Banking API client.
// All fields are exported services; create via New().
//
// It is safe for concurrent use by multiple goroutines.
type Client struct {
	// ── Authentication ─────────────────────────────────────────────────────
	// Auth provides OAuth 2.0 flows: client credentials, code exchange, refresh.
	Auth *auth.Service

	// ── Account aggregation ─────────────────────────────────────────────────
	// Accounts provides bank account data and real-time balances.
	Accounts *accounts.Service
	// Transactions provides standard transaction listing.
	Transactions *transactions.Service
	// TransactionsOneTimeAccess provides single-authorization transaction access.
	TransactionsOneTimeAccess *transactions.OneTimeAccessService
	// TransactionsContinuousAccess provides persistent-user transaction sync.
	TransactionsContinuousAccess *transactions.ContinuousAccessService

	// ── Reference data ──────────────────────────────────────────────────────
	// Providers lists financial institutions supported by Tink (cached 1 hour).
	Providers *providers.Service
	// Categories provides transaction categories (cached 24 hours).
	Categories *categories.Service
	// Statistics returns aggregated financial statistics (cached 1 hour).
	Statistics *statistics.Service

	// ── Users & credentials ──────────────────────────────────────────────────
	// Users manages Tink users and their bank connections.
	Users *users.Service

	// ── Assets ──────────────────────────────────────────────────────────────
	// Investments provides investment accounts and holdings.
	Investments *investments.Service
	// Loans provides loan and mortgage account data.
	Loans *loans.Service

	// ── Finance management ───────────────────────────────────────────────────
	// Budgets provides budget creation, tracking, and history.
	Budgets *budgets.Service
	// CashFlow provides income vs expense summaries by time resolution.
	CashFlow *cashflow.Service
	// FinancialCalendar provides calendar events, attachments, and reconciliation.
	FinancialCalendar *calendar.Service

	// ── Verification ────────────────────────────────────────────────────────
	// AccountCheck verifies bank account ownership (one-time + continuous).
	AccountCheck *accountcheck.Service
	// BalanceCheck provides real-time balance refresh and consent management.
	BalanceCheck *balancecheck.Service
	// BusinessAccountCheck verifies business account ownership.
	BusinessAccountCheck *reports.BusinessAccountCheckService

	// ── Risk & analytics ────────────────────────────────────────────────────
	// IncomeCheck provides income verification reports.
	IncomeCheck *reports.IncomeCheckService
	// ExpenseCheck provides expense analysis reports.
	ExpenseCheck *reports.ExpenseCheckService
	// RiskInsights provides financial risk scoring reports.
	RiskInsights *reports.RiskInsightsService
	// RiskCategorisation provides transaction-level risk categorisation reports.
	RiskCategorisation *reports.RiskCategorisationService

	// ── Infrastructure ───────────────────────────────────────────────────────
	// Connector ingests your own account and transaction data into Tink.
	Connector *connector.Service
	// Link builds Tink Link URLs for all 6 products.
	Link *link.Service
	// Connectivity monitors provider and credential health.
	Connectivity *connectivity.Service

	// ── internals ────────────────────────────────────────────────────────────
	http         *httpclient.HTTPClient
	clientID     string
	clientSecret string
	baseURL      string
}

// New constructs a Client from the provided Config.
// Missing Config fields are filled from environment variables.
//
// Returns an error if neither ClientID nor TINK_CLIENT_ID is set AND no
// AccessToken is provided (because authentication will be impossible).
func New(cfg Config) (*Client, error) {
	// Resolve from env vars
	clientID := cfg.ClientID
	if clientID == "" {
		clientID = os.Getenv("TINK_CLIENT_ID")
	}
	clientSecret := cfg.ClientSecret
	if clientSecret == "" {
		clientSecret = os.Getenv("TINK_CLIENT_SECRET")
	}
	accessToken := cfg.AccessToken
	if accessToken == "" {
		accessToken = os.Getenv("TINK_ACCESS_TOKEN")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("TINK_BASE_URL")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}
	cacheMaxSize := cfg.CacheMaxSize
	if cacheMaxSize == 0 {
		cacheMaxSize = 512
	}

	headers := make(map[string]string, len(cfg.DefaultHeaders)+1)
	for k, v := range cfg.DefaultHeaders {
		headers[k] = v
	}
	headers["User-Agent"] = fmt.Sprintf("tink-client-go/%s", version)

	h := httpclient.New(httpclient.Config{
		BaseURL:        baseURL,
		Timeout:        timeout,
		MaxRetries:     maxRetries,
		CacheEnabled:   !cfg.DisableCache,
		CacheMaxSize:   cacheMaxSize,
		AccessToken:    accessToken,
		UserID:         cfg.UserID,
		DefaultHeaders: headers,
		HTTPClient:     cfg.HTTPClient,
	})

	c := &Client{
		http:         h,
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
	}

	// ── Wire up all 24 service namespaces ─────────────────────────────────
	c.Auth = auth.New(h, baseURL)
	c.Accounts = accounts.New(h)
	c.Transactions = transactions.New(h)
	c.TransactionsOneTimeAccess = transactions.NewOneTimeAccess(h)
	c.TransactionsContinuousAccess = transactions.NewContinuousAccess(h, clientID)
	c.Providers = providers.New(h)
	c.Categories = categories.New(h)
	c.Statistics = statistics.New(h)
	c.Users = users.New(h)
	c.Investments = investments.New(h)
	c.Loans = loans.New(h)
	c.Budgets = budgets.New(h)
	c.CashFlow = cashflow.New(h)
	c.FinancialCalendar = calendar.New(h)
	c.AccountCheck = accountcheck.New(h)
	c.BalanceCheck = balancecheck.New(h)
	c.IncomeCheck = reports.NewIncomeCheck(h)
	c.ExpenseCheck = reports.NewExpenseCheck(h)
	c.RiskInsights = reports.NewRiskInsights(h)
	c.RiskCategorisation = reports.NewRiskCategorisation(h)
	c.BusinessAccountCheck = reports.NewBusinessAccountCheck(h)
	c.Connector = connector.New(h)
	c.Link = link.New()
	c.Connectivity = connectivity.New(h)

	return c, nil
}

// ── Token management ──────────────────────────────────────────────────────

// AccessToken returns the current bearer token.
func (c *Client) AccessToken() string { return c.http.AccessToken() }

// SetAccessToken atomically updates the bearer token for all subsequent requests.
func (c *Client) SetAccessToken(token string) { c.http.SetAccessToken(token) }

// ClientID returns the configured Tink application client ID.
func (c *Client) ClientID() string { return c.clientID }

// Authenticate acquires a client credentials token for the given scope and
// sets it on the client automatically. Returns the full TokenResponse.
//
// Example:
//
//	if err := client.Authenticate(ctx, "accounts:read transactions:read"); err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) Authenticate(ctx context.Context, scope string) error {
	if c.clientID == "" || c.clientSecret == "" {
		return tinkErrors.Validation(
			"clientID and clientSecret are required. " +
				"Set them in Config or via TINK_CLIENT_ID / TINK_CLIENT_SECRET environment variables.",
		)
	}
	token, err := c.Auth.GetAccessToken(ctx, c.clientID, c.clientSecret, scope)
	if err != nil {
		return err
	}
	c.http.SetAccessToken(token.AccessToken)
	return nil
}

// ── Cache management ──────────────────────────────────────────────────────

// ClearCache removes all cached API responses.
func (c *Client) ClearCache() { c.http.InvalidateCache() }

// InvalidateCache removes cached entries whose URL path starts with prefix.
// Useful for selectively refreshing specific resource types:
//
//	client.InvalidateCache("/api/v1/providers")
func (c *Client) InvalidateCache(prefix string) { c.http.InvalidateCache(prefix) }

// ── Webhook factories ─────────────────────────────────────────────────────

// NewWebhookHandler creates a Handler for receiving and dispatching Tink webhooks.
// secret is your webhook signing secret from the Tink console.
//
// Example:
//
//	wh := client.NewWebhookHandler(os.Getenv("TINK_WEBHOOK_SECRET"))
//	wh.On(types.WebhookEventCredentialsUpdated, func(ctx context.Context, e *types.WebhookEvent) error {
//	    log.Printf("user %s: credentials updated", e.Data["userId"])
//	    return nil
//	})
func (c *Client) NewWebhookHandler(secret string) *webhooks.Handler {
	return webhooks.NewHandler(secret)
}

// NewWebhookVerifier creates a standalone Verifier for manual HMAC-SHA256 checking.
func (c *Client) NewWebhookVerifier(secret string) *webhooks.Verifier {
	return webhooks.NewVerifier(secret)
}

// ── Meta ──────────────────────────────────────────────────────────────────

// Info returns metadata about this client instance.
func (c *Client) Info() Info {
	return Info{
		Version:  version,
		BaseURL:  c.baseURL,
		HasToken: c.http.AccessToken() != "",
	}
}

// Info holds metadata returned by Client.Info().
type Info struct {
	Version  string
	BaseURL  string
	HasToken bool
}

// ── Functional options (alternative constructor) ──────────────────────────

// Option is a functional option for configuring a Client.
type Option func(*Config)

// WithCredentials sets the client ID and secret.
func WithCredentials(clientID, clientSecret string) Option {
	return func(cfg *Config) {
		cfg.ClientID = clientID
		cfg.ClientSecret = clientSecret
	}
}

// WithAccessToken sets a pre-existing access token.
func WithAccessToken(token string) Option {
	return func(cfg *Config) { cfg.AccessToken = token }
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(u string) Option {
	return func(cfg *Config) { cfg.BaseURL = u }
}

// WithTimeout sets the per-request timeout.
func WithTimeout(d time.Duration) Option {
	return func(cfg *Config) { cfg.Timeout = d }
}

// WithMaxRetries sets the maximum retry attempts.
func WithMaxRetries(n int) Option {
	return func(cfg *Config) { cfg.MaxRetries = n }
}

// WithHTTPClient overrides the underlying net/http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(cfg *Config) { cfg.HTTPClient = hc }
}

// WithHeader adds a default HTTP header to every request.
func WithHeader(key, value string) Option {
	return func(cfg *Config) {
		if cfg.DefaultHeaders == nil {
			cfg.DefaultHeaders = make(map[string]string)
		}
		cfg.DefaultHeaders[key] = value
	}
}

// WithDisableCache disables the in-memory LRU response cache.
func WithDisableCache() Option {
	return func(cfg *Config) { cfg.DisableCache = true }
}

// NewWithOptions constructs a Client from functional options.
//
// Example:
//
//	client, err := client.NewWithOptions(
//	    client.WithCredentials(clientID, clientSecret),
//	    client.WithTimeout(10 * time.Second),
//	    client.WithHeader("X-Request-ID", requestID),
//	)
func NewWithOptions(opts ...Option) (*Client, error) {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}
	return New(cfg)
}

// ── TokenExpiryHelper is a convenience for managing access tokens ─────────

// TokenInfo holds parsed token expiry information.
type TokenInfo struct {
	ExpiresAt    time.Time
	Scope        string
	RefreshToken string
	ExpiresIn    int
}

// ParseToken extracts expiry information from a TokenResponse.
func ParseToken(t *types.TokenResponse) TokenInfo {
	return TokenInfo{
		ExpiresIn:    t.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(t.ExpiresIn) * time.Second),
		Scope:        t.Scope,
		RefreshToken: t.RefreshToken,
	}
}

// IsExpired reports whether a token has expired (with a 5-minute buffer).
func IsExpired(expiresAt time.Time) bool {
	return time.Now().Add(5 * time.Minute).After(expiresAt)
}
