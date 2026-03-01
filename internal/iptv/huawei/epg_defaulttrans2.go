package huawei

import (
	"context"
	"encoding/json"
	"errors"
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
	RegisterEPGStrategy(&Defaulttrans2Strategy{})
}

// Defaulttrans2Strategy implements the EPGFetchStrategy for the "defaulttrans2" EPG interface (e.g. Shandong)
type Defaulttrans2Strategy struct{}

func (s *Defaulttrans2Strategy) Name() string {
	return "defaulttrans2"
}

// defaulttrans2 JSON types
type defaulttrans2Response struct {
	Data  []defaulttrans2Prog `json:"data"`
	Title []string            `json:"title"`
}

type defaulttrans2Prog struct {
	ProgName    string `json:"progName"`
	ScrollFlag  int    `json:"scrollFlag"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	SubProgName string `json:"subProgName"`
	State       string `json:"state"`
	ProgId      string `json:"progId"`
}

func (s *Defaulttrans2Strategy) Fetch(ctx context.Context, client *iptv.HTTPClient, channels []iptv.Channel, authInfo map[string]string, limiter *iptv.RateLimiter) ([]iptv.ChannelProgramList, error) {
	host := authInfo["Host"]
	jsessionID := authInfo["JSESSIONID"]
	customHeaders := extractCustomHeaders(authInfo)

	var result []iptv.ChannelProgramList
	for _, channel := range channels {
		if limiter != nil {
			if err := limiter.Acquire(ctx); err != nil {
				return nil, err
			}
		}

		progList, err := s.fetchChannelPrograms(ctx, client, host, jsessionID, channel, customHeaders)

		if limiter != nil {
			limiter.Release()
		}

		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err
			}
			fmt.Printf("Warning: Failed to fetch defaulttrans2 EPG for channel %s: %v\n", channel.Name, err)
			continue
		}

		if progList != nil && len(progList.Programs) > 0 {
			result = append(result, *progList)
		}
	}

	return result, nil
}

func (s *Defaulttrans2Strategy) fetchChannelPrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID string, channel iptv.Channel, customHeaders map[string]string) (*iptv.ChannelProgramList, error) {
	var allPrograms []iptv.Program

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// On first request, we discover the actual number of available date slots from the response title array
	dateSize := 7

	for i := 0; i < dateSize; i++ {
		date := today.AddDate(0, 0, -i)

		programs, actualDateSize, err := s.fetchDatePrograms(ctx, client, host, jsessionID, channel, date, -i, customHeaders)
		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err
			}
			continue
		}

		// Update dateSize from first successful response
		if i == 0 && actualDateSize > 0 {
			dateSize = actualDateSize
		}

		allPrograms = append(allPrograms, programs...)
	}

	sort.Slice(allPrograms, func(i, j int) bool {
		return allPrograms[i].StartTime.Before(allPrograms[j].StartTime)
	})

	return &iptv.ChannelProgramList{
		Channel:  channel,
		Programs: allPrograms,
	}, nil
}

func (s *Defaulttrans2Strategy) fetchDatePrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID string, channel iptv.Channel, date time.Time, index int, customHeaders map[string]string) ([]iptv.Program, int, error) {
	reqURL := fmt.Sprintf("http://%s/EPG/jsp/defaulttrans2/en/datajsp/getTvodProgListByIndex.jsp", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}

	params := req.URL.Query()
	params.Add("CHANNELID", channel.ID)
	params.Add("index", strconv.Itoa(index))
	req.URL.RawQuery = params.Encode()

	applyHeaders(req, host, customHeaders)
	req.Header.Set("Referer", fmt.Sprintf("http://%s/EPG/jsp/defaulttrans2/en/chanMiniList.html", host))

	// defaulttrans2 requires specific cookies
	cookies := []*http.Cookie{
		{Name: "maidianFlag", Value: "1"},
		{Name: "navNameFocus", Value: "3"},
		{Name: "jumpTime", Value: "0"},
		{Name: "channelTip", Value: "1"},
		{Name: "lastChanNum", Value: "1"},
		{Name: "STARV_TIMESHFTCID", Value: channel.ID},
		{Name: "STARV_TIMESHFTCNAME", Value: url.QueryEscape(channel.Name)},
		{Name: "JSESSIONID", Value: jsessionID},
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode >= http.StatusInternalServerError {
		return nil, 0, ErrEPGApiNotFound
	} else if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("http status code: %d", resp.StatusCode)
	}

	var response defaulttrans2Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, 0, fmt.Errorf("parse response failed: %w", err)
	}

	return parseDefaulttrans2Programs(response, date, index)
}

func parseDefaulttrans2Programs(response defaulttrans2Response, date time.Time, index int) ([]iptv.Program, int, error) {
	if len(response.Data) == 0 {
		return nil, 0, ErrChProgListIsEmpty
	}
	if len(response.Title) == 0 {
		return nil, 0, fmt.Errorf("no date title list")
	}

	// Validate date position
	datePos := len(response.Title) - 1 + index
	if datePos >= len(response.Title) || datePos < 0 {
		return nil, 0, fmt.Errorf("invalid date position: %d", datePos)
	}
	if !strings.HasPrefix(response.Title[datePos], date.Format("02")) {
		return nil, 0, fmt.Errorf("program date does not match query date")
	}

	dateStr := date.Format("20060102")
	programs := make([]iptv.Program, 0, len(response.Data))

	for i, prog := range response.Data {
		startTimeStr := prog.StartTime
		if i == 0 {
			// First program of the day starts at midnight
			startTimeStr = "00:00"
		} else if len(startTimeStr) > 5 {
			startTimeStr = startTimeStr[:5]
		}
		endTimeStr := prog.EndTime
		if len(endTimeStr) > 5 {
			endTimeStr = endTimeStr[:5]
		}

		startTime, err := time.ParseInLocation("20060102 15:04", dateStr+" "+startTimeStr, time.Local)
		if err != nil {
			continue
		}
		endTime, err := time.ParseInLocation("20060102 15:04", dateStr+" "+endTimeStr, time.Local)
		if err != nil {
			continue
		}

		// Handle cross-midnight: if start > end, the program ends the next day
		if startTime.After(endTime) {
			nextDay := date.AddDate(0, 0, 1)
			endTime = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, nextDay.Location())
		}

		programs = append(programs, iptv.Program{
			Title:     prog.ProgName,
			StartTime: startTime,
			EndTime:   endTime,
		})

		// Stop after cross-midnight program
		if startTime.After(endTime) || endTimeStr == "23:59" {
			break
		}
	}

	return programs, len(response.Title), nil
}
