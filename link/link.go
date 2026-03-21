// Package link provides Tink Link URL construction for all six products.
//
// Tink Link is the browser-based flow that end users go through to connect
// their bank account. This package builds the correct URL for each product.
//
// Supported products:
//   - Transactions   — connect bank accounts for transaction access
//   - AccountCheck   — verify account ownership
//   - IncomeCheck    — verify income
//   - Payment        — initiate a payment
//   - ExpenseCheck   — analyse spending
//   - RiskInsights   — generate risk report
//
// https://docs.tink.com/resources/tink-link
package link

import (
	"fmt"
	"net/url"

	"github.com/iamkanishka/tink-client-go/types"
)

const baseURL = "https://link.tink.com/1.0"

// productPaths maps each LinkProduct to its Tink Link URL path segment.
var productPaths = map[types.LinkProduct]string{
	types.LinkProductTransactions: "transactions/connect-accounts",
	types.LinkProductAccountCheck: "account-check/connect-accounts",
	types.LinkProductIncomeCheck:  "income-check/connect-accounts",
	types.LinkProductPayment:      "pay/execute-payment",
	types.LinkProductExpenseCheck: "expense-check/connect-accounts",
	types.LinkProductRiskInsights: "risk-insights/connect-accounts",
}

// Service provides Tink Link URL building.
// It is stateless and does not require an HTTP client.
type Service struct{}

// New constructs a Link service.
func New() *Service { return &Service{} }

// BuildURL constructs a Tink Link URL for any supported product.
//
// Example:
//
//	u := client.Link.BuildURL(types.LinkProductTransactions, types.LinkURLOptions{
//	    ClientID:          "your_client_id",
//	    RedirectURI:       "https://yourapp.com/callback",
//	    Market:            "GB",
//	    Locale:            "en_US",
//	    AuthorizationCode: code,
//	})
func (s *Service) BuildURL(product types.LinkProduct, opts types.LinkURLOptions) string {
	path, ok := productPaths[product]
	if !ok {
		return ""
	}
	q := url.Values{
		"client_id":    {opts.ClientID},
		"redirect_uri": {opts.RedirectURI},
		"market":       {opts.Market},
		"locale":       {opts.Locale},
	}
	if opts.AuthorizationCode != "" {
		q.Set("authorization_code", opts.AuthorizationCode)
	}
	if product == types.LinkProductPayment && opts.PaymentRequestID != "" {
		q.Set("payment_request_id", opts.PaymentRequestID)
	}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Iframe {
		q.Set("iframe", "true")
	}
	if opts.Test {
		q.Set("test", "true")
		if opts.InputProvider != "" {
			q.Set("input_provider", opts.InputProvider)
		}
		if opts.InputUsername != "" {
			q.Set("input_username", opts.InputUsername)
		}
	}
	return fmt.Sprintf("%s/%s?%s", baseURL, path, q.Encode())
}

// TransactionsURL is a convenience wrapper for the transactions product.
func (s *Service) TransactionsURL(authorizationCode string, opts types.LinkURLOptions) string {
	opts.AuthorizationCode = authorizationCode
	return s.BuildURL(types.LinkProductTransactions, opts)
}

// AccountCheckURL is a convenience wrapper for the account check product.
func (s *Service) AccountCheckURL(authorizationCode string, opts types.LinkURLOptions) string {
	opts.AuthorizationCode = authorizationCode
	return s.BuildURL(types.LinkProductAccountCheck, opts)
}

// PaymentURL is a convenience wrapper for the payment product.
func (s *Service) PaymentURL(paymentRequestID string, opts types.LinkURLOptions) string {
	opts.PaymentRequestID = paymentRequestID
	return s.BuildURL(types.LinkProductPayment, opts)
}
