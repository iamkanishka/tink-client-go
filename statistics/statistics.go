// Package statistics provides the Tink Statistics API client.
//
// Returns aggregated financial statistics across configurable time periods
// and resolutions. Results are cached for 1 hour.
//
// Required scopes: statistics:read, transactions:read
// https://docs.tink.com/api#statistics
package statistics

import (
	"context"
	"fmt"
	"net/url"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Statistics operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Statistics service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// GetStatistics returns aggregated income/expense statistics for a time period.
// Resolution defaults to "MONTHLY" when empty.
func (s *Service) GetStatistics(ctx context.Context, opts types.StatisticsOptions) (*types.StatisticsResponse, error) {
	var out types.StatisticsResponse
	if err := s.http.Get(ctx, "/api/v1/statistics", statsQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCategoryStatistics returns statistics filtered to a specific transaction category.
func (s *Service) GetCategoryStatistics(ctx context.Context, categoryID string, opts types.StatisticsOptions) (*types.StatisticsResponse, error) {
	var out types.StatisticsResponse
	path := fmt.Sprintf("/api/v1/statistics/categories/%s", categoryID)
	if err := s.http.Get(ctx, path, statsQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAccountStatistics returns statistics filtered to a specific account.
func (s *Service) GetAccountStatistics(ctx context.Context, accountID string, opts types.StatisticsOptions) (*types.StatisticsResponse, error) {
	var out types.StatisticsResponse
	path := fmt.Sprintf("/api/v1/statistics/accounts/%s", accountID)
	if err := s.http.Get(ctx, path, statsQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func statsQuery(opts types.StatisticsOptions) url.Values {
	resolution := opts.Resolution
	if resolution == "" {
		resolution = "MONTHLY"
	}
	q := url.Values{
		"periodGte":  {opts.PeriodGte},
		"periodLte":  {opts.PeriodLte},
		"resolution": {resolution},
	}
	for _, id := range opts.AccountIDIn {
		q.Add("accountIdIn", id)
	}
	for _, id := range opts.CategoryIDIn {
		q.Add("categoryIdIn", id)
	}
	return q
}
