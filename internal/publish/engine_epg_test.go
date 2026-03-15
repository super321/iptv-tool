package publish

import (
	"encoding/json"
	"testing"
	"time"

	"iptv-tool-v2/internal/model"
)

// buildTestEPG creates an AggregatedEPG for testing FormatDIYP.
func buildTestEPG() *AggregatedEPG {
	loc := time.FixedZone("CST", 8*3600)
	epg := &AggregatedEPG{
		Channels: make(map[string]*EPGChannelPrograms),
	}

	// Channel: CCTV1 (with alias "CCTV-1")
	cctv1 := &EPGChannelPrograms{
		ChannelID:   "cctv1",
		ChannelName: "CCTV1",
		Alias:       "CCTV-1",
		DatePrograms: map[string][]EPGProgram{
			"2024-03-15": {
				{Title: "新闻联播", Desc: "晚间新闻", StartTime: time.Date(2024, 3, 15, 19, 0, 0, 0, loc), EndTime: time.Date(2024, 3, 15, 19, 30, 0, 0, loc)},
				{Title: "焦点访谈", Desc: "", StartTime: time.Date(2024, 3, 15, 19, 30, 0, 0, loc), EndTime: time.Date(2024, 3, 15, 20, 0, 0, 0, loc)},
			},
			"2024-03-16": {
				{Title: "朝闻天下", Desc: "", StartTime: time.Date(2024, 3, 16, 6, 0, 0, 0, loc), EndTime: time.Date(2024, 3, 16, 9, 0, 0, 0, loc)},
			},
		},
	}
	// Key by lowercase effective name (alias first)
	epg.Channels["cctv-1"] = cctv1
	epg.ChannelOrder = append(epg.ChannelOrder, "cctv-1")

	// Channel: 湖南卫视 (no alias)
	hunan := &EPGChannelPrograms{
		ChannelID:   "hunan",
		ChannelName: "湖南卫视",
		Alias:       "",
		DatePrograms: map[string][]EPGProgram{
			"2024-03-15": {
				{Title: "快乐大本营", Desc: "综艺节目", StartTime: time.Date(2024, 3, 15, 20, 0, 0, 0, loc), EndTime: time.Date(2024, 3, 15, 22, 0, 0, 0, loc)},
			},
		},
	}
	epg.Channels["湖南卫视"] = hunan
	epg.ChannelOrder = append(epg.ChannelOrder, "湖南卫视")

	return epg
}

func TestFormatDIYP_NormalQuery(t *testing.T) {
	epg := buildTestEPG()
	eng := &Engine{iface: model.PublishInterface{}}

	result := eng.FormatDIYP(epg, "CCTV-1", "2024-03-15")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ChannelName != "CCTV-1" {
		t.Errorf("ChannelName = %q, want %q", resp.ChannelName, "CCTV-1")
	}
	if resp.Date != "2024-03-15" {
		t.Errorf("Date = %q, want %q", resp.Date, "2024-03-15")
	}
	if len(resp.EPGData) != 2 {
		t.Fatalf("EPGData length = %d, want 2", len(resp.EPGData))
	}
	if resp.EPGData[0].Title != "新闻联播" {
		t.Errorf("EPGData[0].Title = %q, want %q", resp.EPGData[0].Title, "新闻联播")
	}
	if resp.EPGData[0].Desc != "晚间新闻" {
		t.Errorf("EPGData[0].Desc = %q, want %q", resp.EPGData[0].Desc, "晚间新闻")
	}
	if resp.EPGData[0].Start != "19:00" {
		t.Errorf("EPGData[0].Start = %q, want %q", resp.EPGData[0].Start, "19:00")
	}
	if resp.EPGData[0].End != "19:30" {
		t.Errorf("EPGData[0].End = %q, want %q", resp.EPGData[0].End, "19:30")
	}
}

func TestFormatDIYP_NoMatchingChannel(t *testing.T) {
	epg := buildTestEPG()
	eng := &Engine{iface: model.PublishInterface{}}

	result := eng.FormatDIYP(epg, "不存在的频道", "2024-03-15")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ChannelName != "不存在的频道" {
		t.Errorf("ChannelName = %q, want %q", resp.ChannelName, "不存在的频道")
	}
	if len(resp.EPGData) != 0 {
		t.Errorf("EPGData length = %d, want 0", len(resp.EPGData))
	}
}

func TestFormatDIYP_NoChannelParam(t *testing.T) {
	epg := buildTestEPG()
	eng := &Engine{iface: model.PublishInterface{}}

	result := eng.FormatDIYP(epg, "", "2024-03-15")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ChannelName != "未提供" {
		t.Errorf("ChannelName = %q, want %q", resp.ChannelName, "未提供")
	}
	// Should return 24-hour default placeholder data
	if len(resp.EPGData) != 24 {
		t.Fatalf("EPGData length = %d, want 24", len(resp.EPGData))
	}
	if resp.EPGData[0].Start != "00:00" {
		t.Errorf("EPGData[0].Start = %q, want %q", resp.EPGData[0].Start, "00:00")
	}
	if resp.EPGData[23].Start != "23:00" {
		t.Errorf("EPGData[23].Start = %q, want %q", resp.EPGData[23].Start, "23:00")
	}
}

func TestFormatDIYP_CaseInsensitive(t *testing.T) {
	epg := buildTestEPG()
	eng := &Engine{iface: model.PublishInterface{}}

	// Query with different case
	result := eng.FormatDIYP(epg, "cctv-1", "2024-03-15")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.EPGData) != 2 {
		t.Errorf("EPGData length = %d, want 2 (case-insensitive match)", len(resp.EPGData))
	}
}

func TestFormatDIYP_AliasQuery(t *testing.T) {
	epg := buildTestEPG()
	eng := &Engine{iface: model.PublishInterface{}}

	// CCTV1 has alias "CCTV-1", map key is "cctv-1"
	// Query using the alias should work
	result := eng.FormatDIYP(epg, "CCTV-1", "2024-03-15")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.EPGData) != 2 {
		t.Fatalf("EPGData length = %d, want 2", len(resp.EPGData))
	}
	if resp.EPGData[1].Title != "焦点访谈" {
		t.Errorf("EPGData[1].Title = %q, want %q", resp.EPGData[1].Title, "焦点访谈")
	}
}

func TestFormatDIYP_NoAliasChannel(t *testing.T) {
	epg := buildTestEPG()
	eng := &Engine{iface: model.PublishInterface{}}

	// 湖南卫视 has no alias, map key is "湖南卫视"
	result := eng.FormatDIYP(epg, "湖南卫视", "2024-03-15")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.EPGData) != 1 {
		t.Fatalf("EPGData length = %d, want 1", len(resp.EPGData))
	}
	if resp.EPGData[0].Title != "快乐大本营" {
		t.Errorf("EPGData[0].Title = %q, want %q", resp.EPGData[0].Title, "快乐大本营")
	}
}

func TestFormatDIYP_DifferentDate(t *testing.T) {
	epg := buildTestEPG()
	eng := &Engine{iface: model.PublishInterface{}}

	// CCTV-1 has programs on 2024-03-16 too
	result := eng.FormatDIYP(epg, "CCTV-1", "2024-03-16")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.EPGData) != 1 {
		t.Fatalf("EPGData length = %d, want 1", len(resp.EPGData))
	}
	if resp.EPGData[0].Title != "朝闻天下" {
		t.Errorf("EPGData[0].Title = %q, want %q", resp.EPGData[0].Title, "朝闻天下")
	}
}

func TestFormatDIYP_NilEPG(t *testing.T) {
	eng := &Engine{iface: model.PublishInterface{}}

	result := eng.FormatDIYP(nil, "CCTV-1", "2024-03-15")

	var resp DIYPResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ChannelName != "CCTV-1" {
		t.Errorf("ChannelName = %q, want %q", resp.ChannelName, "CCTV-1")
	}
	if len(resp.EPGData) != 0 {
		t.Errorf("EPGData length = %d, want 0", len(resp.EPGData))
	}
}
