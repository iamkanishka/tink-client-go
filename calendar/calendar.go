// Package calendar provides the Tink Financial Calendar API client.
//
// Schedule and track financial events such as bills, salary payments, and
// recurring expenses. Supports attachments, recurring groups, and transaction
// reconciliation.
//
// Requires a user bearer token (not client credentials).
// https://docs.tink.com/api#finance-management/financial-calendar
package calendar

import (
	"context"
	"fmt"
	"net/url"

	"github.com/iamkanishka/tink-client-go/internal/httpclient"
	"github.com/iamkanishka/tink-client-go/types"
)

const (
	eventsBase    = "/finance-management/v1/financial-calendar-events"
	summariesBase = "/finance-management/v1/financial-calendar-summaries"
)

// Service provides Tink Financial Calendar operations.
type Service struct {
	http *httpclient.HTTPClient
}

// New constructs a FinancialCalendar service.
func New(h *httpclient.HTTPClient) *Service { return &Service{http: h} }

// CreateEvent creates a new financial calendar event.
func (s *Service) CreateEvent(ctx context.Context, params types.CreateCalendarEventParams) (*types.CalendarEvent, error) {
	var out types.CalendarEvent
	if err := s.http.Post(ctx, eventsBase, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetEvent fetches a single calendar event by ID.
func (s *Service) GetEvent(ctx context.Context, eventID string) (*types.CalendarEvent, error) {
	var out types.CalendarEvent
	if err := s.http.Get(ctx, fmt.Sprintf("%s/%s", eventsBase, eventID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateEvent patches a calendar event. Pass only fields to change.
func (s *Service) UpdateEvent(ctx context.Context, eventID string, updates map[string]interface{}) (*types.CalendarEvent, error) {
	var out types.CalendarEvent
	if err := s.http.Patch(ctx, fmt.Sprintf("%s/%s", eventsBase, eventID), updates, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEvents lists calendar events with optional filtering.
// Pass an empty url.Values for no filter.
func (s *Service) ListEvents(ctx context.Context, query url.Values) (*types.CalendarEventsResponse, error) {
	var out types.CalendarEventsResponse
	if err := s.http.Get(ctx, eventsBase, query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteEvent deletes a calendar event.
// opt controls which occurrences are deleted for recurring events.
// Defaults to types.RecurringSingle when empty.
func (s *Service) DeleteEvent(ctx context.Context, eventID string, opt types.RecurringOption) error {
	if opt == "" {
		opt = types.RecurringSingle
	}
	path := fmt.Sprintf("%s/%s/?recurring=%s", eventsBase, eventID, opt)
	return s.http.Delete(ctx, path)
}

// GetSummaries returns summarised calendar data at a given resolution.
func (s *Service) GetSummaries(ctx context.Context, opts types.CalendarSummariesOptions) (map[string]interface{}, error) {
	q := url.Values{
		"periodGte": {opts.PeriodGte},
		"periodLte": {opts.PeriodLte},
	}
	path := fmt.Sprintf("%s/%s", summariesBase, opts.Resolution)
	var out map[string]interface{}
	if err := s.http.Get(ctx, path, q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddAttachment attaches metadata (e.g. an invoice URL) to a calendar event.
func (s *Service) AddAttachment(ctx context.Context, eventID string, params map[string]interface{}) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%s/attachments", eventsBase, eventID)
	if err := s.http.Post(ctx, path, params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteAttachment removes an attachment from a calendar event.
func (s *Service) DeleteAttachment(ctx context.Context, eventID, attachmentID string) error {
	return s.http.Delete(ctx, fmt.Sprintf("%s/%s/attachments/%s/", eventsBase, eventID, attachmentID))
}

// CreateRecurringGroup creates a recurring event group for an existing event.
func (s *Service) CreateRecurringGroup(ctx context.Context, eventID string, params map[string]interface{}) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := s.http.Post(ctx, fmt.Sprintf("%s/%s/recurring-group", eventsBase, eventID), params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateReconciliation links a calendar event to an actual transaction.
func (s *Service) CreateReconciliation(ctx context.Context, eventID string, params map[string]interface{}) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := s.http.Post(ctx, fmt.Sprintf("%s/%s/reconciliations", eventsBase, eventID), params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetReconciliationDetails returns reconciliation details for an event.
func (s *Service) GetReconciliationDetails(ctx context.Context, eventID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := s.http.Get(ctx, fmt.Sprintf("%s/%s/reconciliations/details", eventsBase, eventID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetReconciliationSuggestions returns AI-suggested transactions to reconcile.
func (s *Service) GetReconciliationSuggestions(ctx context.Context, eventID string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := s.http.Get(ctx, fmt.Sprintf("%s/%s/reconciliations/suggestions", eventsBase, eventID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteReconciliation removes the reconciliation link between an event and a transaction.
func (s *Service) DeleteReconciliation(ctx context.Context, eventID, transactionID string) error {
	return s.http.Delete(ctx, fmt.Sprintf("%s/%s/reconciliations/%s", eventsBase, eventID, transactionID))
}
