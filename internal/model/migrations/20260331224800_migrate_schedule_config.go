package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up20260331224800, down20260331224800)
}

// Schedule config constants and types duplicated here to avoid circular imports
// with the model package. Keep in sync with model.ScheduleConfig.
const (
	scheduleModeInterval = "interval"
	scheduleModeDaily    = "daily"
	minIntervalHours     = 1
	maxIntervalHours     = 48
	maxGeoIPDays         = 30
)

type scheduleConfig struct {
	Mode  string   `json:"mode"`
	Hours int      `json:"hours,omitempty"`
	Days  int      `json:"days,omitempty"`
	Times []string `json:"times,omitempty"`
}

// migrateOldInterval converts an old-format interval string (e.g. "6h") to a
// ScheduleConfig JSON string. Returns the original string if already in JSON
// format, or "" if empty/invalid.
func migrateOldInterval(oldInterval string) string {
	oldInterval = strings.TrimSpace(oldInterval)
	if oldInterval == "" {
		return ""
	}
	// Already a JSON object? Don't migrate.
	if strings.HasPrefix(oldInterval, "{") {
		return oldInterval
	}
	// Parse old format like "1h", "2h", "4h", "6h", "12h", "24h"
	dur, err := time.ParseDuration(oldInterval)
	if err != nil {
		slog.Warn("Cannot migrate old interval, clearing", "value", oldInterval)
		return ""
	}
	hours := int(dur.Hours())
	if hours < minIntervalHours {
		hours = minIntervalHours
	}
	if hours > maxIntervalHours {
		hours = maxIntervalHours
	}
	cfg := scheduleConfig{Mode: scheduleModeInterval, Hours: hours}
	data, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	return string(data)
}

// up20260331224800 migrates old-format interval strings in cron_time/cron_detect
// fields to the new JSON ScheduleConfig format, and converts GeoIP auto-update
// settings from the old intervalDays key to a ScheduleConfig.
func up20260331224800(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Running schedule config migration...")
	migrated := 0

	// --- Migrate LiveSource cron_time and cron_detect ---
	rows, err := tx.QueryContext(ctx, "SELECT id, cron_time, cron_detect FROM live_sources")
	if err != nil {
		return fmt.Errorf("query live_sources: %w", err)
	}
	defer rows.Close()

	type liveUpdate struct {
		id         uint
		cronTime   string
		cronDetect string
	}
	var liveUpdates []liveUpdate

	for rows.Next() {
		var id uint
		var cronTime, cronDetect string
		if err := rows.Scan(&id, &cronTime, &cronDetect); err != nil {
			return fmt.Errorf("scan live_sources: %w", err)
		}
		newCronTime := migrateOldInterval(cronTime)
		newCronDetect := migrateOldInterval(cronDetect)
		if newCronTime != cronTime || newCronDetect != cronDetect {
			liveUpdates = append(liveUpdates, liveUpdate{id, newCronTime, newCronDetect})
		}
	}
	rows.Close()

	for _, u := range liveUpdates {
		if _, err := tx.ExecContext(ctx, "UPDATE live_sources SET cron_time = ?, cron_detect = ? WHERE id = ?",
			u.cronTime, u.cronDetect, u.id); err != nil {
			return fmt.Errorf("update live_sources id=%d: %w", u.id, err)
		}
		migrated++
	}

	// --- Migrate EPGSource cron_time ---
	rows2, err := tx.QueryContext(ctx, "SELECT id, cron_time FROM epg_sources")
	if err != nil {
		return fmt.Errorf("query epg_sources: %w", err)
	}
	defer rows2.Close()

	type epgUpdate struct {
		id       uint
		cronTime string
	}
	var epgUpdates []epgUpdate

	for rows2.Next() {
		var id uint
		var cronTime string
		if err := rows2.Scan(&id, &cronTime); err != nil {
			return fmt.Errorf("scan epg_sources: %w", err)
		}
		newCronTime := migrateOldInterval(cronTime)
		if newCronTime != cronTime {
			epgUpdates = append(epgUpdates, epgUpdate{id, newCronTime})
		}
	}
	rows2.Close()

	for _, u := range epgUpdates {
		if _, err := tx.ExecContext(ctx, "UPDATE epg_sources SET cron_time = ? WHERE id = ?",
			u.cronTime, u.id); err != nil {
			return fmt.Errorf("update epg_sources id=%d: %w", u.id, err)
		}
		migrated++
	}

	// --- Migrate GeoIP auto-update settings ---
	var oldValue string
	err = tx.QueryRowContext(ctx, "SELECT value FROM system_settings WHERE key = ?", "geoip_update_interval_days").Scan(&oldValue)
	if err == nil {
		// Parse old days value
		days := 0
		for _, ch := range oldValue {
			if ch >= '0' && ch <= '9' {
				days = days*10 + int(ch-'0')
			}
		}
		if days < 1 {
			days = 1
		}
		if days > maxGeoIPDays {
			days = maxGeoIPDays
		}

		cfg := scheduleConfig{Mode: scheduleModeDaily, Days: days}
		cfgJSON, _ := json.Marshal(cfg)

		// Upsert the new setting
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM system_settings WHERE key = ?", "geoip_schedule_config").Scan(&exists); err != nil {
			return fmt.Errorf("check geoip_schedule_config: %w", err)
		}
		if exists > 0 {
			if _, err := tx.ExecContext(ctx, "UPDATE system_settings SET value = ? WHERE key = ?",
				string(cfgJSON), "geoip_schedule_config"); err != nil {
				return fmt.Errorf("update geoip_schedule_config: %w", err)
			}
		} else {
			if _, err := tx.ExecContext(ctx, "INSERT INTO system_settings (key, value) VALUES (?, ?)",
				"geoip_schedule_config", string(cfgJSON)); err != nil {
				return fmt.Errorf("insert geoip_schedule_config: %w", err)
			}
		}

		// Remove old key
		if _, err := tx.ExecContext(ctx, "DELETE FROM system_settings WHERE key = ?", "geoip_update_interval_days"); err != nil {
			return fmt.Errorf("delete old geoip key: %w", err)
		}
		migrated++
	}

	slog.Info("Schedule config migration completed", "migrated_records", migrated)
	return nil
}

// down20260331224800 is a no-op because converting back to old format would risk data loss.
func down20260331224800(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Rollback of schedule config migration is a no-op")
	return nil
}
