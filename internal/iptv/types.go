package iptv

import (
	"fmt"
	"time"
)

// Config represents the IPTV configuration stored as JSON in LiveSource.
// It includes all necessary parameters to emulate various STBs (Huawei, ZTE, etc.).
// JSON tags must match the old project's camelCase convention exactly.
type Config struct {
	Platform       string `json:"platform"`       // huawei, zte, etc.
	ProviderSuffix string `json:"providerSuffix"` // e.g., CTC (Telecom), CU (Unicom)
	InterfaceName  string `json:"interfaceName"`  // Bind to a specific network interface (e.g., eth1.50)
	IP             string `json:"ip"`             // Override IP if needed
	ServerHost     string `json:"serverHost"`     // IPTV server host:port (e.g., 182.138.3.142:8082)

	Password    string            `json:"password,omitempty"`          // IPTV password
	EPGStrategy string            `json:"channelProgramAPI,omitempty"` // auto, liveplay_30, vsp, gdhdpublic, etc.
	Key         string            `json:"key"`                         // 3DES Key (manually provided or cracked)
	Headers     map[string]string `json:"headers"`                     // Custom headers for requests

	AuthParams map[string]interface{} `json:"authParams"`
}

func (c *Config) GetAuthParam(key string) string {
	if c.AuthParams != nil {
		if val, ok := c.AuthParams[key]; ok {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// DefaultConfig returns a configuration with the default required headers and basic fields.
func DefaultConfig() Config {
	return Config{
		Platform:       "huawei",
		ProviderSuffix: "CTC",
		Headers: map[string]string{
			"Accept":           "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"User-Agent":       "Mozilla/5.0 (X11; Linux x86_64; Fhbw2.0) AppleWebKit",
			"Accept-Language":  "zh-CN,en-US;q=0.8",
			"X-Requested-With": "com.fiberhome.iptv",
		},
	}
}

// Channel represents a single parsed channel from IPTV
type Channel struct {
	ID          string
	Name        string
	URL         string
	OriginalURL string
	CatchupURL  string
	CatchupDays int
}

// Program represents a single TV program in EPG
type Program struct {
	Title     string
	Desc      string
	StartTime time.Time
	EndTime   time.Time
}

// ChannelProgramList contains all programs for a specific channel
type ChannelProgramList struct {
	Channel  Channel
	Programs []Program
}
