package huawei

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"iptv-tool-v2/internal/iptv"
)

func init() {
	RegisterEPGStrategy(&VspStrategy{})
}

// VspStrategy implements the EPGFetchStrategy for the "vsp" EPG interface (e.g. Hubei)
type VspStrategy struct{}

func (s *VspStrategy) Name() string {
	return "vsp"
}

// VSP request/response types
type vspQueryChannel struct {
	ChannelIDs []int64 `json:"channelIDs"`
}

type vspQueryPlaybill struct {
	Type          string `json:"type"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	Count         string `json:"count"`
	Offset        string `json:"offset"`
	IsFillProgram string `json:"isFillProgram"`
	MustIncluded  string `json:"mustIncluded"`
}

type vspQueryPayload struct {
	QueryChannel  *vspQueryChannel  `json:"queryChannel"`
	QueryPlaybill *vspQueryPlaybill `json:"queryPlaybill"`
	NeedChannel   string            `json:"needChannel"`
}

type vspResponseResult struct {
	RetMsg  string `json:"retMsg"`
	RetCode string `json:"retCode"`
}

type vspResponsePlaybillLite struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Name      string `json:"name"`
	ID        string `json:"ID"`
	ChannelID string `json:"channelID"`
}

type vspResponseChannelPlaybills struct {
	PlaybillCount string                    `json:"playbillCount"`
	PlaybillLites []vspResponsePlaybillLite `json:"playbillLites"`
}

type vspResponse struct {
	Result           *vspResponseResult            `json:"result"`
	Total            string                        `json:"total"`
	ChannelPlaybills []vspResponseChannelPlaybills `json:"channelPlaybills"`
}

func (s *VspStrategy) Fetch(ctx context.Context, client *iptv.HTTPClient, channels []iptv.Channel, authInfo map[string]string, limiter *iptv.RateLimiter) ([]iptv.ChannelProgramList, error) {
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
			fmt.Printf("Warning: Failed to fetch vsp EPG for channel %s: %v\n", channel.Name, err)
			continue
		}

		if progList != nil && len(progList.Programs) > 0 {
			result = append(result, *progList)
		}
	}

	return result, nil
}

func (s *VspStrategy) fetchChannelPrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID string, channel iptv.Channel, maxDays int, customHeaders map[string]string) (*iptv.ChannelProgramList, error) {
	var allPrograms []iptv.Program

	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrow = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())

	for i := 0; i <= maxDays; i++ {
		startDate := tomorrow.AddDate(0, 0, -i)
		endDate := startDate.AddDate(0, 0, 1)

		programs, err := s.fetchDatePrograms(ctx, client, host, jsessionID, channel.ID, startDate.UnixMilli(), endDate.UnixMilli(), 0, customHeaders)
		if err != nil {
			if errors.Is(err, ErrEPGApiNotFound) {
				return nil, err
			}
			continue
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

func (s *VspStrategy) fetchDatePrograms(ctx context.Context, client *iptv.HTTPClient, host, jsessionID, channelID string, startTime, endTime int64, offset int, customHeaders map[string]string) ([]iptv.Program, error) {
	channelIDInt, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return nil, err
	}

	payload := &vspQueryPayload{
		QueryChannel: &vspQueryChannel{
			ChannelIDs: []int64{channelIDInt},
		},
		QueryPlaybill: &vspQueryPlaybill{
			Type:          "0",
			StartTime:     strconv.FormatInt(startTime, 10),
			EndTime:       strconv.FormatInt(endTime, 10),
			Count:         "100",
			Offset:        strconv.Itoa(offset),
			IsFillProgram: "0",
			MustIncluded:  "0",
		},
		NeedChannel: "0",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("http://%s/VSP/V3/QueryPlaybillList", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}

	applyHeaders(req, host, customHeaders)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
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

	var response vspResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}
	if response.Result == nil || response.Result.RetCode != "000000000" || len(response.ChannelPlaybills) == 0 {
		return nil, fmt.Errorf("VSP API returned failure: %+v", response.Result)
	}

	channelPlaybills := response.ChannelPlaybills[0]
	programs, err := parseVspPlaybillLites(channelPlaybills.PlaybillLites)
	if err != nil {
		return nil, err
	}

	// Handle pagination recursively
	count, err := strconv.Atoi(channelPlaybills.PlaybillCount)
	if err != nil {
		return nil, err
	}
	if count > (offset + 100) {
		nextPrograms, err := s.fetchDatePrograms(ctx, client, host, jsessionID, channelID, startTime, endTime, offset+100, customHeaders)
		if err != nil {
			return nil, err
		}
		programs = append(programs, nextPrograms...)
	}

	return programs, nil
}

func parseVspPlaybillLites(lites []vspResponsePlaybillLite) ([]iptv.Program, error) {
	if len(lites) == 0 {
		return nil, ErrChProgListIsEmpty
	}

	programs := make([]iptv.Program, 0, len(lites))
	for _, lite := range lites {
		startTimeMs, err := strconv.ParseInt(lite.StartTime, 10, 64)
		if err != nil {
			continue
		}
		endTimeMs, err := strconv.ParseInt(lite.EndTime, 10, 64)
		if err != nil {
			continue
		}

		programs = append(programs, iptv.Program{
			Title:     lite.Name,
			StartTime: time.UnixMilli(startTimeMs),
			EndTime:   time.UnixMilli(endTimeMs),
		})
	}
	return programs, nil
}
