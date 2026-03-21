package model

import (
	"fmt"
	"log/slog"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(dsn string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// Enable WAL mode: allows concurrent reads during writes,
	// preventing scheduled task bulk inserts from blocking web queries.
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	slog.Info("SQLite WAL mode enabled")

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

	slog.Info("Database initialized and migrated successfully.")
	return nil
}
