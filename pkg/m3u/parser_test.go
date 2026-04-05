package m3u

import (
	"strings"
	"testing"
)

// --- ParseM3U ---

func TestParseM3U_Basic(t *testing.T) {
	content := `#EXTM3U
#EXTINF:-1 tvg-id="cctv1" tvg-name="CCTV-1" tvg-logo="http://logo.png" group-title="央视",CCTV-1 综合
http://example.com/cctv1.m3u8
#EXTINF:-1 tvg-id="cctv2" group-title="央视",CCTV-2 财经
http://example.com/cctv2.m3u8`

	channels, err := ParseM3U(content)
	if err != nil {
		t.Fatalf("ParseM3U error: %v", err)
	}

	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	ch := channels[0]
	if ch.Name != "CCTV-1 综合" {
		t.Errorf("Name = %q, want %q", ch.Name, "CCTV-1 综合")
	}
	if ch.TVGId != "cctv1" {
		t.Errorf("TVGId = %q, want %q", ch.TVGId, "cctv1")
	}
	if ch.TVGName != "CCTV-1" {
		t.Errorf("TVGName = %q, want %q", ch.TVGName, "CCTV-1")
	}
	if ch.Logo != "http://logo.png" {
		t.Errorf("Logo = %q, want %q", ch.Logo, "http://logo.png")
	}
	if ch.Group != "央视" {
		t.Errorf("Group = %q, want %q", ch.Group, "央视")
	}
	if ch.URL != "http://example.com/cctv1.m3u8" {
		t.Errorf("URL = %q, want %q", ch.URL, "http://example.com/cctv1.m3u8")
	}
}

func TestParseM3U_TVGNameFallback(t *testing.T) {
	content := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" group-title="Test",My Channel
http://example.com/stream`

	channels, err := ParseM3U(content)
	if err != nil {
		t.Fatal(err)
	}

	// tvg-name not set, should fall back to the channel name
	if channels[0].TVGName != "My Channel" {
		t.Errorf("TVGName = %q, want %q (fallback to Name)", channels[0].TVGName, "My Channel")
	}
}

func TestParseM3U_CatchupDays(t *testing.T) {
	content := `#EXTM3U
#EXTINF:-1 catchup-days="7" catchup-source="http://catchup/{0}",Catchup Channel
http://example.com/live`

	channels, err := ParseM3U(content)
	if err != nil {
		t.Fatal(err)
	}

	ch := channels[0]
	if ch.CatchupDays != 7 {
		t.Errorf("CatchupDays = %d, want 7", ch.CatchupDays)
	}
	if ch.CatchupSrc != "http://catchup/{0}" {
		t.Errorf("CatchupSrc = %q, want %q", ch.CatchupSrc, "http://catchup/{0}")
	}
}

func TestParseM3U_EmptyInput(t *testing.T) {
	channels, err := ParseM3U("")
	if err != nil {
		t.Fatalf("ParseM3U empty input error: %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(channels))
	}
}

func TestParseM3U_SkipsComments(t *testing.T) {
	content := `#EXTM3U
# This is a comment
#EXTVLCOPT:some-option
#EXTINF:-1,Channel 1
http://example.com/1`

	channels, err := ParseM3U(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if channels[0].Name != "Channel 1" {
		t.Errorf("Name = %q, want %q", channels[0].Name, "Channel 1")
	}
}

func TestParseM3U_MalformedEXTINF(t *testing.T) {
	content := `#EXTM3U
#EXTINF:malformed line without comma
http://example.com/bad
#EXTINF:-1,Good Channel
http://example.com/good`

	channels, err := ParseM3U(content)
	if err != nil {
		t.Fatal(err)
	}
	// Should skip malformed EXTINF and parse the good one
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel (skip malformed), got %d", len(channels))
	}
	if channels[0].Name != "Good Channel" {
		t.Errorf("Name = %q, want %q", channels[0].Name, "Good Channel")
	}
}

// --- ParseTXT ---

func TestParseTXT_Basic(t *testing.T) {
	content := `央视,#genre#
CCTV-1,http://example.com/cctv1
CCTV-2,http://example.com/cctv2
卫视,#genre#
湖南卫视,http://example.com/hunan`

	channels, err := ParseTXT(content)
	if err != nil {
		t.Fatalf("ParseTXT error: %v", err)
	}

	if len(channels) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(channels))
	}

	if channels[0].Name != "CCTV-1" || channels[0].Group != "央视" {
		t.Errorf("ch[0] = {Name: %q, Group: %q}, want {CCTV-1, 央视}", channels[0].Name, channels[0].Group)
	}
	if channels[2].Name != "湖南卫视" || channels[2].Group != "卫视" {
		t.Errorf("ch[2] = {Name: %q, Group: %q}, want {湖南卫视, 卫视}", channels[2].Name, channels[2].Group)
	}
}

func TestParseTXT_NoGroup(t *testing.T) {
	content := `Channel1,http://example.com/1
Channel2,http://example.com/2`

	channels, err := ParseTXT(content)
	if err != nil {
		t.Fatal(err)
	}

	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	if channels[0].Group != "" {
		t.Errorf("Group = %q, want empty (no group header)", channels[0].Group)
	}
}

func TestParseTXT_EmptyInput(t *testing.T) {
	channels, err := ParseTXT("")
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(channels))
	}
}

func TestParseTXT_SkipsBadLines(t *testing.T) {
	content := `央视,#genre#
this line has no comma
CCTV-1,http://example.com/1`

	channels, err := ParseTXT(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel (skip bad line), got %d", len(channels))
	}
}

// --- DetectFormat ---

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"M3U format", "#EXTM3U\n#EXTINF:-1,Ch\nhttp://url", "m3u"},
		{"TXT format", "央视,#genre#\nCCTV-1,http://url", "txt"},
		{"plain lines", "Channel1,http://url\nChannel2,http://url2", "txt"},
		{"M3U with spaces", "  #EXTM3U\n#EXTINF:-1,Ch\nhttp://url", "m3u"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.content)
			if got != tt.want {
				t.Errorf("DetectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- ExtractTvgURL ---

func TestExtractTvgURL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			"present",
			`#EXTM3U x-tvg-url="http://epg.example.com/epg.xml"
#EXTINF:-1,Ch
http://url`,
			"http://epg.example.com/epg.xml",
		},
		{
			"absent",
			"#EXTM3U\n#EXTINF:-1,Ch\nhttp://url",
			"",
		},
		{
			"no EXTM3U header",
			"#EXTINF:-1,Ch\nhttp://url",
			"",
		},
		{
			"empty input",
			"",
			"",
		},
		{
			"case insensitive",
			`#EXTM3U X-TVG-URL="http://epg.example.com/epg.xml"`,
			"http://epg.example.com/epg.xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTvgURL(tt.content)
			if got != tt.want {
				t.Errorf("ExtractTvgURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- FormatM3U ---

func TestFormatM3U_Basic(t *testing.T) {
	channels := []Channel{
		{
			Name:    "CCTV-1",
			TVGId:   "cctv1",
			TVGName: "CCTV-1",
			Logo:    "cctv1.png",
			Group:   "央视",
			URL:     "http://example.com/cctv1",
		},
	}

	output := FormatM3U(channels, "", "http://host/logo/")
	if !strings.HasPrefix(output, "#EXTM3U\n") {
		t.Error("FormatM3U should start with #EXTM3U")
	}
	if !strings.Contains(output, `tvg-id="cctv1"`) {
		t.Error("should contain tvg-id")
	}
	if !strings.Contains(output, `tvg-logo="http://host/logo/cctv1.png"`) {
		t.Error("should contain tvg-logo with placeholder prefix")
	}
	if !strings.Contains(output, `group-title="央视"`) {
		t.Error("should contain group-title")
	}
	if !strings.Contains(output, "http://example.com/cctv1") {
		t.Error("should contain URL")
	}
}

func TestFormatM3U_CatchupTemplate(t *testing.T) {
	channels := []Channel{
		{Name: "Ch1", TVGId: "ch1", TVGName: "Ch1", Group: "G", URL: "http://url"},
	}

	output := FormatM3U(channels, "playseek=${start}-${end}", "")
	if !strings.Contains(output, `catchup="append"`) {
		t.Error("should contain catchup=append when template is set and no catchup-source")
	}
}

func TestFormatM3U_CatchupSource(t *testing.T) {
	channels := []Channel{
		{Name: "Ch1", TVGId: "ch1", TVGName: "Ch1", Group: "G", URL: "http://url", CatchupSrc: "http://catchup/{0}"},
	}

	output := FormatM3U(channels, "template", "")
	if !strings.Contains(output, `catchup="default"`) {
		t.Error("should use catchup=default when catchup-source is present")
	}
	if !strings.Contains(output, `catchup-source="http://catchup/{0}"`) {
		t.Error("should contain the original catchup-source")
	}
}

func TestFormatM3U_NoLogo(t *testing.T) {
	channels := []Channel{
		{Name: "Ch1", TVGId: "ch1", TVGName: "Ch1", Group: "G", URL: "http://url"},
	}

	output := FormatM3U(channels, "", "http://host/logo/")
	if strings.Contains(output, "tvg-logo") {
		t.Error("should not contain tvg-logo when Logo is empty")
	}
}

// --- FormatTXT ---

func TestFormatTXT_Basic(t *testing.T) {
	channels := []Channel{
		{Name: "CCTV-1", Group: "央视", URL: "http://cctv1"},
		{Name: "CCTV-2", Group: "央视", URL: "http://cctv2"},
		{Name: "湖南卫视", Group: "卫视", URL: "http://hunan"},
	}

	output := FormatTXT(channels)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have: 央视,#genre#  CCTV-1,url  CCTV-2,url  卫视,#genre#  湖南卫视,url
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "央视,#genre#" {
		t.Errorf("line 0 = %q, want %q", lines[0], "央视,#genre#")
	}
}

func TestFormatTXT_EmptyGroup(t *testing.T) {
	channels := []Channel{
		{Name: "Ch1", Group: "", URL: "http://url"},
	}

	output := FormatTXT(channels)
	if !strings.Contains(output, "未分组,#genre#") {
		t.Error("empty group should default to 未分组")
	}
}

// --- Round-trip ---

func TestParseFormatRoundTrip_TXT(t *testing.T) {
	original := []Channel{
		{Name: "CCTV-1", Group: "央视", URL: "http://cctv1"},
		{Name: "CCTV-2", Group: "央视", URL: "http://cctv2"},
		{Name: "湖南卫视", Group: "卫视", URL: "http://hunan"},
	}

	formatted := FormatTXT(original)
	parsed, err := ParseTXT(formatted)
	if err != nil {
		t.Fatalf("ParseTXT error: %v", err)
	}

	if len(parsed) != len(original) {
		t.Fatalf("round trip: got %d channels, want %d", len(parsed), len(original))
	}

	for i, ch := range parsed {
		if ch.Name != original[i].Name {
			t.Errorf("ch[%d].Name = %q, want %q", i, ch.Name, original[i].Name)
		}
		if ch.URL != original[i].URL {
			t.Errorf("ch[%d].URL = %q, want %q", i, ch.URL, original[i].URL)
		}
		if ch.Group != original[i].Group {
			t.Errorf("ch[%d].Group = %q, want %q", i, ch.Group, original[i].Group)
		}
	}
}
