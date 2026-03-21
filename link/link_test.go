package link_test

import (
	"strings"
	"testing"

	"github.com/iamkanishka/tink-client-go/link"
	"github.com/iamkanishka/tink-client-go/types"
)

func TestBuildURL_AllProducts(t *testing.T) {
	svc := link.New()
	baseOpts := types.LinkURLOptions{
		ClientID:          "test_client",
		RedirectURI:       "https://app.example.com/callback",
		Market:            "GB",
		Locale:            "en_US",
		AuthorizationCode: "auth_code_123",
	}

	cases := []struct {
		product      types.LinkProduct
		expectedPath string
	}{
		{types.LinkProductTransactions, "transactions/connect-accounts"},
		{types.LinkProductAccountCheck, "account-check/connect-accounts"},
		{types.LinkProductIncomeCheck, "income-check/connect-accounts"},
		{types.LinkProductExpenseCheck, "expense-check/connect-accounts"},
		{types.LinkProductRiskInsights, "risk-insights/connect-accounts"},
	}

	for _, tc := range cases {
		t.Run(string(tc.product), func(t *testing.T) {
			u := svc.BuildURL(tc.product, baseOpts)
			if !strings.HasPrefix(u, "https://link.tink.com/1.0/") {
				t.Errorf("URL must start with https://link.tink.com/1.0/, got: %s", u)
			}
			if !strings.Contains(u, tc.expectedPath) {
				t.Errorf("URL %q does not contain expected path %q", u, tc.expectedPath)
			}
		})
	}
}

func TestBuildURL_Payment(t *testing.T) {
	svc := link.New()
	u := svc.BuildURL(types.LinkProductPayment, types.LinkURLOptions{
		ClientID:         "cid",
		RedirectURI:      "https://x.com/cb",
		Market:           "SE",
		Locale:           "sv_SE",
		PaymentRequestID: "pay_req_123",
	})
	if !strings.Contains(u, "pay/execute-payment") {
		t.Errorf("payment URL missing path: %s", u)
	}
	if !strings.Contains(u, "payment_request_id=pay_req_123") {
		t.Errorf("payment URL missing payment_request_id: %s", u)
	}
}

func TestBuildURL_ContainsRequiredParams(t *testing.T) {
	svc := link.New()
	u := svc.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
		ClientID:          "my_client",
		RedirectURI:       "https://example.com/callback",
		Market:            "GB",
		Locale:            "en_US",
		AuthorizationCode: "code_xyz",
	})
	required := []string{"client_id=my_client", "market=GB", "locale=en_US", "authorization_code=code_xyz"}
	for _, r := range required {
		if !strings.Contains(u, r) {
			t.Errorf("URL missing required param %q: %s", r, u)
		}
	}
}

func TestBuildURL_TestMode(t *testing.T) {
	svc := link.New()
	u := svc.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
		ClientID: "cid", RedirectURI: "https://x.com", Market: "GB", Locale: "en_US",
		Test: true, InputProvider: "uk-ob-barclays", InputUsername: "test_user",
	})
	if !strings.Contains(u, "test=true") {
		t.Errorf("test URL missing test=true: %s", u)
	}
	if !strings.Contains(u, "input_provider=uk-ob-barclays") {
		t.Errorf("test URL missing input_provider: %s", u)
	}
	if !strings.Contains(u, "input_username=test_user") {
		t.Errorf("test URL missing input_username: %s", u)
	}
}

func TestBuildURL_IframeMode(t *testing.T) {
	svc := link.New()
	u := svc.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
		ClientID: "cid", RedirectURI: "https://x.com", Market: "GB", Locale: "en_US",
		Iframe: true,
	})
	if !strings.Contains(u, "iframe=true") {
		t.Errorf("iframe URL missing iframe=true: %s", u)
	}
}

func TestBuildURL_StateParam(t *testing.T) {
	svc := link.New()
	u := svc.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
		ClientID: "cid", RedirectURI: "https://x.com", Market: "GB", Locale: "en_US",
		State: "csrf_token_abc",
	})
	if !strings.Contains(u, "state=csrf_token_abc") {
		t.Errorf("URL missing state param: %s", u)
	}
}

func TestTransactionsURL(t *testing.T) {
	svc := link.New()
	u := svc.TransactionsURL("code_abc", types.LinkURLOptions{
		ClientID: "cid", RedirectURI: "https://x.com", Market: "GB", Locale: "en_US",
	})
	if !strings.Contains(u, "transactions/connect-accounts") {
		t.Errorf("TransactionsURL missing product path: %s", u)
	}
	if !strings.Contains(u, "authorization_code=code_abc") {
		t.Errorf("TransactionsURL missing code: %s", u)
	}
}

func TestAccountCheckURL(t *testing.T) {
	svc := link.New()
	u := svc.AccountCheckURL("code_abc", types.LinkURLOptions{
		ClientID: "cid", RedirectURI: "https://x.com", Market: "GB", Locale: "en_US",
	})
	if !strings.Contains(u, "account-check/connect-accounts") {
		t.Errorf("AccountCheckURL missing product path: %s", u)
	}
}

func TestPaymentURL(t *testing.T) {
	svc := link.New()
	u := svc.PaymentURL("pay_123", types.LinkURLOptions{
		ClientID: "cid", RedirectURI: "https://x.com", Market: "SE", Locale: "sv_SE",
	})
	if !strings.Contains(u, "pay/execute-payment") {
		t.Errorf("PaymentURL missing product path: %s", u)
	}
	if !strings.Contains(u, "payment_request_id=pay_123") {
		t.Errorf("PaymentURL missing payment_request_id: %s", u)
	}
}

func TestBuildURL_UnknownProductReturnsEmpty(t *testing.T) {
	svc := link.New()
	u := svc.BuildURL("unknown_product", types.LinkURLOptions{
		ClientID: "cid", RedirectURI: "https://x.com", Market: "GB", Locale: "en_US",
	})
	if u != "" {
		t.Errorf("expected empty string for unknown product, got: %s", u)
	}
}
