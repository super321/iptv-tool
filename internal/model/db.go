package model

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(dsn string) error {
	// Append SQLite performance tuning pragmas to the DSN:
	//   journal_mode=WAL   - allows concurrent reads during writes
	//   synchronous=NORMAL - reduces fsync calls (safe with WAL)
	//   busy_timeout=5000  - wait up to 5s when DB is locked instead of failing immediately
	dsn += "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)"

	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	// SQLite is a single-file database — limiting the pool avoids lock contention
	// and reduces memory usage on resource-constrained devices (routers / soft-routers).
	// WAL mode already allows concurrent reads at the SQLite engine level.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0) // Don't expire the single connection

	slog.Info("Database opened with WAL mode, synchronous=NORMAL, busy_timeout=5000")

	// Auto-migrate the schema
	err = DB.AutoMigrate(
		&User{},
		&LiveSource{},
		&EPGSource{},
		&ChannelLogo{},
		&PublishInterface{},
		&AggregationRule{},
		&ParsedChannel{},
		&ParsedEPG{},
		&SystemSetting{},
		&AccessControlEntry{},
		&AccessStat{},
	)
	if err != nil {
		return err
	}

	// Drop legacy individual indexes on parsed_epgs that are now covered by the composite index.
	// These existed before the composite index idx_epg_source_channel_start was introduced.
	for _, idx := range []string{"idx_parsed_epgs_source_id", "idx_parsed_epgs_channel", "idx_parsed_epgs_start_time", "idx_parsed_epgs_end_time"} {
		sqlDB.Exec("DROP INDEX IF EXISTS " + idx)
	}

	// Defensive reset: set is_syncing and is_detecting to false for all sources on startup
	DB.Model(&LiveSource{}).Where("is_syncing = ?", true).Update("is_syncing", false)
	DB.Model(&LiveSource{}).Where("is_detecting = ?", true).Update("is_detecting", false)
	DB.Model(&EPGSource{}).Where("is_syncing = ?", true).Update("is_syncing", false)

	// Migrate schedule config from old interval strings to new JSON format
	migrateScheduleConfig()

	slog.Info("Database initialized and migrated successfully.")
	return nil
}

// migrateScheduleConfig converts old-format interval strings (e.g. "6h") in cron_time/cron_detect
// fields to the new JSON format (e.g. {"mode":"interval","hours":6}).
// Also migrates GeoIP auto-update settings from intervalDays to ScheduleConfig.
// Uses a migration marker to ensure it only runs once.
func migrateScheduleConfig() {
	const migrationKey = "migration_schedule_v2"

	// Check if migration has already been performed
	var marker SystemSetting
	if err := DB.Where("key = ?", migrationKey).First(&marker).Error; err == nil {
		return // Already migrated
	}

	slog.Info("Running schedule config migration (v2)...")
	migrated := 0

	// Migrate LiveSource cron_time and cron_detect
	var liveSources []LiveSource
	DB.Find(&liveSources)
	for _, src := range liveSources {
		updates := map[string]interface{}{}
		if newVal := MigrateOldInterval(src.CronTime); newVal != src.CronTime {
			updates["cron_time"] = newVal
		}
		if newVal := MigrateOldInterval(src.CronDetect); newVal != src.CronDetect {
			updates["cron_detect"] = newVal
		}
		if len(updates) > 0 {
			DB.Model(&src).Updates(updates)
			migrated++
		}
	}

	// Migrate EPGSource cron_time
	var epgSources []EPGSource
	DB.Find(&epgSources)
	for _, src := range epgSources {
		if newVal := MigrateOldInterval(src.CronTime); newVal != src.CronTime {
			DB.Model(&src).Update("cron_time", newVal)
			migrated++
		}
	}

	// Migrate GeoIP auto-update settings from old key to new format
	var oldDaysSetting SystemSetting
	if err := DB.Where("key = ?", "geoip_update_interval_days").First(&oldDaysSetting).Error; err == nil {
		days := 0
		for _, ch := range oldDaysSetting.Value {
			if ch >= '0' && ch <= '9' {
				days = days*10 + int(ch-'0')
			}
		}
		if days < 1 {
			days = 1
		}
		if days > MaxGeoIPDays {
			days = MaxGeoIPDays
		}

		cfg := ScheduleConfig{Mode: ScheduleModeDaily, Days: days}
		cfgJSON, _ := json.Marshal(cfg)
		// Upsert the new setting
		DB.Where("key = ?", "geoip_schedule_config").Assign(SystemSetting{Value: string(cfgJSON)}).FirstOrCreate(&SystemSetting{Key: "geoip_schedule_config"})
		// Remove old key
		DB.Where("key = ?", "geoip_update_interval_days").Delete(&SystemSetting{})
		migrated++
	}

	// Mark migration as complete
	DB.Create(&SystemSetting{Key: migrationKey, Value: "done"})
	slog.Info("Schedule config migration completed", "migrated_records", migrated)
}

// MigrateOldInterval converts an old-format interval string (e.g. "6h") to a ScheduleConfig JSON string.
// Returns the original string if already in JSON format, or "" if empty/invalid.
func MigrateOldInterval(oldInterval string) string {
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
	if hours < MinIntervalHours {
		hours = MinIntervalHours
	}
	if hours > MaxIntervalHours {
		hours = MaxIntervalHours
	}
	cfg := ScheduleConfig{Mode: ScheduleModeInterval, Hours: hours}
	data, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	return string(data)
}
