// Package httpclient is the transport layer for all Tink API calls.
//
// Features:
//   - Bearer token injection with thread-safe atomic-style RWMutex updates
//   - JSON and form-encoded request/response bodies
//   - In-memory LRU caching with per-resource TTLs
//   - Automatic retry with exponential back-off (via internal/retry)
//   - Context-propagated per-request timeouts
//   - Structured TinkError on every failure
//   - Cache invalidation after mutating requests
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	stderrors "errors"

	"github.com/iamkanishka/tink-client-go/errors"
	"github.com/iamkanishka/tink-client-go/internal/cache"
	"github.com/iamkanishka/tink-client-go/internal/retry"
)

// cacheTTLs maps resource type names to their LRU TTL.
// Values reflect Tink API data freshness characteristics.
var cacheTTLs = map[string]time.Duration{
	"providers":    1 * time.Hour,
	"categories":   24 * time.Hour,
	"accounts":     5 * time.Minute,
	"transactions": 5 * time.Minute,
	"statistics":   1 * time.Hour,
	"credentials":  30 * time.Second,
	"balances":     1 * time.Minute,
	"users":        10 * time.Minute,
	"reports":      24 * time.Hour,
	"default":      5 * time.Minute,
}

// cacheablePatterns are URL path prefixes whose responses are safe to cache.
var cacheablePatterns = []string{
	"/api/v1/providers", "/api/v1/categories", "/api/v1/statistics",
	"/api/v1/credentials", "/data/v2/accounts",
	"/data/v2/investment-accounts", "/data/v2/loan-accounts",
	"/data/v2/transactions", "/data/v2/identities",
	"/finance-management/v1/business-budgets",
	"/finance-management/v1/cash-flow-summaries",
	"/finance-management/v1/financial-calendar",
}

// nonCacheablePatterns are URL path prefixes that must never be cached
// (auth flows, mutations, real-time endpoints).
var nonCacheablePatterns = []string{
	"/oauth", "/user/create", "/user/delete",
	"/authorization-grant", "/link/v1/session",
	"/risk/", "/connector/", "/balance-refresh",
}

// Config holds options for creating an HTTPClient.
// Fields are ordered for minimal struct padding:
// map/pointer fields first, then strings, then numeric and bool last.
type Config struct {
	DefaultHeaders map[string]string
	HTTPClient     *http.Client
	BaseURL        string
	AccessToken    string
	UserID         string
	Timeout        time.Duration
	MaxRetries     int
	CacheMaxSize   int
	CacheEnabled   bool
}

// HTTPClient is the production-grade HTTP client for the Tink API.
// It is safe for concurrent use by multiple goroutines.
//
// Field ordering deliberately places timeout and cacheEnabled before mu
// so that the 24-byte sync.RWMutex lands at the end without forcing
// padding after the bool field.
//
//nolint:govet // fieldalignment: sync.RWMutex must not be copied; this layout is intentional
type HTTPClient struct {
	defaultHeaders map[string]string
	lru            *cache.LRU
	httpClient     *http.Client
	retryPolicy    retry.Policy
	baseURL        string
	token          string
	userID         string
	timeout        time.Duration
	cacheEnabled   bool
	mu             sync.RWMutex
}

// New constructs an HTTPClient from Config, applying safe defaults.
func New(cfg Config) *HTTPClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.CacheMaxSize == 0 {
		cfg.CacheMaxSize = 512
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.tink.com"
	}

	policy := retry.DefaultPolicy()
	policy.MaxAttempts = cfg.MaxRetries
	policy.ShouldRetry = func(err error) bool {
		var te *errors.TinkError
		if stderrors.As(err, &te) {
			return te.Retryable()
		}
		return false
	}

	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{}
	}

	headers := make(map[string]string, len(cfg.DefaultHeaders))
	for k, v := range cfg.DefaultHeaders {
		headers[k] = v
	}

	return &HTTPClient{
		baseURL:        base,
		timeout:        cfg.Timeout,
		retryPolicy:    policy,
		cacheEnabled:   cfg.CacheEnabled,
		lru:            cache.New(cfg.CacheMaxSize),
		httpClient:     hc,
		defaultHeaders: headers,
		token:          cfg.AccessToken,
		userID:         cfg.UserID,
	}
}

// AccessToken returns the current bearer token (RLock-safe).
func (c *HTTPClient) AccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// SetAccessToken atomically replaces the bearer token.
func (c *HTTPClient) SetAccessToken(t string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = t
}

// UserID returns the current user-ID scope.
func (c *HTTPClient) UserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}

// SetUserID sets the user ID used for cache scoping.
func (c *HTTPClient) SetUserID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userID = id
}

// InvalidateUser removes all cached entries for the current (or given) user.
// Called automatically after every mutating request.
func (c *HTTPClient) InvalidateUser(userID ...string) {
	c.mu.RLock()
	uid := c.userID
	c.mu.RUnlock()
	if len(userID) > 0 && userID[0] != "" {
		uid = userID[0]
	}
	if uid != "" {
		c.lru.InvalidatePrefix(uid + ":")
	}
}

// InvalidateCache clears all entries, or those matching a path prefix.
func (c *HTTPClient) InvalidateCache(prefix ...string) {
	if len(prefix) > 0 && prefix[0] != "" {
		c.lru.InvalidatePrefix(prefix[0])
	} else {
		c.lru.Flush()
	}
}

// ── HTTP verbs ─────────────────────────────────────────────────────────────

// Get performs a GET request, decoding the JSON response into dst.
// Responses for cacheable paths are stored in the LRU cache.
func (c *HTTPClient) Get(ctx context.Context, path string, query url.Values, dst any) error {
	full := buildPath(path, query)
	if c.cacheEnabled && isCacheable(full) {
		key := c.cacheKey(full)
		if hit, ok := c.lru.Get(key); ok {
			return roundTripJSON(hit, dst)
		}
		if err := c.dispatch(ctx, http.MethodGet, full, nil, dst, ""); err != nil {
			return err
		}
		c.lru.Set(key, copyVal(dst), ttlFor(full))
		return nil
	}
	return c.dispatch(ctx, http.MethodGet, full, nil, dst, "")
}

// GetRaw performs a GET and returns the raw response bytes.
// Use for binary endpoints (PDF downloads, etc.).
func (c *HTTPClient) GetRaw(ctx context.Context, path string, query url.Values) ([]byte, error) {
	var result []byte
	err := retry.Do(ctx, c.retryPolicy, func() error {
		var e error
		result, e = c.execRaw(ctx, http.MethodGet, buildPath(path, query), nil, "")
		return e
	})
	return result, err
}

// Post sends a JSON-encoded POST and decodes the response into dst.
// The user cache is invalidated on success.
func (c *HTTPClient) Post(ctx context.Context, path string, body, dst any) error {
	if err := c.dispatch(ctx, http.MethodPost, path, body, dst, "application/json"); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// PostForm sends an application/x-www-form-urlencoded POST.
// The user cache is invalidated on success.
func (c *HTTPClient) PostForm(ctx context.Context, path string, form url.Values, dst any) error {
	if err := retry.Do(ctx, c.retryPolicy, func() error {
		return c.execForm(ctx, path, form, dst)
	}); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// Patch sends a JSON-encoded PATCH and decodes the response into dst.
// The user cache is invalidated on success.
func (c *HTTPClient) Patch(ctx context.Context, path string, body, dst any) error {
	if err := c.dispatch(ctx, http.MethodPatch, path, body, dst, "application/json"); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// Delete sends a DELETE request.
// The user cache is invalidated on success.
func (c *HTTPClient) Delete(ctx context.Context, path string) error {
	if err := c.dispatch(ctx, http.MethodDelete, path, nil, nil, ""); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// ── Core dispatch ──────────────────────────────────────────────────────────

func (c *HTTPClient) dispatch(ctx context.Context, method, path string, body, dst any, ct string) error {
	return retry.Do(ctx, c.retryPolicy, func() error {
		raw, err := c.execRaw(ctx, method, path, body, ct)
		if err != nil {
			return err
		}
		if dst == nil || len(raw) == 0 {
			return nil
		}
		if err := json.Unmarshal(raw, dst); err != nil {
			return errors.FromDecodeError(fmt.Errorf("json.Unmarshal: %w", err))
		}
		return nil
	})
}

func (c *HTTPClient) execRaw(ctx context.Context, method, path string, body any, ct string) ([]byte, error) {
	rctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, errors.FromDecodeError(fmt.Errorf("json.Marshal: %w", err))
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(rctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, errors.FromNetworkError(err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		if ct == "" {
			ct = "application/json"
		}
		req.Header.Set("Content-Type", ct)
	}
	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.FromNetworkError(err)
	}
	defer func() {
		// Drain remaining bytes so the underlying TCP connection can be reused,
		// then close. Discard any drain/close error — the read result takes priority.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.FromDecodeError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.DebugContext(rctx, "tink-client-go: non-2xx response",
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", resp.StatusCode),
		)
		return nil, errors.FromResponse(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func (c *HTTPClient) execForm(ctx context.Context, path string, form url.Values, dst any) error {
	rctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(rctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.FromNetworkError(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.FromNetworkError(err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.FromDecodeError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.FromResponse(resp.StatusCode, respBody)
	}
	if dst != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dst); err != nil {
			return errors.FromDecodeError(fmt.Errorf("json.Unmarshal: %w", err))
		}
	}
	return nil
}

// applyHeaders injects the Authorization header and any configured defaults.
func (c *HTTPClient) applyHeaders(req *http.Request) {
	c.mu.RLock()
	tok := c.token
	c.mu.RUnlock()
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	for k, v := range c.defaultHeaders {
		req.Header.Set(k, v)
	}
}

// cacheKey builds a key scoped to the current user or token (last 16 chars).
func (c *HTTPClient) cacheKey(path string) string {
	c.mu.RLock()
	scope := c.userID
	if scope == "" {
		scope = c.token
	}
	c.mu.RUnlock()

	const scopeLen = 16
	if len(scope) > scopeLen {
		scope = scope[len(scope)-scopeLen:]
	}
	if scope == "" {
		scope = "public"
	}
	return scope + ":" + path
}

// ── Private helpers ────────────────────────────────────────────────────────

// isCacheable reports whether path is safe to cache.
func isCacheable(path string) bool {
	for _, p := range nonCacheablePatterns {
		if strings.Contains(path, p) {
			return false
		}
	}
	for _, p := range cacheablePatterns {
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

// resourceType maps a URL path to a cache-TTL bucket name.
func resourceType(path string) string {
	switch {
	case strings.Contains(path, "/providers"):
		return "providers"
	case strings.Contains(path, "/categories"):
		return "categories"
	case strings.Contains(path, "investment-accounts"):
		return "accounts"
	case strings.Contains(path, "loan-accounts"):
		return "accounts"
	case strings.Contains(path, "/balances"):
		return "balances"
	case strings.Contains(path, "/accounts"):
		return "accounts"
	case strings.Contains(path, "/transactions"):
		return "transactions"
	case strings.Contains(path, "/statistics"):
		return "statistics"
	case strings.Contains(path, "/credentials"):
		return "credentials"
	case strings.Contains(path, "/identities"):
		return "users"
	case strings.Contains(path, "/income-check"),
		strings.Contains(path, "/expense-check"),
		strings.Contains(path, "/risk-insight"),
		strings.Contains(path, "/risk-categori"),
		strings.Contains(path, "business-account-verification"):
		return "reports"
	default:
		return "default"
	}
}

// ttlFor returns the cache TTL for the given URL path.
func ttlFor(path string) time.Duration {
	if ttl, ok := cacheTTLs[resourceType(path)]; ok {
		return ttl
	}
	return cacheTTLs["default"]
}

// buildPath appends query parameters to path.
func buildPath(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

// roundTripJSON marshals src and unmarshals into dst (used for cache hits).
func roundTripJSON(src, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return errors.FromDecodeError(err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return errors.FromDecodeError(err)
	}
	return nil
}

// copyVal makes a JSON deep-copy of v for storage in the LRU cache.
func copyVal(v any) any {
	b, _ := json.Marshal(v)
	var m any
	_ = json.Unmarshal(b, &m)
	return m
}
