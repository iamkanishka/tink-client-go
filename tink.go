// Package tink is the public entrypoint for the Tink Open Banking Go SDK.
//
// Import this package to create a client and access all Tink services.
package tink

import "github.com/iamkanishka/tink-client-go/client"

// Client is the main Tink API client.
type Client = client.Client

// Config configures a new client.
type Config = client.Config

// New creates a new Tink API client.
func New(cfg Config) (*Client, error) {
	return client.New(cfg)
}

// Option configures the client using functional options.
type Option = client.Option

// NewWithOptions constructs a client using options.
func NewWithOptions(opts ...Option) (*Client, error) {
	return client.NewWithOptions(opts...)
}

var (
	WithCredentials  = client.WithCredentials
	WithAccessToken  = client.WithAccessToken
	WithBaseURL      = client.WithBaseURL
	WithTimeout      = client.WithTimeout
	WithMaxRetries   = client.WithMaxRetries
	WithHTTPClient   = client.WithHTTPClient
	WithHeader       = client.WithHeader
	WithDisableCache = client.WithDisableCache
)

// TokenInfo contains parsed token expiry metadata.
type TokenInfo = client.TokenInfo

var (
	ParseToken = client.ParseToken
	IsExpired  = client.IsExpired
)
