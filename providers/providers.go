// Package providers provides the Tink Providers API client.
//
// Provider data (financial institutions supported by Tink) is stable reference
// data. Responses from /api/v1/providers are cached for 1 hour automatically.
//
// Required scopes: none (unauthenticated), or provider:read (authenticated)
// https://docs.tink.com/api#providers
package providers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Providers operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Providers service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ListProviders lists available providers, optionally filtered by market or capability.
// Results are cached for 1 hour. Pass nil opts for no filter.
//
// Example:
//
//	resp, err := client.Providers.ListProviders(ctx, &types.ProvidersListOptions{Market: "GB"})
func (s *Service) ListProviders(ctx context.Context, opts *types.ProvidersListOptions) (*types.ProvidersResponse, error) {
	q := url.Values{}
	if opts != nil {
		if opts.Market != "" {
			q.Set("market", opts.Market)
		}
		if len(opts.Capabilities) > 0 {
			q.Set("capabilities", strings.Join(opts.Capabilities, ","))
		}
	}
	var out types.ProvidersResponse
	if err := s.http.Get(ctx, "/api/v1/providers", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetProvider fetches a single provider by its name identifier (e.g. "uk-ob-barclays").
// Result is cached for 1 hour.
func (s *Service) GetProvider(ctx context.Context, providerID string) (*types.Provider, error) {
	var out types.Provider
	if err := s.http.Get(ctx, fmt.Sprintf("/api/v1/providers/%s", providerID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
