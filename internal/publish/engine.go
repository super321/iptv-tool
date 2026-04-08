package publish

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"iptv-tool-v2/internal/model"
)

// SourceOutputConfig holds per-source output settings that can override the global interface config
type SourceOutputConfig struct {
	AddressType        string `json:"address_type"`
	MulticastType      string `json:"multicast_type"`
	UDPxyURL           string `json:"udpxy_url"`
	FCCEnabled         bool   `json:"fcc_enabled"`
	FCCType            string `json:"fcc_type"`
	CustomParams       string `json:"custom_params"`
	M3UCatchupTemplate string `json:"m3u_catchup_template"`
	UnicastType        string `json:"unicast_type"`
	UnicastProxyRules  string `json:"unicast_proxy_rules"`
}

// UnicastProxyRule defines a regex-based URL transformation rule for unicast addresses
type UnicastProxyRule struct {
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	regex       *regexp.Regexp
}

// AggregatedChannel is the result of applying rules to a parsed channel
type AggregatedChannel struct {
	Name            string
	Alias           string
	URL             string
	Group           string
	Logo            string // 台标管理匹配到的相对路径 (e.g. /logo/cctv1.png)
	SourceLogo      string // 数据源解析时自带的原始 logo URL
	TVGId           string
	TVGName         string
	CatchupSrc      string
	CatchupDays     int    // 回看天数
	FCCIP           string // FCC server IP
	FCCPort         string // FCC server port
	SourceID        uint   // Source ID for per-source config lookup
	CatchupTemplate string // Per-source catchup template (empty = use global)
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
	Target    string          `json:"target"` // "name", "alias", or "group"
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
	iface              model.PublishInterface
	aliasRules         []AliasRule
	filterRules        []FilterRule
	groupRules         []GroupRuleConfig
	logoMapCache       map[string]string           // Maps lowercased logo name to url_path
	sourceConfigs      map[uint]SourceOutputConfig // Per-source output configs (nil = all global)
	unicastRules       []UnicastProxyRule          // Pre-compiled global unicast proxy rules
	sourceUnicastRules map[uint][]UnicastProxyRule // Pre-compiled per-source unicast proxy rules
	globalSourceCfg    SourceOutputConfig          // Global config as SourceOutputConfig (unified code path)
	globalCustomParams []customParam               // Pre-parsed global custom params
	sourceCustomParams map[uint][]customParam      // Pre-parsed per-source custom params
}

// NewEngine creates a new publish engine for the given interface
func NewEngine(iface model.PublishInterface) (*Engine, error) {
	e := &Engine{iface: iface}

	// Parse per-source output configs if present
	if iface.SourceOutputConfigs != "" {
		var rawMap map[string]SourceOutputConfig
		if err := json.Unmarshal([]byte(iface.SourceOutputConfigs), &rawMap); err == nil {
			e.sourceConfigs = make(map[uint]SourceOutputConfig, len(rawMap))
			for k, v := range rawMap {
				if id, err := strconv.ParseUint(k, 10, 32); err == nil {
					e.sourceConfigs[uint(id)] = v
					// Pre-compile per-source unicast proxy rules
					if v.UnicastType == "proxy" {
						if e.sourceUnicastRules == nil {
							e.sourceUnicastRules = make(map[uint][]UnicastProxyRule)
						}
						e.sourceUnicastRules[uint(id)] = parseUnicastProxyRules(v.UnicastProxyRules)
					}
				}
			}
		}
	}

	// Pre-compile global unicast proxy rules
	if iface.UnicastType == "proxy" {
		e.unicastRules = parseUnicastProxyRules(iface.UnicastProxyRules)
	}

	// Build global source config for unified code paths (eliminates duplicate functions)
	e.globalSourceCfg = SourceOutputConfig{
		AddressType:   iface.AddressType,
		MulticastType: iface.MulticastType,
		UDPxyURL:      iface.UDPxyURL,
		FCCEnabled:    iface.FCCEnabled,
		FCCType:       iface.FCCType,
		CustomParams:  iface.CustomParams,
		UnicastType:   iface.UnicastType,
	}

	// Pre-parse custom params to avoid repeated JSON deserialization per channel
	e.globalCustomParams = parseCustomParams(iface.CustomParams)
	if len(e.sourceConfigs) > 0 {
		e.sourceCustomParams = make(map[uint][]customParam, len(e.sourceConfigs))
		for id, cfg := range e.sourceConfigs {
			e.sourceCustomParams[id] = parseCustomParams(cfg.CustomParams)
		}
	}

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

// buildLogoMap builds an ignore-case map of uploaded logos.
// Stores relative paths (e.g. /logo/cctv1.png) rather than full URLs,
// so that the cached data is independent of the client's request host.
func (e *Engine) buildLogoMap() {
	e.logoMapCache = make(map[string]string)
	var logos []model.ChannelLogo
	if err := model.DB.Find(&logos).Error; err == nil {
		for _, l := range logos {
			e.logoMapCache[strings.ToLower(l.Name)] = l.URLPath
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
func (e *Engine) shouldFilter(name, alias, group string, skipGroupRules bool) bool {
	for _, fr := range e.filterRules {
		if skipGroupRules && fr.Target == "group" {
			continue
		}
		targetVal := name
		if fr.Target == "alias" {
			targetVal = alias
			if targetVal == "" {
				targetVal = name
			}
		} else if fr.Target == "group" {
			targetVal = group
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

// applyGroup returns the matched group name.
// When hasGroupRules is true (group rules are configured), unmatched channels get empty group
// (source's original group is ignored). When false, the original group is preserved as-is.
func (e *Engine) applyGroup(name, alias, originalGroup string, hasGroupRules bool) string {
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
	if hasGroupRules {
		return ""
	}
	return originalGroup
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

// AggregateLiveChannels loads channels from sources, applies rules sequentially, and returns result.
// Logo fields contain relative paths (e.g. /logo/cctv1.png); full URL resolution
// is deferred to format-time (FormatM3U) so the result is cacheable across hosts.
func (e *Engine) AggregateLiveChannels() ([]AggregatedChannel, error) {
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

	// For Auto-logo, load map once (stores relative paths)
	e.buildLogoMap()

	// Build set of source IDs that should filter timeout channels
	filterInvalidSet := make(map[uint]bool)
	for _, id := range parseSourceIDs(e.iface.FilterInvalidSourceIDs) {
		filterInvalidSet[id] = true
	}

	var result []AggregatedChannel
	seen := make(map[string]bool)
	hasGroupRules := len(e.groupRules) > 0

	for _, ch := range parsedChannels {
		// Stage 0: Skip channels that have been detected as timeout (latency == -1)
		// Only filter if the channel's source is in the filter-invalid set
		// Channels that have not been detected (latency == nil) are NOT filtered
		if ch.Latency != nil && *ch.Latency == -1 && filterInvalidSet[ch.SourceID] {
			continue
		}

		// Stage 1: Alias
		alias := e.applyAlias(ch.Name)

		// Stage 2: Group (before filter, so filter can reference computed group)
		group := e.applyGroup(ch.Name, alias, ch.Group, hasGroupRules)

		// Stage 3: Filter (can now filter by group name)
		if e.shouldFilter(ch.Name, alias, group, false) {
			continue
		}

		// Stage 4: Logo (Auto-match only)
		logo := e.applyLogo(ch.Name, alias)

		// Dedup URL
		if seen[ch.URL] {
			continue
		}
		seen[ch.URL] = true

		// Determine URL and catchup template based on per-source or global config
		var channelURL string
		var catchupTemplate string
		var catchupSrc string
		if srcCfg, ok := e.sourceConfigs[ch.SourceID]; ok {
			channelURL = e.extractBestURLWithConfig(srcCfg, e.sourceUnicastRules[ch.SourceID], e.sourceCustomParams[ch.SourceID], ch.URL, ch.CatchupURL, ch.FCCIP, ch.FCCPort)
			catchupTemplate = srcCfg.M3UCatchupTemplate
			// Apply pre-compiled per-source unicast proxy rules to catchup source
			if rules, ok := e.sourceUnicastRules[ch.SourceID]; ok {
				catchupSrc = transformUnicastURL(ch.CatchupURL, rules)
			} else {
				catchupSrc = ch.CatchupURL
			}
		} else {
			channelURL = e.extractBestURLWithConfig(e.globalSourceCfg, e.unicastRules, e.globalCustomParams, ch.URL, ch.CatchupURL, ch.FCCIP, ch.FCCPort)
			// Apply global unicast proxy rules to catchup source
			catchupSrc = transformUnicastURL(ch.CatchupURL, e.unicastRules)
		}

		agg := AggregatedChannel{
			Name:            ch.Name,
			Alias:           alias,
			URL:             channelURL,
			Group:           group,
			Logo:            logo,
			SourceLogo:      ch.Logo,
			TVGId:           ch.TVGId,
			TVGName:         ch.TVGName,
			CatchupSrc:      catchupSrc,
			CatchupDays:     ch.CatchupDays,
			FCCIP:           ch.FCCIP,
			FCCPort:         ch.FCCPort,
			SourceID:        ch.SourceID,
			CatchupTemplate: catchupTemplate,
		}

		result = append(result, agg)
	}

	return result, nil
}

// customParam represents a single custom URL parameter
type customParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// parseCustomParams parses a JSON string of custom params into a slice.
// Called once at Engine creation to avoid per-channel JSON deserialization.
func parseCustomParams(raw string) []customParam {
	if raw == "" {
		return nil
	}
	var params []customParam
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return nil
	}
	var result []customParam
	for _, p := range params {
		if strings.TrimSpace(p.Key) != "" {
			result = append(result, customParam{
				Key:   strings.TrimSpace(p.Key),
				Value: strings.TrimSpace(p.Value),
			})
		}
	}
	return result
}

// extractMulticastAddr strips the igmp:// or rtp:// prefix from a multicast URL
// and returns the raw address (e.g. "239.1.2.3:5140" or "[ff0e::1]:5140").
// This works for both IPv4 and IPv6 multicast addresses.
func extractMulticastAddr(multicastURL string) (addr string, ok bool) {
	if strings.HasPrefix(multicastURL, "igmp://") {
		return strings.TrimPrefix(multicastURL, "igmp://"), true
	}
	if strings.HasPrefix(multicastURL, "rtp://") {
		return strings.TrimPrefix(multicastURL, "rtp://"), true
	}
	return "", false
}

// parseUnicastProxyRules parses a JSON string into compiled UnicastProxyRule slice
func parseUnicastProxyRules(raw string) []UnicastProxyRule {
	if raw == "" {
		return nil
	}
	var rules []UnicastProxyRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		slog.Error("Failed to parse unicast proxy rules", "error", err)
		return nil
	}
	var compiled []UnicastProxyRule
	for _, r := range rules {
		if strings.TrimSpace(r.Pattern) == "" {
			continue
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			slog.Warn("Skipping invalid unicast proxy rule regex", "pattern", r.Pattern, "error", err)
			continue
		}
		compiled = append(compiled, UnicastProxyRule{
			Pattern:     r.Pattern,
			Replacement: r.Replacement,
			regex:       re,
		})
	}
	return compiled
}

// transformUnicastURL applies unicast proxy rules to a unicast URL.
// Rules are matched in order; the first match wins.
func transformUnicastURL(url string, rules []UnicastProxyRule) string {
	for _, r := range rules {
		if r.regex != nil && r.regex.MatchString(url) {
			return r.regex.ReplaceAllString(url, r.Replacement)
		}
	}
	return url
}

// isMulticastURLStr checks if a URL is multicast using the given udpxyURL
func isMulticastURLStr(url, udpxyURL string) bool {
	if strings.HasPrefix(url, "igmp://") || strings.HasPrefix(url, "rtp://") {
		return true
	}
	if udpxyURL != "" && strings.HasPrefix(url, strings.TrimRight(udpxyURL, "/")) {
		return true
	}
	return false
}

// transformMulticastURLWithConfig transforms a multicast URL using a SourceOutputConfig.
// params are pre-parsed custom params (parsed once at Engine creation, not per-channel).
func transformMulticastURLWithConfig(cfg SourceOutputConfig, params []customParam, multicastURL, fccIP, fccPort string) string {
	switch cfg.MulticastType {
	case "udpxy":
		if addr, ok := extractMulticastAddr(multicastURL); ok && cfg.UDPxyURL != "" {
			result := strings.TrimRight(cfg.UDPxyURL, "/") + "/rtp/" + addr
			if cfg.FCCEnabled && fccIP != "" && fccPort != "" {
				result += "?fcc=" + fccIP + ":" + fccPort
				if cfg.FCCType == "huawei" {
					result += "&fcc-type=huawei"
				}
			}
			for _, p := range params {
				if strings.Contains(result, "?") {
					result += "&" + p.Key + "=" + p.Value
				} else {
					result += "?" + p.Key + "=" + p.Value
				}
			}
			return result
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

// extractBestURLWithConfig selects the best URL from rawURLs based on the given config.
func (e *Engine) extractBestURLWithConfig(cfg SourceOutputConfig, unicastRules []UnicastProxyRule, params []customParam, rawURLs, catchupURL, fccIP, fccPort string) string {
	urls := strings.Split(rawURLs, "|")

	var multicastURL string
	var unicastURL string

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
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

	if cfg.AddressType == "unicast" {
		if unicastURL != "" {
			return transformUnicastURL(unicastURL, unicastRules)
		}
		if multicastURL != "" && catchupURL != "" && !isMulticastURLStr(catchupURL, cfg.UDPxyURL) {
			return transformUnicastURL(catchupURL, unicastRules)
		}
		if multicastURL != "" {
			return transformMulticastURLWithConfig(cfg, params, multicastURL, fccIP, fccPort)
		}
		return rawURLs
	}

	// multicast priority (default)
	if multicastURL != "" {
		return transformMulticastURLWithConfig(cfg, params, multicastURL, fccIP, fccPort)
	}

	if unicastURL != "" {
		return transformUnicastURL(unicastURL, unicastRules)
	}

	return rawURLs
}

func (e *Engine) FormatM3U(channels []AggregatedChannel, requestHost string) string {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	// Global catchup template (fallback)
	globalTemplateParams := strings.TrimLeft(e.iface.M3UCatchupTemplate, "?&")
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

		// ====== tvg-logo 三级回退逻辑 ======
		// 优先级 1: 台标管理匹配的 logo（相对路径，根据客户端请求地址组装完整 URL）
		// 优先级 2: 数据源解析时自带的原始 logo URL（直接使用原始地址）
		// 优先级 3: 均无 logo，不生成 tvg-logo 属性
		if ch.Logo != "" {
			logoURL := fmt.Sprintf("http://%s%s", requestHost, ch.Logo)
			sb.WriteString(fmt.Sprintf(` tvg-logo="%s"`, logoURL))
		} else if ch.SourceLogo != "" {
			sb.WriteString(fmt.Sprintf(` tvg-logo="%s"`, ch.SourceLogo))
		}
		// ====== 核心功能：处理 Catchup 时移参数 ======
		// When per-source mode is active, only use per-source template (no global fallback)
		var templateParams string
		if ch.CatchupTemplate != "" {
			templateParams = strings.TrimLeft(ch.CatchupTemplate, "?&")
		} else if len(e.sourceConfigs) == 0 {
			templateParams = globalTemplateParams
		}
		if templateParams != "" {
			// Determine multicast detection using per-source config if available
			isMulticast := false
			if srcCfg, ok := e.sourceConfigs[ch.SourceID]; ok {
				isMulticast = isMulticastURLStr(ch.URL, srcCfg.UDPxyURL)
			} else {
				isMulticast = isMulticastURLStr(ch.URL, e.globalSourceCfg.UDPxyURL)
			}
			if ch.CatchupSrc != "" && ch.CatchupSrc != ch.URL {
				// 有专属的 TimeShiftURL，且与直播地址不同，使用 default 模式
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
				// 单播源且无专属 TimeShiftURL（或直播地址已使用回看地址），使用 append 模式直接追加参数
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
		if group != "" {
			sb.WriteString(fmt.Sprintf("%s,#genre#\n", group))
		}
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

// EPGProgram holds a single program's display data
type EPGProgram struct {
	Title     string
	Desc      string
	StartTime time.Time
	EndTime   time.Time
}

// EPGChannelPrograms holds all programs for one channel, grouped by date
type EPGChannelPrograms struct {
	ChannelID    string                  // XMLTV channel id (ParsedEPG.Channel)
	ChannelName  string                  // Original channel name
	Alias        string                  // Applied alias (may be empty)
	DatePrograms map[string][]EPGProgram // key: "2006-01-02", value: programs sorted by StartTime
}

// AggregatedEPG is the top-level cached structure
type AggregatedEPG struct {
	Channels     map[string]*EPGChannelPrograms // key: lowercase effective name (优先使用 alias，无 alias 则用原始名称)
	ChannelOrder []string                       // preserves insertion order (lowercase keys) for deterministic XMLTV output
}

func (e *Engine) AggregateEPG() (*AggregatedEPG, error) {
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

	// Filter cache by channel to avoid re-evaluating rules for every program
	channelStateCache := make(map[string]struct {
		ShouldDrop bool
		Alias      string
	})

	result := &AggregatedEPG{
		Channels: make(map[string]*EPGChannelPrograms),
	}

	// Track last start time per channel for dedup.
	// Since the query results are ordered by (channel, start_time),
	// duplicate start times are adjacent, so we only need to remember the last one.
	prevStartKey := make(map[string]string) // lowerKey -> last start time key

	for _, p := range programs {
		cache, exists := channelStateCache[p.ChannelName]
		if !exists {
			// Stage 1: Alias
			alias := e.applyAlias(p.ChannelName)
			// Stage 2: Filter
			drop := e.shouldFilter(p.ChannelName, alias, "", true)

			cache = struct {
				ShouldDrop bool
				Alias      string
			}{ShouldDrop: drop, Alias: alias}
			channelStateCache[p.ChannelName] = cache
		}

		if cache.ShouldDrop {
			continue
		}

		// Effective display name: alias first, fallback to original name
		effName := cache.Alias
		if effName == "" {
			effName = p.ChannelName
		}
		lowerKey := strings.ToLower(effName)

		chEntry, ok := result.Channels[lowerKey]
		if !ok {
			chEntry = &EPGChannelPrograms{
				ChannelID:    p.Channel,
				ChannelName:  p.ChannelName,
				Alias:        cache.Alias,
				DatePrograms: make(map[string][]EPGProgram),
			}
			result.Channels[lowerKey] = chEntry
			result.ChannelOrder = append(result.ChannelOrder, lowerKey)
		}

		// Dedup: since results are ordered by (channel, start_time), duplicates
		// for the same channel will be adjacent. Compare with the previous record.
		startKey := p.StartTime.Format("20060102150405")
		if prevStartKey[lowerKey] == startKey {
			continue
		}
		prevStartKey[lowerKey] = startKey

		dateKey := p.StartTime.Format("2006-01-02")
		chEntry.DatePrograms[dateKey] = append(chEntry.DatePrograms[dateKey], EPGProgram{
			Title:     p.Title,
			Desc:      p.Desc,
			StartTime: p.StartTime,
			EndTime:   p.EndTime,
		})
	}

	return result, nil
}

func (e *Engine) FormatXMLTV(epg *AggregatedEPG) string {
	var sb strings.Builder
	_ = e.FormatXMLTVToWriter(epg, &sb)
	return sb.String()
}

// FormatXMLTVToWriter writes XMLTV content directly to the given writer,
// avoiding building the entire XML string in memory first.
func (e *Engine) FormatXMLTVToWriter(epg *AggregatedEPG, w io.Writer) error {
	if epg == nil {
		_, err := io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n<tv generator-info-name=\"iptv-tool\">\n</tv>\n")
		return err
	}

	channelMap := make(map[string]string) // XMLTV channel id -> DisplayName
	var xmltvChIDOrder []string

	type xmltvProg struct {
		start string
		end   string
		title string
		desc  string
	}
	channelProgs := make(map[string][]xmltvProg)

	for _, key := range epg.ChannelOrder {
		chEntry := epg.Channels[key]

		displayName := chEntry.ChannelName
		if chEntry.Alias != "" {
			displayName = chEntry.Alias
		}

		xmltvChID := chEntry.ChannelID
		if e.iface.TvgIDMode == "name" {
			xmltvChID = displayName
		}
		if xmltvChID == "" {
			xmltvChID = displayName
		}

		if _, exists := channelMap[xmltvChID]; !exists {
			xmltvChIDOrder = append(xmltvChIDOrder, xmltvChID)
		}
		channelMap[xmltvChID] = displayName

		dates := make([]string, 0, len(chEntry.DatePrograms))
		for d := range chEntry.DatePrograms {
			dates = append(dates, d)
		}
		sort.Strings(dates)

		for _, date := range dates {
			for _, prog := range chEntry.DatePrograms[date] {
				channelProgs[xmltvChID] = append(channelProgs[xmltvChID], xmltvProg{
					start: prog.StartTime.Format("20060102150405 -0700"),
					end:   prog.EndTime.Format("20060102150405 -0700"),
					title: prog.Title,
					desc:  prog.Desc,
				})
			}
		}
	}

	// Write XML header
	if _, err := io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, `<tv generator-info-name="iptv-tool">`+"\n"); err != nil {
		return err
	}

	// Write <channel> elements
	for _, chID := range xmltvChIDOrder {
		dispName := channelMap[chID]
		if _, err := fmt.Fprintf(w, "  <channel id=\"%s\">\n    <display-name lang=\"zh\">%s</display-name>\n  </channel>\n",
			xmlEscape(chID), xmlEscape(dispName)); err != nil {
			return err
		}
	}

	// Write <programme> elements
	for _, chID := range xmltvChIDOrder {
		for _, prog := range channelProgs[chID] {
			if _, err := fmt.Fprintf(w, "  <programme start=\"%s\" stop=\"%s\" channel=\"%s\">\n    <title lang=\"zh\">%s</title>\n",
				prog.start, prog.end, xmlEscape(chID), xmlEscape(prog.title)); err != nil {
				return err
			}
			if prog.desc != "" {
				if _, err := fmt.Fprintf(w, "    <desc lang=\"zh\">%s</desc>\n", xmlEscape(prog.desc)); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, "  </programme>\n"); err != nil {
				return err
			}
		}
	}

	_, err := io.WriteString(w, "</tv>\n")
	return err
}

func (e *Engine) FormatXMLTVGzip(epg *AggregatedEPG, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment; filename=epg.xml.gz")

	gz := gzip.NewWriter(w)
	defer gz.Close()

	return e.FormatXMLTVToWriter(epg, gz)
}

func (e *Engine) FormatDIYP(epg *AggregatedEPG, filterChannelName, dateStr string) string {
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

	// O(1) lookup by channel name (case-insensitive)
	var epgData []DIYPProgram

	if epg != nil {
		if chEntry, ok := epg.Channels[strings.ToLower(filterChannelName)]; ok {
			if progs, ok := chEntry.DatePrograms[dateStr]; ok {
				epgData = make([]DIYPProgram, 0, len(progs))
				for _, p := range progs {
					epgData = append(epgData, DIYPProgram{
						Title: p.Title,
						Desc:  p.Desc,
						Start: p.StartTime.Format("15:04"),
						End:   p.EndTime.Format("15:04"),
					})
				}
			}
		}
	}

	if epgData == nil {
		epgData = []DIYPProgram{}
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
