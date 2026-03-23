// Package budgets provides the Tink Finance Management Budgets API client.
//
// Create and track financial budgets for income or expense categories.
// Supports one-off and recurring budgets with flexible allocation rules.
//
// Requires a user bearer token (not client credentials).
// https://docs.tink.com/api#finance-management
package budgets

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

const basePath = "/finance-management/v1/business-budgets"

// Service provides Tink Budgets operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a Budgets service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// CreateBudget creates a new budget.
//
// Example:
//
//	budget, err := client.Budgets.CreateBudget(ctx, types.CreateBudgetParams{
//	    Title:        "Office Supplies",
//	    Type:         types.BudgetTypeExpense,
//	    TargetAmount: types.TargetAmount{Value: types.ExactAmount{UnscaledValue: 50000, Scale: 2}, CurrencyCode: "GBP"},
//	    Recurrence:   types.BudgetRecurrence{Frequency: types.BudgetFrequencyMonthly, Start: "2024-01-01"},
//	})
func (s *Service) CreateBudget(ctx context.Context, params types.CreateBudgetParams) (*types.Budget, error) {
	var out types.Budget
	if err := s.http.Post(ctx, basePath, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBudget fetches a budget by ID.
func (s *Service) GetBudget(ctx context.Context, budgetID string) (*types.Budget, error) {
	var out types.Budget
	if err := s.http.Get(ctx, fmt.Sprintf("%s/%s", basePath, budgetID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBudgetHistory returns the spending history for a budget across all periods.
func (s *Service) GetBudgetHistory(ctx context.Context, budgetID string) (*types.BudgetHistoryResponse, error) {
	var out types.BudgetHistoryResponse
	if err := s.http.Get(ctx, fmt.Sprintf("%s/%s/history", basePath, budgetID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListBudgets lists all budgets with optional status filter.
func (s *Service) ListBudgets(ctx context.Context, opts *types.BudgetsListOptions) (*types.BudgetsResponse, error) {
	var out types.BudgetsResponse
	if err := s.http.Get(ctx, basePath, budgetsQuery(opts), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateBudget patches a budget. Pass only the fields to change.
func (s *Service) UpdateBudget(ctx context.Context, budgetID string, updates map[string]any) (*types.Budget, error) {
	var out types.Budget
	if err := s.http.Patch(ctx, fmt.Sprintf("%s/%s", basePath, budgetID), updates, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBudget permanently removes a budget.
func (s *Service) DeleteBudget(ctx context.Context, budgetID string) error {
	return s.http.Delete(ctx, fmt.Sprintf("%s/%s", basePath, budgetID))
}

func budgetsQuery(opts *types.BudgetsListOptions) url.Values {
	if opts == nil {
		return nil
	}
	q := url.Values{}
	if opts.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(opts.PageSize))
	}
	if opts.PageToken != "" {
		q.Set("pageToken", opts.PageToken)
	}
	if len(opts.ProgressStatusIn) > 0 {
		q.Set("progressStatusIn", strings.Join(opts.ProgressStatusIn, ","))
	}
	return q
}
