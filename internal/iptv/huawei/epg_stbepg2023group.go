package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"iptv-tool-v2/internal/iptv"
)

func init() {
	RegisterEPGStrategy(&StbEpg2023GroupStrategy{})
}

// StbEpg2023GroupStrategy implements the EPGFetchStrategy for the "stbepg2023group" EPG interface (e.g. Fujian)
type StbEpg2023GroupStrategy struct{}

func (s *StbEpg2023GroupStrategy) Name() string {
	return "stbepg2023group"
}

// stbepg2023group JSON types
type stbEpg2023GroupResponse[T any] struct {
	Data    T      `json:"data"`
	ErrCode string `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
	Status  string `json:"status"`
}

type stbEpg2023GroupCategory struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type stbEpg2023GroupChannel struct {
	AuthCode string `json:"authCode"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsCharge string `json:"isCharge"`
	ID       string `json:"ID"`
	MixNo    string `json:"mixNo"`
	MediaID  string `json:"mediaID"`
}

type stbEpg2023GroupChannelProg struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	ID        string `json:"ID"`
	EndTime   int64  `json:"endTime"`
	ChannelID string `json:"channelID"`
	Status    string `json:"status"`
}

func (s *StbEpg2023GroupStrategy) Fetch(ctx context.Context, client *iptv.HTTPClient, channels []iptv.Channel, authInfo map[string]string, limiter *iptv.RateLimiter) ([]iptv.ChannelProgramList, error) {
	host := authInfo["Host"]
	jsessionID := authInfo["JSESSIONID"]
	customHeaders := extractCustomHeaders(authInfo)

	// Step 1: Get the "全部" category ID
	categoryID, err := s.getChannelCategoryID(ctx, client, host, jsessionID, "全部", customHeaders)
	if err != nil {
		return nil, err
	}

	// Step 2: Get the channel list with codes
	stbChannels, err := s.getChannelList(ctx, client, host, jsessionID, categoryID, customHeaders)
	if err != nil {
		return nil, err
	}

	// Build channel ID -> code mapping
	chIDCodeMap := make(map[string]string, len(stbChannels))
	for _, ch := range stbChannels {
		chIDCodeMap[ch.ID] = ch.Code
	}

	// Step 3: Fetch EPG for each channel
	const maxDays = 7
	var result []iptv.ChannelProgramList

	for _, channel := range channels {
		chCode, ok := chIDCodeMap[channel.ID]
		if !ok {
			continue
		}

		if limiter != nil {
			if err := limiter.Acquire(ctx); err != nil {
				return nil, err
			}
		}

		progList, err := s.fetchChannelPrograms(ctx, client, host, jsessionID, channel, chCode, maxDays, customHeaders)

		if limiter != nil {
			limiter.Release()
		}

		if err != nil {
			fmt.Printf("Warning: Failed to fetch stbepg2023group EPG for channel %s: %v\n", channel.Name, err)
			continue
		}

		if progList != nil && len(progList.Programs) > 0 {
			result = append(result, *progList)
		}
	}

	return result, nil
}

func (s *StbEpg2023GroupStrategy) getChannelCategoryID(ctx context.Context, client *iptv.HTTPClient, host, jsessionID, categoryName string, customHeaders map[string]string) (string, error) {
	data := url.Values{}
	data.Set("action", "getChannelCate")

	reqURL := fmt.Sprintf("http://%s/EPG/jsp/StbEpg2023Group/en/function/ajax/epg7getProperties.jsp", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	applyHeaders(req, host, customHeaders)
	req.Header.Set("VIS-AJAX", "AjaxHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: jsessionID})

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode >= http.StatusInternalServerError {
		return "", ErrEPGApiNotFound
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	var response stbEpg2023GroupResponse[[]stbEpg2023GroupCategory]
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("parse response failed: %w", err)
	}
	if response.Status != "1" {
		return "", fmt.Errorf("API returned failure, errMsg: %s", response.ErrMsg)
	}
	if len(response.Data) == 0 {
		return "", fmt.Errorf("no channel categories returned")
	}

	for _, cat := range response.Data {
		if cat.Name == categoryName {
			return cat.ID, nil
		}
	}
	return "", fmt.Errorf("channel category '%s' not found", categoryName)
}

func (s *StbEpg2023GroupStrategy) getChannelList(ctx context.Context, client *iptv.HTTPClient, host, jsessionID, categoryID string, customHeaders map[string]string) ([]stbEpg2023GroupChannel, error) {
	data := url.Values{}
	data.Set("action", "getChannelList")
	data.Set("cateID", categoryID)
	data.Set("type", "")

	reqURL := fmt.Sprintf("http://%s/EPG/jsp/StbEpg2023Group/en/function/ajax/epg7getChannelByAjax.jsp", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	applyHeaders(req, host, customHeaders)
	req.Header.Set("VIS-AJAX", "AjaxHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: jsessionID})

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	var response stbEpg2023GroupResponse[[]stbEpg2023GroupChannel]
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}
	if response.Status != "1" {
		return nil, fmt.Errorf("API returned failure, errMsg: %s", response.ErrMsg)
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no channels returned")
	}

	return response.Data, nil
}

func (s *StbEpg2023GroupStrategy) fetchChannelPrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID string, channel iptv.Channel, chCode string, maxDays int, customHeaders map[string]string) (*iptv.ChannelProgramList, error) {
	// Calculate time range
	tomorrow := time.Now().AddDate(0, 0, 1)
	old := tomorrow.AddDate(0, 0, -maxDays)
	startTime := time.Date(old.Year(), old.Month(), old.Day(), 0, 0, 1, 0, tomorrow.Location()).UnixMilli()
	endTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, tomorrow.Location()).UnixMilli()

	data := url.Values{}
	data.Set("action", "getChannelProg")
	data.Set("code", chCode)
	data.Set("channelID", channel.ID)
	data.Set("endTime", strconv.FormatInt(endTime, 10))
	data.Set("startTime", strconv.FormatInt(startTime, 10))
	data.Set("offset", "0")
	data.Set("limit", "2000")

	reqURL := fmt.Sprintf("http://%s/EPG/jsp/StbEpg2023Group/en/function/ajax/epg7getChannelByAjax.jsp", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	applyHeaders(req, host, customHeaders)
	req.Header.Set("VIS-AJAX", "AjaxHttpRequest")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: jsessionID})

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	var response stbEpg2023GroupResponse[[]stbEpg2023GroupChannelProg]
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}
	if response.Status != "1" {
		return nil, fmt.Errorf("API returned failure, errMsg: %s", response.ErrMsg)
	}

	programs := parseStbEpg2023GroupPrograms(response.Data)

	// Sort by StartTime ascending
	sort.Slice(programs, func(i, j int) bool {
		return programs[i].StartTime.Before(programs[j].StartTime)
	})

	return &iptv.ChannelProgramList{
		Channel:  channel,
		Programs: programs,
	}, nil
}

func parseStbEpg2023GroupPrograms(progList []stbEpg2023GroupChannelProg) []iptv.Program {
	programs := make([]iptv.Program, 0, len(progList))
	for _, p := range progList {
		programs = append(programs, iptv.Program{
			Title:     p.Name,
			StartTime: time.UnixMilli(p.StartTime),
			EndTime:   time.UnixMilli(p.EndTime),
		})
	}
	return programs
}
