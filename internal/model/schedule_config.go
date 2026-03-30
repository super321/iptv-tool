package model

import (
	"fmt"
	"strings"
)

// --- Schedule configuration types (shared by task and service packages) ---

const (
	ScheduleModeInterval = "interval"
	ScheduleModeDaily    = "daily"

	MaxTimePoints    = 5
	MinTimeGapMinute = 30
	MinIntervalHours = 1
	MaxIntervalHours = 48
	MinGeoIPDays     = 1
	MaxGeoIPDays     = 30
)

// ScheduleConfig defines a generic scheduled task configuration.
// For live/EPG sources: mode="interval" uses Hours; mode="daily" uses Times (Days defaults to 1).
// For GeoIP auto-update: mode="daily" uses Days + Times.
type ScheduleConfig struct {
	Mode  string   `json:"mode"`            // "interval" or "daily"
	Hours int      `json:"hours,omitempty"` // interval mode: every N hours (1-48)
	Days  int      `json:"days,omitempty"`  // daily mode: every N days (1-30, default 1)
	Times []string `json:"times,omitempty"` // daily mode: time-of-day list ("HH:MM", max 5)
}

// IsEmpty returns true if the config represents "no schedule".
func (c *ScheduleConfig) IsEmpty() bool {
	return c == nil || c.Mode == ""
}

// String returns a human-readable representation.
func (c *ScheduleConfig) String() string {
	if c.IsEmpty() {
		return ""
	}
	switch c.Mode {
	case ScheduleModeInterval:
		return fmt.Sprintf("every %dh", c.Hours)
	case ScheduleModeDaily:
		if c.Days > 1 {
			return fmt.Sprintf("every %dd at %s", c.Days, strings.Join(c.Times, ", "))
		}
		return fmt.Sprintf("daily at %s", strings.Join(c.Times, ", "))
	default:
		return ""
	}
}
