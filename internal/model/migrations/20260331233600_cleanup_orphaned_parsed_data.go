package migrations

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up20260331233600, down20260331233600)
}

// up20260331233600 cleans up orphaned rows in parsed_channels and parsed_epgs
// whose source_id references a live_source or epg_source that no longer exists.
// This can happen if a source was deleted while an async sync was still in progress.
func up20260331233600(ctx context.Context, tx *sql.Tx) error {
	// Delete parsed_channels whose source_id has no matching live_sources row
	result, err := tx.ExecContext(ctx,
		`DELETE FROM parsed_channels WHERE source_id NOT IN (SELECT id FROM live_sources)`)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n > 0 {
		slog.Info("Cleaned up orphaned parsed_channels", "count", n)
	}

	// Delete parsed_epgs whose source_id has no matching epg_sources row
	result, err = tx.ExecContext(ctx,
		`DELETE FROM parsed_epgs WHERE source_id NOT IN (SELECT id FROM epg_sources)`)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n > 0 {
		slog.Info("Cleaned up orphaned parsed_epgs", "count", n)
	}

	return nil
}

// down20260331233600 is a no-op because deleted orphaned data cannot be recovered.
func down20260331233600(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Rollback of orphaned data cleanup is a no-op")
	return nil
}
