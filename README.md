# tink-client-go

[![Go Reference](https://pkg.go.dev/badge/github.com/iamkanishka/tink-client-go.svg)](https://pkg.go.dev/github.com/iamkanishka/tink-client-go)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow)](LICENSE)

**Production-grade Go client for the [Tink Open Banking API](https://docs.tink.com).**

- Zero external dependencies (stdlib only)
- Context-propagated timeouts on every request
- Thread-safe LRU caching with per-resource TTLs
- Exponential back-off retry with jitter
- Typed error handling with `Retryable()` support
- HMAC-SHA256 webhook verification (constant-time)
- Typed webhook event dispatch
- Functional options constructor
- Full coverage of all 24 Tink API product namespaces

---

## Installation

```bash
go get github.com/iamkanishka/tink-client-go
```

Requires **Go ≥ 1.25**.

---

## All 24 service namespaces

| Field | Description |
|---|---|
| `client.Auth` | OAuth 2.0 — client credentials, code exchange, token refresh, delegation |
| `client.Accounts` | Bank accounts and balances (cached 5 min) |
| `client.Transactions` | Transaction listing with filtering |
| `client.TransactionsOneTimeAccess` | Single-authorization flow |
| `client.TransactionsContinuousAccess` | Persistent user with recurring sync |
| `client.Providers` | Financial institutions (cached 1 hour) |
| `client.Categories` | Transaction categories (cached 24 hours) |
| `client.Statistics` | Aggregated financial stats (cached 1 hour) |
| `client.Users` | User and credential management |
| `client.Investments` | Investment accounts and holdings |
| `client.Loans` | Loan and mortgage accounts |
| `client.Budgets` | Budget creation, tracking, and history |
| `client.CashFlow` | Income vs expense summaries |
| `client.FinancialCalendar` | Calendar events, attachments, reconciliation |
| `client.AccountCheck` | Account ownership verification |
| `client.BalanceCheck` | Real-time balance refresh |
| `client.BusinessAccountCheck` | Business account verification |
| `client.IncomeCheck` | Income verification reports |
| `client.ExpenseCheck` | Expense analysis reports |
| `client.RiskInsights` | Financial risk scoring |
| `client.RiskCategorisation` | Transaction-level risk categories |
| `client.Connector` | Ingest your own account/transaction data |
| `client.Link` | Build Tink Link URLs for all 6 products |
| `client.Connectivity` | Monitor provider and credential health |

---

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/iamkanishka/tink-client-go/client"
    "github.com/iamkanishka/tink-client-go/types"
)

func main() {
    ctx := context.Background()

    tink, err := client.NewWithOptions(
        client.WithCredentials(
            os.Getenv("TINK_CLIENT_ID"),
            os.Getenv("TINK_CLIENT_SECRET"),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Acquire token and set it automatically
    if err := tink.Authenticate(ctx, "accounts:read transactions:read"); err != nil {
        log.Fatal(err)
    }

    accounts, err := tink.Accounts.ListAccounts(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    for _, acc := range accounts.Accounts {
        fmt.Println(acc.Name, acc.Type)
    }
}
```

---

## Configuration

```go
// Struct-based
tink, err := client.New(client.Config{
    ClientID:       os.Getenv("TINK_CLIENT_ID"),
    ClientSecret:   os.Getenv("TINK_CLIENT_SECRET"),
    Timeout:        15 * time.Second,
    MaxRetries:     3,
    DisableCache:   false,
    CacheMaxSize:   512,
    DefaultHeaders: map[string]string{"X-Request-ID": requestID},
})

// Functional options
tink, err := client.NewWithOptions(
    client.WithCredentials(clientID, clientSecret),
    client.WithTimeout(15 * time.Second),
    client.WithMaxRetries(5),
    client.WithHeader("X-Correlation-ID", correlationID),
    client.WithHTTPClient(customHTTPClient),
    client.WithDisableCache(),
)
```

Environment variables are read automatically when Config fields are empty:
- `TINK_CLIENT_ID`
- `TINK_CLIENT_SECRET`
- `TINK_ACCESS_TOKEN`
- `TINK_BASE_URL`

---

## Authentication

```go
// Client credentials (server-to-server)
if err := tink.Authenticate(ctx, "accounts:read transactions:read"); err != nil {
    log.Fatal(err)
}

// Authorization code exchange (after user redirect)
token, err := tink.Auth.ExchangeCode(ctx, clientID, clientSecret, code)
tink.SetAccessToken(token.AccessToken)

// Token refresh
token, err = tink.Auth.RefreshAccessToken(ctx, clientID, clientSecret, token.RefreshToken)

// Build Tink OAuth authorization URL for user redirect
authURL := tink.Auth.BuildAuthorizationURL(types.AuthorizationURLOptions{
    ClientID:    clientID,
    RedirectURI: "https://yourapp.com/callback",
    Scope:       "accounts:read",
    Market:      "GB",
})

// Create authorization grant for a user
grant, err := tink.Auth.CreateAuthorization(ctx, types.CreateAuthorizationParams{
    UserID: userID,
    Scope:  "accounts:read transactions:read",
})

// Validate current token
valid := tink.Auth.ValidateToken(ctx) // bool
```

---

## Accounts and Transactions

```go
// List accounts with filter
accounts, err := tink.Accounts.ListAccounts(ctx, &types.AccountsListOptions{
    TypeIn: []string{"CHECKING", "SAVINGS"},
    PaginationOptions: types.PaginationOptions{PageSize: 50},
})

// Single account
account, err := tink.Accounts.GetAccount(ctx, "acc_id")

// Balances
balances, err := tink.Accounts.GetBalances(ctx, "acc_id")

// Transactions
txResp, err := tink.Transactions.ListTransactions(ctx, &types.TransactionsListOptions{
    BookedDateGte: "2024-01-01",
    BookedDateLte: "2024-12-31",
    StatusIn:      []string{"BOOKED"},
    PaginationOptions: types.PaginationOptions{PageSize: 100},
})
```

---

## Continuous Access Flow

```go
// 1. Create a permanent Tink user
user, err := tink.TransactionsContinuousAccess.CreateUser(ctx,
    transactions.CreateUserParams{
        ExternalUserID: "your_user_id",
        Locale:         "en_US",
        Market:         "GB",
    })

// 2. Grant user Tink Link access
grant, err := tink.TransactionsContinuousAccess.GrantUserAccess(ctx,
    transactions.GrantUserAccessParams{
        UserID: user.UserID,
        IDHint: "user@example.com",
        Scope:  "authorization:read,credentials:read,credentials:write,providers:read,user:read",
    })

// 3. Build Tink Link URL → redirect user
tinkLink := tink.TransactionsContinuousAccess.BuildTinkLink(grant.Code,
    transactions.BuildTinkLinkOptions{
        ClientID:    clientID,
        RedirectURI: "https://yourapp.com/callback",
        Market:      "GB",
        Locale:      "en_US",
    })

// 4. Create data access authorization
auth, err := tink.TransactionsContinuousAccess.CreateAuthorization(ctx, user.UserID, "accounts:read transactions:read")

// 5. Exchange for user access token
token, err := tink.TransactionsContinuousAccess.GetUserAccessToken(ctx, clientID, clientSecret, auth.Code)
tink.SetAccessToken(token.AccessToken)

// 6. Fetch data
accounts, err := tink.TransactionsContinuousAccess.ListAccounts(ctx, nil)
transactions, err := tink.TransactionsContinuousAccess.ListTransactions(ctx, nil)
```

---

## Account Check

```go
// One-time verification
session, err := tink.AccountCheck.CreateSession(ctx, types.CreateSessionParams{
    User:   types.AccountCheckUser{FirstName: "Jane", LastName: "Smith"},
    Market: "GB",
})
url := tink.AccountCheck.BuildLinkURL(session, accountcheck.BuildLinkURLOptions{
    ClientID: clientID, Market: "GB",
})
// → Redirect user to url
report, err := tink.AccountCheck.GetReport(ctx, reportID)
// report.Verification.Status: "MATCH" | "NO_MATCH" | "INDETERMINATE"
pdf, err := tink.AccountCheck.GetReportPDF(ctx, reportID, "standard-1.0")

// Continuous access
user, err := tink.AccountCheck.CreateUser(ctx, accountcheck.CreateUserParams{ExternalUserID: "u1", Market: "GB", Locale: "en_US"})
grant, err := tink.AccountCheck.GrantUserAccess(ctx, types.GrantUserAccessParams{UserID: user.UserID, IDHint: "hint", Scope: scope}, clientID)
link := tink.AccountCheck.BuildContinuousAccessLink(grant.Code, types.ContinuousAccessLinkOptions{ClientID: clientID, Market: "GB", Locale: "en_US", RedirectURI: redirectURI})
parties, err := tink.AccountCheck.GetAccountParties(ctx, "acc_id")
identities, err := tink.AccountCheck.ListIdentities(ctx)
```

---

## Balance Check

```go
// Initiate async refresh
refresh, err := tink.BalanceCheck.RefreshBalance(ctx, "account_id")

// Poll for completion
for {
    status, err := tink.BalanceCheck.GetRefreshStatus(ctx, refresh.BalanceRefreshID)
    if status.Status == types.BalanceRefreshCompleted || status.Status == types.BalanceRefreshFailed {
        break
    }
    time.Sleep(time.Second)
}

// Read updated balance
balance, err := tink.BalanceCheck.GetAccountBalance(ctx, "account_id")

// Build Tink Link for initial connection
link := tink.BalanceCheck.BuildAccountCheckLink("grant_code", types.BuildAccountCheckLinkOptions{
    ClientID: clientID, Market: "SE", RedirectURI: redirectURI,
    Test: false, State: "csrf_token",
})
```

---

## Finance Management

```go
// Budgets
budget, err := tink.Budgets.CreateBudget(ctx, types.CreateBudgetParams{
    Title: "Marketing",
    Type:  types.BudgetTypeExpense,
    TargetAmount: types.TargetAmount{
        Value:        types.ExactAmount{UnscaledValue: 50000, Scale: 2},
        CurrencyCode: "GBP",
    },
    Recurrence: types.BudgetRecurrence{Frequency: types.BudgetFrequencyMonthly, Start: "2024-01-01"},
})
history, err := tink.Budgets.GetBudgetHistory(ctx, budget.ID)

// Cash Flow
flow, err := tink.CashFlow.GetSummaries(ctx, types.CashFlowOptions{
    Resolution: types.CashFlowResolutionMonthly,
    FromGte: "2024-01-01",
    ToLte:   "2024-12-31",
})

// Financial Calendar
event, err := tink.FinancialCalendar.CreateEvent(ctx, types.CreateCalendarEventParams{
    Title:   "Electricity Bill",
    DueDate: "2024-02-15",
    EventAmount: &types.CalendarEventAmount{
        CurrencyCode: "GBP",
        Value:        types.ExactAmount{UnscaledValue: 12500, Scale: 2},
    },
})
_, err = tink.FinancialCalendar.CreateReconciliation(ctx, event.ID, map[string]interface{}{"transactionId": "txn_1"})
err = tink.FinancialCalendar.DeleteEvent(ctx, event.ID, types.RecurringAll)
```

---

## Tink Link URL Builder

```go
// All 6 products
u := tink.Link.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
    ClientID: clientID, RedirectURI: redirectURI, Market: "GB", Locale: "en_US",
    AuthorizationCode: code,
})

tink.Link.BuildURL(types.LinkProductPayment, types.LinkURLOptions{
    ClientID: clientID, RedirectURI: redirectURI, Market: "SE", Locale: "sv_SE",
    PaymentRequestID: payID,
})

// Sandbox test mode
tink.Link.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
    ClientID: clientID, RedirectURI: redirectURI, Market: "GB", Locale: "en_US",
    AuthorizationCode: code, Test: true, InputProvider: "uk-ob-barclays",
})

// Convenience wrappers
tink.Link.TransactionsURL(code, opts)
tink.Link.AccountCheckURL(code, opts)
tink.Link.PaymentURL(paymentRequestID, opts)
```

---

## Webhooks

```go
// Create handler
wh := tink.NewWebhookHandler(os.Getenv("TINK_WEBHOOK_SECRET"))

// Register typed handlers (chainable)
wh.On(types.WebhookEventCredentialsUpdated, func(ctx context.Context, e *types.WebhookEvent) error {
    log.Printf("credentials updated: user %s", e.Data["userId"])
    return nil
}).
On(types.WebhookEventCredentialsRefreshFailed, func(ctx context.Context, e *types.WebhookEvent) error {
    return notifyUser(e.Data["userId"].(string))
}).
OnAll(func(ctx context.Context, e *types.WebhookEvent) error {
    log.Printf("webhook: %s", e.Type) // wildcard
    return nil
})

// net/http handler
http.HandleFunc("/webhooks/tink", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    sig := r.Header.Get("X-Tink-Signature")
    if err := wh.HandleRequest(r.Context(), body, sig); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

---

## Error Handling

```go
import tinkErrors "github.com/iamkanishka/tink-client-go/errors"

result, err := tink.Accounts.ListAccounts(ctx, nil)
if err != nil {
    var te *tinkErrors.TinkError
    if errors.As(err, &te) {
        switch te.Type {
        case types.ErrorTypeAuthentication:
            // Token expired — re-authenticate
            tink.Authenticate(ctx, scope)
        case types.ErrorTypeRateLimit:
            // Back off
            time.Sleep(time.Minute)
        case types.ErrorTypeNetwork, types.ErrorTypeTimeout:
            // Retryable — the client retries automatically, but you can check:
            fmt.Println("Is retryable:", te.Retryable()) // true
        }
        fmt.Println(te.StatusCode) // 401 | 429 | 500 | 0
        fmt.Println(te.ErrorCode)  // "TOKEN_INVALID" etc.
        fmt.Println(te.Format())   // "[401] Unauthorized (TOKEN_INVALID)"
        fmt.Println(te.RequestID)  // for Tink support tickets
    }
}
```

---

## Cache Management

```go
tink.ClearCache()                             // clear everything
tink.InvalidateCache("/api/v1/providers")     // clear by path prefix
```

---

## Package Layout

```
tink-client-go/
├── client/          # TinkClient — main entry point
├── types/           # All domain types and constants
├── errors/          # TinkError with typed error classification
├── auth/            # OAuth 2.0 flows
├── accounts/        # Accounts and balances
├── transactions/    # Transactions, one-time, continuous access
├── providers/       # Financial institutions
├── categories/      # Transaction categories
├── statistics/      # Aggregated financial stats
├── users/           # User and credential management
├── investments/     # Investment accounts and holdings
├── loans/           # Loan and mortgage accounts
├── budgets/         # Budget management
├── cashflow/        # Cash flow summaries
├── calendar/        # Financial calendar
├── accountcheck/    # Account ownership verification
├── balancecheck/    # Real-time balance refresh
├── reports/         # Income, expense, risk, business reports
├── connector/       # Data ingestion
├── link/            # Tink Link URL builder
├── connectivity/    # Provider and credential health
├── webhooks/        # HMAC-SHA256 verification + typed dispatch
├── internal/
│   ├── cache/       # Thread-safe LRU cache with TTL
│   ├── httpclient/  # HTTP client with retry, cache, token management
│   ├── retry/       # Exponential back-off with jitter
│   └── ratelimit/   # Sliding window rate limiter
└── examples/
    └── quickstart/  # Complete working example
```

---

## Running Tests

```bash
go test ./...
go test ./... -race -count=1
go test ./webhooks/... -v
```

---

## License

MIT © Kanishka Naik
