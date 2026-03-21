// Package httpclient is the transport layer for all Tink API calls.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/iamkanishka/tink-client-go/errors"
	"github.com/iamkanishka/tink-client-go/internal/cache"
	"github.com/iamkanishka/tink-client-go/internal/retry"
)

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

var cacheablePatterns = []string{
	"/api/v1/providers", "/api/v1/categories", "/api/v1/statistics",
	"/api/v1/credentials", "/data/v2/accounts",
	"/data/v2/investment-accounts", "/data/v2/loan-accounts",
	"/data/v2/transactions", "/data/v2/identities",
	"/finance-management/v1/business-budgets",
	"/finance-management/v1/cash-flow-summaries",
	"/finance-management/v1/financial-calendar",
}

var nonCacheablePatterns = []string{
	"/oauth", "/user/create", "/user/delete",
	"/authorization-grant", "/link/v1/session",
	"/risk/", "/connector/", "/balance-refresh",
}

// Config holds options for creating an HTTPClient.
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

// HTTPClient is the production HTTP client. Safe for concurrent use.
type HTTPClient struct {
	defaultHeaders map[string]string
	lru            *cache.LRU
	httpClient     *http.Client
	retryPolicy    retry.Policy
	baseURL        string
	token          string
	userID         string
	mu             sync.RWMutex
	timeout        time.Duration
	cacheEnabled   bool
}

// New constructs an HTTPClient from Config.
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
		if te, ok := err.(*errors.TinkError); ok {
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

func (c *HTTPClient) AccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}
func (c *HTTPClient) SetAccessToken(t string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = t
}
func (c *HTTPClient) UserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}
func (c *HTTPClient) SetUserID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userID = id
}

// InvalidateUser removes all cache entries for the current (or given) user.
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

// Get performs a cached GET request and decodes into dst.
func (c *HTTPClient) Get(ctx context.Context, path string, query url.Values, dst interface{}) error {
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

// GetRaw performs a GET and returns raw bytes (for PDF/binary responses).
func (c *HTTPClient) GetRaw(ctx context.Context, path string, query url.Values) ([]byte, error) {
	var result []byte
	err := retry.Do(ctx, c.retryPolicy, func() error {
		var e error
		result, e = c.execRaw(ctx, http.MethodGet, buildPath(path, query), nil, "")
		return e
	})
	return result, err
}

// Post sends a JSON-encoded POST. Invalidates user cache on success.
func (c *HTTPClient) Post(ctx context.Context, path string, body, dst interface{}) error {
	if err := c.dispatch(ctx, http.MethodPost, path, body, dst, "application/json"); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// PostForm sends an application/x-www-form-urlencoded POST. Invalidates cache on success.
func (c *HTTPClient) PostForm(ctx context.Context, path string, form url.Values, dst interface{}) error {
	if err := retry.Do(ctx, c.retryPolicy, func() error {
		return c.execForm(ctx, path, form, dst)
	}); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// Patch sends a JSON-encoded PATCH. Invalidates cache on success.
func (c *HTTPClient) Patch(ctx context.Context, path string, body, dst interface{}) error {
	if err := c.dispatch(ctx, http.MethodPatch, path, body, dst, "application/json"); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// Delete sends a DELETE. Invalidates cache on success.
func (c *HTTPClient) Delete(ctx context.Context, path string) error {
	if err := c.dispatch(ctx, http.MethodDelete, path, nil, nil, ""); err != nil {
		return err
	}
	c.InvalidateUser()
	return nil
}

// ── core dispatch ──────────────────────────────────────────────────────────

func (c *HTTPClient) dispatch(ctx context.Context, method, path string, body, dst interface{}, ct string) error {
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

func (c *HTTPClient) execRaw(ctx context.Context, method, path string, body interface{}, ct string) ([]byte, error) {
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
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.FromDecodeError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.FromResponse(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func (c *HTTPClient) execForm(ctx context.Context, path string, form url.Values, dst interface{}) error {
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
	defer resp.Body.Close()

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

func (c *HTTPClient) cacheKey(path string) string {
	c.mu.RLock()
	scope := c.userID
	if scope == "" {
		scope = c.token
	}
	c.mu.RUnlock()
	if len(scope) > 16 {
		scope = scope[len(scope)-16:]
	}
	if scope == "" {
		scope = "public"
	}
	return scope + ":" + path
}

// ── helpers ────────────────────────────────────────────────────────────────

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

func ttlFor(path string) time.Duration {
	if ttl, ok := cacheTTLs[resourceType(path)]; ok {
		return ttl
	}
	return cacheTTLs["default"]
}

func buildPath(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

func roundTripJSON(src, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return errors.FromDecodeError(err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return errors.FromDecodeError(err)
	}
	return nil
}

func copyVal(v interface{}) interface{} {
	b, _ := json.Marshal(v)
	var m interface{}
	_ = json.Unmarshal(b, &m)
	return m
}
