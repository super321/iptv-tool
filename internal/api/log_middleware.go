package api

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/service"
)

// AccessLogMiddleware returns a Gin middleware that records access logs
// into the given AccessLogBuffer. It only logs backend API (/api/) and
// subscription (/sub/) requests, skipping static resources and log APIs.
// It also records IP access statistics via the AccessStatService.
func AccessLogMiddleware(buf *AccessLogBuffer, accessStatSvc *service.AccessStatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Only record backend API and subscription requests
		// Skip static resources, frontend files, logos, and log APIs (feedback loops)
		isAPI := strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/api/logs/")
		isSub := strings.HasPrefix(path, "/sub/")
		if !isAPI && !isSub {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		buf.Append(AccessLogEntry{
			Time:      start.Format("2006-01-02 15:04:05"),
			ClientIP:  c.ClientIP(),
			Method:    c.Request.Method,
			Path:      path,
			Status:    c.Writer.Status(),
			Latency:   formatLatency(latency),
			UserAgent: c.Request.UserAgent(),
		})

		// Record IP access stat (non-blocking)
		if accessStatSvc != nil {
			accessStatSvc.Record(c.ClientIP(), isSub)
		}
	}
}

// formatLatency formats duration as a human-readable string
func formatLatency(d time.Duration) string {
	if d < time.Millisecond {
		return d.Truncate(time.Microsecond).String()
	}
	return d.Truncate(time.Millisecond).String()
}
