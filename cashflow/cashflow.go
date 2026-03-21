// Package cashflow provides the Tink Cash Flow API client.
//
// Income vs expense summaries across configurable time resolutions.
//
// Requires a user bearer token (not client credentials).
// https://docs.tink.com/api#finance-management/cash-flow
package cashflow

import (
	"context"
	"net/url"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Cash Flow operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a CashFlow service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// GetSummaries returns cash flow summaries for a date range at the given resolution.
//
// Example:
//
//	resp, err := client.CashFlow.GetSummaries(ctx, types.CashFlowOptions{
//	    Resolution: types.CashFlowResolutionMonthly,
//	    FromGte:    "2024-01-01",
//	    ToLte:      "2024-12-31",
//	})
func (s *Service) GetSummaries(ctx context.Context, opts types.CashFlowOptions) (*types.CashFlowResponse, error) {
	q := url.Values{
		"fromGte": {opts.FromGte},
		"toLte":   {opts.ToLte},
	}
	path := "/finance-management/v1/cash-flow-summaries/" + string(opts.Resolution)
	var out types.CashFlowResponse
	if err := s.http.Get(ctx, path, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
