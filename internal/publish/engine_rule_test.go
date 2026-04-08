package publish

import (
	"regexp"
	"testing"

	"iptv-tool-v2/internal/model"
)

func TestApplyGroup(t *testing.T) {
	// Build engine with group rules: CCTV -> "央视", 卫视 -> "卫视"
	e := &Engine{
		groupRules: []GroupRuleConfig{
			{
				GroupName: "央视",
				Rules: []struct {
					Target    string          `json:"target"`
					MatchMode model.MatchMode `json:"match_mode"`
					Pattern   string          `json:"pattern"`
					regex     *regexp.Regexp
				}{
					{Target: "name", MatchMode: model.MatchModeString, Pattern: "cctv"},
				},
			},
			{
				GroupName: "卫视",
				Rules: []struct {
					Target    string          `json:"target"`
					MatchMode model.MatchMode `json:"match_mode"`
					Pattern   string          `json:"pattern"`
					regex     *regexp.Regexp
				}{
					{Target: "name", MatchMode: model.MatchModeString, Pattern: "卫视"},
				},
			},
		},
	}

	tests := []struct {
		name          string
		chName        string
		alias         string
		originalGroup string
		hasGroupRules bool
		want          string
	}{
		{
			name:          "match_cctv_with_group_rules",
			chName:        "CCTV1",
			alias:         "",
			originalGroup: "原始分组",
			hasGroupRules: true,
			want:          "央视",
		},
		{
			name:          "match_satellite_with_group_rules",
			chName:        "湖南卫视",
			alias:         "",
			originalGroup: "原始分组",
			hasGroupRules: true,
			want:          "卫视",
		},
		{
			name:          "no_match_with_group_rules_returns_empty",
			chName:        "凤凰中文",
			alias:         "",
			originalGroup: "原始分组",
			hasGroupRules: true,
			want:          "",
		},
		{
			name:          "no_match_without_group_rules_returns_original",
			chName:        "凤凰中文",
			alias:         "",
			originalGroup: "原始分组",
			hasGroupRules: false,
			want:          "原始分组",
		},
		{
			name:          "no_match_without_group_rules_empty_original",
			chName:        "凤凰中文",
			alias:         "",
			originalGroup: "",
			hasGroupRules: false,
			want:          "",
		},
		{
			name:          "match_by_alias_target",
			chName:        "SomeChannel",
			alias:         "CCTV1-alias",
			originalGroup: "原始分组",
			hasGroupRules: true,
			want:          "央视", // "cctv" is found in alias "CCTV1-alias" (case-insensitive via strings.Contains)
		},
	}

	// Add a group rule that matches by alias
	eWithAlias := &Engine{
		groupRules: []GroupRuleConfig{
			{
				GroupName: "央视",
				Rules: []struct {
					Target    string          `json:"target"`
					MatchMode model.MatchMode `json:"match_mode"`
					Pattern   string          `json:"pattern"`
					regex     *regexp.Regexp
				}{
					{Target: "alias", MatchMode: model.MatchModeString, Pattern: "cctv"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := e
			if tt.name == "match_by_alias_target" {
				eng = eWithAlias
			}
			got := eng.applyGroup(tt.chName, tt.alias, tt.originalGroup, tt.hasGroupRules)
			if got != tt.want {
				t.Errorf("applyGroup(%q, %q, %q, %v) = %q, want %q",
					tt.chName, tt.alias, tt.originalGroup, tt.hasGroupRules, got, tt.want)
			}
		})
	}
}

func TestShouldFilterWithGroup(t *testing.T) {
	tests := []struct {
		name           string
		filterRules    []FilterRule
		chName         string
		alias          string
		group          string
		skipGroupRules bool
		wantDrop       bool
	}{
		{
			name: "filter_by_name_match",
			filterRules: []FilterRule{
				{Target: "name", MatchMode: model.MatchModeString, Pattern: "购物"},
			},
			chName:   "购物频道",
			alias:    "",
			group:    "购物",
			wantDrop: true,
		},
		{
			name: "filter_by_name_no_match",
			filterRules: []FilterRule{
				{Target: "name", MatchMode: model.MatchModeString, Pattern: "购物"},
			},
			chName:   "CCTV1",
			alias:    "",
			group:    "央视",
			wantDrop: false,
		},
		{
			name: "filter_by_group_match",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeString, Pattern: "购物"},
			},
			chName:   "东方购物1",
			alias:    "",
			group:    "购物频道",
			wantDrop: true,
		},
		{
			name: "filter_by_group_no_match",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeString, Pattern: "购物"},
			},
			chName:   "CCTV1",
			alias:    "",
			group:    "央视",
			wantDrop: false,
		},
		{
			name: "filter_by_group_regex_match_empty",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeRegex, Pattern: "^$", regex: regexp.MustCompile("^$")},
			},
			chName:   "未知频道",
			alias:    "",
			group:    "",
			wantDrop: true,
		},
		{
			name: "filter_by_group_regex_does_not_match_nonempty",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeRegex, Pattern: "^$", regex: regexp.MustCompile("^$")},
			},
			chName:   "CCTV1",
			alias:    "",
			group:    "央视",
			wantDrop: false,
		},
		{
			name: "filter_by_alias_fallback_to_name",
			filterRules: []FilterRule{
				{Target: "alias", MatchMode: model.MatchModeString, Pattern: "test"},
			},
			chName:   "test_channel",
			alias:    "",
			group:    "测试",
			wantDrop: true, // alias is empty, falls back to name
		},
		{
			name: "filter_by_group_does_not_affect_name",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeString, Pattern: "央视"},
			},
			chName:   "央视新闻",
			alias:    "",
			group:    "新闻",
			wantDrop: false, // group is "新闻", not "央视"
		},
		// EPG context tests: skipGroupRules=true should skip group-target rules
		{
			name: "epg_skip_group_regex_empty",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeRegex, Pattern: "^$", regex: regexp.MustCompile("^$")},
			},
			chName:         "CCTV1",
			alias:          "",
			group:          "",
			skipGroupRules: true,
			wantDrop:       false, // group rule skipped in EPG context
		},
		{
			name: "epg_skip_group_string",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeString, Pattern: "央视"},
			},
			chName:         "CCTV1",
			alias:          "",
			group:          "央视",
			skipGroupRules: true,
			wantDrop:       false, // group rule skipped in EPG context
		},
		{
			name: "epg_still_applies_name_filter",
			filterRules: []FilterRule{
				{Target: "group", MatchMode: model.MatchModeRegex, Pattern: "^$", regex: regexp.MustCompile("^$")},
				{Target: "name", MatchMode: model.MatchModeString, Pattern: "购物"},
			},
			chName:         "购物频道",
			alias:          "",
			group:          "",
			skipGroupRules: true,
			wantDrop:       true, // group rule skipped, but name rule still matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Engine{filterRules: tt.filterRules}
			got := e.shouldFilter(tt.chName, tt.alias, tt.group, tt.skipGroupRules)
			if got != tt.wantDrop {
				t.Errorf("shouldFilter(%q, %q, %q, %v) = %v, want %v",
					tt.chName, tt.alias, tt.group, tt.skipGroupRules, got, tt.wantDrop)
			}
		})
	}
}
