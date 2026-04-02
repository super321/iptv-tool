package huawei

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"iptv-tool-v2/internal/iptv"
)

func init() {
	RegisterEPGStrategy(&XagqftkdbStrategy{})
}

// XagqftkdbStrategy implements the EPGFetchStrategy for the "xagqftkdb" EPG interface.
// It queries channel program guides starting from tomorrow (current date + 1) and
// iterates backwards day by day. The response Content-Type is text/html but the body
// contains JSON data.
type XagqftkdbStrategy struct{}

func (s *XagqftkdbStrategy) Name() string {
	return "xagqftkdb"
}

// prevueList JSON response types
type prevueListResponse struct {
	TotalSize         int             `json:"totalSize"`
	CurPage           int             `json:"curPage"`
	TotalPage         int             `json:"totalPage"`
	ChannelName       string          `json:"channelName"`
	ChannelPrevueList []prevueProgram `json:"channelPrevueList"`
}

type prevueProgram struct {
	PrevueName  string `json:"prevueName"`
	ContentName string `json:"contentName"`
	StartTime   string `json:"startTime"` // "HH:MM:SS"
	EndTime     string `json:"endTime"`   // "HH:MM:SS"
	CurrDate    string `json:"currDate"`  // "YYYY-MM-DD"
}

func (s *XagqftkdbStrategy) Fetch(ctx context.Context, client *iptv.HTTPClient, channels []iptv.Channel, authInfo map[string]string, limiter *iptv.RateLimiter) ([]iptv.ChannelProgramList, error) {
	host := authInfo["Host"]
	jsessionID := authInfo["JSESSIONID"]
	customHeaders := extractCustomHeaders(authInfo)

	const maxDays = 7

	var result []iptv.ChannelProgramList
	for _, channel := range channels {
		if limiter != nil {
			if err := limiter.Acquire(ctx); err != nil {
				return nil, err
			}
		}

		progList, err := s.fetchChannelPrograms(ctx, client, host, jsessionID, channel, maxDays, customHeaders)

		if limiter != nil {
			limiter.Release()
		}

		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err
			}
			fmt.Printf("Warning: Failed to fetch xagqftkdb EPG for channel %s: %v\n", channel.Name, err)
			continue
		}

		if progList != nil && len(progList.Programs) > 0 {
			result = append(result, *progList)
		}
	}

	return result, nil
}

func (s *XagqftkdbStrategy) fetchChannelPrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID string, channel iptv.Channel, maxDays int, customHeaders map[string]string) (*iptv.ChannelProgramList, error) {
	var allPrograms []iptv.Program

	// Start from tomorrow and go backwards
	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrow = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())

	for i := 0; i <= maxDays; i++ {
		date := tomorrow.AddDate(0, 0, -i)
		dateStr := date.Format("20060102")

		programs, err := s.fetchDatePrograms(ctx, client, host, jsessionID, channel.ID, dateStr, customHeaders)
		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err
			}
			// Skip failed dates but continue with others
			continue
		}
		allPrograms = append(allPrograms, programs...)
	}

	// Sort all programs by start time ascending
	sort.Slice(allPrograms, func(i, j int) bool {
		return allPrograms[i].StartTime.Before(allPrograms[j].StartTime)
	})

	return &iptv.ChannelProgramList{
		Channel:  channel,
		Programs: allPrograms,
	}, nil
}

func (s *XagqftkdbStrategy) fetchDatePrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID, channelID, dateStr string, customHeaders map[string]string) ([]iptv.Program, error) {
	reqURL := fmt.Sprintf("http://%s/EPG/jsp/xagqftkdb/en/GDIndex/hwdatajsp/prevueList.jsp", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	params := req.URL.Query()
	params.Add("channelID", channelID)
	params.Add("curdate", dateStr)
	params.Add("pageIndex", "1")
	params.Add("pageSize", "9999")
	params.Add("isJson", "-1")
	params.Add("isAjax", "1")
	req.URL.RawQuery = params.Encode()

	// Apply custom headers
	applyHeaders(req, host, customHeaders)
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: jsessionID})

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode >= http.StatusInternalServerError {
		return nil, ErrEPGApiNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// The response Content-Type is text/html but the body is JSON.
	// Trim any leading/trailing whitespace that might be present.
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr == "" {
		return nil, ErrChProgListIsEmpty
	}

	return parsePrevueListPrograms([]byte(bodyStr))
}

func parsePrevueListPrograms(data []byte) ([]iptv.Program, error) {
	var resp prevueListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse prevuelist response failed: %w", err)
	}

	if len(resp.ChannelPrevueList) == 0 {
		return nil, ErrChProgListIsEmpty
	}

	programs := make([]iptv.Program, 0, len(resp.ChannelPrevueList))
	loc := time.Local

	for _, prog := range resp.ChannelPrevueList {
		name := prog.PrevueName
		if name == "" {
			name = prog.ContentName
		}
		if name == "" {
			continue
		}

		dateStr := prog.CurrDate // "YYYY-MM-DD"
		if dateStr == "" {
			continue
		}

		startTimeStr := prog.StartTime // "HH:MM:SS"
		endTimeStr := prog.EndTime     // "HH:MM:SS"
		if startTimeStr == "" || endTimeStr == "" {
			continue
		}

		startTime, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr+" "+startTimeStr, loc)
		if err != nil {
			continue
		}
		endTime, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr+" "+endTimeStr, loc)
		if err != nil {
			continue
		}

		// Handle cross-midnight: if endTime <= startTime, the program ends the next day
		if !endTime.After(startTime) {
			endTime = endTime.AddDate(0, 0, 1)
		}

		programs = append(programs, iptv.Program{
			Title:     name,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return programs, nil
}
