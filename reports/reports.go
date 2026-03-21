// Package reports provides clients for Tink risk and verification reports.
//
// Five read-only report services, all returning pre-generated analysis reports.
// Reports are immutable once created and are cached for 24 hours.
//
//   - IncomeCheck          — income stream analysis and PDF export
//   - ExpenseCheck         — categorised expense analysis
//   - RiskInsights         — financial risk scoring
//   - RiskCategorisation   — transaction-level risk categories
//   - BusinessAccountCheck — business account ownership verification
//
// https://docs.tink.com/api
package reports

import (
	"context"
	"fmt"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

// ── IncomeCheck ───────────────────────────────────────────────────────────

// IncomeCheckService retrieves income verification reports.
type IncomeCheckService struct {
	http *httpclient.HTTPClient
}

// NewIncomeCheck constructs an IncomeCheck service.
func NewIncomeCheck(h *httpclient.HTTPClient) *IncomeCheckService {
	return &IncomeCheckService{http: h}
}

// GetReport retrieves an income verification report by ID.
func (s *IncomeCheckService) GetReport(ctx context.Context, reportID string) (*types.IncomeCheckReport, error) {
	var out types.IncomeCheckReport
	if err := s.http.Get(ctx, fmt.Sprintf("/v2/income-checks/%s", reportID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetReportPDF downloads an income check report as a PDF binary.
func (s *IncomeCheckService) GetReportPDF(ctx context.Context, reportID string) ([]byte, error) {
	return s.http.GetRaw(ctx, fmt.Sprintf("/v2/income-checks/%s:generate-pdf", reportID), nil)
}

// ── ExpenseCheck ──────────────────────────────────────────────────────────

// ExpenseCheckService retrieves expense analysis reports.
type ExpenseCheckService struct {
	http *httpclient.HTTPClient
}

// NewExpenseCheck constructs an ExpenseCheck service.
func NewExpenseCheck(h *httpclient.HTTPClient) *ExpenseCheckService {
	return &ExpenseCheckService{http: h}
}

// GetReport retrieves an expense analysis report by ID.
func (s *ExpenseCheckService) GetReport(ctx context.Context, reportID string) (*types.ExpenseCheckReport, error) {
	var out types.ExpenseCheckReport
	if err := s.http.Get(ctx, fmt.Sprintf("/risk/v1/expense-checks/%s", reportID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── RiskInsights ──────────────────────────────────────────────────────────

// RiskInsightsService retrieves financial risk scoring reports.
type RiskInsightsService struct {
	http *httpclient.HTTPClient
}

// NewRiskInsights constructs a RiskInsights service.
func NewRiskInsights(h *httpclient.HTTPClient) *RiskInsightsService {
	return &RiskInsightsService{http: h}
}

// GetReport retrieves a risk insights report by ID.
func (s *RiskInsightsService) GetReport(ctx context.Context, reportID string) (*types.RiskInsightsReport, error) {
	var out types.RiskInsightsReport
	if err := s.http.Get(ctx, fmt.Sprintf("/risk/v1/risk-insights/%s", reportID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── RiskCategorisation ────────────────────────────────────────────────────

// RiskCategorisationService retrieves transaction-level risk categorisation reports.
type RiskCategorisationService struct {
	http *httpclient.HTTPClient
}

// NewRiskCategorisation constructs a RiskCategorisation service.
func NewRiskCategorisation(h *httpclient.HTTPClient) *RiskCategorisationService {
	return &RiskCategorisationService{http: h}
}

// GetReport retrieves a risk categorisation report by ID.
func (s *RiskCategorisationService) GetReport(ctx context.Context, reportID string) (*types.RiskCategorisationReport, error) {
	var out types.RiskCategorisationReport
	path := fmt.Sprintf("/risk/v2/risk-categorisation/reports/%s", reportID)
	if err := s.http.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── BusinessAccountCheck ──────────────────────────────────────────────────

// BusinessAccountCheckService retrieves business account verification reports.
type BusinessAccountCheckService struct {
	http *httpclient.HTTPClient
}

// NewBusinessAccountCheck constructs a BusinessAccountCheck service.
func NewBusinessAccountCheck(h *httpclient.HTTPClient) *BusinessAccountCheckService {
	return &BusinessAccountCheckService{http: h}
}

// GetReport retrieves a business account verification report by ID.
// Endpoint: /data/v1/business-account-verification-reports/:id
func (s *BusinessAccountCheckService) GetReport(ctx context.Context, reportID string) (*types.BusinessAccountCheckReport, error) {
	var out types.BusinessAccountCheckReport
	path := fmt.Sprintf("/data/v1/business-account-verification-reports/%s", reportID)
	if err := s.http.Get(ctx, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
