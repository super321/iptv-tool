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
	RegisterEPGStrategy(&GdhdpublicStrategy{})
}

// GdhdpublicStrategy implements the EPGFetchStrategy for the "gdhdpublic" EPG interface (e.g. Guangdong/Zhejiang)
type GdhdpublicStrategy struct{}

func (s *GdhdpublicStrategy) Name() string {
	return "gdhdpublic"
}

// gdhdpublic JSON response types
type gdhdpublicResult struct {
	Result []gdhdpublicProgram `json:"result"`
}

type gdhdpublicProgram struct {
	Code    string `json:"code"`
	ProID   string `json:"proID"`
	ProFlag string `json:"proflag"`
	Name    string `json:"name"`
	Time    string `json:"time"`
	Endtime string `json:"endtime"`
	Day     string `json:"day"`
}

func (s *GdhdpublicStrategy) Fetch(ctx context.Context, client *iptv.HTTPClient, channels []iptv.Channel, authInfo map[string]string, limiter *iptv.RateLimiter) ([]iptv.ChannelProgramList, error) {
	host := authInfo["Host"]
	jsessionID := authInfo["JSESSIONID"]
	userToken := authInfo["UserToken"]

	// Extract custom headers
	customHeaders := extractCustomHeaders(authInfo)

	// Fetch EPG for 7 days (today + 6 days back)
	const maxDays = 7

	var result []iptv.ChannelProgramList
	for _, channel := range channels {
		if limiter != nil {
			if err := limiter.Acquire(ctx); err != nil {
				return nil, err
			}
		}

		progList, err := s.fetchChannelPrograms(ctx, client, host, jsessionID, userToken, channel, maxDays, customHeaders)

		if limiter != nil {
			limiter.Release()
		}

		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err
			}
			fmt.Printf("Warning: Failed to fetch gdhdpublic EPG for channel %s: %v\n", channel.Name, err)
			continue
		}

		if progList != nil && len(progList.Programs) > 0 {
			result = append(result, *progList)
		}
	}

	return result, nil
}

func (s *GdhdpublicStrategy) fetchChannelPrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID, userToken string, channel iptv.Channel, maxDays int, customHeaders map[string]string) (*iptv.ChannelProgramList, error) {
	var allPrograms []iptv.Program

	// Query from tomorrow backwards
	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrow = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())

	for i := 0; i <= maxDays; i++ {
		date := tomorrow.AddDate(0, 0, -i)
		dateStr := date.Format("20060102")

		programs, err := s.fetchDatePrograms(ctx, client, host, jsessionID, userToken, channel.ID, dateStr, customHeaders)
		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err
			}
			// Skip failed dates but continue
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

func (s *GdhdpublicStrategy) fetchDatePrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID, userToken, channelID, dateStr string, customHeaders map[string]string) ([]iptv.Program, error) {
	reqURL := fmt.Sprintf("http://%s/EPG/jsp/gdhdpublic/Ver.3/common/data.jsp", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	params := req.URL.Query()
	params.Add("Action", "channelProgramList")
	params.Add("channelId", channelID)
	params.Add("date", dateStr)
	req.URL.RawQuery = params.Encode()

	// Apply custom headers
	applyHeaders(req, host, customHeaders)

	// Cookies
	req.AddCookie(&http.Cookie{Name: "JSESSIONID", Value: jsessionID})
	req.AddCookie(&http.Cookie{Name: "telecomToken", Value: userToken})

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

	return parseGdhdpublicPrograms(body)
}

func parseGdhdpublicPrograms(data []byte) ([]iptv.Program, error) {
	var resp gdhdpublicResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if len(resp.Result) == 0 {
		return nil, ErrChProgListIsEmpty
	}

	programs := make([]iptv.Program, 0, len(resp.Result))
	for _, raw := range resp.Result {
		startTime, err := time.ParseInLocation(time.DateTime, raw.Day+" "+raw.Time, time.Local)
		if err != nil {
			continue
		}
		endTime, err := time.ParseInLocation(time.DateTime, raw.Day+" "+raw.Endtime, time.Local)
		if err != nil {
			continue
		}

		programs = append(programs, iptv.Program{
			Title:     raw.Name,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}
	return programs, nil
}

// --- shared helpers used by all strategies ---

// extractCustomHeaders extracts user-configured headers from authInfo (keys prefixed with "Header-")
func extractCustomHeaders(authInfo map[string]string) map[string]string {
	headers := make(map[string]string)
	for k, v := range authInfo {
		if strings.HasPrefix(k, "Header-") {
			headers[strings.TrimPrefix(k, "Header-")] = v
		}
	}
	return headers
}

// applyHeaders applies custom headers and ensures Host + User-Agent defaults
func applyHeaders(req *http.Request, host string, customHeaders map[string]string) {
	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; Fhbw2.0) AppleWebKit")
	}
	req.Header.Set("Host", host)
}
