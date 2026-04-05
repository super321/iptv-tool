package epg

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateXMLTV_Basic(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	programs := []Program{
		{
			Channel:     "ch1",
			ChannelName: "CCTV-1",
			Title:       "新闻联播",
			Desc:        "每日新闻",
			StartTime:   time.Date(2026, 3, 12, 20, 0, 0, 0, loc),
			EndTime:     time.Date(2026, 3, 12, 21, 0, 0, 0, loc),
		},
	}
	channelMap := map[string]string{"ch1": "CCTV-1"}

	output := GenerateXMLTV(programs, channelMap)

	if !strings.Contains(output, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(output, `generator-info-name="iptv-tool-v2"`) {
		t.Error("missing generator info")
	}
	if !strings.Contains(output, `<channel id="ch1">`) {
		t.Error("missing channel element")
	}
	if !strings.Contains(output, `<display-name>CCTV-1</display-name>`) {
		t.Error("missing display-name")
	}
	if !strings.Contains(output, `channel="ch1"`) {
		t.Error("missing programme channel attribute")
	}
	if !strings.Contains(output, `<title>新闻联播</title>`) {
		t.Error("missing title")
	}
	if !strings.Contains(output, `<desc>每日新闻</desc>`) {
		t.Error("missing desc")
	}
	if !strings.Contains(output, "</tv>") {
		t.Error("missing closing tv tag")
	}
}

func TestGenerateXMLTV_NoDesc(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	programs := []Program{
		{
			Channel:   "ch1",
			Title:     "Test",
			StartTime: time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
			EndTime:   time.Date(2026, 1, 1, 1, 0, 0, 0, loc),
		},
	}

	output := GenerateXMLTV(programs, map[string]string{"ch1": "Test"})
	if strings.Contains(output, "<desc>") {
		t.Error("should not contain <desc> when Desc is empty")
	}
}

func TestGenerateXMLTV_EmptyPrograms(t *testing.T) {
	output := GenerateXMLTV(nil, map[string]string{})
	if !strings.Contains(output, "</tv>") {
		t.Error("should still produce valid XML structure")
	}
	if strings.Contains(output, "<programme") {
		t.Error("should not contain programme elements for nil programs")
	}
}

func TestGenerateXMLTV_XMLEscape(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	programs := []Program{
		{
			Channel:   "ch&1",
			Title:     `"Title" <with> 'special' & chars`,
			Desc:      `A & B < C > D "E" 'F'`,
			StartTime: time.Date(2026, 1, 1, 0, 0, 0, 0, loc),
			EndTime:   time.Date(2026, 1, 1, 1, 0, 0, 0, loc),
		},
	}
	channelMap := map[string]string{"ch&1": `Channel "A" & <B>`}

	output := GenerateXMLTV(programs, channelMap)

	// Channel id should be escaped
	if !strings.Contains(output, `channel id="ch&amp;1"`) {
		t.Error("channel id not escaped")
	}
	// Title should have escaped characters
	if !strings.Contains(output, "&amp;") {
		t.Error("& not escaped in output")
	}
	if !strings.Contains(output, "&lt;") {
		t.Error("< not escaped in output")
	}
	if !strings.Contains(output, "&gt;") {
		t.Error("> not escaped in output")
	}
	if !strings.Contains(output, "&quot;") {
		t.Error("\" not escaped in output")
	}
	if !strings.Contains(output, "&apos;") {
		t.Error("' not escaped in output")
	}
}

func TestGenerateXMLTV_RoundTrip(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	original := []Program{
		{
			Channel:     "ch1",
			ChannelName: "CCTV-1",
			Title:       "新闻联播",
			Desc:        "每日新闻",
			StartTime:   time.Date(2026, 3, 12, 20, 0, 0, 0, loc),
			EndTime:     time.Date(2026, 3, 12, 21, 0, 0, 0, loc),
		},
		{
			Channel:     "ch2",
			ChannelName: "CCTV-2",
			Title:       "财经报道",
			StartTime:   time.Date(2026, 3, 12, 19, 0, 0, 0, loc),
			EndTime:     time.Date(2026, 3, 12, 20, 0, 0, 0, loc),
		},
	}
	channelMap := map[string]string{"ch1": "CCTV-1", "ch2": "CCTV-2"}

	generated := GenerateXMLTV(original, channelMap)

	parsed, err := ParseXMLTV(generated)
	if err != nil {
		t.Fatalf("ParseXMLTV error: %v", err)
	}

	if len(parsed) != len(original) {
		t.Fatalf("round trip: got %d programs, want %d", len(parsed), len(original))
	}

	// Build a map by channel+title for comparison (order may differ due to channelMap iteration)
	type key struct{ ch, title string }
	origMap := make(map[key]Program)
	for _, p := range original {
		origMap[key{p.Channel, p.Title}] = p
	}

	for _, p := range parsed {
		k := key{p.Channel, p.Title}
		orig, ok := origMap[k]
		if !ok {
			t.Errorf("unexpected program: channel=%q title=%q", p.Channel, p.Title)
			continue
		}
		if p.ChannelName != orig.ChannelName {
			t.Errorf("ChannelName = %q, want %q", p.ChannelName, orig.ChannelName)
		}
		if !p.StartTime.Equal(orig.StartTime) {
			t.Errorf("StartTime = %v, want %v", p.StartTime, orig.StartTime)
		}
		if !p.EndTime.Equal(orig.EndTime) {
			t.Errorf("EndTime = %v, want %v", p.EndTime, orig.EndTime)
		}
		if p.Desc != orig.Desc {
			t.Errorf("Desc = %q, want %q", p.Desc, orig.Desc)
		}
	}
}

func TestXMLEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ampersand", "A&B", "A&amp;B"},
		{"less than", "A<B", "A&lt;B"},
		{"greater than", "A>B", "A&gt;B"},
		{"double quote", `A"B`, "A&quot;B"},
		{"single quote", "A'B", "A&apos;B"},
		{"no special", "plain text", "plain text"},
		{"empty", "", ""},
		{"all special", `&<>"'`, "&amp;&lt;&gt;&quot;&apos;"},
		{"multiple ampersands", "a&b&c", "a&amp;b&amp;c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xmlEscape(tt.input)
			if got != tt.want {
				t.Errorf("xmlEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
