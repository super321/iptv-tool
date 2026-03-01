package huawei

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"iptv-tool-v2/internal/iptv"
)

var (
	ErrParseChProgList   = errors.New("failed to parse channel program list")
	ErrChProgListIsEmpty = errors.New("the list of programs is empty")
	ErrEPGApiNotFound    = errors.New("epg api not found")
)

// GetAllChannelProgramList fetches the EPG for all channels using the configured strategy.
func (c *Client) GetAllChannelProgramList(ctx context.Context, channels []iptv.Channel) ([]iptv.ChannelProgramList, error) {
	if c.Token == nil || c.Token.UserToken == "" {
		return nil, errors.New("not authenticated, call Authenticate first")
	}

	authInfo := map[string]string{
		"UserToken":      c.Token.UserToken,
		"JSESSIONID":     c.Token.JSESSIONID,
		"Stbid":          c.Token.Stbid,
		"UserID":         c.config.GetAuthParam("UserID"),
		"ProviderSuffix": c.config.ProviderSuffix,
		"Lang":           c.config.GetAuthParam("Lang"),
		"Conntype":       c.config.GetAuthParam("conntype"),
		"Host":           c.host, // The specific redirected EDS host
	}

	// Pass user-configured custom headers with "Header-" prefix so strategies can apply them
	for k, v := range c.config.Headers {
		authInfo["Header-"+k] = v
	}

	strategyName := strings.ToLower(strings.TrimSpace(c.config.EPGStrategy))

	// Create a limiter if we need to fall back to per-channel fetching
	// Limit to 3 concurrent STB simulation requests to avoid 503 errors
	limiter := iptv.NewRateLimiter(3)

	var result []iptv.ChannelProgramList
	var err error

	// 1. Registry matching or Auto Detect
	if strategyName != "" && strategyName != "auto" {
		strategy := GetEPGStrategy(strategyName)
		if strategy == nil {
			return nil, fmt.Errorf("epg strategy '%s' not found or not registered", strategyName)
		}
		result, err = strategy.Fetch(ctx, c.httpClient, channels, authInfo, limiter)
	} else {
		// AUTO DETECT MODE
		// Try strategies one by one until one succeeds without returning ErrEPGApiNotFound
		result, err = c.autoDetectEPGStrategy(ctx, channels, authInfo, limiter)
	}

	return result, err
}

func (c *Client) autoDetectEPGStrategy(ctx context.Context, channels []iptv.Channel, authInfo map[string]string, limiter *iptv.RateLimiter) ([]iptv.ChannelProgramList, error) {
	strategies := GetAllEPGStrategies()

	for _, strategy := range strategies {
		result, err := strategy.Fetch(ctx, c.httpClient, channels, authInfo, limiter)
		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				// The API endpoint for this strategy does not exist on this server, try the next one
				continue
			}
			// It's a network error or parsing error for a *valid* endpoint
			return nil, fmt.Errorf("strategy %s failed: %w", strategy.Name(), err)
		}

		// If it succeeds, update the config so we don't have to brute force next time
		c.config.EPGStrategy = strategy.Name()
		return result, nil
	}

	return nil, errors.New("no suitable EPG API strategy found on this IPTV server")
}
