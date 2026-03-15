package version

import (
	"strconv"
	"strings"
)

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X iptv-tool-v2/internal/version.Version=v1.0.0"
var Version = "dev"

// CompareVersions compares two semver version strings.
// Returns -1 if current < latest, 0 if equal, 1 if current > latest.
// Handles "v" prefix and "dev" (always considered older than any release).
func CompareVersions(current, latest string) int {
	current = strings.TrimPrefix(strings.TrimSpace(current), "v")
	latest = strings.TrimPrefix(strings.TrimSpace(latest), "v")

	// "dev" is always older than any release version
	if current == "dev" && latest == "dev" {
		return 0
	}
	if current == "dev" {
		return -1
	}
	if latest == "dev" {
		return 1
	}

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	// Compare each segment numerically
	maxLen := len(currentParts)
	if len(latestParts) > maxLen {
		maxLen = len(latestParts)
	}

	for i := 0; i < maxLen; i++ {
		var c, l int
		if i < len(currentParts) {
			c, _ = strconv.Atoi(currentParts[i])
		}
		if i < len(latestParts) {
			l, _ = strconv.Atoi(latestParts[i])
		}
		if c < l {
			return -1
		}
		if c > l {
			return 1
		}
	}
	return 0
}
