package model

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"

	// Blank import to trigger init() registration of all goose migrations
	_ "iptv-tool-v2/internal/model/migrations"
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

	// Run goose migrations (Go functions registered via init() in the migrations package)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		return err
	}

	// Defensive reset: set is_syncing and is_detecting to false for all sources on startup
	DB.Model(&LiveSource{}).Where("is_syncing = ?", true).Update("is_syncing", false)
	DB.Model(&LiveSource{}).Where("is_detecting = ?", true).Update("is_detecting", false)
	DB.Model(&EPGSource{}).Where("is_syncing = ?", true).Update("is_syncing", false)

	slog.Info("Database initialized and migrated successfully.")
	return nil
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
