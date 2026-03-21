// Package types defines all domain types for the tink-client-go client.
package types

import "time"

// ── Config ────────────────────────────────────────────────────────────────

// Config holds all options for constructing a Client.
// Fields are ordered for minimal struct padding: maps and pointers first,
// strings next, then numeric types, booleans last.
type Config struct {
	DefaultHeaders map[string]string
	ClientID       string
	ClientSecret   string
	AccessToken    string
	UserID         string
	BaseURL        string
	Timeout        time.Duration
	MaxRetries     int
	CacheMaxSize   int
	DisableCache   bool
}

// ── Error ─────────────────────────────────────────────────────────────────

// ErrorType classifies a TinkError.
type ErrorType string

const (
	ErrorTypeAPI            ErrorType = "api_error"
	ErrorTypeAuthentication ErrorType = "authentication_error"
	ErrorTypeRateLimit      ErrorType = "rate_limit_error"
	ErrorTypeValidation     ErrorType = "validation_error"
	ErrorTypeNetwork        ErrorType = "network_error"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeDecode         ErrorType = "decode_error"
	ErrorTypeMarketMismatch ErrorType = "market_mismatch"
	ErrorTypeUnknown        ErrorType = "unknown"
)

// ── Auth ──────────────────────────────────────────────────────────────────

// TokenResponse is the OAuth 2.0 token response from the Tink API.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

// AuthorizationURLOptions holds parameters for building a Tink OAuth URL.
type AuthorizationURLOptions struct {
	ClientID    string
	RedirectURI string
	Scope       string
	State       string
	Market      string
	Locale      string
}

// CreateAuthorizationParams holds parameters for creating an authorization grant.
type CreateAuthorizationParams struct {
	UserID string `json:"user_id"`
	Scope  string `json:"scope"`
}

// DelegateAuthorizationParams holds parameters for delegating a grant.
type DelegateAuthorizationParams struct {
	UserID        string `json:"user_id"`
	IDHint        string `json:"id_hint"`
	Scope         string `json:"scope"`
	ActorClientID string `json:"actor_client_id,omitempty"`
}

// AuthorizationCode wraps a short-lived authorization code.
type AuthorizationCode struct {
	Code string `json:"code"`
}

// ── Primitives ────────────────────────────────────────────────────────────

// Amount is a monetary value with an ISO 4217 currency code.
type Amount struct {
	Value        string `json:"value"`
	CurrencyCode string `json:"currencyCode"`
}

// ExactAmount represents a decimal as unscaledValue / 10^scale.
type ExactAmount struct {
	UnscaledValue int64 `json:"unscaledValue"`
	Scale         int   `json:"scale"`
}

// TargetAmount combines ExactAmount with a currency code.
// currencyCode (16B string) after Value (ExactAmount = 16B) for tightest packing.
type TargetAmount struct {
	CurrencyCode string      `json:"currencyCode"`
	Value        ExactAmount `json:"value"`
}

// PaginationOptions holds common pagination parameters.
type PaginationOptions struct {
	PageToken string `json:"pageToken,omitempty"`
	PageSize  int    `json:"pageSize,omitempty"`
}

// ── Accounts ──────────────────────────────────────────────────────────────

// AccountBalanceItem holds a single balance entry.
type AccountBalanceItem struct {
	Amount Amount `json:"amount"`
}

// AccountBalances holds all balance types for a bank account.
type AccountBalances struct {
	Booked      *AccountBalanceItem `json:"booked,omitempty"`
	Available   *AccountBalanceItem `json:"available,omitempty"`
	Reserved    *AccountBalanceItem `json:"reserved,omitempty"`
	CreditLimit *AccountBalanceItem `json:"creditLimit,omitempty"`
}

// AccountIdentifiers holds IBAN, sort code, and PAN identifiers.
type AccountIdentifiers struct {
	IBAN *struct {
		IBAN string `json:"iban"`
		BBAN string `json:"bban,omitempty"`
	} `json:"iban,omitempty"`
	SortCode *struct {
		Code          string `json:"code"`
		AccountNumber string `json:"accountNumber"`
	} `json:"sortCode,omitempty"`
	PAN *struct {
		Masked string `json:"masked"`
	} `json:"pan,omitempty"`
}

// FinancialInstitution identifies a bank.
type FinancialInstitution struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Account represents a bank account.
// Pointer fields lead to minimise padding.
type Account struct {
	Balances             *AccountBalances      `json:"balances,omitempty"`
	Identifiers          *AccountIdentifiers   `json:"identifiers,omitempty"`
	FinancialInstitution *FinancialInstitution `json:"financialInstitution,omitempty"`
	Flags                []string              `json:"flags,omitempty"`
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	Type                 string                `json:"type"`
	Subtype              string                `json:"subtype,omitempty"`
	Currency             string                `json:"currency,omitempty"`
	ProviderName         string                `json:"providerName,omitempty"`
	Ownership            string                `json:"ownership,omitempty"`
	CredentialsID        string                `json:"credentialsId,omitempty"`
}

// AccountsResponse is a paginated list of accounts.
type AccountsResponse struct {
	Accounts      []Account `json:"accounts"`
	NextPageToken string    `json:"nextPageToken,omitempty"`
}

// AccountsListOptions are filter options for listing accounts.
type AccountsListOptions struct {
	PaginationOptions
	TypeIn []string
}

// ── Transactions ──────────────────────────────────────────────────────────

// transactionDescriptions holds display text.
type transactionDescriptions struct {
	Original string `json:"original,omitempty"`
	Display  string `json:"display,omitempty"`
}

// transactionDates holds the booked and value dates.
type transactionDates struct {
	Booked string `json:"booked,omitempty"`
	Value  string `json:"value,omitempty"`
}

// transactionMerchant holds merchant name and category.
type transactionMerchant struct {
	MerchantName         string `json:"merchantName,omitempty"`
	MerchantCategoryCode string `json:"merchantCategoryCode,omitempty"`
}

// TransactionPFM is the personal finance management category.
type TransactionPFM struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// transactionCategories holds categorisation data.
type transactionCategories struct {
	PFM *TransactionPFM `json:"pfm,omitempty"`
}

// Transaction is a single financial transaction.
// Pointer fields lead to minimise padding.
type Transaction struct {
	Descriptions        *transactionDescriptions `json:"descriptions,omitempty"`
	Dates               *transactionDates        `json:"dates,omitempty"`
	MerchantInformation *transactionMerchant     `json:"merchantInformation,omitempty"`
	Categories          *transactionCategories   `json:"categories,omitempty"`
	Amount              Amount                   `json:"amount"`
	ID                  string                   `json:"id"`
	AccountID           string                   `json:"accountId,omitempty"`
	Status              string                   `json:"status"`
}

// TransactionsResponse is a paginated list of transactions.
type TransactionsResponse struct {
	Transactions  []Transaction `json:"transactions"`
	NextPageToken string        `json:"nextPageToken,omitempty"`
}

// TransactionsListOptions are filter options for listing transactions.
type TransactionsListOptions struct {
	PaginationOptions
	AccountIDIn   []string
	BookedDateGte string
	BookedDateLte string
	StatusIn      []string
	CategoryIDIn  []string
}

// ── Providers ─────────────────────────────────────────────────────────────

// Provider is a financial institution supported by Tink.
type Provider struct {
	Capabilities []string `json:"capabilities,omitempty"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	Type         string   `json:"type,omitempty"`
	Status       string   `json:"status,omitempty"`
	Market       string   `json:"market"`
}

// ProvidersResponse holds a list of providers.
type ProvidersResponse struct {
	Providers []Provider `json:"providers"`
}

// ProvidersListOptions are filter options for listing providers.
type ProvidersListOptions struct {
	Capabilities []string
	Market       string
}

// ── Categories ────────────────────────────────────────────────────────────

// Category is a transaction category.
type Category struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DisplayName string `json:"displayName,omitempty"`
}

// CategoriesResponse holds a list of categories.
type CategoriesResponse struct {
	Categories []Category `json:"categories"`
}

// ── Statistics ────────────────────────────────────────────────────────────

// statisticsAmount wraps a single Amount.
type statisticsAmount struct {
	Amount Amount `json:"amount"`
}

// StatisticsPeriod holds financial statistics for one period.
// Pointer fields lead for alignment.
type StatisticsPeriod struct {
	Income   *statisticsAmount `json:"income,omitempty"`
	Expenses *statisticsAmount `json:"expenses,omitempty"`
	Period   string            `json:"period"`
}

// StatisticsResponse is the statistics API response.
type StatisticsResponse struct {
	Periods []StatisticsPeriod `json:"periods"`
}

// StatisticsOptions are options for requesting financial statistics.
type StatisticsOptions struct {
	AccountIDIn  []string
	CategoryIDIn []string
	PeriodGte    string
	PeriodLte    string
	Resolution   string
}

// ── Users ─────────────────────────────────────────────────────────────────

// CreateUserParams are parameters for creating a new Tink user.
type CreateUserParams struct {
	ExternalUserID string `json:"external_user_id"`
	Locale         string `json:"locale"`
	Market         string `json:"market"`
}

// TinkUser represents a Tink user.
type TinkUser struct {
	UserID         string `json:"userId,omitempty"`
	UserIDSnake    string `json:"user_id,omitempty"` // snake_case variant returned by some endpoints
	ExternalUserID string `json:"externalUserId,omitempty"`
}

// Credential is a Tink bank connection.
type Credential struct {
	ID            string `json:"id"`
	ProviderName  string `json:"providerName"`
	Status        string `json:"status,omitempty"`
	StatusUpdated string `json:"statusUpdated,omitempty"`
	StatusPayload string `json:"statusPayload,omitempty"`
}

// CredentialsResponse holds a list of credentials.
type CredentialsResponse struct {
	Credentials []Credential `json:"credentials"`
}

// ── Investments ───────────────────────────────────────────────────────────

// investmentBalance wraps an Amount for investment balance fields.
type investmentBalance struct {
	Amount Amount `json:"amount"`
}

// InvestmentAccount is an investment account.
// Pointer fields lead for alignment.
type InvestmentAccount struct {
	Balance *investmentBalance `json:"balance,omitempty"`
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Type    string             `json:"type"`
}

// InvestmentAccountsResponse is a paginated list of investment accounts.
type InvestmentAccountsResponse struct {
	Accounts      []InvestmentAccount `json:"accounts"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
}

// HoldingValue wraps an Amount for holding price/value fields.
type HoldingValue struct {
	Amount Amount `json:"amount"`
}

// holdingInstrument identifies the financial instrument.
type holdingInstrument struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol,omitempty"`
	ISIN   string `json:"isin,omitempty"`
}

// Holding is a single position within an investment account.
// Pointer fields lead for alignment.
type Holding struct {
	Instrument  *holdingInstrument `json:"instrument,omitempty"`
	Quantity    *float64           `json:"quantity,omitempty"`
	MarketValue *HoldingValue      `json:"marketValue,omitempty"`
	ID          string             `json:"id"`
}

// HoldingsResponse holds a list of holdings.
// Pointer fields lead for alignment.
type HoldingsResponse struct {
	TotalValue *HoldingValue `json:"totalValue,omitempty"`
	Holdings   []Holding     `json:"holdings"`
}

// ── Loans ─────────────────────────────────────────────────────────────────

// loanBalance wraps an Amount for loan balance fields.
type loanBalance struct {
	Amount Amount `json:"amount"`
}

// LoanAccount is a loan or mortgage account.
// Pointer fields lead for alignment.
type LoanAccount struct {
	Balance      *loanBalance `json:"balance,omitempty"`
	InterestRate *float64     `json:"interestRate,omitempty"`
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	MaturityDate string       `json:"maturityDate,omitempty"`
}

// LoanAccountsResponse is a paginated list of loan accounts.
type LoanAccountsResponse struct {
	Accounts      []LoanAccount `json:"accounts"`
	NextPageToken string        `json:"nextPageToken,omitempty"`
}

// ── Budgets ───────────────────────────────────────────────────────────────

// BudgetType is the type of budget.
type BudgetType string

// BudgetFrequency is the recurrence frequency.
type BudgetFrequency string

const (
	// BudgetTypeIncome tracks income.
	BudgetTypeIncome BudgetType = "INCOME"
	// BudgetTypeExpense tracks expenses.
	BudgetTypeExpense        BudgetType      = "EXPENSE"
	BudgetFrequencyOneOff    BudgetFrequency = "ONE_OFF"
	BudgetFrequencyMonthly   BudgetFrequency = "MONTHLY"
	BudgetFrequencyQuarterly BudgetFrequency = "QUARTERLY"
	BudgetFrequencyYearly    BudgetFrequency = "YEARLY"
)

// BudgetRecurrence defines how a budget recurs.
type BudgetRecurrence struct {
	Frequency BudgetFrequency `json:"frequency"`
	Start     string          `json:"start"`
	End       string          `json:"end,omitempty"`
}

// CreateBudgetParams are parameters for creating a budget.
type CreateBudgetParams struct {
	Recurrence   BudgetRecurrence `json:"recurrence"`
	Title        string           `json:"title"`
	Type         BudgetType       `json:"type"`
	TargetAmount TargetAmount     `json:"targetAmount"`
}

// Budget is a Tink budget.
type Budget struct {
	TargetAmount   *TargetAmount     `json:"targetAmount,omitempty"`
	Recurrence     *BudgetRecurrence `json:"recurrence,omitempty"`
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Type           BudgetType        `json:"type"`
	ProgressStatus string            `json:"progressStatus,omitempty"`
}

// BudgetsResponse is a paginated list of budgets.
type BudgetsResponse struct {
	Budgets       []Budget `json:"budgets"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// BudgetHistoryEntry holds actual vs target for one budget period.
type BudgetHistoryEntry struct {
	Spent     *TargetAmount `json:"spent,omitempty"`
	Remaining *TargetAmount `json:"remaining,omitempty"`
	Period    string        `json:"period"`
}

// BudgetHistoryResponse holds the full history of a budget.
type BudgetHistoryResponse struct {
	History []BudgetHistoryEntry `json:"history"`
}

// BudgetsListOptions are filter options for listing budgets.
type BudgetsListOptions struct {
	PaginationOptions
	ProgressStatusIn []string
}

// ── Cash Flow ─────────────────────────────────────────────────────────────

// CashFlowResolution is the time resolution for cash flow summaries.
type CashFlowResolution string

const (
	// CashFlowResolutionDaily aggregates daily.
	CashFlowResolutionDaily   CashFlowResolution = "DAILY"
	CashFlowResolutionWeekly  CashFlowResolution = "WEEKLY"
	CashFlowResolutionMonthly CashFlowResolution = "MONTHLY"
	CashFlowResolutionYearly  CashFlowResolution = "YEARLY"
)

// cashFlowAmount wraps an Amount for cash-flow period fields.
type cashFlowAmount struct {
	Amount Amount `json:"amount"`
}

// CashFlowPeriod holds cash flow data for a single time period.
// Pointer fields lead for alignment.
type CashFlowPeriod struct {
	Income      *cashFlowAmount `json:"income,omitempty"`
	Expenses    *cashFlowAmount `json:"expenses,omitempty"`
	NetAmount   *Amount         `json:"netAmount,omitempty"`
	PeriodStart string          `json:"periodStart"`
	PeriodEnd   string          `json:"periodEnd"`
}

// CashFlowResponse is the cash flow API response.
type CashFlowResponse struct {
	Periods []CashFlowPeriod `json:"periods"`
}

// CashFlowOptions are options for requesting cash flow summaries.
type CashFlowOptions struct {
	Resolution CashFlowResolution
	FromGte    string
	ToLte      string
}

// ── Financial Calendar ────────────────────────────────────────────────────

// CalendarEventAmount is the monetary amount for a calendar event.
type CalendarEventAmount struct {
	CurrencyCode string      `json:"currencyCode"`
	Value        ExactAmount `json:"value"`
}

// CalendarEvent is a financial calendar event.
type CalendarEvent struct {
	EventAmount *CalendarEventAmount `json:"eventAmount,omitempty"`
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	DueDate     string               `json:"dueDate,omitempty"`
	Status      string               `json:"status,omitempty"`
}

// CalendarEventsResponse is a paginated list of calendar events.
type CalendarEventsResponse struct {
	Events        []CalendarEvent `json:"events"`
	NextPageToken string          `json:"nextPageToken,omitempty"`
}

// CreateCalendarEventParams are parameters for creating a calendar event.
type CreateCalendarEventParams struct {
	EventAmount *CalendarEventAmount `json:"eventAmount,omitempty"`
	Title       string               `json:"title"`
	DueDate     string               `json:"dueDate,omitempty"`
}

// CalendarSummariesOptions are options for requesting calendar summaries.
type CalendarSummariesOptions struct {
	Resolution string
	PeriodGte  string
	PeriodLte  string
}

// RecurringOption controls which occurrences of a recurring event are affected.
type RecurringOption string

const (
	// RecurringSingle affects only the selected occurrence.
	RecurringSingle           RecurringOption = "SINGLE"
	RecurringThisAndFollowing RecurringOption = "THIS_AND_FOLLOWING"
	RecurringAll              RecurringOption = "ALL"
)

// ── Account Check ─────────────────────────────────────────────────────────

// AccountCheckUser holds the user identity for an account check session.
type AccountCheckUser struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// CreateSessionParams are parameters for creating an account check session.
type CreateSessionParams struct {
	User        AccountCheckUser `json:"user"`
	Market      string           `json:"market,omitempty"`
	Locale      string           `json:"locale,omitempty"`
	RedirectURI string           `json:"redirectUri,omitempty"`
}

// AccountCheckSession is a Tink Link account check session.
type AccountCheckSession struct {
	User      *AccountCheckUser `json:"user,omitempty"`
	SessionID string            `json:"sessionId"`
	ExpiresAt string            `json:"expiresAt,omitempty"`
}

// accountCheckVerification holds the verification result.
type accountCheckVerification struct {
	NameMatched     *bool  `json:"nameMatched,omitempty"`
	Status          string `json:"status"`
	MatchConfidence string `json:"matchConfidence,omitempty"`
}

// accountCheckDetails holds the verified account details.
type accountCheckDetails struct {
	IBAN              string `json:"iban,omitempty"`
	AccountNumber     string `json:"accountNumber,omitempty"`
	AccountHolderName string `json:"accountHolderName,omitempty"`
}

// AccountCheckReport is an account ownership verification report.
type AccountCheckReport struct {
	Verification   *accountCheckVerification `json:"verification,omitempty"`
	AccountDetails *accountCheckDetails      `json:"accountDetails,omitempty"`
	ID             string                    `json:"id"`
	Timestamp      string                    `json:"timestamp,omitempty"`
}

// AccountCheckReportsResponse is a paginated list of reports.
type AccountCheckReportsResponse struct {
	Reports       []AccountCheckReport `json:"reports"`
	NextPageToken string               `json:"nextPageToken,omitempty"`
}

// AccountParty is an account owner or co-owner.
type AccountParty struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// AccountPartiesResponse holds a list of account parties.
type AccountPartiesResponse struct {
	Parties []AccountParty `json:"parties"`
}

// GrantUserAccessParams are parameters for granting user access.
type GrantUserAccessParams struct {
	UserID        string `json:"user_id"`
	IDHint        string `json:"id_hint"`
	Scope         string `json:"scope"`
	ActorClientID string `json:"actor_client_id,omitempty"`
}

// ContinuousAccessLinkOptions are options for building a continuous access Tink Link URL.
type ContinuousAccessLinkOptions struct {
	ClientID    string
	Market      string
	Locale      string
	RedirectURI string
	Products    string
}

// ── Balance Check ─────────────────────────────────────────────────────────

// BalanceRefreshResponse is returned when initiating a balance refresh.
type BalanceRefreshResponse struct {
	BalanceRefreshID string `json:"balanceRefreshId"`
	Status           string `json:"status"`
}

// BalanceRefreshStatus is the status of a balance refresh operation.
type BalanceRefreshStatus string

const (
	// BalanceRefreshInitiated means the refresh has been accepted.
	BalanceRefreshInitiated  BalanceRefreshStatus = "INITIATED"
	BalanceRefreshInProgress BalanceRefreshStatus = "IN_PROGRESS"
	BalanceRefreshCompleted  BalanceRefreshStatus = "COMPLETED"
	BalanceRefreshFailed     BalanceRefreshStatus = "FAILED"
)

// BalanceRefreshStatusResponse holds the full balance refresh status.
type BalanceRefreshStatusResponse struct {
	BalanceRefreshID string               `json:"balanceRefreshId"`
	Status           BalanceRefreshStatus `json:"status"`
	Updated          string               `json:"updated,omitempty"`
}

// BuildAccountCheckLinkOptions are options for building a balance check link.
// String fields lead; bool last.
type BuildAccountCheckLinkOptions struct {
	ClientID    string
	Market      string
	RedirectURI string
	State       string
	Test        bool
}

// ConsentUpdateLinkOptions are options for building a consent update link.
type ConsentUpdateLinkOptions struct {
	ClientID      string
	CredentialsID string
	Market        string
	RedirectURI   string
}

// ── Reports ───────────────────────────────────────────────────────────────

// IncomeCheckReport is an income verification report.
type IncomeCheckReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

// ExpenseCheckReport is an expense analysis report.
type ExpenseCheckReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

// RiskInsightsReport is a financial risk insights report.
type RiskInsightsReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

// RiskCategorisationReport is a risk categorisation report.
type RiskCategorisationReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

// BusinessAccountCheckReport is a business account verification report.
type BusinessAccountCheckReport struct {
	ID      string `json:"id"`
	Status  string `json:"status,omitempty"`
	Created string `json:"created,omitempty"`
}

// ── Connector ─────────────────────────────────────────────────────────────

// ConnectorAccount is an account to ingest via the Connector API.
type ConnectorAccount struct {
	ExternalID string  `json:"externalId"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Balance    float64 `json:"balance"`
}

// ConnectorTransaction is a transaction to ingest via the Connector API.
// Strings (16B each) before numerics (8B each) for optimal alignment.
type ConnectorTransaction struct {
	ExternalID  string  `json:"externalId"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Date        int64   `json:"date"`
}

// ConnectorTransactionAccount groups transactions under one account.
type ConnectorTransactionAccount struct {
	Transactions []ConnectorTransaction `json:"transactions"`
	ExternalID   string                 `json:"externalId"`
	Balance      float64                `json:"balance"`
}

// IngestAccountsParams are parameters for ingesting accounts.
type IngestAccountsParams struct {
	Accounts []ConnectorAccount `json:"accounts"`
}

// IngestType controls real-time vs batch ingestion.
type IngestType string

const (
	// IngestTypeRealTime is used for live transaction feeds.
	IngestTypeRealTime IngestType = "REAL_TIME"
	IngestTypeBatch    IngestType = "BATCH"
)

// IngestTransactionsParams are parameters for ingesting transactions.
type IngestTransactionsParams struct {
	TransactionAccounts []ConnectorTransactionAccount `json:"transactionAccounts"`
	Type                IngestType                    `json:"type"`
}

// ── Link ──────────────────────────────────────────────────────────────────

// LinkProduct is a supported Tink Link product.
type LinkProduct string

const (
	// LinkProductTransactions connects bank accounts for transaction access.
	LinkProductTransactions LinkProduct = "transactions"
	LinkProductAccountCheck LinkProduct = "account_check"
	LinkProductIncomeCheck  LinkProduct = "income_check"
	LinkProductPayment      LinkProduct = "payment"
	LinkProductExpenseCheck LinkProduct = "expense_check"
	LinkProductRiskInsights LinkProduct = "risk_insights"
)

// LinkURLOptions are parameters for building a Tink Link URL.
// String fields lead; bools last.
type LinkURLOptions struct {
	ClientID          string
	RedirectURI       string
	Market            string
	Locale            string
	AuthorizationCode string
	PaymentRequestID  string
	State             string
	InputProvider     string
	InputUsername     string
	Test              bool
	Iframe            bool
}

// ── Connectivity ──────────────────────────────────────────────────────────

// ProviderStatusResult is the result of a provider status check.
// Pointer leads; bool last.
type ProviderStatusResult struct {
	Provider *Provider `json:"provider,omitempty"`
	Active   bool      `json:"active"`
}

// CredentialConnectivity holds the health status of a credential.
// Strings lead; bool last.
type CredentialConnectivity struct {
	CredentialID  string `json:"credentialId"`
	ProviderName  string `json:"providerName"`
	Status        string `json:"status"`
	LastRefreshed string `json:"lastRefreshed,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	Healthy       bool   `json:"healthy"`
}

// ConnectivitySummary holds an aggregated connectivity report.
type ConnectivitySummary struct {
	Credentials []CredentialConnectivity `json:"credentials"`
	Healthy     int                      `json:"healthy"`
	Unhealthy   int                      `json:"unhealthy"`
	Total       int                      `json:"total"`
}

// ConnectivityOptions controls which credentials are included in a summary.
type ConnectivityOptions struct {
	IncludeHealthy   *bool
	IncludeUnhealthy *bool
}

// ── Webhooks ──────────────────────────────────────────────────────────────

// WebhookEventType is a known Tink webhook event type.
type WebhookEventType string

const (
	// WebhookEventCredentialsUpdated fires when credentials are updated.
	WebhookEventCredentialsUpdated          WebhookEventType = "credentials.updated"
	WebhookEventCredentialsRefreshSucceeded WebhookEventType = "credentials.refresh.succeeded"
	WebhookEventCredentialsRefreshFailed    WebhookEventType = "credentials.refresh.failed"
	WebhookEventProviderConsentsCreated     WebhookEventType = "provider_consents.created"
	WebhookEventProviderConsentsRevoked     WebhookEventType = "provider_consents.revoked"
	WebhookEventTest                        WebhookEventType = "test"
)

// WebhookEvent is a parsed Tink webhook event.
// Maps lead; strings follow.
type WebhookEvent struct {
	Data      map[string]interface{} `json:"data"`
	Raw       map[string]interface{} `json:"-"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp,omitempty"`
}
