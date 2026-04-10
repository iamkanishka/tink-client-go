# Changelog

All notable changes to `tink-client-go` are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [1.0.0] — 2026-03-21

### Added

**Core infrastructure**

- `client.Client` — main entry point with 24 service namespaces, wired at construction
- `client.New(Config)` — struct-based constructor; reads `TINK_CLIENT_ID`, `TINK_CLIENT_SECRET`, `TINK_ACCESS_TOKEN`, `TINK_BASE_URL` from environment when fields are empty
- `client.NewWithOptions(...Option)` — functional options constructor
- Functional options: `WithCredentials`, `WithAccessToken`, `WithBaseURL`, `WithTimeout`, `WithMaxRetries`, `WithHTTPClient`, `WithHeader`, `WithDisableCache`
- `Client.Authenticate(ctx, scope)` — acquires a client credentials token and sets it automatically
- `Client.SetAccessToken(token)` / `Client.AccessToken()` — thread-safe token management
- `Client.ClearCache()` / `Client.InvalidateCache(prefix)` — manual cache control
- `Client.NewWebhookHandler(secret)` / `Client.NewWebhookVerifier(secret)` — webhook factories
- `Client.Info()` — returns version, base URL, and `HasToken` status
- `client.ParseToken(TokenResponse)` — extracts expiry info from an OAuth token response
- `client.IsExpired(time.Time)` — checks expiry with a 5-minute safety buffer

**`types` package** — all domain types with JSON tags

- `Config`, `ErrorType` constants (`ErrorTypeAuthentication`, `ErrorTypeRateLimit`, `ErrorTypeNetwork`, `ErrorTypeTimeout`, `ErrorTypeValidation`, `ErrorTypeDecode`, `ErrorTypeAPI`, `ErrorTypeUnknown`)
- `TokenResponse`, `AuthorizationURLOptions`, `CreateAuthorizationParams`, `DelegateAuthorizationParams`, `AuthorizationCode`
- `Amount`, `ExactAmount`, `TargetAmount`, `PaginationOptions`
- `Account`, `AccountBalances`, `AccountBalanceItem`, `AccountIdentifiers`, `AccountsResponse`, `AccountsListOptions`
- `Transaction`, `TransactionsResponse`, `TransactionsListOptions`
- `Provider`, `ProvidersResponse`, `ProvidersListOptions`
- `Category`, `CategoriesResponse`
- `StatisticsPeriod`, `StatisticsResponse`, `StatisticsOptions`
- `CreateUserParams`, `TinkUser`, `Credential`, `CredentialsResponse`
- `InvestmentAccount`, `InvestmentAccountsResponse`, `Holding`, `HoldingValue`, `HoldingsResponse`
- `LoanAccount`, `LoanAccountsResponse`
- `BudgetType`, `BudgetFrequency`, `BudgetRecurrence`, `CreateBudgetParams`, `Budget`, `BudgetsResponse`, `BudgetHistoryEntry`, `BudgetHistoryResponse`, `BudgetsListOptions`
- `CashFlowResolution`, `CashFlowPeriod`, `CashFlowResponse`, `CashFlowOptions`
- `CalendarEvent`, `CalendarEventsResponse`, `CreateCalendarEventParams`, `CalendarSummariesOptions`, `RecurringOption` constants (`RecurringSingle`, `RecurringThisAndFollowing`, `RecurringAll`)
- `AccountCheckUser`, `CreateSessionParams`, `AccountCheckSession`, `AccountCheckReport`, `AccountCheckReportsResponse`, `AccountParty`, `AccountPartiesResponse`, `GrantUserAccessParams`, `ContinuousAccessLinkOptions`
- `BalanceRefreshResponse`, `BalanceRefreshStatus` constants (`BalanceRefreshInitiated`, `BalanceRefreshInProgress`, `BalanceRefreshCompleted`, `BalanceRefreshFailed`), `BalanceRefreshStatusResponse`, `BuildAccountCheckLinkOptions`, `ConsentUpdateLinkOptions`
- `IncomeCheckReport`, `ExpenseCheckReport`, `RiskInsightsReport`, `RiskCategorisationReport`, `BusinessAccountCheckReport`
- `ConnectorAccount`, `ConnectorTransaction`, `ConnectorTransactionAccount`, `IngestAccountsParams`, `IngestType` constants (`IngestTypeRealTime`, `IngestTypeBatch`), `IngestTransactionsParams`
- `LinkProduct` constants (`LinkProductTransactions`, `LinkProductAccountCheck`, `LinkProductIncomeCheck`, `LinkProductPayment`, `LinkProductExpenseCheck`, `LinkProductRiskInsights`), `LinkURLOptions`
- `ProviderStatusResult`, `CredentialConnectivity`, `ConnectivitySummary`, `ConnectivityOptions`
- `WebhookEventType` constants (`WebhookEventCredentialsUpdated`, `WebhookEventCredentialsRefreshSucceeded`, `WebhookEventCredentialsRefreshFailed`, `WebhookEventProviderConsentsCreated`, `WebhookEventProviderConsentsRevoked`, `WebhookEventTest`), `WebhookEvent`

**`errors` package**

- `TinkError` struct with fields: `Type`, `Message`, `StatusCode`, `ErrorCode`, `RequestID`, `ErrorDetails`, `Cause`
- `TinkError.Retryable() bool` — true for `network_error`, `timeout`, HTTP 408/429/500/502/503/504
- `TinkError.Format() string` — human-readable `"[401] Unauthorized (TOKEN_INVALID)"`
- `TinkError.Error() string` — implements `error` interface
- `TinkError.Unwrap() error` — enables `errors.Is` / `errors.As` chain traversal
- `FromResponse(statusCode int, body []byte)` — constructs from HTTP response
- `FromNetworkError(cause error)` — wraps transport failures; classifies timeouts
- `FromDecodeError(cause error)` — wraps JSON parse failures
- `Validation(msg string)` — creates validation errors for config issues
- `New(ErrorType, msg)` — generic constructor

**`auth` package**

- `Service.GetAccessToken(ctx, clientID, clientSecret, scope)` — client credentials grant
- `Service.ExchangeCode(ctx, clientID, clientSecret, code)` — authorization code exchange
- `Service.RefreshAccessToken(ctx, clientID, clientSecret, refreshToken)` — token refresh
- `Service.BuildAuthorizationURL(AuthorizationURLOptions)` — OAuth redirect URL
- `Service.CreateAuthorization(ctx, CreateAuthorizationParams)` — create authorization grant
- `Service.DelegateAuthorization(ctx, DelegateAuthorizationParams)` — delegate grant for Tink Link
- `Service.ValidateToken(ctx)` — boolean token health probe

**`accounts` package**

- `Service.ListAccounts(ctx, *AccountsListOptions)` — paginated account listing with type filter
- `Service.GetAccount(ctx, accountID)` — single account
- `Service.GetBalances(ctx, accountID)` — real-time booked/available/reserved/credit balances

**`transactions` package**

- `Service.ListAccounts` / `Service.ListTransactions` — standard access
- `OneTimeAccessService.ListAccounts` / `OneTimeAccessService.ListTransactions` — single-authorization flow
- `ContinuousAccessService.CreateUser` — create permanent Tink user
- `ContinuousAccessService.GrantUserAccess` — delegate Tink Link access
- `ContinuousAccessService.BuildTinkLink` — build bank-connection URL
- `ContinuousAccessService.CreateAuthorization` — data access grant
- `ContinuousAccessService.GetUserAccessToken` — exchange code for user token
- `ContinuousAccessService.ListAccounts` / `ContinuousAccessService.ListTransactions`

**`providers` package**

- `Service.ListProviders(ctx, *ProvidersListOptions)` — market and capability filters; cached 1 hour
- `Service.GetProvider(ctx, providerID)` — single provider; cached 1 hour

**`categories` package**

- `Service.ListCategories(ctx, locale)` — locale-aware; cached 24 hours
- `Service.GetCategory(ctx, categoryID, locale)` — single category; cached 24 hours

**`statistics` package**

- `Service.GetStatistics(ctx, StatisticsOptions)` — aggregated income/expense across periods
- `Service.GetCategoryStatistics(ctx, categoryID, StatisticsOptions)` — per-category breakdown
- `Service.GetAccountStatistics(ctx, accountID, StatisticsOptions)` — per-account breakdown
- All results cached 1 hour

**`users` package**

- `Service.CreateUser(ctx, CreateUserParams)` — creates a Tink user
- `Service.DeleteUser(ctx, userID)` — permanently deletes user and all data
- `Service.ListCredentials(ctx)` — bank connections; cached 30 seconds
- `Service.GetCredential(ctx, credentialID)` — single credential
- `Service.DeleteCredential(ctx, credentialID)`
- `Service.RefreshCredential(ctx, credentialID)` — triggers bank data refresh; invalidates cache
- `Service.CreateAuthorization(ctx, userID, scope)` — creates authorization grant
- `Service.GetUserAccessToken(ctx, clientID, clientSecret, code)` — token exchange

**`investments` package**

- `Service.ListAccounts(ctx)` — brokerage, ISA, pension accounts
- `Service.GetAccount(ctx, accountID)`
- `Service.GetHoldings(ctx, accountID)` — positions with instrument details and market value

**`loans` package**

- `Service.ListAccounts(ctx)` — mortgages, personal loans, auto loans
- `Service.GetAccount(ctx, accountID)` — interest rate, maturity date, payment schedule

**`budgets` package**

- `Service.CreateBudget(ctx, CreateBudgetParams)` — income or expense budget with recurrence
- `Service.GetBudget(ctx, budgetID)`
- `Service.GetBudgetHistory(ctx, budgetID)` — spending history across periods
- `Service.ListBudgets(ctx, *BudgetsListOptions)` — with progress status filter
- `Service.UpdateBudget(ctx, budgetID, updates)` — partial update via map
- `Service.DeleteBudget(ctx, budgetID)`

**`cashflow` package**

- `Service.GetSummaries(ctx, CashFlowOptions)` — income/expense summaries with DAILY/WEEKLY/MONTHLY/YEARLY resolution

**`calendar` package**

- `Service.CreateEvent(ctx, CreateCalendarEventParams)` — bills, salaries, subscriptions
- `Service.GetEvent(ctx, eventID)`
- `Service.UpdateEvent(ctx, eventID, updates)` — partial update
- `Service.ListEvents(ctx, query)` — with arbitrary query parameters
- `Service.DeleteEvent(ctx, eventID, RecurringOption)` — SINGLE / THIS_AND_FOLLOWING / ALL
- `Service.GetSummaries(ctx, CalendarSummariesOptions)` — period summaries
- `Service.AddAttachment(ctx, eventID, params)` — attach invoice URLs
- `Service.DeleteAttachment(ctx, eventID, attachmentID)`
- `Service.CreateRecurringGroup(ctx, eventID, params)` — iCalendar RRULE recurrence
- `Service.CreateReconciliation(ctx, eventID, params)` — link event to transaction
- `Service.GetReconciliationDetails(ctx, eventID)`
- `Service.GetReconciliationSuggestions(ctx, eventID)` — AI-suggested transaction matches
- `Service.DeleteReconciliation(ctx, eventID, transactionID)`

**`accountcheck` package**

- `Service.CreateSession(ctx, CreateSessionParams)` — creates Tink Link session for one-time verification
- `Service.BuildLinkURL(session, BuildLinkURLOptions)` — verification redirect URL
- `Service.GetReport(ctx, reportID)` — MATCH / NO_MATCH / INDETERMINATE
- `Service.GetReportPDF(ctx, reportID, template)` — PDF binary download
- `Service.ListReports(ctx, *PaginationOptions)` — paginated report list
- `Service.CreateUser(ctx, CreateUserParams)` — persistent user for continuous access
- `Service.GrantUserAccess(ctx, GrantUserAccessParams, defaultClientID)` — delegate Tink Link
- `Service.BuildContinuousAccessLink(authCode, ContinuousAccessLinkOptions)` — persistent connection URL
- `Service.CreateAuthorization(ctx, userID, scope)`
- `Service.GetUserAccessToken(ctx, clientID, clientSecret, code)`
- `Service.ListAccounts(ctx, *PaginationOptions)`
- `Service.GetAccountParties(ctx, accountID)` — account owners and co-owners
- `Service.ListIdentities(ctx)` — name, address, national ID data
- `Service.ListTransactions(ctx, *TransactionsListOptions)`
- `Service.DeleteUser(ctx, userID)`

**`balancecheck` package**

- `Service.CreateUser(ctx, CreateUserParams)`
- `Service.GrantUserAccess(ctx, GrantUserAccessParams, defaultClientID)`
- `Service.BuildAccountCheckLink(authCode, BuildAccountCheckLinkOptions)` — includes `test` and `state` params
- `Service.GetAccountCheckReport(ctx, reportID)`
- `Service.CreateAuthorization(ctx, userID, scope)`
- `Service.GetUserAccessToken(ctx, clientID, clientSecret, code)`
- `Service.RefreshBalance(ctx, accountID)` — initiates async real-time balance refresh
- `Service.GetRefreshStatus(ctx, refreshID)` — INITIATED / IN_PROGRESS / COMPLETED / FAILED
- `Service.GetAccountBalance(ctx, accountID)` — read updated balance after completion
- `Service.GrantConsentUpdate(ctx, GrantUserAccessParams, defaultClientID)` — consent renewal
- `Service.BuildConsentUpdateLink(authCode, ConsentUpdateLinkOptions)` — renewal redirect URL

**`reports` package**

- `IncomeCheckService.GetReport(ctx, reportID)` — income stream analysis
- `IncomeCheckService.GetReportPDF(ctx, reportID)` — PDF binary with `:generate-pdf` endpoint
- `ExpenseCheckService.GetReport(ctx, reportID)` — categorised expense analysis
- `RiskInsightsService.GetReport(ctx, reportID)` — financial risk scoring
- `RiskCategorisationService.GetReport(ctx, reportID)` — transaction-level risk categories
- `BusinessAccountCheckService.GetReport(ctx, reportID)` — business account verification (`/data/v1/` path)

**`connector` package**

- `Service.CreateUser(ctx, CreateUserParams)`
- `Service.IngestAccounts(ctx, externalUserID, IngestAccountsParams)` — push account data
- `Service.IngestTransactions(ctx, externalUserID, IngestTransactionsParams)` — REAL_TIME or BATCH

**`link` package**

- `Service.BuildURL(product, LinkURLOptions)` — all six products; supports test mode, iframe, state
- `Service.TransactionsURL(authCode, LinkURLOptions)` — convenience wrapper
- `Service.AccountCheckURL(authCode, LinkURLOptions)` — convenience wrapper
- `Service.PaymentURL(paymentRequestID, LinkURLOptions)` — convenience wrapper
- Product URL paths: `transactions/connect-accounts`, `account-check/connect-accounts`, `income-check/connect-accounts`, `pay/execute-payment`, `expense-check/connect-accounts`, `risk-insights/connect-accounts`

**`connectivity` package**

- `Service.ListProvidersByMarket(ctx, market)` — unauthenticated
- `Service.ListProvidersByMarketAuthenticated(ctx, market)` — authenticated
- `Service.CheckProviderStatus(ctx, providerID, market)` — ENABLED check with optional market validation; never returns error (returns `active: false` on failure)
- `Service.ProviderOperational(ctx, providerID, market)` — boolean wrapper
- `Service.CheckCredentialConnectivity(ctx, *ConnectivityOptions)` — healthy/unhealthy summary
- `Service.GetCredentialConnectivity(ctx, credentialID)` — single credential status
- `Service.CheckAPIHealth(ctx)` — probes `/api/v1/providers/GB`

**`webhooks` package**

- `Verifier.Verify(payload, signature)` — HMAC-SHA256 constant-time verification (`crypto/hmac.Equal`)
- `Verifier.GenerateSignature(payload)` — raw HMAC bytes
- `Verifier.GenerateSignatureHex(payload)` — hex-encoded for testing
- `Handler.On(eventType, HandlerFunc)` — typed event registration; chainable
- `Handler.OnAll(HandlerFunc)` — wildcard handler; chainable
- `Handler.Off(eventType)` — remove all handlers for a type
- `Handler.HandleRequest(ctx, body, signature)` — verify + parse + dispatch; returns nil for test webhooks; uses `errors.Join` to aggregate handler errors
- `Handler.Handlers()` — snapshot of registered handler counts
- `VerificationError` — typed error with `Code` field (`missing_signature`, `invalid_signature`, `invalid_json`, `missing_type`, `missing_data`)

**`internal/cache` package**

- Thread-safe LRU cache using `sync.Mutex` + `container/list`
- `LRU.Get(key)` — miss on expired entries; moves hit to front (O(1))
- `LRU.Set(key, value, ttl)` — evicts LRU entry when at capacity
- `LRU.Delete(key)`, `LRU.InvalidatePrefix(prefix)`, `LRU.Flush()`, `LRU.Len()`

**`internal/retry` package**

- `Do(ctx, Policy, fn)` — context-aware retry loop; respects cancellation during delays
- `CalculateDelay(attempt, base, max, jitter)` — exponential back-off: `min(base * 2^(n-1), max) ± jitter`
- `Policy.ShouldRetry` — customisable via function field; falls back to `Retryable() bool` interface
- `DefaultPolicy()` — 3 attempts, 1s base, 30s max, 10% jitter

**`internal/ratelimit` package**

- Sliding-window rate limiter with per-key buckets
- `Allow(key)`, `Remaining(key)`, `Reset(key)`, `ResetAll()`
- `Inspect(key)` — non-mutating status snapshot
- `SetEnabled(bool)` — disable for tests

**`internal/httpclient` package**

- `HTTPClient` with thread-safe token management (`sync.RWMutex`)
- `Get` — automatic caching for read-only endpoints
- `Post`, `PostForm`, `Patch`, `Put`, `Delete` — cache invalidation on success
- `GetRaw` — returns `[]byte` for PDF/binary endpoints
- `InvalidateUser()`, `InvalidateCache()` — targeted cache invalidation
- Per-resource cache TTLs matching Tink API data freshness characteristics
- Non-cacheable pattern list prevents OAuth/mutation/real-time endpoints from being cached

**Test suite**

- 147 tests across 7 packages
- `client/client_test.go` — 54 tests using `httptest.Server` for full integration coverage
- `webhooks/webhooks_test.go` — 25 tests including concurrent dispatch and error isolation
- `errors/errors_test.go` — 17 tests covering all constructors and error classification
- `internal/ratelimit/ratelimit_test.go` — 14 tests including concurrent access safety
- `internal/cache/cache_test.go` — 14 tests including LRU eviction and concurrent safety
- `internal/retry/retry_test.go` — 13 tests including context cancellation and delay math
- `link/link_test.go` — 10 tests covering all 6 products and URL parameter encoding

**Documentation**

- `README.md` — quick start, all 24 namespaces table, full API reference
- `USAGE.md` — complete usage guide with runnable examples for every API
- `CHANGELOG.md` — this file
- `examples/quickstart/main.go` — end-to-end demonstration of all major features
- GoDoc comments on every exported type, function, method, and constant

---

[Unreleased]: https://github.com/iamkanishka/tink-client-go/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/iamkanishka/tink-client-go/releases/tag/v1.0.0

---

## [1.0.1] — 2026-03-23

### Changed

**Struct field alignment (zero-change in behaviour)**

Six additional structs in `types/types.go` were reordered so that fields with
higher alignment requirements come first, eliminating implicit padding bytes:

| Struct                        | Before                                          | After                                           | Saving |
| ----------------------------- | ----------------------------------------------- | ----------------------------------------------- | ------ |
| `Account`                     | `Flags []string` last                           | pointer fields first, slice second              | 8 B    |
| `BudgetsResponse`             | `[]Budget` → `string`                           | `string` → `[]Budget`                           | 8 B    |
| `CalendarEventsResponse`      | `[]CalendarEvent` → `string`                    | `string` → `[]CalendarEvent`                    | 8 B    |
| `AccountCheckReportsResponse` | `[]AccountCheckReport` → `string`               | `string` → `[]AccountCheckReport`               | 8 B    |
| `ConnectorTransactionAccount` | `[]ConnectorTransaction` → `string` → `float64` | `string` → `[]ConnectorTransaction` → `float64` | 8 B    |
| `IngestTransactionsParams`    | `[]ConnectorTransactionAccount` → `IngestType`  | `IngestType` → `[]ConnectorTransactionAccount`  | 8 B    |

JSON serialisation is unaffected because all literals use named fields.

**Code quality**

- `internal/cache/lru.go`: Removed alignment-comment padding (trailing spaces
  after field names) that caused `gofmt` to flag the file as unformatted.
- `internal/retry/retry.go`: `nolint:gosec` directive now includes an inline
  explanation (`// non-cryptographic jitter for back-off`) satisfying the
  `gocritic` `whyNoLint` rule.
- `transactions/transactions.go`: Removed hand-aligned spaces in `url.Values`
  map literals that `gofmt` does not preserve.

## [1.0.2] — 2026-04-10

### Fixed

**Code quality and linting**

- Resolved minor `golangci-lint` issues across the codebase.
- Adjusted struct field ordering to satisfy `govet` alignment checks.
- Removed unnecessary whitespace and formatting inconsistencies flagged by `gofmt`.
- Minor refactoring to improve static analysis results from `staticcheck` and `gocritic`.

### Changed

- Internal code cleanups and formatting improvements with no behavioural changes.


[1.0.1]: https://github.com/iamkanishka/tink-client-go/compare/v1.0.0...v1.0.1
[1.0.2]: https://github.com/iamkanishka/tink-client-go/compare/v1.0.1...v1.0.2


