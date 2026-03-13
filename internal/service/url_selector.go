package service

import "strings"

// SelectDetectURL picks the best URL for detection (ffprobe) based on the given strategy.
//
// Strategy values:
//   - "unicast": prefer unicast addresses, fallback to unicast catchup, then multicast, then raw
//   - "multicast" (or any other value): prefer multicast addresses, fallback to unicast, then raw
//
// For multicast URLs, igmp:// is automatically converted to rtp:// for ffprobe compatibility.
func SelectDetectURL(rawURLs, catchupURL, strategy string) string {
	urls := strings.Split(rawURLs, "|")

	var multicastURL string
	var unicastURL string

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if strings.HasPrefix(u, "igmp://") || strings.HasPrefix(u, "rtp://") {
			if multicastURL == "" {
				multicastURL = u
			}
		} else if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") ||
			strings.HasPrefix(u, "rtsp://") || strings.HasPrefix(u, "rtmp://") {
			if unicastURL == "" {
				unicastURL = u
			}
		}
	}

	if strategy == "unicast" {
		// Priority 1: direct unicast URL
		if unicastURL != "" {
			return unicastURL
		}
		// Priority 2: only multicast available, but catchup is unicast → use catchup
		if multicastURL != "" && catchupURL != "" && !isMulticastAddr(catchupURL) {
			return catchupURL
		}
		// Priority 3: fallback to multicast (convert igmp→rtp for ffprobe)
		if multicastURL != "" {
			return igmpToRtp(multicastURL)
		}
		// Priority 4: fallback to raw first URL
		return firstURL(rawURLs)
	}

	// Multicast priority (default)
	if multicastURL != "" {
		return igmpToRtp(multicastURL)
	}
	if unicastURL != "" {
		return unicastURL
	}
	return firstURL(rawURLs)
}

// isMulticastAddr checks if a URL is a multicast address
func isMulticastAddr(url string) bool {
	return strings.HasPrefix(url, "igmp://") || strings.HasPrefix(url, "rtp://")
}

// igmpToRtp converts igmp:// to rtp:// for ffprobe compatibility
func igmpToRtp(url string) string {
	if strings.HasPrefix(url, "igmp://") {
		return "rtp://" + strings.TrimPrefix(url, "igmp://")
	}
	return url
}

// firstURL returns the first pipe-separated URL, trimmed
func firstURL(rawURLs string) string {
	if idx := strings.Index(rawURLs, "|"); idx > 0 {
		return strings.TrimSpace(rawURLs[:idx])
	}
	return strings.TrimSpace(rawURLs)
}
