package migrations

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up20260316233800, down20260316233800)
}

// up20260316233800 drops legacy individual indexes on parsed_epgs that are now
// covered by the composite index idx_epg_source_channel_start.
func up20260316233800(ctx context.Context, tx *sql.Tx) error {
	indexes := []string{
		"idx_parsed_epgs_source_id",
		"idx_parsed_epgs_channel",
		"idx_parsed_epgs_start_time",
		"idx_parsed_epgs_end_time",
	}
	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS "+idx); err != nil {
			return err
		}
	}
	slog.Info("Dropped legacy EPG indexes")
	return nil
}

// down20260316233800 is a no-op because the legacy indexes are no longer needed.
func down20260316233800(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Rollback of legacy EPG index drop is a no-op")
	return nil
}
