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
		fccEnabled    bool
		fccType       string
		customParams  string
		rawURLs       string
		catchupURL    string
		fccIP         string
		fccPort       string
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

		// ===== FCC 相关测试 =====
		{
			name:          "fcc_telecom_default_protocol",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    true,
			fccType:       "telecom",
			rawURLs:       "igmp://239.253.64.120:5140",
			fccIP:         "10.255.14.152",
			fccPort:       "15970",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140?fcc=10.255.14.152:15970",
		},
		{
			name:          "fcc_huawei_protocol",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    true,
			fccType:       "huawei",
			rawURLs:       "igmp://239.253.64.120:5140",
			fccIP:         "10.255.14.152",
			fccPort:       "8027",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140?fcc=10.255.14.152:8027&fcc-type=huawei",
		},
		{
			name:          "fcc_enabled_no_channel_fcc_data",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    true,
			fccType:       "telecom",
			rawURLs:       "igmp://239.93.42.42:5140",
			fccIP:         "",
			fccPort:       "",
			want:          "http://192.168.1.1:5140/rtp/239.93.42.42:5140",
		},
		{
			name:          "fcc_disabled_with_channel_fcc_data",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    false,
			rawURLs:       "igmp://239.253.64.120:5140",
			fccIP:         "10.255.14.152",
			fccPort:       "8027",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140",
		},
		{
			name:          "fcc_unicast_fallback_to_multicast_with_fcc",
			addressType:   "unicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    true,
			fccType:       "telecom",
			rawURLs:       "igmp://239.253.64.120:5140",
			fccIP:         "10.7.10.172",
			fccPort:       "8027",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140?fcc=10.7.10.172:8027",
		},

		// ===== Custom Params 相关测试 =====
		{
			name:          "custom_params_only_no_fcc",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    false,
			customParams:  `[{"key":"r2h-token","value":"abc123"}]`,
			rawURLs:       "igmp://239.253.64.120:5140",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140?r2h-token=abc123",
		},
		{
			name:          "fcc_and_custom_params_combined",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    true,
			fccType:       "telecom",
			customParams:  `[{"key":"r2h-token","value":"abc123"}]`,
			rawURLs:       "igmp://239.253.64.120:5140",
			fccIP:         "10.255.14.152",
			fccPort:       "15970",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140?fcc=10.255.14.152:15970&r2h-token=abc123",
		},
		{
			name:          "multiple_custom_params",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    false,
			customParams:  `[{"key":"r2h-token","value":"abc123"},{"key":"r2h-ifname","value":"eth0"}]`,
			rawURLs:       "igmp://239.253.64.120:5140",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140?r2h-token=abc123&r2h-ifname=eth0",
		},
		{
			name:          "empty_custom_params_no_effect",
			addressType:   "multicast",
			multicastType: "udpxy",
			udpxyURL:      "http://192.168.1.1:5140",
			fccEnabled:    false,
			customParams:  "",
			rawURLs:       "igmp://239.253.64.120:5140",
			want:          "http://192.168.1.1:5140/rtp/239.253.64.120:5140",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Engine{
				iface: model.PublishInterface{
					AddressType:   tt.addressType,
					MulticastType: tt.multicastType,
					UDPxyURL:      tt.udpxyURL,
					FCCEnabled:    tt.fccEnabled,
					FCCType:       tt.fccType,
					CustomParams:  tt.customParams,
				},
			}
			got := e.extractBestURL(tt.rawURLs, tt.catchupURL, tt.fccIP, tt.fccPort)
			if got != tt.want {
				t.Errorf("extractBestURL(%q, %q) = %q, want %q", tt.rawURLs, tt.catchupURL, got, tt.want)
			}
		})
	}
}
