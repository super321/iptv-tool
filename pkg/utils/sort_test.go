package utils

import "testing"

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		// Identical strings
		{"identical", "abc", "abc", false},
		{"identical numeric", "123", "123", false},
		{"empty strings", "", "", false},

		// Pure alphabetical
		{"alpha less", "abc", "abd", true},
		{"alpha greater", "abd", "abc", false},
		{"alpha case sensitive", "A", "a", true}, // 'A' (65) < 'a' (97)

		// Pure numeric
		{"numeric less", "2", "10", true},
		{"numeric greater", "10", "2", false},
		{"numeric equal value different length", "01", "1", false}, // same value, "01" has longer repr so NOT less

		// Mixed channel names — the core use case
		{"CCTV channels", "CCTV-2", "CCTV-10", true},
		{"CCTV channels reversed", "CCTV-10", "CCTV-2", false},
		{"CCTV same", "CCTV-1", "CCTV-1", false},

		// Prefix relationship
		{"prefix shorter", "abc", "abcd", true},
		{"prefix longer", "abcd", "abc", false},

		// Empty vs non-empty
		{"empty vs non-empty", "", "a", true},
		{"non-empty vs empty", "a", "", false},

		// Numbers at different positions
		{"numbers at start", "2abc", "10abc", true},
		{"numbers at end", "abc2", "abc10", true},

		// Multiple numeric segments
		{"multi numeric segments", "ch1ep2", "ch1ep10", true},
		{"multi numeric segments equal first", "ch2ep1", "ch10ep1", true},

		// Leading zeros
		{"leading zeros same value", "007", "7", false},      // 7==7, but "007" has longer repr → NOT less
		{"leading zeros different value", "009", "10", true}, // 9 < 10

		// Unicode / Chinese characters
		{"chinese characters", "频道1", "频道2", true},
		{"chinese characters reversed", "频道2", "频道1", false},
		{"chinese equal", "频道1", "频道1", false},

		// Large numbers
		{"large numbers", "channel100", "channel200", true},
		{"large numbers reversed", "channel200", "channel100", false},

		// Only numbers differ in multi-part
		{"version-like", "v1.2.3", "v1.2.10", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NaturalLess(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("NaturalLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestNaturalLess_Transitivity(t *testing.T) {
	// If a < b and b < c, then a < c
	a, b, c := "CCTV-1", "CCTV-5", "CCTV-13"
	if !NaturalLess(a, b) {
		t.Errorf("expected %q < %q", a, b)
	}
	if !NaturalLess(b, c) {
		t.Errorf("expected %q < %q", b, c)
	}
	if !NaturalLess(a, c) {
		t.Errorf("expected %q < %q (transitivity)", a, c)
	}
}
