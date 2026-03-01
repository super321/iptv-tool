package m3u

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// Channel represents a single channel parsed from M3U or TXT format
type Channel struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	OriginalURL string            `json:"original_url"` // 新增
	Group       string            `json:"group"`
	Logo        string            `json:"logo"`
	TVGId       string            `json:"tvg_id"`
	TVGName     string            `json:"tvg_name"`
	CatchupSrc  string            `json:"catchup_src"`
	CatchupDays int               `json:"catchup_days"` // 新增
	Extra       map[string]string `json:"extra"`
}

var (
	extinfRegex = regexp.MustCompile(`#EXTINF:-?\d+\s*(.*)?,(.*)`)
	attrRegex   = regexp.MustCompile(`(\S+?)="(.*?)"`)
)

// ParseM3U parses M3U format content and returns a list of channels
func ParseM3U(content string) ([]Channel, error) {
	var channels []Channel
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentAttrs map[string]string
	var currentName string
	inEntry := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip the header line
		if strings.HasPrefix(line, "#EXTM3U") {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			matches := extinfRegex.FindStringSubmatch(line)
			if len(matches) < 3 {
				continue
			}

			attrStr := matches[1]
			currentName = strings.TrimSpace(matches[2])

			// Parse all key="value" attributes
			currentAttrs = make(map[string]string)
			attrMatches := attrRegex.FindAllStringSubmatch(attrStr, -1)
			for _, am := range attrMatches {
				if len(am) == 3 {
					currentAttrs[strings.ToLower(am[1])] = am[2]
				}
			}
			inEntry = true
			continue
		}

		// Skip other directives
		if strings.HasPrefix(line, "#") {
			continue
		}

		// This line is the URL for the previous #EXTINF entry
		if inEntry {
			var catchupDays int
			if dStr, ok := currentAttrs["catchup-days"]; ok {
				fmt.Sscanf(dStr, "%d", &catchupDays)
			}
			ch := Channel{
				Name:        currentName,
				URL:         line,
				OriginalURL: line,
				Group:       currentAttrs["group-title"],
				Logo:        currentAttrs["tvg-logo"],
				TVGId:       currentAttrs["tvg-id"],
				TVGName:     currentAttrs["tvg-name"],
				CatchupSrc:  currentAttrs["catchup-source"],
				CatchupDays: catchupDays,
				Extra:       currentAttrs,
			}
			if ch.TVGName == "" {
				ch.TVGName = currentName
			}
			channels = append(channels, ch)
			inEntry = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning M3U content: %w", err)
	}

	return channels, nil
}

// ParseTXT parses DIYP TXT format content and returns a list of channels.
// Format:
//
//	GroupName,#genre#
//	ChannelName,URL
//	ChannelName,URL
func ParseTXT(content string) ([]Channel, error) {
	var channels []Channel
	scanner := bufio.NewScanner(strings.NewReader(content))
	currentGroup := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Group header line: "GroupName,#genre#"
		if value == "#genre#" {
			currentGroup = name
			continue
		}

		// Channel line: "ChannelName,URL"
		ch := Channel{
			Name:  name,
			URL:   value,
			Group: currentGroup,
		}
		channels = append(channels, ch)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning TXT content: %w", err)
	}

	return channels, nil
}

// DetectFormat detects whether the content is M3U or TXT format
func DetectFormat(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "#EXTM3U") {
		return "m3u"
	}
	// Check for DIYP TXT pattern: contains ",#genre#"
	if strings.Contains(trimmed, ",#genre#") {
		return "txt"
	}
	// Fallback: if lines look like "name,url" pairs
	return "txt"
}

// FormatM3U generates M3U format string from channels.
// logoPlaceholder is a template like "{HOST}/logo/" that will be replaced at serve time.
func FormatM3U(channels []Channel, catchupTemplate string, logoPlaceholder string) string {
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")

	for _, ch := range channels {
		sb.WriteString(fmt.Sprintf(`#EXTINF:-1 tvg-id="%s" tvg-name="%s"`, ch.TVGId, ch.TVGName))

		if ch.Logo != "" {
			logoURL := ch.Logo
			if logoPlaceholder != "" {
				logoURL = logoPlaceholder + ch.Logo
			}
			sb.WriteString(fmt.Sprintf(` tvg-logo="%s"`, logoURL))
		}

		if catchupTemplate != "" && ch.CatchupSrc != "" {
			sb.WriteString(fmt.Sprintf(` catchup="default" catchup-source="%s"`, ch.CatchupSrc))
		} else if catchupTemplate != "" {
			sb.WriteString(fmt.Sprintf(` catchup="append" catchup-source="?%s"`, catchupTemplate))
		}

		sb.WriteString(fmt.Sprintf(` group-title="%s",%s`, ch.Group, ch.Name))
		sb.WriteString("\n")
		sb.WriteString(ch.URL)
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatTXT generates DIYP TXT format string from channels
func FormatTXT(channels []Channel) string {
	// Group channels by group name while preserving order
	groupOrder := make([]string, 0)
	grouped := make(map[string][]Channel)

	for _, ch := range channels {
		group := ch.Group
		if group == "" {
			group = "未分组"
		}
		if _, exists := grouped[group]; !exists {
			groupOrder = append(groupOrder, group)
		}
		grouped[group] = append(grouped[group], ch)
	}

	var sb strings.Builder
	for _, group := range groupOrder {
		sb.WriteString(fmt.Sprintf("%s,#genre#\n", group))
		for _, ch := range grouped[group] {
			sb.WriteString(fmt.Sprintf("%s,%s\n", ch.Name, ch.URL))
		}
	}

	return sb.String()
}
