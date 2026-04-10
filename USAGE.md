# tink-client-go Usage Guide

A production-grade Go client for the [Tink Open Banking API](https://docs.tink.com/api).  
Module: `github.com/iamkanishka/tink-client-go` · Requires Go ≥ 1.25 · Zero external dependencies.

---

## Table of Contents

1. [Installation](#installation)
2. [Creating a Client](#creating-a-client)
3. [Environment Variables](#environment-variables)
4. [Authentication](#authentication)
5. [Error Handling](#error-handling)
6. [Accounts](#accounts)
7. [Transactions](#transactions)
8. [Providers](#providers)
9. [Categories](#categories)
10. [Statistics](#statistics)
11. [Users & Credentials](#users--credentials)
12. [Investments](#investments)
13. [Loans](#loans)
14. [Budgets](#budgets)
15. [Cash Flow](#cash-flow)
16. [Financial Calendar](#financial-calendar)
17. [Account Check](#account-check)
18. [Balance Check](#balance-check)
19. [Risk & Verification Reports](#risk--verification-reports)
20. [Connector (Data Ingestion)](#connector-data-ingestion)
21. [Tink Link URL Builder](#tink-link-url-builder)
22. [Connectivity Monitoring](#connectivity-monitoring)
23. [Webhooks](#webhooks)
24. [Caching](#caching)
25. [Retry Behaviour](#retry-behaviour)
26. [Testing](#testing)
27. [Package Layout](#package-layout)

---

## Installation

```bash
go get github.com/iamkanishka/tink-client-go
```

---

## Creating a Client

### Struct-based construction

```go
import "github.com/iamkanishka/tink-client-go/client"

c, err := client.New(client.Config{
    ClientID:     "your_client_id",
    ClientSecret: "your_client_secret",
    Timeout:      15 * time.Second,
    MaxRetries:   3,
})
if err != nil {
    log.Fatal(err)
}
```

### Functional options (recommended)

```go
c, err := client.NewWithOptions(
    client.WithCredentials("your_client_id", "your_client_secret"),
    client.WithTimeout(15 * time.Second),
    client.WithMaxRetries(3),
    client.WithHeader("X-Request-ID", generateRequestID()),
    client.WithHeader("X-Correlation-ID", correlationID),
)
```

All available options:

| Option | Description |
|---|---|
| `WithCredentials(id, secret)` | Set client ID and secret |
| `WithAccessToken(token)` | Use a pre-existing bearer token |
| `WithBaseURL(url)` | Override `https://api.tink.com` |
| `WithTimeout(d)` | Per-request timeout (default: 30s) |
| `WithMaxRetries(n)` | Max retry attempts (default: 3) |
| `WithHTTPClient(hc)` | Provide a custom `*http.Client` |
| `WithHeader(key, value)` | Add a default header to every request |
| `WithDisableCache()` | Disable the in-memory LRU cache |

---

## Environment Variables

When the corresponding `Config` field is empty, the client reads:

| Variable | Field |
|---|---|
| `TINK_CLIENT_ID` | `Config.ClientID` |
| `TINK_CLIENT_SECRET` | `Config.ClientSecret` |
| `TINK_ACCESS_TOKEN` | `Config.AccessToken` |
| `TINK_BASE_URL` | `Config.BaseURL` |

```bash
export TINK_CLIENT_ID=your_client_id
export TINK_CLIENT_SECRET=your_client_secret
```

```go
// Reads credentials from environment automatically
c, err := client.New(client.Config{})
```

---

## Authentication

### Client credentials (server-to-server)

Acquires a token and sets it on the client automatically:

```go
ctx := context.Background()

err := c.Authenticate(ctx, "accounts:read transactions:read")
if err != nil {
    log.Fatal(err)
}
// All subsequent API calls now use the acquired token.
fmt.Println("Token set:", c.AccessToken() != "")
```

### Authorization code exchange (after user redirect)

```go
// 1. Build the redirect URL
authURL := c.Auth.BuildAuthorizationURL(types.AuthorizationURLOptions{
    ClientID:    os.Getenv("TINK_CLIENT_ID"),
    RedirectURI: "https://yourapp.com/callback",
    Scope:       "accounts:read transactions:read",
    Market:      "GB",
    State:       csrfToken,
})
// Redirect your user to authURL

// 2. After the user returns with ?code=...
token, err := c.Auth.ExchangeCode(ctx, clientID, clientSecret, code)
if err != nil {
    log.Fatal(err)
}
c.SetAccessToken(token.AccessToken)
```

### Token refresh

```go
token, err := c.Auth.RefreshAccessToken(ctx, clientID, clientSecret, token.RefreshToken)
if err != nil {
    log.Fatal(err)
}
c.SetAccessToken(token.AccessToken)
```

### Token expiry helpers

```go
tokenInfo := client.ParseToken(tokenResponse)
fmt.Println("Expires at:", tokenInfo.ExpiresAt)
fmt.Println("Scope:", tokenInfo.Scope)

// IsExpired applies a 5-minute safety buffer
if client.IsExpired(tokenInfo.ExpiresAt) {
    // Re-authenticate before the token actually expires
    c.Authenticate(ctx, scope)
}
```

### Validate current token

```go
valid := c.Auth.ValidateToken(ctx) // bool — probes /api/v1/user
```

### Authorization grants (for user-scoped access)

```go
// Create a grant → returns a short-lived code
grant, err := c.Auth.CreateAuthorization(ctx, types.CreateAuthorizationParams{
    UserID: tinkUserID,
    Scope:  "accounts:read transactions:read",
})

// Delegate a grant to an actor client (for Tink Link)
delegated, err := c.Auth.DelegateAuthorization(ctx, types.DelegateAuthorizationParams{
    UserID:        tinkUserID,
    IDHint:        "user@example.com",
    Scope:         "authorization:read,credentials:read",
    ActorClientID: clientID,
})
```

---

## Error Handling

All client methods return `*errors.TinkError` on failure. Use `errors.As` to inspect:

```go
import (
    "errors"
    tinkErrors "github.com/iamkanishka/tink-client-go/errors"
    "github.com/iamkanishka/tink-client-go/types"
)

resp, err := c.Accounts.ListAccounts(ctx, nil)
if err != nil {
    var te *tinkErrors.TinkError
    if errors.As(err, &te) {
        switch te.Type {
        case types.ErrorTypeAuthentication: // 401
            log.Println("Token expired — re-authenticating")
            c.Authenticate(ctx, scope)

        case types.ErrorTypeRateLimit: // 429
            log.Println("Rate limited — backing off")
            time.Sleep(60 * time.Second)

        case types.ErrorTypeNetwork, types.ErrorTypeTimeout:
            log.Printf("Transient error (retryable=%v): %v", te.Retryable(), te)

        case types.ErrorTypeValidation: // 400
            log.Printf("Bad request: %s (%s)", te.Message, te.ErrorCode)

        default:
            log.Printf("API error [%d]: %s", te.StatusCode, te.Format())
        }

        // Useful fields for support tickets:
        _ = te.StatusCode   // e.g. 401, 429, 500
        _ = te.ErrorCode    // e.g. "TOKEN_INVALID"
        _ = te.RequestID    // from the Tink API response header
        _ = te.Retryable()  // true for network_error, timeout, 5xx, 429
        _ = te.Format()     // "[401] Unauthorized (TOKEN_INVALID)"
    }
}
```

### Error types

| `types.ErrorType` | HTTP Status | `Retryable()` | Description |
|---|---|---|---|
| `ErrorTypeAuthentication` | 401 | false | Invalid or expired token |
| `ErrorTypeRateLimit` | 429 | **true** | Too many requests |
| `ErrorTypeValidation` | 400 | false | Bad request parameters |
| `ErrorTypeAPI` | 4xx/5xx | true for 5xx | General API error |
| `ErrorTypeNetwork` | — | **true** | Network-level failure |
| `ErrorTypeTimeout` | — | **true** | Request deadline exceeded |
| `ErrorTypeDecode` | — | false | JSON parse failure |

---

## Accounts

```go
// List all accounts (returns CHECKING, SAVINGS, CREDIT_CARD, etc.)
resp, err := c.Accounts.ListAccounts(ctx, nil)

// Filter by type with pagination
resp, err = c.Accounts.ListAccounts(ctx, &types.AccountsListOptions{
    TypeIn: []string{"CHECKING", "SAVINGS"},
    PaginationOptions: types.PaginationOptions{
        PageSize:  50,
        PageToken: resp.NextPageToken, // for subsequent pages
    },
})
for _, acc := range resp.Accounts {
    fmt.Printf("%s (%s): %s %s\n",
        acc.Name, acc.Type,
        acc.Balances.Booked.Amount.Value,
        acc.Balances.Booked.Amount.CurrencyCode,
    )
}

// Single account
acc, err := c.Accounts.GetAccount(ctx, "acc_id_here")

// Real-time balances
balances, err := c.Accounts.GetBalances(ctx, "acc_id_here")
fmt.Println("Booked:", balances.Booked.Amount.Value)
fmt.Println("Available:", balances.Available.Amount.Value)
```

---

## Transactions

### Standard access

```go
resp, err := c.Transactions.ListTransactions(ctx, &types.TransactionsListOptions{
    BookedDateGte: "2024-01-01",
    BookedDateLte: "2024-12-31",
    StatusIn:      []string{"BOOKED"},
    PaginationOptions: types.PaginationOptions{PageSize: 100},
})
for _, txn := range resp.Transactions {
    fmt.Printf("%s  %s %s\n",
        txn.Dates.Booked,
        txn.Amount.Value,
        txn.Amount.CurrencyCode,
    )
}
```

### One-time access flow

```go
// After user completes Tink Link and you have their user access token:
c.SetAccessToken(userToken)

accounts, err  := c.TransactionsOneTimeAccess.ListAccounts(ctx, nil)
txns, err       := c.TransactionsOneTimeAccess.ListTransactions(ctx, nil)
```

### Continuous access flow

```go
// Step 1 — Create a permanent Tink user (once per customer, store the ID)
user, err := c.TransactionsContinuousAccess.CreateUser(ctx, transactions.CreateUserParams{
    ExternalUserID: "your_internal_user_id",
    Market:         "GB",
    Locale:         "en_US",
})
db.SaveTinkUserID(userID, user.UserID)

// Step 2 — Grant Tink Link access → build redirect URL
grant, err := c.TransactionsContinuousAccess.GrantUserAccess(ctx, transactions.GrantUserAccessParams{
    UserID: user.UserID,
    IDHint: "user@example.com",
    Scope:  "authorization:read,credentials:read,credentials:write,credentials:refresh,providers:read,user:read",
})
tinkLink := c.TransactionsContinuousAccess.BuildTinkLink(grant.Code, transactions.BuildTinkLinkOptions{
    ClientID:    clientID,
    RedirectURI: "https://yourapp.com/callback",
    Market:      "GB",
    Locale:      "en_US",
})
// Redirect user to tinkLink

// Step 3 — After bank connection: create data authorization
auth, err := c.TransactionsContinuousAccess.CreateAuthorization(ctx, tinkUserID, "accounts:read transactions:read")
token, err := c.TransactionsContinuousAccess.GetUserAccessToken(ctx, clientID, clientSecret, auth.Code)

// Step 4 — Fetch data with user token
c.SetAccessToken(token.AccessToken)
accounts, err   := c.TransactionsContinuousAccess.ListAccounts(ctx, nil)
txns, err        := c.TransactionsContinuousAccess.ListTransactions(ctx, &types.TransactionsListOptions{
    BookedDateGte: "2024-01-01",
})
```

---

## Providers

```go
// List providers in a market (cached 1 hour)
resp, err := c.Providers.ListProviders(ctx, &types.ProvidersListOptions{
    Market:       "GB",
    Capabilities: []string{"CHECKING_ACCOUNTS"},
})
activeProviders := make([]types.Provider, 0)
for _, p := range resp.Providers {
    if p.Status == "ENABLED" {
        activeProviders = append(activeProviders, p)
    }
}

// Single provider
provider, err := c.Providers.GetProvider(ctx, "uk-ob-barclays")
fmt.Println(provider.DisplayName, provider.Market, provider.Status)
```

---

## Categories

```go
// List all categories for a locale (cached 24 hours)
resp, err := c.Categories.ListCategories(ctx, "en_US")
for _, cat := range resp.Categories {
    fmt.Println(cat.Code, cat.DisplayName)
}

// Single category
cat, err := c.Categories.GetCategory(ctx, "expenses:food.groceries", "en_US")
```

---

## Statistics

```go
resp, err := c.Statistics.GetStatistics(ctx, types.StatisticsOptions{
    PeriodGte:  "2024-01-01",
    PeriodLte:  "2024-12-31",
    Resolution: "MONTHLY", // DAILY, WEEKLY, MONTHLY, YEARLY
})
for _, p := range resp.Periods {
    fmt.Printf("%s: income=%s expenses=%s\n",
        p.Period,
        p.Income.Amount.Value,
        p.Expenses.Amount.Value,
    )
}

// Filter by category
catStats, err := c.Statistics.GetCategoryStatistics(ctx,
    "expenses:food.groceries",
    types.StatisticsOptions{PeriodGte: "2024-01-01", PeriodLte: "2024-12-31"},
)

// Filter by account
accStats, err := c.Statistics.GetAccountStatistics(ctx,
    "acc_id_here",
    types.StatisticsOptions{PeriodGte: "2024-01-01", PeriodLte: "2024-12-31"},
)
```

---

## Users & Credentials

```go
// Create a user
user, err := c.Users.CreateUser(ctx, types.CreateUserParams{
    ExternalUserID: "your_internal_id",
    Locale:         "en_US",
    Market:         "GB",
})

// Delete a user (irreversible)
err = c.Users.DeleteUser(ctx, tinkUserID)

// List bank connections (cached 30 seconds)
resp, err := c.Users.ListCredentials(ctx)
for _, cred := range resp.Credentials {
    fmt.Printf("%s — %s (%s)\n", cred.ProviderName, cred.Status, cred.StatusUpdated)
}

// Single credential
cred, err := c.Users.GetCredential(ctx, "cred_id_here")

// Trigger data refresh from bank
refreshed, err := c.Users.RefreshCredential(ctx, "cred_id_here")

// Delete a credential
err = c.Users.DeleteCredential(ctx, "cred_id_here")

// Create authorization grant
grant, err := c.Users.CreateAuthorization(ctx, tinkUserID, "accounts:read")
token, err  := c.Users.GetUserAccessToken(ctx, clientID, clientSecret, grant.Code)
```

---

## Investments

```go
// List investment accounts (brokerage, ISA, pension, etc.)
resp, err := c.Investments.ListAccounts(ctx)
for _, acc := range resp.Accounts {
    fmt.Println(acc.Name, acc.Type, acc.Balance.Amount.Value)
}

// Single investment account
acc, err := c.Investments.GetAccount(ctx, "inv_acc_id")

// Portfolio holdings
holdings, err := c.Investments.GetHoldings(ctx, "inv_acc_id")
for _, h := range holdings.Holdings {
    fmt.Printf("%s (%s): qty=%.2f value=%s\n",
        h.Instrument.Symbol, h.Instrument.Type,
        *h.Quantity, h.MarketValue.Amount.Value,
    )
}
```

---

## Loans

```go
// List loan and mortgage accounts
resp, err := c.Loans.ListAccounts(ctx)
for _, loan := range resp.Accounts {
    fmt.Printf("%s: balance=%s rate=%.2f%%\n",
        loan.Name, loan.Balance.Amount.Value, *loan.InterestRate,
    )
}

// Single loan account
loan, err := c.Loans.GetAccount(ctx, "loan_id_here")
fmt.Println("Matures:", loan.MaturityDate)
```

---

## Budgets

```go
// Create a monthly expense budget of £500
budget, err := c.Budgets.CreateBudget(ctx, types.CreateBudgetParams{
    Title: "Office Supplies",
    Type:  types.BudgetTypeExpense,
    TargetAmount: types.TargetAmount{
        Value:        types.ExactAmount{UnscaledValue: 50000, Scale: 2}, // 500.00
        CurrencyCode: "GBP",
    },
    Recurrence: types.BudgetRecurrence{
        Frequency: types.BudgetFrequencyMonthly,
        Start:     "2024-01-01",
    },
})

// Fetch a budget
b, err := c.Budgets.GetBudget(ctx, budget.ID)
fmt.Println("Status:", b.ProgressStatus) // ON_TRACK, OVER_BUDGET, etc.

// Spending history across periods
history, err := c.Budgets.GetBudgetHistory(ctx, budget.ID)
for _, h := range history.History {
    fmt.Printf("%s: spent=%s remaining=%s\n",
        h.Period, h.Spent.Value.UnscaledValue, h.Remaining.Value.UnscaledValue,
    )
}

// List with filter
resp, err := c.Budgets.ListBudgets(ctx, &types.BudgetsListOptions{
    ProgressStatusIn: []string{"OVER_BUDGET"},
})

// Patch a budget
updated, err := c.Budgets.UpdateBudget(ctx, budget.ID, map[string]interface{}{
    "title": "Office Supplies Q2",
})

// Delete
err = c.Budgets.DeleteBudget(ctx, budget.ID)
```

---

## Cash Flow

```go
resp, err := c.CashFlow.GetSummaries(ctx, types.CashFlowOptions{
    Resolution: types.CashFlowResolutionMonthly,
    FromGte:    "2024-01-01",
    ToLte:      "2024-12-31",
})
for _, p := range resp.Periods {
    net := p.NetAmount
    fmt.Printf("%s → %s: income=%s expenses=%s net=%s\n",
        p.PeriodStart, p.PeriodEnd,
        p.Income.Amount.Value, p.Expenses.Amount.Value,
        net.Value,
    )
}
```

Resolutions: `CashFlowResolutionDaily`, `CashFlowResolutionWeekly`,
`CashFlowResolutionMonthly`, `CashFlowResolutionYearly`.

---

## Financial Calendar

```go
// Create an event (bill, salary, subscription, etc.)
event, err := c.FinancialCalendar.CreateEvent(ctx, types.CreateCalendarEventParams{
    Title:   "Electricity Bill",
    DueDate: "2024-02-15",
    EventAmount: &types.CalendarEventAmount{
        CurrencyCode: "GBP",
        Value:        types.ExactAmount{UnscaledValue: 12500, Scale: 2}, // £125.00
    },
})

// List events in a date range
q := url.Values{}
q.Set("dueDateGte", "2024-02-01")
q.Set("dueDateLte", "2024-02-28")
events, err := c.FinancialCalendar.ListEvents(ctx, q)

// Update an event
updated, err := c.FinancialCalendar.UpdateEvent(ctx, event.ID, map[string]interface{}{
    "title": "Electricity Bill — Feb 2024",
})

// Delete — choose which occurrences
err = c.FinancialCalendar.DeleteEvent(ctx, event.ID, types.RecurringSingle)
err = c.FinancialCalendar.DeleteEvent(ctx, event.ID, types.RecurringThisAndFollowing)
err = c.FinancialCalendar.DeleteEvent(ctx, event.ID, types.RecurringAll)

// Attach an invoice PDF URL
_, err = c.FinancialCalendar.AddAttachment(ctx, event.ID, map[string]interface{}{
    "title": "Invoice #2024-02",
    "url":   "https://invoices.example.com/feb-2024.pdf",
})

// Remove an attachment
err = c.FinancialCalendar.DeleteAttachment(ctx, event.ID, "attachment_id_here")

// Set up a recurring group (e.g. monthly for 12 months)
_, err = c.FinancialCalendar.CreateRecurringGroup(ctx, event.ID, map[string]interface{}{
    "rrulePattern": "FREQ=MONTHLY;COUNT=12",
})

// Reconcile with an actual transaction
_, err = c.FinancialCalendar.CreateReconciliation(ctx, event.ID, map[string]interface{}{
    "transactionId": "txn_id_here",
})

// Get reconciliation suggestions from AI
suggestions, err := c.FinancialCalendar.GetReconciliationSuggestions(ctx, event.ID)

// Remove a reconciliation link
err = c.FinancialCalendar.DeleteReconciliation(ctx, event.ID, "txn_id_here")
```

---

## Account Check

### One-time verification

```go
// Step 1 — Create a session with user identity
session, err := c.AccountCheck.CreateSession(ctx, types.CreateSessionParams{
    User:        types.AccountCheckUser{FirstName: "Jane", LastName: "Smith"},
    Market:      "GB",
    RedirectURI: "https://yourapp.com/account-check/callback",
})

// Step 2 — Build Tink Link URL and redirect user
linkURL := c.AccountCheck.BuildLinkURL(session, accountcheck.BuildLinkURLOptions{
    ClientID: clientID,
    Market:   "GB",
})
// Redirect user → after bank auth, Tink redirects to your callback with ?account_verification_report_id=...

// Step 3 — Fetch the verification report
report, err := c.AccountCheck.GetReport(ctx, reportID)
switch report.Verification.Status {
case "MATCH":
    fmt.Println("Name matched — account verified")
case "NO_MATCH":
    fmt.Println("Name did not match bank records")
case "INDETERMINATE":
    fmt.Println("Insufficient data from bank")
}

// Download as PDF
pdf, err := c.AccountCheck.GetReportPDF(ctx, reportID, "standard-1.0")
os.WriteFile("report.pdf", pdf, 0644)

// List all reports
reports, err := c.AccountCheck.ListReports(ctx, nil)
```

### Continuous access

```go
// Create a persistent user
user, err := c.AccountCheck.CreateUser(ctx, accountcheck.CreateUserParams{
    ExternalUserID: "your_user_id",
    Market:         "GB",
    Locale:         "en_US",
})

// Delegate Tink Link access
grant, err := c.AccountCheck.GrantUserAccess(ctx, types.GrantUserAccessParams{
    UserID: user.UserID,
    IDHint: "user@example.com",
    Scope:  "authorization:read,credentials:read,credentials:write",
}, clientID)

// Build continuous access link
link := c.AccountCheck.BuildContinuousAccessLink(grant.Code, types.ContinuousAccessLinkOptions{
    ClientID:    clientID,
    Market:      "GB",
    Locale:      "en_US",
    RedirectURI: "https://yourapp.com/callback",
    Products:    "ACCOUNT_CHECK,TRANSACTIONS",
})

// Get account parties (owners, co-owners)
parties, err := c.AccountCheck.GetAccountParties(ctx, "acc_id_here")

// Get identity records
identities, err := c.AccountCheck.ListIdentities(ctx)

// Delete user
err = c.AccountCheck.DeleteUser(ctx, tinkUserID)
```

---

## Balance Check

```go
// Step 1 — Initial bank connection (one-time setup per user)
user, err := c.BalanceCheck.CreateUser(ctx, balancecheck.CreateUserParams{
    ExternalUserID: "your_user_id",
    Market:         "SE",
    Locale:         "sv_SE",
})

grant, err := c.BalanceCheck.GrantUserAccess(ctx, types.GrantUserAccessParams{
    UserID: user.UserID,
    IDHint: "user@example.com",
    Scope:  "authorization:read,credentials:read,credentials:write,accounts:read,account-verification-reports:read",
}, clientID)

link := c.BalanceCheck.BuildAccountCheckLink(grant.Code, types.BuildAccountCheckLinkOptions{
    ClientID:    clientID,
    Market:      "SE",
    RedirectURI: "https://yourapp.com/callback",
    Test:        false,
    State:       csrfToken,
})
// Redirect user to link

// Step 2 — On-demand balance refresh
refresh, err := c.BalanceCheck.RefreshBalance(ctx, "acc_id_here")
fmt.Println("Refresh ID:", refresh.BalanceRefreshID)

// Step 3 — Poll for completion
for {
    status, err := c.BalanceCheck.GetRefreshStatus(ctx, refresh.BalanceRefreshID)
    if err != nil { log.Fatal(err) }
    fmt.Println("Status:", status.Status)
    if status.Status == types.BalanceRefreshCompleted || status.Status == types.BalanceRefreshFailed {
        break
    }
    time.Sleep(time.Second)
}

// Step 4 — Read updated balance
balance, err := c.BalanceCheck.GetAccountBalance(ctx, "acc_id_here")

// Consent renewal
consentGrant, err := c.BalanceCheck.GrantConsentUpdate(ctx, types.GrantUserAccessParams{
    UserID: tinkUserID,
    IDHint: "user@example.com",
    Scope:  "credentials:write,authorization:grant",
}, clientID)
consentLink := c.BalanceCheck.BuildConsentUpdateLink(consentGrant.Code, types.ConsentUpdateLinkOptions{
    ClientID:      clientID,
    CredentialsID: "cred_id_here",
    Market:        "SE",
    RedirectURI:   "https://yourapp.com/callback",
})
```

---

## Risk & Verification Reports

All report services follow the same pattern: provide a report ID (obtained from the Tink webhook or previous API call) and receive a structured report.

```go
// Income verification
income, err := c.IncomeCheck.GetReport(ctx, "report_id_here")
pdf, err    := c.IncomeCheck.GetReportPDF(ctx, "report_id_here")
os.WriteFile("income.pdf", pdf, 0644)

// Expense analysis
expense, err := c.ExpenseCheck.GetReport(ctx, "report_id_here")

// Risk insights (fraud/financial health scoring)
risk, err := c.RiskInsights.GetReport(ctx, "report_id_here")

// Transaction-level risk categorisation
riskCat, err := c.RiskCategorisation.GetReport(ctx, "report_id_here")

// Business account verification
biz, err := c.BusinessAccountCheck.GetReport(ctx, "report_id_here")
fmt.Println("Status:", biz.Status)
```

---

## Connector (Data Ingestion)

Use the Connector API when you already have account and transaction data and want Tink's enrichment and analytics applied.

```go
// Step 1 — Create a Tink user to own the data
user, err := c.Connector.CreateUser(ctx, connector.CreateUserParams{
    ExternalUserID: "your_internal_user_id",
    Market:         "GB",
    Locale:         "en_US",
})

// Step 2 — Ingest accounts
_, err = c.Connector.IngestAccounts(ctx, user.User_ID, types.IngestAccountsParams{
    Accounts: []types.ConnectorAccount{
        {
            ExternalID: "acc_external_001",
            Name:       "Main Checking",
            Type:       "CHECKING",
            Balance:    1500.00,
        },
        {
            ExternalID: "acc_external_002",
            Name:       "Savings",
            Type:       "SAVINGS",
            Balance:    8200.00,
        },
    },
})

// Step 3 — Ingest transactions
_, err = c.Connector.IngestTransactions(ctx, user.User_ID, types.IngestTransactionsParams{
    Type: types.IngestTypeRealTime, // or IngestTypeBatch for historical data
    TransactionAccounts: []types.ConnectorTransactionAccount{
        {
            ExternalID: "acc_external_001",
            Balance:    1485.00, // updated balance after transactions
            Transactions: []types.ConnectorTransaction{
                {
                    ExternalID:  "txn_external_001",
                    Amount:      -15.00,
                    Date:        time.Now().UnixMilli(),
                    Description: "Coffee Shop",
                    Type:        "DEFAULT",
                },
                {
                    ExternalID:  "txn_external_002",
                    Amount:      -32.50,
                    Date:        time.Now().Add(-24 * time.Hour).UnixMilli(),
                    Description: "Grocery Store",
                    Type:        "DEFAULT",
                },
            },
        },
    },
})
```

---

## Tink Link URL Builder

Build URLs for all six Tink Link products without making any API calls.

```go
// All six products
url := c.Link.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
    ClientID:          clientID,
    RedirectURI:       "https://yourapp.com/callback",
    Market:            "GB",
    Locale:            "en_US",
    AuthorizationCode: grant.Code,
})

url = c.Link.BuildURL(types.LinkProductAccountCheck, types.LinkURLOptions{
    ClientID:          clientID,
    RedirectURI:       "https://yourapp.com/callback",
    Market:            "GB",
    Locale:            "en_US",
    AuthorizationCode: grant.Code,
})

// Payment initiation (uses PaymentRequestID instead of AuthorizationCode)
url = c.Link.BuildURL(types.LinkProductPayment, types.LinkURLOptions{
    ClientID:         clientID,
    RedirectURI:      "https://yourapp.com/callback",
    Market:           "SE",
    Locale:           "sv_SE",
    PaymentRequestID: "pay_req_id_here",
})

// Other products: LinkProductIncomeCheck, LinkProductExpenseCheck, LinkProductRiskInsights

// Convenience wrappers
transURL := c.Link.TransactionsURL(grant.Code, types.LinkURLOptions{
    ClientID: clientID, RedirectURI: redirectURI, Market: "GB", Locale: "en_US",
})
acURL := c.Link.AccountCheckURL(grant.Code, types.LinkURLOptions{
    ClientID: clientID, RedirectURI: redirectURI, Market: "GB", Locale: "en_US",
})
payURL := c.Link.PaymentURL("pay_req_id", types.LinkURLOptions{
    ClientID: clientID, RedirectURI: redirectURI, Market: "SE", Locale: "sv_SE",
})

// Sandbox test mode with pre-selected provider
sandboxURL := c.Link.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
    ClientID:          clientID,
    RedirectURI:       redirectURI,
    Market:            "GB",
    Locale:            "en_US",
    AuthorizationCode: grant.Code,
    Test:              true,
    InputProvider:     "uk-ob-barclays",
})

// Add CSRF state and iframe mode
url = c.Link.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
    ClientID:          clientID,
    RedirectURI:       redirectURI,
    Market:            "GB",
    Locale:            "en_US",
    AuthorizationCode: grant.Code,
    State:             csrfToken,
    Iframe:            true,
})
```

---

## Connectivity Monitoring

```go
// List all providers in a market
resp, err := c.Connectivity.ListProvidersByMarket(ctx, "GB")
resp, err  = c.Connectivity.ListProvidersByMarketAuthenticated(ctx, "GB")

// Check if a specific provider is active
result, err := c.Connectivity.CheckProviderStatus(ctx, "uk-ob-barclays", "GB")
if !result.Active {
    log.Println("Barclays is currently unavailable")
}

// Boolean helper
active, err := c.Connectivity.ProviderOperational(ctx, "uk-ob-barclays", "GB")

// Check all user credentials
summary, err := c.Connectivity.CheckCredentialConnectivity(ctx, nil)
fmt.Printf("%d/%d credentials healthy\n", summary.Healthy, summary.Total)
for _, cred := range summary.Credentials {
    if !cred.Healthy {
        fmt.Printf("  ✗ %s — %s: %s\n",
            cred.ProviderName, cred.Status, cred.ErrorMessage,
        )
    }
}

// Filter — include only unhealthy
t := true; f := false
summary, err = c.Connectivity.CheckCredentialConnectivity(ctx, &types.ConnectivityOptions{
    IncludeHealthy:   &f,
    IncludeUnhealthy: &t,
})

// Single credential
cred, err := c.Connectivity.GetCredentialConnectivity(ctx, "cred_id_here")
fmt.Println("Healthy:", cred.Healthy, "Last refreshed:", cred.LastRefreshed)

// API health probe
if err := c.Connectivity.CheckAPIHealth(ctx); err != nil {
    alertOps("Tink API is unreachable: " + err.Error())
}
```

---

## Webhooks

### Handler setup

```go
import (
    "io"
    "net/http"
    "github.com/iamkanishka/tink-client-go/types"
)

// Create a handler (one per application)
wh := c.NewWebhookHandler(os.Getenv("TINK_WEBHOOK_SECRET"))

// Register typed event handlers (chainable)
wh.On(types.WebhookEventCredentialsUpdated, func(ctx context.Context, e *types.WebhookEvent) error {
    userID := e.Data["userId"].(string)
    log.Printf("credentials updated for user %s at %s", userID, e.Timestamp)
    return syncUserCredentials(ctx, userID)
}).
On(types.WebhookEventCredentialsRefreshFailed, func(ctx context.Context, e *types.WebhookEvent) error {
    return notifyUserToReconnect(ctx, e.Data["userId"].(string))
}).
On(types.WebhookEventCredentialsRefreshSucceeded, func(ctx context.Context, e *types.WebhookEvent) error {
    return updateLastSyncTimestamp(ctx, e.Data["userId"].(string))
}).
On(types.WebhookEventProviderConsentsRevoked, func(ctx context.Context, e *types.WebhookEvent) error {
    return handleConsentRevoked(ctx, e.Data["userId"].(string))
}).
OnAll(func(ctx context.Context, e *types.WebhookEvent) error {
    // Wildcard — called for every non-test event
    metrics.IncrementWebhookCounter(e.Type)
    return nil
})
```

All event types: `WebhookEventCredentialsUpdated`, `WebhookEventCredentialsRefreshSucceeded`,
`WebhookEventCredentialsRefreshFailed`, `WebhookEventProviderConsentsCreated`,
`WebhookEventProviderConsentsRevoked`, `WebhookEventTest`.

### net/http integration

```go
http.HandleFunc("/webhooks/tink", func(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "cannot read body", http.StatusBadRequest)
        return
    }

    sig := r.Header.Get("X-Tink-Signature")
    if err := wh.HandleRequest(r.Context(), body, sig); err != nil {
        // Invalid signature or malformed payload
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    // Return 200 immediately — handler errors are logged internally
    w.WriteHeader(http.StatusOK)
})
```

Test webhooks (`type: "test"`) are silently acknowledged and do not trigger handlers.

### Manual signature verification

```go
// When you only need to verify the signature (not dispatch):
verifier := c.NewWebhookVerifier(os.Getenv("TINK_WEBHOOK_SECRET"))
if err := verifier.Verify(rawBody, r.Header.Get("X-Tink-Signature")); err != nil {
    http.Error(w, "invalid signature", http.StatusUnauthorized)
    return
}

// Generate signatures for your own test payloads:
payload := []byte(`{"type":"credentials.updated","data":{"userId":"u1"}}`)
sig := verifier.GenerateSignatureHex(payload)
```

---

## Caching

The client maintains an in-memory LRU cache to reduce API calls for stable read-only endpoints.

| Resource | Cache TTL | Notes |
|---|---|---|
| Providers | **1 hour** | Rarely changes |
| Categories | **24 hours** | Static reference data |
| Statistics | **1 hour** | Aggregated data |
| Accounts | **5 minutes** | Changes after refresh |
| Transactions | **5 minutes** | Changes after refresh |
| Balances | **1 minute** | Real-time data |
| Credentials | **30 seconds** | Changes during auth flows |
| Users | **10 minutes** | |
| Reports | **24 hours** | Immutable once generated |

Cache is automatically invalidated:
- After every `POST`, `PATCH`, `PUT`, `DELETE` request
- Per-user when `InvalidateUser()` is called

Manual cache management:
```go
c.ClearCache()                             // clear everything
c.InvalidateCache("/api/v1/providers")     // clear by path prefix
c.InvalidateCache("/data/v2/accounts")     // force-refresh account data
```

Disable cache (e.g. for testing):
```go
c, _ := client.NewWithOptions(
    client.WithCredentials(id, secret),
    client.WithDisableCache(),
)
```

---

## Retry Behaviour

The client automatically retries requests that fail with retryable errors using exponential back-off with jitter.

**Retryable errors:** `network_error`, `timeout`, HTTP 408, 429, 500, 502, 503, 504.  
**Not retried:** 400, 401, 403, 404 (client errors are not transient).

Default policy: 3 attempts, 1 second base delay, 30 second max delay, 10% jitter.

Override max retries:
```go
c, _ := client.NewWithOptions(
    client.WithCredentials(id, secret),
    client.WithMaxRetries(5), // 5 total attempts
)
```

Back-off schedule (no jitter, for illustration):

| Attempt | Delay |
|---|---|
| 1 | immediate |
| 2 | 1 second |
| 3 | 2 seconds |
| 4 | 4 seconds |
| 5 | 8 seconds |

All retries respect the request context — if the context is cancelled, the retry loop stops immediately.

---

## Testing

The package is designed to be easy to test. Swap the underlying `http.Client` with one backed by `httptest.Server`:

```go
import "net/http/httptest"

func TestMyHandler(t *testing.T) {
    // Create a mock server
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "accounts": []map[string]interface{}{
                {"id": "acc_1", "name": "Test Account", "type": "CHECKING"},
            },
        })
    }))
    defer srv.Close()

    // Point the client at the mock server
    c, _ := client.NewWithOptions(
        client.WithBaseURL(srv.URL),
        client.WithHTTPClient(srv.Client()),
        client.WithAccessToken("test-token"),
        client.WithDisableCache(),
        client.WithMaxRetries(1),
    )

    resp, err := c.Accounts.ListAccounts(context.Background(), nil)
    if err != nil {
        t.Fatal(err)
    }
    if len(resp.Accounts) != 1 {
        t.Errorf("expected 1 account, got %d", len(resp.Accounts))
    }
}
```

Run all tests with race detection:

```bash
go test ./... -race -count=1
```

Run specific package tests:

```bash
go test ./webhooks/... -v
go test ./client/... -v -run TestAccounts
go test ./internal/... -v
```

---

## Package Layout

```
github.com/iamkanishka/tink-client-go/
├── client/          Client struct with all 24 namespaces; New, NewWithOptions
├── types/           All domain types, error type constants, webhook event constants
├── errors/          TinkError: Retryable(), Unwrap(), Format(), FromResponse()
├── auth/            GetAccessToken, ExchangeCode, RefreshAccessToken, BuildAuthorizationURL
├── accounts/        ListAccounts, GetAccount, GetBalances
├── transactions/    Service + OneTimeAccessService + ContinuousAccessService
├── providers/       ListProviders, GetProvider
├── categories/      ListCategories, GetCategory
├── statistics/      GetStatistics, GetCategoryStatistics, GetAccountStatistics
├── users/           CreateUser, DeleteUser, ListCredentials, RefreshCredential
├── investments/     ListAccounts, GetAccount, GetHoldings
├── loans/           ListAccounts, GetAccount
├── budgets/         CreateBudget, GetBudget, GetBudgetHistory, ListBudgets, UpdateBudget, DeleteBudget
├── cashflow/        GetSummaries
├── calendar/        CreateEvent, GetEvent, UpdateEvent, ListEvents, DeleteEvent, reconciliation
├── accountcheck/    CreateSession, BuildLinkURL, GetReport, GetReportPDF + continuous access
├── balancecheck/    RefreshBalance, GetRefreshStatus, GetAccountBalance + consent management
├── reports/         IncomeCheck, ExpenseCheck, RiskInsights, RiskCategorisation, BusinessAccountCheck
├── connector/       CreateUser, IngestAccounts, IngestTransactions
├── link/            BuildURL for all 6 products; TransactionsURL, AccountCheckURL, PaymentURL
├── connectivity/    CheckAPIHealth, CheckProviderStatus, CheckCredentialConnectivity
├── webhooks/        Verifier (HMAC-SHA256), Handler (typed dispatch)
└── internal/        cache, retry, ratelimit, httpclient (not for external use)
```
