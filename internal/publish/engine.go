package publish

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"iptv-tool-v2/internal/model"
)

// AggregatedChannel is the result of applying rules to a parsed channel
type AggregatedChannel struct {
	Name        string
	Alias       string
	URL         string
	Group       string
	Logo        string
	TVGId       string
	TVGName     string
	CatchupSrc  string
	CatchupDays int // 回看天数
}

// DIYPProgram is a single EPG entry in the DIYP JSON format
type DIYPProgram struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type DIYPResponse struct {
	ChannelName string        `json:"channel_name"`
	Date        string        `json:"date"`
	EPGData     []DIYPProgram `json:"epg_data"`
}

// Rule Configuration Structures
type AliasRule struct {
	MatchMode   model.MatchMode `json:"match_mode"`
	Pattern     string          `json:"pattern"`
	Replacement string          `json:"replacement"`
	regex       *regexp.Regexp
}

type FilterRule struct {
	MatchMode model.MatchMode `json:"match_mode"`
	Target    string          `json:"target"` // "name" or "alias"
	Pattern   string          `json:"pattern"`
	regex     *regexp.Regexp
}

type GroupRuleConfig struct {
	GroupName string `json:"group_name"`
	Rules     []struct {
		Target    string          `json:"target"`
		MatchMode model.MatchMode `json:"match_mode"`
		Pattern   string          `json:"pattern"`
		regex     *regexp.Regexp
	} `json:"rules"`
}

// Engine handles the aggregation logic
type Engine struct {
	iface        model.PublishInterface
	aliasRules   []AliasRule
	filterRules  []FilterRule
	groupRules   []GroupRuleConfig
	logoMapCache map[string]string // Maps lowercased logo name to url_path
}

// NewEngine creates a new publish engine for the given interface
func NewEngine(iface model.PublishInterface) (*Engine, error) {
	e := &Engine{iface: iface}

	// Load and compile rules
	ruleIDs := parseSourceIDs(iface.RuleIDs)
	if len(ruleIDs) > 0 {
		var rules []model.AggregationRule
		if err := model.DB.Where("id IN ? AND status = ?", ruleIDs, true).Find(&rules).Error; err == nil {
			for _, r := range rules {
				switch r.Type {
				case model.RuleTypeAlias:
					var ar []AliasRule
					if json.Unmarshal([]byte(r.Config), &ar) == nil {
						for i := range ar {
							if ar[i].MatchMode == model.MatchModeRegex {
								ar[i].regex, _ = regexp.Compile(ar[i].Pattern)
								// Replace $G1, $G2 with $1, $2 for Go regexp
								ar[i].Replacement = strings.ReplaceAll(ar[i].Replacement, "$G", "$")
							}
						}
						e.aliasRules = append(e.aliasRules, ar...)
					}
				case model.RuleTypeFilter:
					var fr []FilterRule
					if json.Unmarshal([]byte(r.Config), &fr) == nil {
						for i := range fr {
							if fr[i].MatchMode == model.MatchModeRegex {
								fr[i].regex, _ = regexp.Compile(fr[i].Pattern)
							}
						}
						e.filterRules = append(e.filterRules, fr...)
					}
				case model.RuleTypeGroup:
					var gr []GroupRuleConfig
					if json.Unmarshal([]byte(r.Config), &gr) == nil {
						for i := range gr {
							for j := range gr[i].Rules {
								if gr[i].Rules[j].MatchMode == model.MatchModeRegex {
									gr[i].Rules[j].regex, _ = regexp.Compile(gr[i].Rules[j].Pattern)
								}
							}
						}
						e.groupRules = append(e.groupRules, gr...)
					}
				}
			}
		}
	}

	return e, nil
}

// buildLogoMap builds an ignore-case map of uploaded logos
func (e *Engine) buildLogoMap(requestHost string) {
	e.logoMapCache = make(map[string]string)
	var logos []model.ChannelLogo
	if err := model.DB.Find(&logos).Error; err == nil {
		for _, l := range logos {
			url := l.URLPath
			if strings.HasPrefix(url, "/") {
				url = fmt.Sprintf("http://%s%s", requestHost, url)
			}
			e.logoMapCache[strings.ToLower(l.Name)] = url
		}
	}
}

// applyAlias returns the alias name if matched, otherwise empty string
func (e *Engine) applyAlias(name string) string {
	for _, ar := range e.aliasRules {
		if ar.MatchMode == model.MatchModeRegex && ar.regex != nil {
			if ar.regex.MatchString(name) {
				return ar.regex.ReplaceAllString(name, ar.Replacement)
			}
		} else if ar.MatchMode == model.MatchModeString {
			if strings.Contains(strings.ToLower(name), strings.ToLower(ar.Pattern)) {
				return ar.Replacement
			}
		}
	}
	return ""
}

// shouldFilter returns true if the channel should be dropped
func (e *Engine) shouldFilter(name, alias string) bool {
	for _, fr := range e.filterRules {
		targetVal := name
		if fr.Target == "alias" {
			targetVal = alias
			if targetVal == "" {
				targetVal = name
			}
		}

		if fr.MatchMode == model.MatchModeRegex && fr.regex != nil {
			if fr.regex.MatchString(targetVal) {
				return true
			}
		} else if fr.MatchMode == model.MatchModeString {
			if strings.Contains(strings.ToLower(targetVal), strings.ToLower(fr.Pattern)) {
				return true
			}
		}
	}
	return false
}

// applyGroup returns the matched group name or original if not matched
func (e *Engine) applyGroup(name, alias, originalGroup string) string {
	for _, g := range e.groupRules {
		for _, r := range g.Rules {
			targetVal := name
			if r.Target == "alias" {
				targetVal = alias
				if targetVal == "" {
					targetVal = name
				}
			}

			if r.MatchMode == model.MatchModeRegex && r.regex != nil {
				if r.regex.MatchString(targetVal) {
					return g.GroupName
				}
			} else if r.MatchMode == model.MatchModeString {
				if strings.Contains(strings.ToLower(targetVal), strings.ToLower(r.Pattern)) {
					return g.GroupName
				}
			}
		}
	}
	if originalGroup != "" {
		return originalGroup
	}
	return "未分组"
}

// applyLogo matches the channel against uploaded logos
func (e *Engine) applyLogo(name, alias string) string {
	if e.logoMapCache == nil {
		return ""
	}
	// Try alias first
	if alias != "" {
		if url, ok := e.logoMapCache[strings.ToLower(alias)]; ok {
			return url
		}
	}
	// Fallback to name
	if url, ok := e.logoMapCache[strings.ToLower(name)]; ok {
		return url
	}
	return ""
}

// --- Live channel aggregation ---

// AggregateLiveChannels loads channels from sources, applies rules sequentially, and returns result
func (e *Engine) AggregateLiveChannels(requestHost string) ([]AggregatedChannel, error) {
	sourceIDs := parseSourceIDs(e.iface.SourceIDs)
	if len(sourceIDs) == 0 {
		return nil, nil
	}

	// Filter out disabled live sources
	var activeSourceIDs []uint
	if err := model.DB.Model(&model.LiveSource{}).Where("id IN ? AND status = ?", sourceIDs, true).Pluck("id", &activeSourceIDs).Error; err != nil {
		slog.Error("Publish Engine: Failed to filter active sources", "error", err, "iface_id", e.iface.ID)
		return nil, fmt.Errorf("failed to filter active sources: %w", err)
	}

	if len(activeSourceIDs) == 0 {
		return nil, nil // No active sources
	}

	var parsedChannels []model.ParsedChannel
	if err := model.DB.Where("source_id IN ?", activeSourceIDs).Find(&parsedChannels).Error; err != nil {
		slog.Error("Publish Engine: Failed to load channels", "error", err, "iface_id", e.iface.ID)
		return nil, fmt.Errorf("failed to load channels: %w", err)
	}

	// For Auto-logo, load map once
	e.buildLogoMap(requestHost)

	// Build set of source IDs that should filter timeout channels
	filterInvalidSet := make(map[uint]bool)
	for _, id := range parseSourceIDs(e.iface.FilterInvalidSourceIDs) {
		filterInvalidSet[id] = true
	}

	var result []AggregatedChannel
	seen := make(map[string]bool)

	for _, ch := range parsedChannels {
		// Stage 0: Skip channels that have been detected as timeout (latency == -1)
		// Only filter if the channel's source is in the filter-invalid set
		// Channels that have not been detected (latency == nil) are NOT filtered
		if ch.Latency != nil && *ch.Latency == -1 && filterInvalidSet[ch.SourceID] {
			continue
		}

		// Stage 1: Alias
		alias := e.applyAlias(ch.Name)

		// Stage 2: Filter
		if e.shouldFilter(ch.Name, alias) {
			continue
		}

		// Stage 3: Group
		group := e.applyGroup(ch.Name, alias, ch.Group)

		// Stage 4: Logo (Auto-match only)
		logo := e.applyLogo(ch.Name, alias)

		// Dedup URL
		if seen[ch.URL] {
			continue
		}
		seen[ch.URL] = true

		agg := AggregatedChannel{
			Name:        ch.Name,
			Alias:       alias,
			URL:         e.extractBestURL(ch.URL, ch.CatchupURL),
			Group:       group,
			Logo:        logo,
			TVGId:       ch.TVGId,
			TVGName:     ch.TVGName,
			CatchupSrc:  ch.CatchupURL,
			CatchupDays: ch.CatchupDays,
		}

		result = append(result, agg)
	}

	return result, nil
}

// isMulticastURL 判断给定的最终播放地址是否为组播类型
// 包括: igmp://, rtp://, 以及通过 UDPXY 代理的组播地址
func (e *Engine) isMulticastURL(url string) bool {
	if strings.HasPrefix(url, "igmp://") || strings.HasPrefix(url, "rtp://") {
		return true
	}
	// UDPXY 代理的组播地址形如 http://udpxy-host:port/rtp/239.x.x.x:1234
	if e.iface.UDPxyURL != "" && strings.HasPrefix(url, strings.TrimRight(e.iface.UDPxyURL, "/")) {
		return true
	}
	return false
}

func (e *Engine) extractBestURL(rawURLs, catchupURL string) string {
	urls := strings.Split(rawURLs, "|")

	var multicastURL string
	var unicastURL string

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if strings.HasPrefix(u, "igmp://") || strings.HasPrefix(u, "rtp://") {
			if multicastURL == "" {
				multicastURL = u
			}
		} else if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "rtsp://") || strings.HasPrefix(u, "rtmp://") {
			if unicastURL == "" {
				unicastURL = u
			}
		}
	}

	// 单播优先策略 (unicast)
	if e.iface.AddressType == "unicast" {
		if unicastURL != "" {
			return unicastURL
		}
		// 如果只有组播，但配置了单播优先，尝试使用回看/时移地址作为单播地址
		if catchupURL != "" {
			return catchupURL
		}
		// 实在没有办法，只能退回返回组播地址
		return multicastURL
	}

	// 组播优先策略 (multicast) - 默认
	if multicastURL != "" {
		switch e.iface.MulticastType {
		case "udpxy":
			if e.iface.UDPxyURL != "" && strings.HasPrefix(multicastURL, "igmp://") {
				addr := strings.TrimPrefix(multicastURL, "igmp://")
				return strings.TrimRight(e.iface.UDPxyURL, "/") + "/rtp/" + addr
			}
		case "rtp":
			if strings.HasPrefix(multicastURL, "igmp://") {
				return "rtp://" + strings.TrimPrefix(multicastURL, "igmp://")
			}
		case "igmp":
			return multicastURL
		}
		return multicastURL
	}

	// 选了组播优先，但源列表里根本没组播地址，降级使用单播
	if unicastURL != "" {
		return unicastURL
	}

	// 兜底返回最原始的值
	return rawURLs
}

func (e *Engine) FormatM3U(channels []AggregatedChannel) string {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	// 去除用户输入的回看模板前面的多余符号
	templateParams := strings.TrimLeft(e.iface.M3UCatchupTemplate, "?&")
	for _, ch := range channels {
		displayName := ch.Name
		if ch.Alias != "" {
			displayName = ch.Alias
		}

		// ====== tvg-id 处理逻辑 ======
		tvgID := ch.TVGId
		if e.iface.TvgIDMode == "name" {
			tvgID = displayName // 优先使用别名(displayName已经处理过逻辑了)，若无别名则使用原名
		}

		// 只有当 tvgID 不为空时才输出 tvg-id 属性
		if tvgID != "" {
			sb.WriteString(fmt.Sprintf(`#EXTINF:-1 tvg-id="%s" tvg-name="%s"`, tvgID, displayName))
		} else {
			sb.WriteString(fmt.Sprintf(`#EXTINF:-1 tvg-name="%s"`, displayName))
		}

		if ch.Logo != "" {
			sb.WriteString(fmt.Sprintf(` tvg-logo="%s"`, ch.Logo))
		}
		// ====== 核心功能：处理 Catchup 时移参数 ======
		if templateParams != "" {
			isMulticast := e.isMulticastURL(ch.URL)
			if ch.CatchupSrc != "" {
				// 有专属的 TimeShiftURL（无论组播/单播），使用 default 模式
				chCatchupSource := ch.CatchupSrc
				if strings.Contains(chCatchupSource, "?") {
					chCatchupSource += "&" + templateParams
				} else {
					chCatchupSource += "?" + templateParams
				}
				sb.WriteString(fmt.Sprintf(` catchup="default" catchup-source="%s"`, chCatchupSource))
				if ch.CatchupDays > 0 {
					sb.WriteString(fmt.Sprintf(` catchup-days="%d"`, ch.CatchupDays))
				}
			} else if !isMulticast {
				// 单播源且无专属 TimeShiftURL，使用 append 模式直接追加参数
				sb.WriteString(fmt.Sprintf(` catchup="append" catchup-source="?%s"`, templateParams))
				if ch.CatchupDays > 0 {
					sb.WriteString(fmt.Sprintf(` catchup-days="%d"`, ch.CatchupDays))
				}
			}
			// 组播源且无 TimeShiftURL → 不生成任何 catchup 参数
		}
		sb.WriteString(fmt.Sprintf(` group-title="%s",%s`, ch.Group, displayName))
		sb.WriteString("\n")
		sb.WriteString(ch.URL)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (e *Engine) FormatTXT(channels []AggregatedChannel) string {
	groupOrder := make([]string, 0)
	grouped := make(map[string][]AggregatedChannel)

	for _, ch := range channels {
		if _, exists := grouped[ch.Group]; !exists {
			groupOrder = append(groupOrder, ch.Group)
		}
		grouped[ch.Group] = append(grouped[ch.Group], ch)
	}

	var sb strings.Builder
	for _, group := range groupOrder {
		sb.WriteString(fmt.Sprintf("%s,#genre#\n", group))
		for _, ch := range grouped[group] {
			displayName := ch.Name
			if ch.Alias != "" {
				displayName = ch.Alias
			}
			sb.WriteString(fmt.Sprintf("%s,%s\n", displayName, ch.URL))
		}
	}

	return sb.String()
}

// --- EPG aggregation ---

// AggregatedEPGProgram includes alias for EPG
type AggregatedEPGProgram struct {
	model.ParsedEPG
	Alias string
}

func (e *Engine) AggregateEPGPrograms() ([]AggregatedEPGProgram, error) {
	sourceIDs := parseSourceIDs(e.iface.SourceIDs)
	if len(sourceIDs) == 0 {
		return nil, nil
	}

	// Filter out disabled EPG sources
	var activeSourceIDs []uint
	if err := model.DB.Model(&model.EPGSource{}).Where("id IN ? AND status = ?", sourceIDs, true).Pluck("id", &activeSourceIDs).Error; err != nil {
		slog.Error("Publish Engine: Failed to filter active EPG sources", "error", err, "iface_id", e.iface.ID)
		return nil, fmt.Errorf("failed to filter active EPG sources: %w", err)
	}

	if len(activeSourceIDs) == 0 {
		return nil, nil // No active sources
	}

	query := model.DB.Where("source_id IN ?", activeSourceIDs)

	if e.iface.EPGDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -e.iface.EPGDays)
		query = query.Where("start_time >= ?", cutoff)
	}

	var programs []model.ParsedEPG
	if err := query.Order("channel, start_time").Find(&programs).Error; err != nil {
		slog.Error("Publish Engine: Failed to load EPG programs", "error", err, "iface_id", e.iface.ID)
		return nil, fmt.Errorf("failed to load EPG programs: %w", err)
	}

	var result []AggregatedEPGProgram

	// Filter cache by channel to avoid re-evaluating rules for every program
	channelStateCache := make(map[string]struct {
		ShouldDrop bool
		Alias      string
	})

	for _, p := range programs {
		cache, exists := channelStateCache[p.ChannelName]
		if !exists {
			// Stage 1: Alias
			alias := e.applyAlias(p.ChannelName)
			// Stage 2: Filter
			drop := e.shouldFilter(p.ChannelName, alias)

			cache = struct {
				ShouldDrop bool
				Alias      string
			}{ShouldDrop: drop, Alias: alias}
			channelStateCache[p.ChannelName] = cache
		}

		if cache.ShouldDrop {
			continue
		}

		result = append(result, AggregatedEPGProgram{
			ParsedEPG: p,
			Alias:     cache.Alias,
		})
	}

	return result, nil
}

func (e *Engine) FormatXMLTV(programs []AggregatedEPGProgram) string {
	channelMap := make(map[string]string) // XMLTV channel id -> DisplayName

	// 针对频道/节目映射，准备存储结构
	type uniqueProg struct {
		start string
		end   string
		title string
		desc  string
	}
	// uniqueProgramMap 结构: map[xmltv_channel_id]map[start_time]uniqueProg
	// 用来防止同一 channel_id 下存在起止时间完全相同的重复节目
	uniqueProgramMap := make(map[string]map[string]uniqueProg)

	for _, p := range programs {
		displayName := p.ChannelName
		if p.Alias != "" {
			displayName = p.Alias
		}

		// ====== channel id 处理逻辑 ======
		xmltvChID := p.Channel
		if e.iface.TvgIDMode == "name" {
			xmltvChID = displayName
		}

		// 如果没有获取到有效ID，为了保证 XMLTV 规范，用 displayName 兜底
		if xmltvChID == "" {
			xmltvChID = displayName
		}

		// 填充唯一的顶部频道映射
		channelMap[xmltvChID] = displayName

		// 填充排重节目表
		if uniqueProgramMap[xmltvChID] == nil {
			uniqueProgramMap[xmltvChID] = make(map[string]uniqueProg)
		}

		start := p.StartTime.Format("20060102150405 -0700")
		// 根据 start time 作为排重 key，如果同时间存在节目则保留第一个（抛弃后续的）
		if _, exists := uniqueProgramMap[xmltvChID][start]; !exists {
			uniqueProgramMap[xmltvChID][start] = uniqueProg{
				start: start,
				end:   p.EndTime.Format("20060102150405 -0700"),
				title: p.Title,
				desc:  p.Desc,
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(`<tv generator-info-name="iptv-tool">`)
	sb.WriteString("\n")

	// 1. 生成 <channel> 头部，完全去重
	for chID, dispName := range channelMap {
		sb.WriteString(fmt.Sprintf(`  <channel id="%s">`, xmlEscape(chID)))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`    <display-name lang="zh">%s</display-name>`, xmlEscape(dispName)))
		sb.WriteString("\n")
		sb.WriteString("  </channel>\n")
	}

	// 2. 生成 <programme> 内容
	for chID, progsByStart := range uniqueProgramMap {
		for _, prog := range progsByStart {
			sb.WriteString(fmt.Sprintf(`  <programme start="%s" stop="%s" channel="%s">`,
				prog.start, prog.end, xmlEscape(chID)))
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf(`    <title lang="zh">%s</title>`, xmlEscape(prog.title)))
			sb.WriteString("\n")
			if prog.desc != "" {
				sb.WriteString(fmt.Sprintf(`    <desc lang="zh">%s</desc>`, xmlEscape(prog.desc)))
				sb.WriteString("\n")
			}
			sb.WriteString("  </programme>\n")
		}
	}

	sb.WriteString("</tv>\n")
	return sb.String()
}

func (e *Engine) FormatXMLTVGzip(programs []AggregatedEPGProgram, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment; filename=epg.xml.gz")

	gz := gzip.NewWriter(w)
	defer gz.Close()

	content := e.FormatXMLTV(programs)
	_, err := gz.Write([]byte(content))
	return err
}

func (e *Engine) FormatDIYP(programs []AggregatedEPGProgram, filterChannelName, dateStr string) string {
	// 强制设定为空时的默认查询日期（今日）
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	// 需求：如果未传入频道名称参数，返回一整天的默认空提示数据
	if filterChannelName == "" {
		var defaultEpgData []DIYPProgram
		for i := 0; i < 24; i++ {
			defaultEpgData = append(defaultEpgData, DIYPProgram{
				Start: fmt.Sprintf("%02d:00", i),
				End:   fmt.Sprintf("%02d:59", i),
				Title: "精彩节目-暂未提供节目预告信息",
			})
		}

		resp := DIYPResponse{
			ChannelName: "未提供",
			Date:        dateStr,
			EPGData:     defaultEpgData,
		}

		data, _ := json.Marshal(resp)
		return string(data)
	}

	var filtered []AggregatedEPGProgram
	for _, p := range programs {
		effName := p.Alias
		if effName == "" {
			effName = p.ChannelName
		}
		if !strings.EqualFold(effName, filterChannelName) {
			continue
		}

		// 此时 dateStr 必有值，精确匹配该日期的节目
		progDate := p.StartTime.Format("2006-01-02")
		if progDate != dateStr {
			continue
		}

		filtered = append(filtered, p)
	}

	epgData := make([]DIYPProgram, 0, len(filtered))
	for _, p := range filtered {
		epgData = append(epgData, DIYPProgram{
			Title: p.Title,
			Desc:  p.Desc,
			Start: p.StartTime.Format("15:04"),
			End:   p.EndTime.Format("15:04"),
		})
	}

	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	resp := DIYPResponse{
		ChannelName: filterChannelName,
		Date:        dateStr,
		EPGData:     epgData,
	}

	data, _ := json.Marshal(resp)
	return string(data)
}

func parseSourceIDs(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var ids []uint
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}
	return ids
}

func xmlEscape(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
