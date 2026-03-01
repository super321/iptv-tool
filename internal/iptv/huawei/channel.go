package huawei

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"iptv-tool-v2/internal/iptv"
)

// GetAllChannelList retrieves the list of live channels from Huawei STB API
func (c *Client) GetAllChannelList(ctx context.Context) ([]iptv.Channel, error) {
	if c.Token == nil || c.Token.UserToken == "" || c.Token.JSESSIONID == "" {
		return nil, errors.New("not authenticated, call Authenticate first")
	}

	// MD5 of JSESSIONID is used as tempKey (uppercase hex)
	hash := md5.Sum([]byte(c.Token.JSESSIONID))
	tempKey := strings.ToUpper(hex.EncodeToString(hash[:]))

	data := url.Values{}

	data.Set("conntype", c.config.GetAuthParam("conntype"))
	data.Set("UserToken", c.Token.UserToken)
	data.Set("tempKey", tempKey)
	data.Set("stbid", c.Token.Stbid)
	data.Set("SupportHD", "1")
	data.Set("UserID", c.config.GetAuthParam("UserID"))
	data.Set("Lang", c.config.GetAuthParam("Lang"))

	path := fmt.Sprintf("http://%s/EPG/jsp/getchannellistHW%s.jsp", c.host, c.config.ProviderSuffix)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	c.setCommonHeaders(req)
	req.Header.Set("Referer", fmt.Sprintf("http://%s/EPG/jsp/ValidAuthenticationHW%s.jsp", c.host, c.config.ProviderSuffix))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID",
		Value: c.Token.JSESSIONID,
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Regex to extract channel information
	// ChannelID="xxx",ChannelName="xxx",UserChannelID="xxx",ChannelURL="xxx",TimeShift="xxx",TimeShiftLength="xxx",TimeShiftURL="xxx"
	chRegex := regexp.MustCompile(`ChannelID="(.+?)",ChannelName="(.+?)",UserChannelID="(.+?)",ChannelURL="(.+?)",TimeShift="(.+?)",TimeShiftLength="(\d+?)".+?,TimeShiftURL="(.+?)"`)
	matchesList := chRegex.FindAllSubmatch(result, -1)
	if matchesList == nil {
		return nil, errors.New("failed to extract channel list from response")
	}

	var channels []iptv.Channel
	for _, matches := range matchesList {
		if len(matches) != 8 {
			continue
		}

		channelID := string(matches[1])
		channelName := string(matches[2])
		channelURLsRaw := string(matches[4])
		timeShift := string(matches[5])
		timeShiftLengthStr := string(matches[6])
		catchupURL := string(matches[7]) // TimeShiftURL

		// 处理回看天数
		var catchupDays int
		if timeShift == "1" && timeShiftLengthStr != "" {
			// 华为接口自带的 TimeShiftLength 通常是秒。换算成天数。
			var length int
			if _, err := fmt.Sscanf(timeShiftLengthStr, "%d", &length); err == nil {
				catchupDays = length / 86400
			}
		}

		// 不支持时移的清理干净
		if timeShift != "1" {
			catchupURL = ""
			catchupDays = 0
		}

		// The raw URL might contain multiple addresses (multicast and unicast) separated by '|'
		// e.g. igmp://239.93.1.23:5140|http://113.136.x.x/xxxx
		// We join them with a comma or keep the raw string depending on design.
		// For simplicity, we keep the full raw string and let the Aggregation Engine split and pick.
		channels = append(channels, iptv.Channel{
			ID:          channelID,
			Name:        channelName,
			URL:         channelURLsRaw,
			OriginalURL: channelURLsRaw, // 核心改动：把带有 igmp:// 等未拆分处理的内容原封不动传给下游，用来做组播判断
			CatchupURL:  catchupURL,
			CatchupDays: catchupDays, // 核心改动：时移天数通过提取运算赋值
		})
	}

	return channels, nil
}
