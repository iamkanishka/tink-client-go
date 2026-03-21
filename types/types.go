// Package types defines all domain types for the tink-client-go client.
package types

import "time"

// ── Config ────────────────────────────────────────────────────────────────

// Config holds all options for constructing a Client.
type Config struct {
	ClientID       string
	ClientSecret   string
	AccessToken    string
	UserID         string
	BaseURL        string
	Timeout        time.Duration
	MaxRetries     int
	DisableCache   bool
	CacheMaxSize   int
	DefaultHeaders map[string]string
}

// ── Error ─────────────────────────────────────────────────────────────────

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

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type AuthorizationURLOptions struct {
	ClientID    string
	RedirectURI string
	Scope       string
	State       string
	Market      string
	Locale      string
}

type CreateAuthorizationParams struct {
	UserID string `json:"user_id"`
	Scope  string `json:"scope"`
}

type DelegateAuthorizationParams struct {
	UserID        string `json:"user_id"`
	IDHint        string `json:"id_hint"`
	Scope         string `json:"scope"`
	ActorClientID string `json:"actor_client_id,omitempty"`
}

type AuthorizationCode struct {
	Code string `json:"code"`
}

// ── Primitives ────────────────────────────────────────────────────────────

type Amount struct {
	Value        string `json:"value"`
	CurrencyCode string `json:"currencyCode"`
}

type ExactAmount struct {
	UnscaledValue int64 `json:"unscaledValue"`
	Scale         int   `json:"scale"`
}

type TargetAmount struct {
	Value        ExactAmount `json:"value"`
	CurrencyCode string      `json:"currencyCode"`
}

type PaginationOptions struct {
	PageSize  int    `json:"pageSize,omitempty"`
	PageToken string `json:"pageToken,omitempty"`
}

// ── Accounts ──────────────────────────────────────────────────────────────

type AccountBalanceItem struct {
	Amount Amount `json:"amount"`
}

type AccountBalances struct {
	Booked      *AccountBalanceItem `json:"booked,omitempty"`
	Available   *AccountBalanceItem `json:"available,omitempty"`
	Reserved    *AccountBalanceItem `json:"reserved,omitempty"`
	CreditLimit *AccountBalanceItem `json:"creditLimit,omitempty"`
}

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

type FinancialInstitution struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Account struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	Type                 string                `json:"type"`
	Subtype              string                `json:"subtype,omitempty"`
	Currency             string                `json:"currency,omitempty"`
	Balances             *AccountBalances      `json:"balances,omitempty"`
	Identifiers          *AccountIdentifiers   `json:"identifiers,omitempty"`
	ProviderName         string                `json:"providerName,omitempty"`
	Ownership            string                `json:"ownership,omitempty"`
	Flags                []string              `json:"flags,omitempty"`
	FinancialInstitution *FinancialInstitution `json:"financialInstitution,omitempty"`
	CredentialsID        string                `json:"credentialsId,omitempty"`
}

type AccountsResponse struct {
	Accounts      []Account `json:"accounts"`
	NextPageToken string    `json:"nextPageToken,omitempty"`
}

type AccountsListOptions struct {
	PaginationOptions
	TypeIn []string
}

// ── Transactions ──────────────────────────────────────────────────────────

type Transaction struct {
	ID           string `json:"id"`
	AccountID    string `json:"accountId,omitempty"`
	Amount       Amount `json:"amount"`
	Status       string `json:"status"`
	Descriptions *struct {
		Original string `json:"original,omitempty"`
		Display  string `json:"display,omitempty"`
	} `json:"descriptions,omitempty"`
	Dates *struct {
		Booked string `json:"booked,omitempty"`
		Value  string `json:"value,omitempty"`
	} `json:"dates,omitempty"`
	MerchantInformation *struct {
		MerchantName         string `json:"merchantName,omitempty"`
		MerchantCategoryCode string `json:"merchantCategoryCode,omitempty"`
	} `json:"merchantInformation,omitempty"`
	Categories *struct {
		PFM *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"pfm,omitempty"`
	} `json:"categories,omitempty"`
}

type TransactionsResponse struct {
	Transactions  []Transaction `json:"transactions"`
	NextPageToken string        `json:"nextPageToken,omitempty"`
}

type TransactionsListOptions struct {
	PaginationOptions
	AccountIDIn   []string
	BookedDateGte string
	BookedDateLte string
	StatusIn      []string
	CategoryIDIn  []string
}

// ── Providers ─────────────────────────────────────────────────────────────

type Provider struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	Type         string   `json:"type,omitempty"`
	Status       string   `json:"status,omitempty"`
	Market       string   `json:"market"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type ProvidersResponse struct {
	Providers []Provider `json:"providers"`
}

type ProvidersListOptions struct {
	Market       string
	Capabilities []string
}

// ── Categories ────────────────────────────────────────────────────────────

type Category struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DisplayName string `json:"displayName,omitempty"`
}

type CategoriesResponse struct {
	Categories []Category `json:"categories"`
}

// ── Statistics ────────────────────────────────────────────────────────────

type StatisticsPeriod struct {
	Period string `json:"period"`
	Income *struct {
		Amount Amount `json:"amount"`
	} `json:"income,omitempty"`
	Expenses *struct {
		Amount Amount `json:"amount"`
	} `json:"expenses,omitempty"`
}

type StatisticsResponse struct {
	Periods []StatisticsPeriod `json:"periods"`
}

type StatisticsOptions struct {
	PeriodGte    string
	PeriodLte    string
	Resolution   string
	AccountIDIn  []string
	CategoryIDIn []string
}

// ── Users ─────────────────────────────────────────────────────────────────

type CreateUserParams struct {
	ExternalUserID string `json:"external_user_id"`
	Locale         string `json:"locale"`
	Market         string `json:"market"`
}

type TinkUser struct {
	UserID         string `json:"userId,omitempty"`
	User_ID        string `json:"user_id,omitempty"`
	ExternalUserID string `json:"externalUserId,omitempty"`
}

type Credential struct {
	ID            string `json:"id"`
	ProviderName  string `json:"providerName"`
	Status        string `json:"status,omitempty"`
	StatusUpdated string `json:"statusUpdated,omitempty"`
	StatusPayload string `json:"statusPayload,omitempty"`
}

type CredentialsResponse struct {
	Credentials []Credential `json:"credentials"`
}

// ── Investments ───────────────────────────────────────────────────────────

type InvestmentAccount struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Balance *struct {
		Amount Amount `json:"amount"`
	} `json:"balance,omitempty"`
}

type InvestmentAccountsResponse struct {
	Accounts      []InvestmentAccount `json:"accounts"`
	NextPageToken string              `json:"nextPageToken,omitempty"`
}

type HoldingValue struct {
	Amount Amount `json:"amount"`
}

type Holding struct {
	ID         string `json:"id"`
	Instrument *struct {
		Type   string `json:"type"`
		Symbol string `json:"symbol,omitempty"`
		ISIN   string `json:"isin,omitempty"`
	} `json:"instrument,omitempty"`
	Quantity    *float64      `json:"quantity,omitempty"`
	MarketValue *HoldingValue `json:"marketValue,omitempty"`
}

type HoldingsResponse struct {
	Holdings   []Holding     `json:"holdings"`
	TotalValue *HoldingValue `json:"totalValue,omitempty"`
}

// ── Loans ─────────────────────────────────────────────────────────────────

type LoanAccount struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Balance *struct {
		Amount Amount `json:"amount"`
	} `json:"balance,omitempty"`
	InterestRate *float64 `json:"interestRate,omitempty"`
	MaturityDate string   `json:"maturityDate,omitempty"`
}

type LoanAccountsResponse struct {
	Accounts      []LoanAccount `json:"accounts"`
	NextPageToken string        `json:"nextPageToken,omitempty"`
}

// ── Budgets ───────────────────────────────────────────────────────────────

type BudgetType string
type BudgetFrequency string

const (
	BudgetTypeIncome         BudgetType      = "INCOME"
	BudgetTypeExpense        BudgetType      = "EXPENSE"
	BudgetFrequencyOneOff    BudgetFrequency = "ONE_OFF"
	BudgetFrequencyMonthly   BudgetFrequency = "MONTHLY"
	BudgetFrequencyQuarterly BudgetFrequency = "QUARTERLY"
	BudgetFrequencyYearly    BudgetFrequency = "YEARLY"
)

type BudgetRecurrence struct {
	Frequency BudgetFrequency `json:"frequency"`
	Start     string          `json:"start"`
	End       string          `json:"end,omitempty"`
}

type CreateBudgetParams struct {
	Title        string           `json:"title"`
	Type         BudgetType       `json:"type"`
	TargetAmount TargetAmount     `json:"targetAmount"`
	Recurrence   BudgetRecurrence `json:"recurrence"`
}

type Budget struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Type           BudgetType        `json:"type"`
	TargetAmount   *TargetAmount     `json:"targetAmount,omitempty"`
	Recurrence     *BudgetRecurrence `json:"recurrence,omitempty"`
	ProgressStatus string            `json:"progressStatus,omitempty"`
}

type BudgetsResponse struct {
	Budgets       []Budget `json:"budgets"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

type BudgetHistoryEntry struct {
	Period    string        `json:"period"`
	Spent     *TargetAmount `json:"spent,omitempty"`
	Remaining *TargetAmount `json:"remaining,omitempty"`
}

type BudgetHistoryResponse struct {
	History []BudgetHistoryEntry `json:"history"`
}

type BudgetsListOptions struct {
	PaginationOptions
	ProgressStatusIn []string
}

// ── Cash Flow ─────────────────────────────────────────────────────────────

type CashFlowResolution string

const (
	CashFlowResolutionDaily   CashFlowResolution = "DAILY"
	CashFlowResolutionWeekly  CashFlowResolution = "WEEKLY"
	CashFlowResolutionMonthly CashFlowResolution = "MONTHLY"
	CashFlowResolutionYearly  CashFlowResolution = "YEARLY"
)

type CashFlowPeriod struct {
	PeriodStart string `json:"periodStart"`
	PeriodEnd   string `json:"periodEnd"`
	Income      *struct {
		Amount Amount `json:"amount"`
	} `json:"income,omitempty"`
	Expenses *struct {
		Amount Amount `json:"amount"`
	} `json:"expenses,omitempty"`
	NetAmount *Amount `json:"netAmount,omitempty"`
}

type CashFlowResponse struct {
	Periods []CashFlowPeriod `json:"periods"`
}

type CashFlowOptions struct {
	Resolution CashFlowResolution
	FromGte    string
	ToLte      string
}

// ── Financial Calendar ────────────────────────────────────────────────────

type CalendarEventAmount struct {
	CurrencyCode string      `json:"currencyCode"`
	Value        ExactAmount `json:"value"`
}

type CalendarEvent struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	DueDate     string               `json:"dueDate,omitempty"`
	EventAmount *CalendarEventAmount `json:"eventAmount,omitempty"`
	Status      string               `json:"status,omitempty"`
}

type CalendarEventsResponse struct {
	Events        []CalendarEvent `json:"events"`
	NextPageToken string          `json:"nextPageToken,omitempty"`
}

type CreateCalendarEventParams struct {
	Title       string               `json:"title"`
	DueDate     string               `json:"dueDate,omitempty"`
	EventAmount *CalendarEventAmount `json:"eventAmount,omitempty"`
}

type CalendarSummariesOptions struct {
	Resolution string
	PeriodGte  string
	PeriodLte  string
}

type RecurringOption string

const (
	RecurringSingle           RecurringOption = "SINGLE"
	RecurringThisAndFollowing RecurringOption = "THIS_AND_FOLLOWING"
	RecurringAll              RecurringOption = "ALL"
)

// ── Account Check ─────────────────────────────────────────────────────────

type AccountCheckUser struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type CreateSessionParams struct {
	User        AccountCheckUser `json:"user"`
	Market      string           `json:"market,omitempty"`
	Locale      string           `json:"locale,omitempty"`
	RedirectURI string           `json:"redirectUri,omitempty"`
}

type AccountCheckSession struct {
	SessionID string            `json:"sessionId"`
	User      *AccountCheckUser `json:"user,omitempty"`
	ExpiresAt string            `json:"expiresAt,omitempty"`
}

type AccountCheckReport struct {
	ID           string `json:"id"`
	Verification *struct {
		Status          string `json:"status"`
		NameMatched     *bool  `json:"nameMatched,omitempty"`
		MatchConfidence string `json:"matchConfidence,omitempty"`
	} `json:"verification,omitempty"`
	AccountDetails *struct {
		IBAN              string `json:"iban,omitempty"`
		AccountNumber     string `json:"accountNumber,omitempty"`
		AccountHolderName string `json:"accountHolderName,omitempty"`
	} `json:"accountDetails,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type AccountCheckReportsResponse struct {
	Reports       []AccountCheckReport `json:"reports"`
	NextPageToken string               `json:"nextPageToken,omitempty"`
}

type AccountParty struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type AccountPartiesResponse struct {
	Parties []AccountParty `json:"parties"`
}

type GrantUserAccessParams struct {
	UserID        string `json:"user_id"`
	IDHint        string `json:"id_hint"`
	Scope         string `json:"scope"`
	ActorClientID string `json:"actor_client_id,omitempty"`
}

type ContinuousAccessLinkOptions struct {
	ClientID    string
	Market      string
	Locale      string
	RedirectURI string
	Products    string
}

// ── Balance Check ─────────────────────────────────────────────────────────

type BalanceRefreshResponse struct {
	BalanceRefreshID string `json:"balanceRefreshId"`
	Status           string `json:"status"`
}

type BalanceRefreshStatus string

const (
	BalanceRefreshInitiated  BalanceRefreshStatus = "INITIATED"
	BalanceRefreshInProgress BalanceRefreshStatus = "IN_PROGRESS"
	BalanceRefreshCompleted  BalanceRefreshStatus = "COMPLETED"
	BalanceRefreshFailed     BalanceRefreshStatus = "FAILED"
)

type BalanceRefreshStatusResponse struct {
	BalanceRefreshID string               `json:"balanceRefreshId"`
	Status           BalanceRefreshStatus `json:"status"`
	Updated          string               `json:"updated,omitempty"`
}

type BuildAccountCheckLinkOptions struct {
	ClientID    string
	Market      string
	RedirectURI string
	Test        bool
	State       string
}

type ConsentUpdateLinkOptions struct {
	ClientID      string
	CredentialsID string
	Market        string
	RedirectURI   string
}

// ── Reports ───────────────────────────────────────────────────────────────

type IncomeCheckReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

type ExpenseCheckReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

type RiskInsightsReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

type RiskCategorisationReport struct {
	ID      string `json:"id"`
	Created string `json:"created,omitempty"`
}

type BusinessAccountCheckReport struct {
	ID      string `json:"id"`
	Status  string `json:"status,omitempty"`
	Created string `json:"created,omitempty"`
}

// ── Connector ─────────────────────────────────────────────────────────────

type ConnectorAccount struct {
	ExternalID string  `json:"externalId"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Balance    float64 `json:"balance"`
}

type ConnectorTransaction struct {
	ExternalID  string  `json:"externalId"`
	Amount      float64 `json:"amount"`
	Date        int64   `json:"date"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
}

type ConnectorTransactionAccount struct {
	ExternalID   string                 `json:"externalId"`
	Balance      float64                `json:"balance"`
	Transactions []ConnectorTransaction `json:"transactions"`
}

type IngestAccountsParams struct {
	Accounts []ConnectorAccount `json:"accounts"`
}

type IngestType string

const (
	IngestTypeRealTime IngestType = "REAL_TIME"
	IngestTypeBatch    IngestType = "BATCH"
)

type IngestTransactionsParams struct {
	Type                IngestType                    `json:"type"`
	TransactionAccounts []ConnectorTransactionAccount `json:"transactionAccounts"`
}

// ── Link ──────────────────────────────────────────────────────────────────

type LinkProduct string

const (
	LinkProductTransactions LinkProduct = "transactions"
	LinkProductAccountCheck LinkProduct = "account_check"
	LinkProductIncomeCheck  LinkProduct = "income_check"
	LinkProductPayment      LinkProduct = "payment"
	LinkProductExpenseCheck LinkProduct = "expense_check"
	LinkProductRiskInsights LinkProduct = "risk_insights"
)

type LinkURLOptions struct {
	ClientID          string
	RedirectURI       string
	Market            string
	Locale            string
	AuthorizationCode string
	PaymentRequestID  string
	State             string
	Test              bool
	InputProvider     string
	InputUsername     string
	Iframe            bool
}

// ── Connectivity ──────────────────────────────────────────────────────────

type ProviderStatusResult struct {
	Active   bool      `json:"active"`
	Provider *Provider `json:"provider,omitempty"`
}

type CredentialConnectivity struct {
	CredentialID  string `json:"credentialId"`
	ProviderName  string `json:"providerName"`
	Status        string `json:"status"`
	Healthy       bool   `json:"healthy"`
	LastRefreshed string `json:"lastRefreshed,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type ConnectivitySummary struct {
	Credentials []CredentialConnectivity `json:"credentials"`
	Healthy     int                      `json:"healthy"`
	Unhealthy   int                      `json:"unhealthy"`
	Total       int                      `json:"total"`
}

type ConnectivityOptions struct {
	IncludeHealthy   *bool
	IncludeUnhealthy *bool
}

// ── Webhooks ──────────────────────────────────────────────────────────────

type WebhookEventType string

const (
	WebhookEventCredentialsUpdated          WebhookEventType = "credentials.updated"
	WebhookEventCredentialsRefreshSucceeded WebhookEventType = "credentials.refresh.succeeded"
	WebhookEventCredentialsRefreshFailed    WebhookEventType = "credentials.refresh.failed"
	WebhookEventProviderConsentsCreated     WebhookEventType = "provider_consents.created"
	WebhookEventProviderConsentsRevoked     WebhookEventType = "provider_consents.revoked"
	WebhookEventTest                        WebhookEventType = "test"
)

type WebhookEvent struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp,omitempty"`
	Raw       map[string]interface{} `json:"-"`
}
