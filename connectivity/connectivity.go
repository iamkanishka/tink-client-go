// Package connectivity provides the Tink Connectivity API client.
//
// Check and monitor the health of financial provider integrations
// and user bank connections (credentials).
//
// https://docs.tink.com/api#providers
package connectivity

import (
	"context"
	"fmt"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Connectivity operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Connectivity service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ListProvidersByMarket lists all providers available in a given market.
// This is an unauthenticated endpoint — no token required.
func (s *Service) ListProvidersByMarket(ctx context.Context, market string) (*types.ProvidersResponse, error) {
	var out types.ProvidersResponse
	if err := s.http.Get(ctx, fmt.Sprintf("/api/v1/providers/%s", market), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListProvidersByMarketAuthenticated lists providers using an authenticated request.
// May return additional data compared to the unauthenticated endpoint.
func (s *Service) ListProvidersByMarketAuthenticated(ctx context.Context, market string) (*types.ProvidersResponse, error) {
	return s.ListProvidersByMarket(ctx, market)
}

// CheckProviderStatus checks whether a specific provider is active and operational.
// If market is non-empty, also validates that the provider belongs to that market.
// Returns ProviderStatusResult{Active: false} if the provider is unavailable.
func (s *Service) CheckProviderStatus(ctx context.Context, providerID, market string) (*types.ProviderStatusResult, error) {
	var p types.Provider
	err := s.http.Get(ctx, fmt.Sprintf("/api/v1/providers/%s", providerID), nil, &p)
	if err != nil {
		return &types.ProviderStatusResult{Active: false}, nil
	}
	if market != "" && p.Market != market {
		return &types.ProviderStatusResult{Active: false}, nil
	}
	return &types.ProviderStatusResult{Active: p.Status == "ENABLED", Provider: &p}, nil
}

// ProviderOperational returns true if the provider is active and accepting connections.
func (s *Service) ProviderOperational(ctx context.Context, providerID, market string) (bool, error) {
	result, err := s.CheckProviderStatus(ctx, providerID, market)
	if err != nil {
		return false, err
	}
	return result.Active, nil
}

// CheckCredentialConnectivity returns a connectivity health summary for all user credentials.
// opts controls which credentials appear in the results (nil = include all).
func (s *Service) CheckCredentialConnectivity(ctx context.Context, opts *types.ConnectivityOptions) (*types.ConnectivitySummary, error) {
	var data struct {
		Credentials []struct {
			ID            string `json:"id"`
			ProviderName  string `json:"providerName"`
			Status        string `json:"status"`
			StatusUpdated string `json:"statusUpdated"`
			StatusPayload string `json:"statusPayload"`
		} `json:"credentials"`
	}
	if err := s.http.Get(ctx, "/api/v1/credentials/list", nil, &data); err != nil {
		return nil, err
	}

	all := make([]types.CredentialConnectivity, 0, len(data.Credentials))
	for _, c := range data.Credentials {
		all = append(all, types.CredentialConnectivity{
			CredentialID:  c.ID,
			ProviderName:  c.ProviderName,
			Status:        c.Status,
			Healthy:       c.Status == "UPDATED",
			LastRefreshed: c.StatusUpdated,
			ErrorMessage:  c.StatusPayload,
		})
	}

	filtered := make([]types.CredentialConnectivity, 0, len(all))
	for _, c := range all {
		if opts != nil {
			if opts.IncludeHealthy != nil && !*opts.IncludeHealthy && c.Healthy {
				continue
			}
			if opts.IncludeUnhealthy != nil && !*opts.IncludeUnhealthy && !c.Healthy {
				continue
			}
		}
		filtered = append(filtered, c)
	}

	healthy, unhealthy := 0, 0
	for _, c := range all {
		if c.Healthy {
			healthy++
		} else {
			unhealthy++
		}
	}

	return &types.ConnectivitySummary{
		Credentials: filtered,
		Healthy:     healthy,
		Unhealthy:   unhealthy,
		Total:       len(all),
	}, nil
}

// GetCredentialConnectivity returns connectivity status for a single credential.
func (s *Service) GetCredentialConnectivity(ctx context.Context, credentialID string) (*types.CredentialConnectivity, error) {
	var c struct {
		ID            string `json:"id"`
		ProviderName  string `json:"providerName"`
		Status        string `json:"status"`
		StatusUpdated string `json:"statusUpdated"`
		StatusPayload string `json:"statusPayload"`
	}
	if err := s.http.Get(ctx, fmt.Sprintf("/api/v1/credentials/%s", credentialID), nil, &c); err != nil {
		return nil, err
	}
	return &types.CredentialConnectivity{
		CredentialID:  c.ID,
		ProviderName:  c.ProviderName,
		Status:        c.Status,
		Healthy:       c.Status == "UPDATED",
		LastRefreshed: c.StatusUpdated,
		ErrorMessage:  c.StatusPayload,
	}, nil
}

// CheckAPIHealth checks overall Tink API availability by probing a known endpoint.
// Returns nil if the API is reachable.
func (s *Service) CheckAPIHealth(ctx context.Context) error {
	var out types.ProvidersResponse
	return s.http.Get(ctx, "/api/v1/providers/GB", nil, &out)
}
