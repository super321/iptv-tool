package service

import (
	"testing"
)

func TestSelectDetectURL(t *testing.T) {
	tests := []struct {
		name       string
		rawURLs    string
		catchupURL string
		strategy   string
		want       string
	}{
		// ===== Unicast priority =====
		{
			name:     "unicast_has_both",
			rawURLs:  "igmp://239.93.1.23:5140|http://113.136.1.1/live/ch1",
			strategy: "unicast",
			want:     "http://113.136.1.1/live/ch1",
		},
		{
			name:     "unicast_only_unicast",
			rawURLs:  "http://113.136.1.1/live/ch1",
			strategy: "unicast",
			want:     "http://113.136.1.1/live/ch1",
		},
		{
			name:       "unicast_only_multicast_with_unicast_catchup",
			rawURLs:    "igmp://239.93.1.23:5140",
			catchupURL: "http://113.136.1.1/timeshift/ch1",
			strategy:   "unicast",
			want:       "http://113.136.1.1/timeshift/ch1",
		},
		{
			name:       "unicast_only_multicast_with_multicast_catchup",
			rawURLs:    "igmp://239.93.1.23:5140",
			catchupURL: "rtp://239.93.1.23:5140",
			strategy:   "unicast",
			want:       "rtp://239.93.1.23:5140", // igmp→rtp conversion
		},
		{
			name:     "unicast_only_multicast_no_catchup",
			rawURLs:  "igmp://239.93.1.23:5140",
			strategy: "unicast",
			want:     "rtp://239.93.1.23:5140", // igmp→rtp for ffprobe
		},
		{
			name:       "unicast_has_unicast_ignore_catchup",
			rawURLs:    "igmp://239.93.1.23:5140|http://113.136.1.1/live/ch1",
			catchupURL: "http://113.136.1.1/timeshift/ch1",
			strategy:   "unicast",
			want:       "http://113.136.1.1/live/ch1",
		},

		// ===== Multicast priority =====
		{
			name:     "multicast_has_both",
			rawURLs:  "igmp://239.93.1.23:5140|http://113.136.1.1/live/ch1",
			strategy: "multicast",
			want:     "rtp://239.93.1.23:5140", // igmp→rtp for ffprobe
		},
		{
			name:     "multicast_only_unicast",
			rawURLs:  "http://113.136.1.1/live/ch1",
			strategy: "multicast",
			want:     "http://113.136.1.1/live/ch1",
		},
		{
			name:     "multicast_only_multicast",
			rawURLs:  "igmp://239.93.1.23:5140",
			strategy: "multicast",
			want:     "rtp://239.93.1.23:5140",
		},
		{
			name:     "multicast_rtp_passthrough",
			rawURLs:  "rtp://239.93.1.23:5140",
			strategy: "multicast",
			want:     "rtp://239.93.1.23:5140",
		},

		// ===== Edge cases =====
		{
			name:     "empty_entries_with_pipes",
			rawURLs:  "|igmp://239.93.1.23:5140|",
			strategy: "unicast",
			want:     "rtp://239.93.1.23:5140",
		},
		{
			name:     "spaces_around_urls",
			rawURLs:  " igmp://239.93.1.23:5140 | http://113.136.1.1/live/ch1 ",
			strategy: "unicast",
			want:     "http://113.136.1.1/live/ch1",
		},
		{
			name:     "default_strategy_is_multicast",
			rawURLs:  "igmp://239.93.1.23:5140|http://113.136.1.1/live/ch1",
			strategy: "",
			want:     "rtp://239.93.1.23:5140",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectDetectURL(tt.rawURLs, tt.catchupURL, tt.strategy)
			if got != tt.want {
				t.Errorf("SelectDetectURL(%q, %q, %q) = %q, want %q",
					tt.rawURLs, tt.catchupURL, tt.strategy, got, tt.want)
			}
		})
	}
}
