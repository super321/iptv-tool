package huawei

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"time"

	"iptv-tool-v2/internal/iptv"
)

func init() {
	RegisterEPGStrategy(&LiveplayStrategy{})
}

// LiveplayStrategy implements the EPGFetchStrategy for the "liveplay_30" EPG interface
type LiveplayStrategy struct{}

func (s *LiveplayStrategy) Name() string {
	return "liveplay_30"
}

func (s *LiveplayStrategy) Fetch(ctx context.Context, client *iptv.HTTPClient, channels []iptv.Channel, authInfo map[string]string, limiter *iptv.RateLimiter) ([]iptv.ChannelProgramList, error) {
	var result []iptv.ChannelProgramList
	host := authInfo["Host"]
	jsessionID := authInfo["JSESSIONID"]
	customHeaders := extractCustomHeaders(authInfo)

	for _, channel := range channels {
		// Acquire rate limiter slot to avoid 503 from server
		if limiter != nil {
			if err := limiter.Acquire(ctx); err != nil {
				return nil, err
			}
		}

		progList, err := s.fetchSingleChannel(ctx, client, host, jsessionID, channel, customHeaders)

		if limiter != nil {
			limiter.Release()
		}

		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err // Fast fail to allow auto-detect to try the next strategy
			}
			// Log error and continue with next channel
			fmt.Printf("Warning: Failed to fetch liveplay EPG for channel %s: %v\n", channel.Name, err)
			continue
		}

		if progList != nil && len(progList.Programs) > 0 {
			result = append(result, *progList)
		}
	}

	return result, nil
}

func (s *LiveplayStrategy) fetchSingleChannel(ctx context.Context, client *iptv.HTTPClient, host, jsessionID string, channel iptv.Channel, customHeaders map[string]string) (*iptv.ChannelProgramList, error) {
	reqURL := fmt.Sprintf("http://%s/EPG/jsp/liveplay_30/en/getTvodData.jsp", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	params := req.URL.Query()
	params.Add("channelId", channel.ID)
	req.URL.RawQuery = params.Encode()

	// Apply user-configured custom headers with defaults
	applyHeaders(req, host, customHeaders)
	req.AddCookie(&http.Cookie{
		Name:  "JSESSIONID",
		Value: jsessionID,
	})

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

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract JSON using regex
	regex := regexp.MustCompile(`parent\.jsonBackLookStr\s*=\s*(.+?);`)
	matches := regex.FindSubmatch(result)
	if len(matches) != 2 {
		return nil, ErrParseChProgList
	}

	programs, err := s.parsePrograms(matches[1])
	if err != nil {
		return nil, err
	}

	if len(programs) == 0 {
		return nil, ErrChProgListIsEmpty
	}

	return &iptv.ChannelProgramList{
		Channel:  channel,
		Programs: programs,
	}, nil
}

func (s *LiveplayStrategy) parsePrograms(rawData []byte) ([]iptv.Program, error) {
	var rawArray []any
	if err := json.Unmarshal(rawData, &rawArray); err != nil {
		return nil, err
	}

	if len(rawArray) != 2 {
		return nil, ErrParseChProgList
	}

	dateProgList, ok := rawArray[1].([]any)
	if !ok || len(dateProgList) == 0 {
		return nil, ErrParseChProgList
	}

	var programs []iptv.Program
	loc := time.Local // Adjust to server timezone if needed

	for _, rawProgList := range dateProgList {
		progList, ok := rawProgList.([]any)
		if !ok || len(progList) == 0 {
			continue
		}

		for _, rawProg := range progList {
			prog, ok := rawProg.(map[string]any)
			if !ok || len(prog) == 0 {
				continue
			}

			programName, _ := prog["programName"].(string)
			beginTimeFormatStr, _ := prog["beginTimeFormat"].(string)
			endTimeFormatStr, _ := prog["endTimeFormat"].(string)

			// Handle IPTV bug where end time at 00:00 format is incorrectly set to today's 00:00 instead of tomorrow's
			if len(beginTimeFormatStr) >= 8 && len(endTimeFormatStr) >= 14 {
				if prog["endTime"] == "00:00" && (beginTimeFormatStr[:8]+"000000") == endTimeFormatStr {
					et, err := time.ParseInLocation("20060102150405", endTimeFormatStr, loc)
					if err == nil {
						et = et.Add(24 * time.Hour)
						endTimeFormatStr = et.Format("20060102150405")
					}
				}
			}

			startTime, err := time.ParseInLocation("20060102150405", beginTimeFormatStr, loc)
			if err != nil {
				continue
			}
			endTime, err := time.ParseInLocation("20060102150405", endTimeFormatStr, loc)
			if err != nil {
				continue
			}

			programs = append(programs, iptv.Program{
				Title:     programName,
				StartTime: startTime,
				EndTime:   endTime,
			})
		}
	}

	// Sort programs by StartTime ascending
	sort.Slice(programs, func(i, j int) bool {
		return programs[i].StartTime.Before(programs[j].StartTime)
	})

	return programs, nil
}
