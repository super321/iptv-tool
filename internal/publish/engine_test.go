package publish

import (
	"testing"

	"iptv-tool-v2/internal/model"
)

func TestExtractBestURL(t *testing.T) {
	tests := []struct {
		name          string
		addressType   string // "unicast" or "multicast"
		multicastType string
		udpxyURL      string
		rawURLs       string
		catchupURL    string
		want          string
	}{
		// ===== 单播优先 =====
		{
			name:        "unicast_priority_has_both_unicast_and_multicast",
			addressType: "unicast",
			rawURLs:     "igmp://239.93.1.23:5140|http://113.136.1.1/live/channel1",
			catchupURL:  "",
			want:        "http://113.136.1.1/live/channel1",
		},
		{
			name:        "unicast_priority_only_unicast",
			addressType: "unicast",
			rawURLs:     "http://113.136.1.1/live/channel1",
			catchupURL:  "",
			want:        "http://113.136.1.1/live/channel1",
		},
		{
			name:        "unicast_priority_only_multicast_with_unicast_catchup",
			addressType: "unicast",
			rawURLs:     "igmp://239.93.1.23:5140",
			catchupURL:  "http://113.136.1.1/timeshift/channel1",
			want:        "http://113.136.1.1/timeshift/channel1",
		},
		{
			name:        "unicast_priority_only_multicast_with_multicast_catchup",
			addressType: "unicast",
			rawURLs:     "igmp://239.93.1.23:5140",
			catchupURL:  "rtp://239.93.1.23:5140",
			want:        "igmp://239.93.1.23:5140",
		},
		{
			name:        "unicast_priority_only_multicast_no_catchup",
			addressType: "unicast",
			rawURLs:     "igmp://239.93.1.23:5140",
			catchupURL:  "",
			want:        "igmp://239.93.1.23:5140",
		},
		{
			name:          "unicast_fallback_multicast_with_udpxy",
			addressType:   "unicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:4022",
			rawURLs:       "igmp://239.93.1.23:5140",
			catchupURL:    "",
			want:          "http://192.168.1.1:4022/rtp/239.93.1.23:5140",
		},
		{
			name:          "unicast_fallback_multicast_with_rtp",
			addressType:   "unicast",
			multicastType: "rtp",
			rawURLs:       "igmp://239.93.1.23:5140",
			catchupURL:    "",
			want:          "rtp://239.93.1.23:5140",
		},
		{
			name:          "unicast_fallback_multicast_with_igmp",
			addressType:   "unicast",
			multicastType: "igmp",
			rawURLs:       "igmp://239.93.1.23:5140",
			catchupURL:    "",
			want:          "igmp://239.93.1.23:5140",
		},
		{
			name:        "unicast_priority_multicast_first_then_unicast",
			addressType: "unicast",
			rawURLs:     "rtp://239.93.1.23:5140|http://113.136.1.1/live/channel1",
			catchupURL:  "",
			want:        "http://113.136.1.1/live/channel1",
		},
		{
			name:        "unicast_priority_multiple_unicast",
			addressType: "unicast",
			rawURLs:     "http://first.com/live|http://second.com/live",
			catchupURL:  "",
			want:        "http://first.com/live",
		},
		{
			name:        "unicast_priority_has_unicast_ignore_catchup",
			addressType: "unicast",
			rawURLs:     "igmp://239.93.1.23:5140|http://113.136.1.1/live/channel1",
			catchupURL:  "http://113.136.1.1/timeshift/channel1",
			want:        "http://113.136.1.1/live/channel1",
		},

		// ===== 组播优先 =====
		{
			name:        "multicast_priority_has_both",
			addressType: "multicast",
			rawURLs:     "igmp://239.93.1.23:5140|http://113.136.1.1/live/channel1",
			catchupURL:  "",
			want:        "igmp://239.93.1.23:5140",
		},
		{
			name:        "multicast_priority_only_unicast",
			addressType: "multicast",
			rawURLs:     "http://113.136.1.1/live/channel1",
			catchupURL:  "",
			want:        "http://113.136.1.1/live/channel1",
		},
		{
			name:        "multicast_priority_only_multicast",
			addressType: "multicast",
			rawURLs:     "igmp://239.93.1.23:5140",
			catchupURL:  "",
			want:        "igmp://239.93.1.23:5140",
		},
		{
			name:          "multicast_priority_udpxy_conversion",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:4022",
			rawURLs:       "igmp://239.93.1.23:5140|http://113.136.1.1/live/channel1",
			catchupURL:    "",
			want:          "http://192.168.1.1:4022/rtp/239.93.1.23:5140",
		},
		{
			name:          "multicast_priority_rtp_conversion",
			addressType:   "multicast",
			multicastType: "rtp",
			rawURLs:       "igmp://239.93.1.23:5140",
			catchupURL:    "",
			want:          "rtp://239.93.1.23:5140",
		},
		{
			name:          "multicast_priority_igmp_passthrough",
			addressType:   "multicast",
			multicastType: "igmp",
			rawURLs:       "igmp://239.93.1.23:5140",
			catchupURL:    "",
			want:          "igmp://239.93.1.23:5140",
		},
		{
			name:          "multicast_priority_rtp_already_rtp",
			addressType:   "multicast",
			multicastType: "rtp",
			rawURLs:       "rtp://239.93.1.23:5140",
			catchupURL:    "",
			want:          "rtp://239.93.1.23:5140",
		},

		// ===== 边界情况 =====
		{
			name:        "empty_urls_with_pipes",
			addressType: "unicast",
			rawURLs:     "|igmp://239.93.1.23:5140|",
			catchupURL:  "",
			want:        "igmp://239.93.1.23:5140",
		},
		{
			name:        "spaces_around_urls",
			addressType: "unicast",
			rawURLs:     " igmp://239.93.1.23:5140 | http://113.136.1.1/live/ch1 ",
			catchupURL:  "",
			want:        "http://113.136.1.1/live/ch1",
		},
		{
			name:          "multicast_priority_udpxy_with_catchup_url_as_multicast",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:4022",
			rawURLs:       "igmp://239.93.1.23:5140",
			catchupURL:    "http://113.136.1.1/timeshift/ch1",
			want:          "http://192.168.1.1:4022/rtp/239.93.1.23:5140",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Engine{
				iface: model.PublishInterface{
					AddressType:   tt.addressType,
					MulticastType: tt.multicastType,
					UDPxyURL:      tt.udpxyURL,
				},
			}
			got := e.extractBestURL(tt.rawURLs, tt.catchupURL)
			if got != tt.want {
				t.Errorf("extractBestURL(%q, %q) = %q, want %q", tt.rawURLs, tt.catchupURL, got, tt.want)
			}
		})
	}
}
