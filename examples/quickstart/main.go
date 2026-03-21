// Command quickstart demonstrates the tink-client-go client across all major API surfaces.
//
// Run with:
//
//	TINK_CLIENT_ID=xxx TINK_CLIENT_SECRET=yyy go run ./examples/quickstart
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/iamkanishka/tink-client-go/client"
	"github.com/iamkanishka/tink-client-go/types"
	"github.com/iamkanishka/tink-client-go/webhooks"
)

func main() {
	ctx := context.Background()

	// ── 1. Construct client ────────────────────────────────────────────────
	//
	// Reads TINK_CLIENT_ID and TINK_CLIENT_SECRET from the environment.
	// Uses functional options for clean configuration.
	tink, err := client.NewWithOptions(
		client.WithCredentials(
			os.Getenv("TINK_CLIENT_ID"),
			os.Getenv("TINK_CLIENT_SECRET"),
		),
		client.WithTimeout(15*time.Second),
		client.WithMaxRetries(3),
		client.WithHeader("X-Request-ID", "example-request-001"),
	)
	if err != nil {
		log.Fatalf("client.New: %v", err)
	}
	fmt.Printf("Client ready: version=%s baseURL=%s\n", tink.Info().Version, tink.Info().BaseURL)

	// ── 2. Authenticate (client credentials) ──────────────────────────────
	if err := tink.Authenticate(ctx, "accounts:read transactions:read"); err != nil {
		log.Fatalf("Authenticate: %v", err)
	}
	fmt.Println("Authenticated ✓")

	// ── 3. List providers (cached 1 hour) ─────────────────────────────────
	provResp, err := tink.Providers.ListProviders(ctx, &types.ProvidersListOptions{Market: "GB"})
	if err != nil {
		log.Printf("ListProviders: %v", err)
	} else {
		fmt.Printf("Providers in GB: %d\n", len(provResp.Providers))
		if len(provResp.Providers) > 0 {
			fmt.Printf("  First: %s (%s)\n", provResp.Providers[0].DisplayName, provResp.Providers[0].Status)
		}
	}

	// ── 4. List categories (cached 24 hours) ──────────────────────────────
	catResp, err := tink.Categories.ListCategories(ctx, "en_US")
	if err != nil {
		log.Printf("ListCategories: %v", err)
	} else {
		fmt.Printf("Categories: %d\n", len(catResp.Categories))
	}

	// ── 5. Account data ───────────────────────────────────────────────────
	accResp, err := tink.Accounts.ListAccounts(ctx, &types.AccountsListOptions{
		TypeIn:            []string{"CHECKING", "SAVINGS"},
		PaginationOptions: types.PaginationOptions{PageSize: 25},
	})
	if err != nil {
		log.Printf("ListAccounts: %v (expected without user token)", err)
	} else {
		fmt.Printf("Accounts: %d\n", len(accResp.Accounts))
		for _, acc := range accResp.Accounts {
			fmt.Printf("  %s — %s\n", acc.Name, acc.Type)
		}
	}

	// ── 6. Transactions ───────────────────────────────────────────────────
	txResp, err := tink.Transactions.ListTransactions(ctx, &types.TransactionsListOptions{
		BookedDateGte:     "2024-01-01",
		BookedDateLte:     "2024-12-31",
		PaginationOptions: types.PaginationOptions{PageSize: 10},
	})
	if err != nil {
		log.Printf("ListTransactions: %v (expected without user token)", err)
	} else {
		fmt.Printf("Transactions: %d\n", len(txResp.Transactions))
	}

	// ── 7. Statistics ─────────────────────────────────────────────────────
	statsResp, err := tink.Statistics.GetStatistics(ctx, types.StatisticsOptions{
		PeriodGte:  "2024-01-01",
		PeriodLte:  "2024-12-31",
		Resolution: "MONTHLY",
	})
	if err != nil {
		log.Printf("GetStatistics: %v (expected without user token)", err)
	} else {
		fmt.Printf("Statistics periods: %d\n", len(statsResp.Periods))
	}

	// ── 8. Tink Link URL builder ──────────────────────────────────────────
	txURL := tink.Link.TransactionsURL("authorization_code_from_delegate", types.LinkURLOptions{
		ClientID:    os.Getenv("TINK_CLIENT_ID"),
		RedirectURI: "https://yourapp.com/callback",
		Market:      "GB",
		Locale:      "en_US",
	})
	fmt.Printf("Transactions Tink Link: %s\n", txURL)

	acURL := tink.Link.AccountCheckURL("authorization_code", types.LinkURLOptions{
		ClientID:    os.Getenv("TINK_CLIENT_ID"),
		RedirectURI: "https://yourapp.com/callback",
		Market:      "GB",
		Locale:      "en_US",
	})
	fmt.Printf("AccountCheck Tink Link: %s\n", acURL)

	payURL := tink.Link.PaymentURL("payment_request_id_123", types.LinkURLOptions{
		ClientID:    os.Getenv("TINK_CLIENT_ID"),
		RedirectURI: "https://yourapp.com/callback",
		Market:      "SE",
		Locale:      "sv_SE",
	})
	fmt.Printf("Payment Tink Link: %s\n", payURL)

	// ── 9. Continuous access flow example ─────────────────────────────────
	fmt.Println("\n── Continuous access flow (demonstration) ──")
	user, err := tink.Users.CreateUser(ctx, types.CreateUserParams{
		ExternalUserID: "example_user_001",
		Locale:         "en_US",
		Market:         "GB",
	})
	if err != nil {
		log.Printf("CreateUser: %v (expected without sufficient scope)", err)
	} else {
		fmt.Printf("Created user: %s\n", user.UserID)
	}

	// ── 10. API health check ──────────────────────────────────────────────
	if err := tink.Connectivity.CheckAPIHealth(ctx); err != nil {
		fmt.Printf("API health: UNHEALTHY — %v\n", err)
	} else {
		fmt.Println("API health: OK ✓")
	}

	// ── 11. Webhook signature verification ───────────────────────────────
	fmt.Println("\n── Webhook verification ──")
	secret := "my_webhook_signing_secret"
	verifier := webhooks.NewVerifier(secret)

	payload := []byte(`{"type":"credentials.updated","data":{"userId":"u_123"},"timestamp":"2024-01-15T12:00:00Z"}`)
	sig := verifier.GenerateSignatureHex(payload)
	fmt.Printf("Generated signature: %s...\n", sig[:16])

	if err := verifier.Verify(payload, sig); err != nil {
		log.Printf("Verify failed: %v", err)
	} else {
		fmt.Println("Signature verified ✓")
	}

	// ── 12. Full webhook handler with typed dispatch ───────────────────────
	wh := tink.NewWebhookHandler(secret)

	wh.On(types.WebhookEventCredentialsUpdated, func(ctx context.Context, e *types.WebhookEvent) error {
		fmt.Printf("Handler: credentials.updated for user %v\n", e.Data["userId"])
		return nil
	})
	wh.On(types.WebhookEventCredentialsRefreshFailed, func(ctx context.Context, e *types.WebhookEvent) error {
		fmt.Printf("Handler: credentials.refresh.failed for user %v\n", e.Data["userId"])
		return nil
	})
	wh.OnAll(func(ctx context.Context, e *types.WebhookEvent) error {
		fmt.Printf("Wildcard: received %s\n", e.Type)
		return nil
	})

	if err := wh.HandleRequest(ctx, payload, sig); err != nil {
		log.Printf("HandleRequest: %v", err)
	}

	// ── 13. Test webhook is silently acknowledged ──────────────────────────
	testPayload := []byte(`{"type":"test","data":{}}`)
	testSig := verifier.GenerateSignatureHex(testPayload)
	if err := wh.HandleRequest(ctx, testPayload, testSig); err != nil {
		log.Printf("Test webhook error: %v", err)
	} else {
		fmt.Println("Test webhook silently acknowledged ✓")
	}

	// ── 14. Cache management ──────────────────────────────────────────────
	fmt.Println("\n── Cache management ──")
	tink.ClearCache()
	fmt.Println("Cache cleared")

	tink.InvalidateCache("/api/v1/providers")
	fmt.Println("Provider cache invalidated")

	// ── 15. Token expiry helper ───────────────────────────────────────────
	tokenInfo := client.ParseToken(&types.TokenResponse{
		AccessToken: "tok_example",
		ExpiresIn:   3600,
		Scope:       "accounts:read",
	})
	fmt.Printf("\nToken expires at: %s\n", tokenInfo.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("Is expired (5min buffer): %v\n", client.IsExpired(tokenInfo.ExpiresAt))
	fmt.Printf("Is expired (already expired): %v\n", client.IsExpired(time.Now().Add(-10*time.Minute)))

	// ── 16. HTTP server for webhook integration ───────────────────────────
	fmt.Println("\n── Example webhook HTTP handler (not started) ──")
	_ = webhookHTTPHandler(wh) // show the pattern
	fmt.Println("Webhook HTTP handler ready (use http.Handle)")

	// ── 17. Balance check URL builder ────────────────────────────────────
	balanceLink := tink.BalanceCheck.BuildAccountCheckLink("grant_code", types.BuildAccountCheckLinkOptions{
		ClientID:    os.Getenv("TINK_CLIENT_ID"),
		Market:      "SE",
		RedirectURI: "https://yourapp.com/callback",
		Test:        false,
		State:       "csrf_token_abc",
	})
	fmt.Printf("Balance check link: %s\n", balanceLink)

	// ── 18. Verify hex encode/decode round-trip ───────────────────────────
	raw := verifier.GenerateSignature([]byte("verify me"))
	encoded := hex.EncodeToString(raw)
	if err := verifier.Verify([]byte("verify me"), encoded); err != nil {
		log.Printf("Round-trip verify: %v", err)
	} else {
		fmt.Println("Signature round-trip OK ✓")
	}

	fmt.Println("\nDone.")
}

// webhookHTTPHandler demonstrates integrating the webhook handler with net/http.
func webhookHTTPHandler(wh *webhooks.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read raw body — do NOT use r.Body after this point without re-wrapping.
		buf := make([]byte, 1<<20) // 1 MB max
		n, err := r.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}
		body := buf[:n]

		sig := r.Header.Get("X-Tink-Signature")
		if err := wh.HandleRequest(r.Context(), body, sig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
