// Package categories provides the Tink Categories API client.
//
// Transaction categories are static reference data used to classify
// transactions. They are locale-specific and cached for 24 hours.
//
// https://docs.tink.com/api#categories
package categories

import (
	"context"
	"fmt"
	"net/url"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// Service provides Tink Categories operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Categories service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// ListCategories lists all transaction categories for the given locale.
// locale defaults to "en_US" when empty. Results are cached for 24 hours.
func (s *Service) ListCategories(ctx context.Context, locale string) (*types.CategoriesResponse, error) {
	if locale == "" {
		locale = "en_US"
	}
	q := url.Values{"locale": {locale}}
	var out types.CategoriesResponse
	if err := s.http.Get(ctx, "/api/v1/categories", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCategory fetches a single category by its ID.
// locale defaults to "en_US" when empty. Result is cached for 24 hours.
func (s *Service) GetCategory(ctx context.Context, categoryID, locale string) (*types.Category, error) {
	if locale == "" {
		locale = "en_US"
	}
	q := url.Values{"locale": {locale}}
	var out types.Category
	if err := s.http.Get(ctx, fmt.Sprintf("/api/v1/categories/%s", categoryID), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
