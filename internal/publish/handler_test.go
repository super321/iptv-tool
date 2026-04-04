package publish

import "testing"

func TestCheckUserAgent(t *testing.T) {
	tests := []struct {
		name          string
		reqUA         string
		allowedValues string
		want          bool
	}{
		{
			name:          "empty UA rejected",
			reqUA:         "",
			allowedValues: "Mozilla",
			want:          false,
		},
		{
			name:          "simple match",
			reqUA:         "Mozilla/5.0 Chrome/120",
			allowedValues: "Mozilla",
			want:          true,
		},
		{
			name:          "no match",
			reqUA:         "curl/7.88",
			allowedValues: "Mozilla\nChrome",
			want:          false,
		},
		{
			name:          "multiple values second matches",
			reqUA:         "Mozilla/5.0 Chrome/120",
			allowedValues: "Safari\nChrome",
			want:          true,
		},
		{
			name:          "UA containing commas matches correctly",
			reqUA:         "B700-V2A|Mozilla|5.0|ztebw(Chrome)|1.2.0;Resolution(PAL,720p,1080i) AppleWebKit/535.7 (KHTML, like Gecko) Chrome/16.0.912.63 Safari/535.7",
			allowedValues: "B700-V2A|Mozilla|5.0|ztebw(Chrome)|1.2.0;Resolution(PAL,720p,1080i) AppleWebKit/535.7 (KHTML, like Gecko) Chrome/16.0.912.63 Safari/535.7",
			want:          true,
		},
		{
			name:          "partial match with comma-containing UA",
			reqUA:         "B700-V2A|Mozilla|5.0|ztebw(Chrome)|1.2.0;Resolution(PAL,720p,1080i) AppleWebKit/535.7",
			allowedValues: "Resolution(PAL,720p,1080i)",
			want:          true,
		},
		{
			name:          "blank lines ignored",
			reqUA:         "Mozilla/5.0",
			allowedValues: "\n\nMozilla\n\n",
			want:          true,
		},
		{
			name:          "whitespace trimmed",
			reqUA:         "Mozilla/5.0",
			allowedValues: "  Mozilla  \n  Chrome  ",
			want:          true,
		},
		{
			name:          "empty allowed values",
			reqUA:         "Mozilla/5.0",
			allowedValues: "",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkUserAgent(tt.reqUA, tt.allowedValues)
			if got != tt.want {
				t.Errorf("checkUserAgent(%q, %q) = %v, want %v", tt.reqUA, tt.allowedValues, got, tt.want)
			}
		})
	}
}
