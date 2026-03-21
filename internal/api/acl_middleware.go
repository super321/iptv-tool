package api

import (
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
)

// --- ACL Cache (goroutine-safe, 1-hour TTL) ---

type aclCache struct {
	mu       sync.RWMutex
	mode     string // "disabled", "whitelist", "blacklist"
	entries  []model.AccessControlEntry
	loadedAt time.Time
	ttl      time.Duration
}

var globalACLCache = &aclCache{ttl: 1 * time.Hour}

// load reads the ACL settings from the DB (called under write lock).
func (c *aclCache) load() {
	// Read mode
	var setting model.SystemSetting
	if err := model.DB.Where("key = ?", "access_control_mode").First(&setting).Error; err != nil {
		c.mode = "disabled"
	} else {
		c.mode = setting.Value
	}

	// Read entries
	var entries []model.AccessControlEntry
	model.DB.Find(&entries)
	c.entries = entries
	c.loadedAt = time.Now()
}

// get returns the cached mode and entries, refreshing if stale.
func (c *aclCache) get() (string, []model.AccessControlEntry) {
	c.mu.RLock()
	if !c.loadedAt.IsZero() && time.Since(c.loadedAt) < c.ttl {
		mode, entries := c.mode, c.entries
		c.mu.RUnlock()
		return mode, entries
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring write lock
	if !c.loadedAt.IsZero() && time.Since(c.loadedAt) < c.ttl {
		return c.mode, c.entries
	}
	c.load()
	return c.mode, c.entries
}

// invalidate forces a reload on next access.
func (c *aclCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loadedAt = time.Time{}
}

// --- IP Matching Logic ---

// matchIPSingle checks if clientIP equals the entry IP (handles IPv4/IPv6 normalization).
func matchIPSingle(clientIP, entryIP string) bool {
	cip := net.ParseIP(clientIP)
	eip := net.ParseIP(entryIP)
	if cip == nil || eip == nil {
		return false
	}
	return cip.Equal(eip)
}

// matchIPCIDR checks if clientIP is within the CIDR range.
func matchIPCIDR(clientIP, cidr string) bool {
	cip := net.ParseIP(clientIP)
	if cip == nil {
		return false
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipNet.Contains(cip)
}

// matchIPRange checks if clientIP is within the "startIP~endIP" range.
// Works for both IPv4 and IPv6 by comparing byte representations.
func matchIPRange(clientIP, rangeValue string) bool {
	parts := strings.SplitN(rangeValue, "~", 2)
	if len(parts) != 2 {
		return false
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	cip := net.ParseIP(clientIP)
	startIP := net.ParseIP(startStr)
	endIP := net.ParseIP(endStr)
	if cip == nil || startIP == nil || endIP == nil {
		return false
	}

	// Normalize all to 16-byte representation
	cipB := cip.To16()
	startB := startIP.To16()
	endB := endIP.To16()
	if cipB == nil || startB == nil || endB == nil {
		return false
	}

	return bytesCompare(cipB, startB) >= 0 && bytesCompare(cipB, endB) <= 0
}

// bytesCompare compares two byte slices lexicographically.
func bytesCompare(a, b net.IP) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// MatchEntry checks if a client IP matches a single access control entry.
func MatchEntry(clientIP string, entry model.AccessControlEntry) bool {
	switch entry.EntryType {
	case "single":
		return matchIPSingle(clientIP, entry.Value)
	case "cidr":
		return matchIPCIDR(clientIP, entry.Value)
	case "range":
		return matchIPRange(clientIP, entry.Value)
	default:
		return false
	}
}

// isBlacklistEntryActive checks if a blacklist entry is still active (not expired).
func isBlacklistEntryActive(entry model.AccessControlEntry) bool {
	if entry.BlockDays == nil || *entry.BlockDays == 0 {
		// Permanent block
		return true
	}
	expiry := entry.CreatedAt.AddDate(0, 0, *entry.BlockDays)
	return time.Now().Before(expiry)
}

// isLoopbackOrLocal checks if the IP is a loopback or link-local address.
func isLoopbackOrLocal(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

// IsIPAllowed checks whether a client IP is allowed given the current mode and entries.
// Loopback and link-local addresses are always allowed regardless of configuration.
func IsIPAllowed(clientIP, mode string, entries []model.AccessControlEntry) bool {
	if isLoopbackOrLocal(clientIP) {
		return true
	}

	switch mode {
	case "whitelist":
		for _, e := range entries {
			if e.ListType != "whitelist" {
				continue
			}
			if MatchEntry(clientIP, e) {
				return true
			}
		}
		return false // Not in whitelist → denied
	case "blacklist":
		for _, e := range entries {
			if e.ListType != "blacklist" {
				continue
			}
			if MatchEntry(clientIP, e) && isBlacklistEntryActive(e) {
				return false // Matched active blacklist entry → denied
			}
		}
		return true
	default:
		return true // Disabled or unknown → allowed
	}
}

// --- Gin Middleware ---

// AccessControlMiddleware creates a Gin middleware that enforces IP-based access control.
func AccessControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Exempt locale endpoints so the frontend can always load i18n data
		if path == "/api/locales" || strings.HasPrefix(path, "/api/locales/") {
			c.Next()
			return
		}

		mode, entries := globalACLCache.get()
		if mode == "disabled" || mode == "" {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if !IsIPAllowed(clientIP, mode, entries) {
			slog.Warn("Access denied by ACL", "client_ip", clientIP, "mode", mode)
			c.AbortWithStatus(403)
			return
		}

		c.Next()
	}
}
