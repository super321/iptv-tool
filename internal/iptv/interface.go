package iptv

import (
	"context"
)

// Client is the interface for interacting with different IPTV platforms (Huawei, ZTE, etc.)
type Client interface {
	// Authenticate performs the login to the IPTV platform
	Authenticate(ctx context.Context) error

	// GetAllChannelList retrieves the list of live channels
	GetAllChannelList(ctx context.Context) ([]Channel, error)

	// GetAllChannelProgramList retrieves the EPG for the provided channels
	// This will internally use the configured EPG Strategy.
	GetAllChannelProgramList(ctx context.Context, channels []Channel) ([]ChannelProgramList, error)
}

// EPGFetchStrategy is the Strategy interface for different regional EPG fetch methods
type EPGFetchStrategy interface {
	// Name returns the unique identifier for the strategy (e.g., "liveplay", "vsp")
	Name() string

	// Fetch executes the actual logic to download and parse EPG data.
	// client is the HTTP client with retry/rate-limit capabilities.
	// authInfo contains any tokens or session IDs required.
	// limiter controls concurrency for strategies that cannot bulk-fetch.
	Fetch(ctx context.Context, client *HTTPClient, channels []Channel, authInfo map[string]string, limiter *RateLimiter) ([]ChannelProgramList, error)
}
