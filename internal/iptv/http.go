package iptv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// HTTPClient is a wrapper around http.Client that implements retries and backoff
type HTTPClient struct {
	client     *http.Client
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
}

// NewHTTPClient creates a new HTTP client with retry capabilities
func NewHTTPClient(client *http.Client) *HTTPClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &HTTPClient{
		client:     client,
		maxRetries: 3, // 最多重试3次
		baseDelay:  1 * time.Second,
		maxDelay:   5 * time.Second,
	}
}

// Do executes an HTTP request with retry logic and exponential backoff
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Buffer the request body so it can be replayed on retries.
	// http.Request.Body is an io.Reader stream — once consumed, retries would send an empty body.
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Reset body for retry
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
			// 指数退避算法: 1s, 2s, 4s...
			delay := c.baseDelay * time.Duration(1<<(attempt-1))
			if delay > c.maxDelay {
				delay = c.maxDelay
			}
			// 加入 Jitter (随机抖动) 避免雪崩
			jitter := time.Duration(rand.Int63n(int64(delay / 4)))

			// 如果 context 已经 done，提前退出
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(delay + jitter):
			}
		}

		resp, err = c.client.Do(req)
		if err != nil {
			// 网络错误或超时，进入下一次重试
			continue
		}

		// 检查需要重试的 HTTP 状态码
		if resp.StatusCode == http.StatusTooManyRequests || // 429
			resp.StatusCode == http.StatusBadGateway || // 502
			resp.StatusCode == http.StatusServiceUnavailable || // 503
			resp.StatusCode == http.StatusGatewayTimeout { // 504

			// 关闭 body 释放连接，然后重试
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			continue
		}

		// 成功或者遇到不可重试的错误（如 401 认证失败, 404 找不到），直接返回
		return resp, nil
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, err)
	}
	if resp != nil {
		return nil, fmt.Errorf("request failed after %d retries with status: %d", c.maxRetries, resp.StatusCode)
	}
	return nil, fmt.Errorf("request failed after %d retries", c.maxRetries)
}

// SetCommonHeaders is a helper to apply user configured headers
func SetCommonHeaders(req *http.Request, config *Config) {
	if config != nil && len(config.Headers) > 0 {
		for k, v := range config.Headers {
			req.Header.Set(k, v)
		}
	}
}

// RateLimiter is a simple worker pool / rate limiter for batch requests
type RateLimiter struct {
	pool chan struct{}
}

// NewRateLimiter creates a new limiter with max concurrent requests
func NewRateLimiter(maxConcurrent int) *RateLimiter {
	return &RateLimiter{
		pool: make(chan struct{}, maxConcurrent),
	}
}

// Acquire blocks until a slot is available
func (l *RateLimiter) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.pool <- struct{}{}:
		// Add a base delay to simulate real STB behavior and avoid triggering 503
		time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond) // 200ms - 500ms delay
		return nil
	}
}

// Release frees a slot
func (l *RateLimiter) Release() {
	<-l.pool
}
