package model

import (
	"log/slog"

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
