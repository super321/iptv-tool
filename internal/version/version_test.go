package version

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
	}{
		{"equal versions", "v1.0.0", "v1.0.0", 0},
		{"current older patch", "v1.0.0", "v1.0.1", -1},
		{"current newer minor", "v1.1.0", "v1.0.1", 1},
		{"current newer major", "v2.0.0", "v1.9.9", 1},
		{"current older major", "v1.9.9", "v2.0.0", -1},
		{"dev vs release", "dev", "v1.0.0", -1},
		{"release vs dev", "v1.0.0", "dev", 1},
		{"dev vs dev", "dev", "dev", 0},
		{"no v prefix", "1.0.0", "1.0.1", -1},
		{"mixed prefix", "v1.0.0", "1.0.0", 0},
		{"different segment count", "v1.0", "v1.0.0", 0},
		{"different segment count newer", "v1.0", "v1.0.1", -1},
		{"whitespace", " v1.0.0 ", "v1.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
