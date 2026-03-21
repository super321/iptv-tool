package model

import (
	"time"
)

// User represents the administrator account
type User struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LiveSourceType represents the type of live source
type LiveSourceType string

const (
	LiveSourceTypeIPTV          LiveSourceType = "iptv"
	LiveSourceTypeNetworkURL    LiveSourceType = "network_url"
	LiveSourceTypeNetworkManual LiveSourceType = "network_manual"
)

// LiveSource represents a source of live TV channels (M3U/TXT/IPTV)
type LiveSource struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	Name           string         `gorm:"not null" json:"name"`
	Description    string         `json:"description"`
	Type           LiveSourceType `gorm:"not null" json:"type"` // iptv, network_url, network_manual
	URL            string         `json:"url"`                  // For network_url
	Content        string         `json:"content"`              // For network_manual
	CronTime       string         `json:"cron_time"`            // 1h, 2h, 4h, 6h, 12h, 24h
	CronDetect     string         `json:"cron_detect"`          // Scheduled detection interval, same options as CronTime
	DetectStrategy string         `json:"detect_strategy"`      // unicast, multicast (for detection URL selection)
	Headers        string         `json:"headers"`              // JSON string for network_url custom headers
	Status         bool           `gorm:"default:true" json:"status"`
	IsSyncing      bool           `gorm:"default:false" json:"is_syncing"`
	IsDetecting    bool           `gorm:"default:false" json:"is_detecting"`
	IPTVConfig     string         `gorm:"column:iptv_config" json:"iptv_config"` // JSON string for IPTV specific configs (platform, credentials)
	LastFetchedAt  *time.Time     `json:"last_fetched_at"`
	LastError      string         `json:"last_error"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// EPGSourceType represents the type of EPG source
type EPGSourceType string

const (
	EPGSourceTypeIPTV         EPGSourceType = "iptv"
	EPGSourceTypeNetworkXMLTV EPGSourceType = "network_xmltv"
)

// EPGSource represents a source of EPG (XMLTV/IPTV)
type EPGSource struct {
	ID            uint          `gorm:"primarykey" json:"id"`
	Name          string        `gorm:"not null" json:"name"`
	Description   string        `json:"description"`
	Type          EPGSourceType `gorm:"not null" json:"type"`        // iptv, network_xmltv
	URL           string        `json:"url"`                         // For network_xmltv
	Headers       string        `json:"headers"`                     // JSON string for network_xmltv custom headers
	LiveSourceID  *uint         `gorm:"index" json:"live_source_id"` // FK to LiveSource (only for IPTV type, auto-created)
	CronTime      string        `json:"cron_time"`
	Status        bool          `gorm:"default:true" json:"status"`
	IsSyncing     bool          `gorm:"default:false" json:"is_syncing"`
	IPTVConfig    string        `gorm:"column:iptv_config" json:"iptv_config"` // JSON string for IPTV specific EPG configs (strategy: auto, vsp, etc.)
	LastFetchedAt *time.Time    `json:"last_fetched_at"`
	LastError     string        `json:"last_error"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// ChannelLogo represents an uploaded channel logo
type ChannelLogo struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"not null" json:"name"` // Usually the original filename without extension
	FilePath  string    `gorm:"not null" json:"-"`    // Local file path
	URLPath   string    `gorm:"not null" json:"url_path"`
	CreatedAt time.Time `json:"created_at"`
}

// PublishFormat represents the format of the published interface
type PublishFormat string

const (
	PublishFormatM3U   PublishFormat = "m3u"
	PublishFormatTXT   PublishFormat = "txt"
	PublishFormatXMLTV PublishFormat = "xmltv"
	PublishFormatDIYP  PublishFormat = "diyp"
)

// PublishInterface represents an aggregated endpoint provided by this system
type PublishInterface struct {
	ID          uint          `gorm:"primarykey" json:"id"`
	Name        string        `gorm:"not null" json:"name"`
	Description string        `json:"description"`
	Path        string        `gorm:"uniqueIndex:idx_type_path;not null" json:"path"` // e.g., my_list
	Type        string        `gorm:"uniqueIndex:idx_type_path;not null" json:"type"` // live or epg
	Format      PublishFormat `gorm:"not null" json:"format"`                         // m3u, txt, xmltv, diyp
	SourceIDs   string        `json:"source_ids"`                                     // Comma-separated IDs of LiveSource or EPGSource
	Status      bool          `gorm:"default:true" json:"status"`

	// EPG specific
	EPGDays     int  `json:"epg_days"`     // Number of days to include
	GzipEnabled bool `json:"gzip_enabled"` // For XMLTV

	// Live specific
	AddressType            string `json:"address_type"`                                            // multicast, unicast
	MulticastType          string `json:"multicast_type"`                                          // udpxy, rtp, igmp
	UDPxyURL               string `gorm:"column:udpxy_url" json:"udpxy_url"`                       // e.g., http://192.168.1.1:4022
	FCCEnabled             bool   `json:"fcc_enabled"`                                             // Enable FCC (Fast Channel Change) for rtp2httpd
	FCCType                string `json:"fcc_type"`                                                // telecom (default) or huawei
	M3UCatchupTemplate     string `gorm:"column:m3u_catchup_template" json:"m3u_catchup_template"` // e.g., playseek=${(b)yyyyMMddHHmmss}-${(e)yyyyMMddHHmmss}
	FilterInvalidSourceIDs string `json:"filter_invalid_source_ids"`                               // Comma-separated source IDs that should filter timeout channels

	TvgIDMode string `gorm:"default:'channel_id'" json:"tvg_id_mode"` // channel_id or name
	RuleIDs   string `json:"rule_ids"`                                // Comma-separated IDs of AggregationRule

	// User-Agent validation
	UACheckEnabled  bool   `json:"ua_check_enabled"`  // Enable User-Agent validation
	UAAllowedValues string `json:"ua_allowed_values"` // Comma-separated allowed UA substrings

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RuleType represents the type of publish rule
type RuleType string

const (
	RuleTypeAlias  RuleType = "alias"
	RuleTypeFilter RuleType = "filter"
	RuleTypeGroup  RuleType = "group"
)

type MatchMode string

const (
	MatchModeRegex  MatchMode = "regex"
	MatchModeString MatchMode = "string"
)

// AggregationRule represents an independent rule that can be reused across interfaces
type AggregationRule struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`          // Added description
	Type        RuleType  `gorm:"not null" json:"type"` // alias, filter, group
	Config      string    `json:"config"`               // JSON string for specific rule configuration
	Status      bool      `gorm:"default:true" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Internal structures for parsed channel and EPG data stored in DB or cache

// ParsedChannel represents a channel parsed from any source
type ParsedChannel struct {
	ID              uint       `gorm:"primarykey" json:"id"`
	SourceID        uint       `gorm:"index" json:"source_id"`
	TVGId           string     `json:"tvg_id"`
	TVGName         string     `json:"tvg_name"`
	Name            string     `gorm:"index" json:"name"`
	Group           string     `json:"group"`
	Logo            string     `json:"logo"`
	URL             string     `json:"url"`
	CatchupURL      string     `json:"catchup_url"`      // Original timeshift/catchup base URL
	CatchupDays     int        `json:"catchup_days"`     // Days available for catchup
	FCCIP           string     `json:"fcc_ip"`           // FCC server IP (from ChannelFCCIP)
	FCCPort         string     `json:"fcc_port"`         // FCC server port (from ChannelFCCPort)
	Latency         *int       `json:"latency"`          // Detection latency in ms: nil=not detected, -1=timeout, >0=normal latency
	DetectedAt      *time.Time `json:"detected_at"`      // Last detection time
	VideoCodec      *string    `json:"video_codec"`      // nil=not detected, e.g. "h264", "hevc"
	VideoResolution *string    `json:"video_resolution"` // nil=not detected, e.g. "1920x1080"
}

// SystemSetting stores key-value system configuration
type SystemSetting struct {
	ID    uint   `gorm:"primarykey" json:"id"`
	Key   string `gorm:"uniqueIndex;not null" json:"key"`
	Value string `gorm:"not null" json:"value"`
}

// ParsedEPG represents a single EPG program
type ParsedEPG struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	SourceID    uint      `gorm:"index:idx_epg_source_channel_start,priority:1" json:"source_id"`
	Channel     string    `gorm:"index:idx_epg_source_channel_start,priority:2" json:"channel"` // Channel ID (XMLTV channel id / IPTV ChannelID)
	ChannelName string    `json:"channel_name"`                                                 // Channel display name
	Title       string    `json:"title"`
	Desc        string    `json:"desc"`
	StartTime   time.Time `gorm:"index:idx_epg_source_channel_start,priority:3" json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// AccessControlEntry represents a whitelist or blacklist entry for access control
type AccessControlEntry struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ListType  string    `gorm:"not null;index" json:"list_type"` // "whitelist" or "blacklist"
	EntryType string    `gorm:"not null" json:"entry_type"`      // "single", "cidr", "range"
	Value     string    `gorm:"not null" json:"value"`           // IP, CIDR, or "start~end"
	BlockDays *int      `json:"block_days"`                      // nil = permanent (blacklist only)
	CreatedAt time.Time `json:"created_at"`
}

// AccessStat tracks IP access statistics over the last 7 days
type AccessStat struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	IP             string    `gorm:"uniqueIndex;not null" json:"ip"`
	LastAccessedAt time.Time `gorm:"not null;index" json:"last_accessed_at"`
	TotalRequests  int64     `gorm:"not null;default:0" json:"total_requests"`
	SubRequests    int64     `gorm:"not null;default:0" json:"sub_requests"` // /sub/ endpoint hits
}
