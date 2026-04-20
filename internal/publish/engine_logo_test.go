package publish

import (
	"strings"
	"testing"

	"iptv-tool-v2/internal/model"
)

func TestFormatM3U_LogoFallback(t *testing.T) {
	tests := []struct {
		name      string
		channels  []AggregatedChannel
		wantLogo  string // expected tvg-logo value substring, empty means no tvg-logo
		noTvgLogo bool   // if true, tvg-logo should NOT appear
	}{
		{
			name: "priority1_logo_management_match",
			channels: []AggregatedChannel{
				{
					Name:       "CCTV1",
					URL:        "http://example.com/live/cctv1",
					Group:      "央视",
					Logo:       "/logo/cctv1.png",
					SourceLogo: "http://original.com/logo/cctv1.png",
				},
			},
			wantLogo: `tvg-logo="http://192.168.1.1:8023/logo/cctv1.png"`,
		},
		{
			name: "priority2_source_logo_only",
			channels: []AggregatedChannel{
				{
					Name:       "CCTV2",
					URL:        "http://example.com/live/cctv2",
					Group:      "央视",
					Logo:       "",
					SourceLogo: "http://original.com/logo/cctv2.png",
				},
			},
			wantLogo: `tvg-logo="http://original.com/logo/cctv2.png"`,
		},
		{
			name: "priority3_no_logo_at_all",
			channels: []AggregatedChannel{
				{
					Name:       "CCTV3",
					URL:        "http://example.com/live/cctv3",
					Group:      "央视",
					Logo:       "",
					SourceLogo: "",
				},
			},
			noTvgLogo: true,
		},
		{
			name: "priority1_logo_management_no_source_logo",
			channels: []AggregatedChannel{
				{
					Name:       "CCTV4",
					URL:        "http://example.com/live/cctv4",
					Group:      "央视",
					Logo:       "/logo/cctv4.png",
					SourceLogo: "",
				},
			},
			wantLogo: `tvg-logo="http://192.168.1.1:8023/logo/cctv4.png"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Engine{
				iface: model.PublishInterface{},
			}
			result := e.FormatM3U(tt.channels, "http://192.168.1.1:8023")

			if tt.noTvgLogo {
				if strings.Contains(result, "tvg-logo") {
					t.Errorf("expected no tvg-logo attribute, but found one in:\n%s", result)
				}
			} else {
				if !strings.Contains(result, tt.wantLogo) {
					t.Errorf("expected %q in output, but got:\n%s", tt.wantLogo, result)
				}
			}
		})
	}
}
