package api

import (
	"testing"
	"time"

	"iptv-tool-v2/internal/model"
)

// --- matchIPSingle ---

func TestMatchIPSingle_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		clientIP string
		entryIP  string
		want     bool
	}{
		{"exact match", "192.168.1.1", "192.168.1.1", true},
		{"mismatch", "192.168.1.1", "192.168.1.2", false},
		{"same IP different subnet", "10.0.0.1", "10.0.0.1", true},
		{"different IP", "10.0.0.1", "10.0.0.2", false},
		{"invalid client", "invalid", "192.168.1.1", false},
		{"invalid entry", "192.168.1.1", "invalid", false},
		{"both invalid", "invalid", "invalid", false},
		{"empty strings", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIPSingle(tt.clientIP, tt.entryIP)
			if got != tt.want {
				t.Errorf("matchIPSingle(%q, %q) = %v, want %v", tt.clientIP, tt.entryIP, got, tt.want)
			}
		})
	}
}

func TestMatchIPSingle_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		clientIP string
		entryIP  string
		want     bool
	}{
		{"loopback", "::1", "::1", true},
		{"loopback mismatch", "::1", "::2", false},
		{"global unicast match", "2001:db8::1", "2001:db8::1", true},
		{"global unicast mismatch", "2001:db8::1", "2001:db8::2", false},
		{"full form vs abbreviated", "2001:0db8:0000:0000:0000:0000:0000:0001", "2001:db8::1", true},
		{"link-local match", "fe80::1", "fe80::1", true},
		{"link-local mismatch", "fe80::1", "fe80::2", false},
		{"mixed v4 and v6", "192.168.1.1", "::ffff:192.168.1.1", true}, // IPv4-mapped IPv6
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIPSingle(tt.clientIP, tt.entryIP)
			if got != tt.want {
				t.Errorf("matchIPSingle(%q, %q) = %v, want %v", tt.clientIP, tt.entryIP, got, tt.want)
			}
		})
	}
}

// --- matchIPCIDR ---

func TestMatchIPCIDR_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		clientIP string
		cidr     string
		want     bool
	}{
		{"in /24 range", "192.168.1.1", "192.168.1.0/24", true},
		{"broadcast in /24", "192.168.1.255", "192.168.1.0/24", true},
		{"outside /24 range", "192.168.2.1", "192.168.1.0/24", false},
		{"in /8 range", "10.0.0.5", "10.0.0.0/8", true},
		{"outside /8 range", "11.0.0.1", "10.0.0.0/8", false},
		{"/32 exact match", "192.168.1.1", "192.168.1.1/32", true},
		{"/32 no match", "192.168.1.2", "192.168.1.1/32", false},
		{"invalid client IP", "invalid", "192.168.1.0/24", false},
		{"invalid CIDR", "192.168.1.1", "invalid", false},
		{"malformed CIDR prefix", "192.168.1.1", "192.168.1.0/abc", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIPCIDR(tt.clientIP, tt.cidr)
			if got != tt.want {
				t.Errorf("matchIPCIDR(%q, %q) = %v, want %v", tt.clientIP, tt.cidr, got, tt.want)
			}
		})
	}
}

func TestMatchIPCIDR_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		clientIP string
		cidr     string
		want     bool
	}{
		{"in /32", "2001:db8::1", "2001:db8::/32", true},
		{"in /32 sub-prefix", "2001:db8:1::1", "2001:db8::/32", true},
		{"outside /32", "2001:db9::1", "2001:db8::/32", false},
		{"link-local in /10", "fe80::1", "fe80::/10", true},
		{"link-local broadcast in /10", "fe80::ffff", "fe80::/10", true},
		{"/128 exact match", "2001:db8::1", "2001:db8::1/128", true},
		{"/128 no match", "2001:db8::2", "2001:db8::1/128", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIPCIDR(tt.clientIP, tt.cidr)
			if got != tt.want {
				t.Errorf("matchIPCIDR(%q, %q) = %v, want %v", tt.clientIP, tt.cidr, got, tt.want)
			}
		})
	}
}

// --- matchIPRange ---

func TestMatchIPRange_IPv4(t *testing.T) {
	tests := []struct {
		name       string
		clientIP   string
		rangeValue string
		want       bool
	}{
		{"in range", "192.168.1.5", "192.168.1.1~192.168.1.10", true},
		{"boundary start", "192.168.1.1", "192.168.1.1~192.168.1.10", true},
		{"boundary end", "192.168.1.10", "192.168.1.1~192.168.1.10", true},
		{"above range", "192.168.1.11", "192.168.1.1~192.168.1.10", false},
		{"below range", "192.168.1.0", "192.168.1.1~192.168.1.10", false},
		{"completely different subnet", "10.0.0.1", "192.168.1.1~192.168.1.10", false},
		{"invalid client", "invalid", "192.168.1.1~192.168.1.10", false},
		{"invalid start", "192.168.1.5", "invalid~192.168.1.10", false},
		{"invalid end", "192.168.1.5", "192.168.1.1~invalid", false},
		{"no tilde separator", "192.168.1.5", "no_tilde", false},
		{"single IP range", "192.168.1.1", "192.168.1.1~192.168.1.1", true},
		{"single IP range miss", "192.168.1.2", "192.168.1.1~192.168.1.1", false},
		{"spaces around tilde", "192.168.1.5", "192.168.1.1 ~ 192.168.1.10", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIPRange(tt.clientIP, tt.rangeValue)
			if got != tt.want {
				t.Errorf("matchIPRange(%q, %q) = %v, want %v", tt.clientIP, tt.rangeValue, got, tt.want)
			}
		})
	}
}

func TestMatchIPRange_IPv6(t *testing.T) {
	tests := []struct {
		name       string
		clientIP   string
		rangeValue string
		want       bool
	}{
		{"in range", "2001:db8::5", "2001:db8::1~2001:db8::10", true},
		{"boundary start", "2001:db8::1", "2001:db8::1~2001:db8::10", true},
		{"boundary end", "2001:db8::10", "2001:db8::1~2001:db8::10", true},
		{"above range", "2001:db8::11", "2001:db8::1~2001:db8::10", false},
		{"below range", "2001:db8::0", "2001:db8::1~2001:db8::10", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchIPRange(tt.clientIP, tt.rangeValue)
			if got != tt.want {
				t.Errorf("matchIPRange(%q, %q) = %v, want %v", tt.clientIP, tt.rangeValue, got, tt.want)
			}
		})
	}
}

// --- MatchEntry ---

func TestMatchEntry(t *testing.T) {
	tests := []struct {
		name     string
		clientIP string
		entry    model.AccessControlEntry
		want     bool
	}{
		{"single match", "192.168.1.1", model.AccessControlEntry{EntryType: "single", Value: "192.168.1.1"}, true},
		{"single mismatch", "192.168.1.2", model.AccessControlEntry{EntryType: "single", Value: "192.168.1.1"}, false},
		{"cidr match", "192.168.1.50", model.AccessControlEntry{EntryType: "cidr", Value: "192.168.1.0/24"}, true},
		{"cidr mismatch", "10.0.0.1", model.AccessControlEntry{EntryType: "cidr", Value: "192.168.1.0/24"}, false},
		{"range match", "192.168.1.5", model.AccessControlEntry{EntryType: "range", Value: "192.168.1.1~192.168.1.10"}, true},
		{"range mismatch", "192.168.1.20", model.AccessControlEntry{EntryType: "range", Value: "192.168.1.1~192.168.1.10"}, false},
		{"unknown type", "192.168.1.1", model.AccessControlEntry{EntryType: "unknown", Value: "192.168.1.1"}, false},
		{"empty type", "192.168.1.1", model.AccessControlEntry{EntryType: "", Value: "192.168.1.1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchEntry(tt.clientIP, tt.entry)
			if got != tt.want {
				t.Errorf("MatchEntry(%q, %+v) = %v, want %v", tt.clientIP, tt.entry, got, tt.want)
			}
		})
	}
}

// --- isBlacklistEntryActive ---

func TestIsBlacklistEntryActive(t *testing.T) {
	now := time.Now()
	days7 := 7

	tests := []struct {
		name  string
		entry model.AccessControlEntry
		want  bool
	}{
		{"permanent (nil block_days)", model.AccessControlEntry{BlockDays: nil, CreatedAt: now}, true},
		{"permanent (zero block_days)", model.AccessControlEntry{BlockDays: intPtr(0), CreatedAt: now}, true},
		{"active (1 day into 7-day block)", model.AccessControlEntry{BlockDays: &days7, CreatedAt: now.Add(-1 * 24 * time.Hour)}, true},
		{"active (6 days into 7-day block)", model.AccessControlEntry{BlockDays: &days7, CreatedAt: now.Add(-6 * 24 * time.Hour)}, true},
		{"expired (10 days into 7-day block)", model.AccessControlEntry{BlockDays: &days7, CreatedAt: now.Add(-10 * 24 * time.Hour)}, false},
		{"expired (exactly 7 days ago)", model.AccessControlEntry{BlockDays: &days7, CreatedAt: now.Add(-7*24*time.Hour - time.Second)}, false},
		{"1-day block just created", model.AccessControlEntry{BlockDays: intPtr(1), CreatedAt: now}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBlacklistEntryActive(tt.entry)
			if got != tt.want {
				t.Errorf("isBlacklistEntryActive(%+v) = %v, want %v", tt.entry, got, tt.want)
			}
		})
	}
}

// --- isLoopbackOrLocal ---

func TestIsLoopbackOrLocal(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback other", "127.0.0.2", true},
		{"IPv6 loopback", "::1", true},
		{"IPv4 private - not loopback", "192.168.1.1", false},
		{"IPv4 private 10.x", "10.0.0.1", false},
		{"IPv6 link-local", "fe80::1", true},
		{"IPv6 global unicast", "2001:db8::1", false},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLoopbackOrLocal(tt.ip)
			if got != tt.want {
				t.Errorf("isLoopbackOrLocal(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// --- IsIPAllowed (integration-level tests) ---

func TestIsIPAllowed_Disabled(t *testing.T) {
	got := IsIPAllowed("192.168.1.1", "disabled", nil)
	if !got {
		t.Errorf("IsIPAllowed with disabled mode should always return true")
	}
}

func TestIsIPAllowed_Whitelist(t *testing.T) {
	entries := []model.AccessControlEntry{
		{ListType: "whitelist", EntryType: "single", Value: "192.168.1.1"},
		{ListType: "whitelist", EntryType: "cidr", Value: "10.0.0.0/8"},
		{ListType: "whitelist", EntryType: "range", Value: "172.16.0.1~172.16.0.50"},
	}

	tests := []struct {
		name     string
		clientIP string
		want     bool
	}{
		{"single IP match", "192.168.1.1", true},
		{"CIDR match", "10.0.0.5", true},
		{"range match", "172.16.0.25", true},
		{"range boundary start", "172.16.0.1", true},
		{"range boundary end", "172.16.0.50", true},
		{"not in any entry", "172.16.1.1", false},
		{"loopback always allowed", "127.0.0.1", true},
		{"IPv6 loopback always allowed", "::1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIPAllowed(tt.clientIP, "whitelist", entries)
			if got != tt.want {
				t.Errorf("IsIPAllowed(%q, whitelist) = %v, want %v", tt.clientIP, got, tt.want)
			}
		})
	}
}

func TestIsIPAllowed_WhitelistEmpty(t *testing.T) {
	// Empty whitelist + non-loopback IP = denied
	got := IsIPAllowed("192.168.1.1", "whitelist", []model.AccessControlEntry{})
	if got {
		t.Errorf("IsIPAllowed with empty whitelist should deny non-loopback IPs")
	}
	// Empty whitelist + loopback = allowed
	got = IsIPAllowed("127.0.0.1", "whitelist", []model.AccessControlEntry{})
	if !got {
		t.Errorf("IsIPAllowed with empty whitelist should allow loopback IPs")
	}
}

func TestIsIPAllowed_Blacklist(t *testing.T) {
	now := time.Now()
	days7 := 7

	entries := []model.AccessControlEntry{
		{ListType: "blacklist", EntryType: "single", Value: "192.168.1.100", BlockDays: nil, CreatedAt: now},
		{ListType: "blacklist", EntryType: "single", Value: "192.168.1.200", BlockDays: &days7, CreatedAt: now.Add(-10 * 24 * time.Hour)}, // expired
		{ListType: "blacklist", EntryType: "single", Value: "192.168.1.150", BlockDays: &days7, CreatedAt: now.Add(-1 * 24 * time.Hour)},  // active
	}

	tests := []struct {
		name     string
		clientIP string
		want     bool
	}{
		{"permanent block", "192.168.1.100", false},
		{"expired block - allowed", "192.168.1.200", true},
		{"active time-limited block", "192.168.1.150", false},
		{"not in blacklist", "192.168.1.1", true},
		{"loopback always allowed even if blacklisted would match", "127.0.0.1", true},
		{"IPv6 loopback always allowed", "::1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIPAllowed(tt.clientIP, "blacklist", entries)
			if got != tt.want {
				t.Errorf("IsIPAllowed(%q, blacklist) = %v, want %v", tt.clientIP, got, tt.want)
			}
		})
	}
}

func TestIsIPAllowed_IgnoresWrongListType(t *testing.T) {
	// Whitelist mode should ignore blacklist entries and vice versa
	entries := []model.AccessControlEntry{
		{ListType: "blacklist", EntryType: "single", Value: "192.168.1.1"},
	}
	// In whitelist mode, blacklist entries should be ignored, IP not in whitelist = denied
	got := IsIPAllowed("192.168.1.1", "whitelist", entries)
	if got {
		t.Errorf("Whitelist mode should ignore blacklist-type entries")
	}

	entries2 := []model.AccessControlEntry{
		{ListType: "whitelist", EntryType: "single", Value: "192.168.1.1"},
	}
	// In blacklist mode, whitelist entries should be ignored, IP not blacklisted = allowed
	got2 := IsIPAllowed("192.168.1.1", "blacklist", entries2)
	if !got2 {
		t.Errorf("Blacklist mode should ignore whitelist-type entries")
	}
}

// --- validateEntryValue ---

func TestValidateEntryValue(t *testing.T) {
	tests := []struct {
		name    string
		entry   AccessControlEntryRequest
		wantErr bool
	}{
		// Single IP - valid
		{"valid IPv4", AccessControlEntryRequest{EntryType: "single", Value: "192.168.1.1"}, false},
		{"valid IPv6", AccessControlEntryRequest{EntryType: "single", Value: "2001:db8::1"}, false},
		{"valid IPv6 loopback", AccessControlEntryRequest{EntryType: "single", Value: "::1"}, false},
		// Single IP - invalid
		{"invalid IP", AccessControlEntryRequest{EntryType: "single", Value: "not_an_ip"}, true},
		{"empty value", AccessControlEntryRequest{EntryType: "single", Value: ""}, true},
		{"spaces only", AccessControlEntryRequest{EntryType: "single", Value: "   "}, true},
		{"partial IP", AccessControlEntryRequest{EntryType: "single", Value: "192.168.1"}, true},
		{"IP with port", AccessControlEntryRequest{EntryType: "single", Value: "192.168.1.1:8080"}, true},

		// CIDR - valid
		{"valid CIDR v4", AccessControlEntryRequest{EntryType: "cidr", Value: "192.168.1.0/24"}, false},
		{"valid CIDR v6", AccessControlEntryRequest{EntryType: "cidr", Value: "2001:db8::/32"}, false},
		{"valid CIDR /32", AccessControlEntryRequest{EntryType: "cidr", Value: "192.168.1.1/32"}, false},
		// CIDR - invalid
		{"CIDR no prefix", AccessControlEntryRequest{EntryType: "cidr", Value: "192.168.1.0"}, true},
		{"CIDR invalid prefix", AccessControlEntryRequest{EntryType: "cidr", Value: "192.168.1.0/abc"}, true},
		{"CIDR prefix too large v4", AccessControlEntryRequest{EntryType: "cidr", Value: "192.168.1.0/33"}, true},
		{"CIDR invalid IP", AccessControlEntryRequest{EntryType: "cidr", Value: "invalid/24"}, true},

		// Range - valid
		{"valid range v4", AccessControlEntryRequest{EntryType: "range", Value: "192.168.1.1~192.168.1.10"}, false},
		{"valid range v6", AccessControlEntryRequest{EntryType: "range", Value: "2001:db8::1~2001:db8::10"}, false},
		{"valid range single IP", AccessControlEntryRequest{EntryType: "range", Value: "192.168.1.1~192.168.1.1"}, false},
		// Range - invalid
		{"range no separator", AccessControlEntryRequest{EntryType: "range", Value: "192.168.1.1"}, true},
		{"range invalid start", AccessControlEntryRequest{EntryType: "range", Value: "invalid~192.168.1.10"}, true},
		{"range invalid end", AccessControlEntryRequest{EntryType: "range", Value: "192.168.1.1~invalid"}, true},
		{"range start > end", AccessControlEntryRequest{EntryType: "range", Value: "192.168.1.10~192.168.1.1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEntryValue(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEntryValue(%+v) error = %v, wantErr %v", tt.entry, err, tt.wantErr)
			}
		})
	}
}

func intPtr(v int) *int {
	return &v
}
