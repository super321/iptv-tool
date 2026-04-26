package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up20260426140000, down20260426140000)
}

// up20260426140000 migrates the old filter_invalid_source_ids (comma-separated IDs)
// to the new detect_filter_config (JSON object) column, then drops the old column.
func up20260426140000(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Running detect filter config migration...")

	// Check if the old column exists (it might not exist on fresh installs)
	var oldColumnExists bool
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info(publish_interfaces)")
	if err != nil {
		return fmt.Errorf("query table_info: %w", err)
	}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			rows.Close()
			return fmt.Errorf("scan table_info: %w", err)
		}
		if name == "filter_invalid_source_ids" {
			oldColumnExists = true
		}
	}
	rows.Close()

	if !oldColumnExists {
		slog.Info("Old column filter_invalid_source_ids does not exist, skipping migration")
		return nil
	}

	// Read and convert old data
	dataRows, err := tx.QueryContext(ctx, "SELECT id, filter_invalid_source_ids FROM publish_interfaces WHERE filter_invalid_source_ids != '' AND filter_invalid_source_ids IS NOT NULL")
	if err != nil {
		return fmt.Errorf("query publish_interfaces: %w", err)
	}

	type updateEntry struct {
		id     uint
		config string
	}
	var updates []updateEntry

	for dataRows.Next() {
		var id uint
		var oldIDs string
		if err := dataRows.Scan(&id, &oldIDs); err != nil {
			dataRows.Close()
			return fmt.Errorf("scan publish_interfaces: %w", err)
		}

		// Parse comma-separated IDs into uint slice
		parts := strings.Split(oldIDs, ",")
		var sourceIDs []uint
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			parsed, err := strconv.ParseUint(p, 10, 32)
			if err != nil {
				continue
			}
			sourceIDs = append(sourceIDs, uint(parsed))
		}

		if len(sourceIDs) == 0 {
			continue
		}

		// Build new JSON config
		newConfig := map[string]interface{}{
			"source_ids":            sourceIDs,
			"min_resolution_width":  0,
			"min_resolution_height": 0,
		}
		data, err := json.Marshal(newConfig)
		if err != nil {
			slog.Warn("Failed to marshal detect filter config", "id", id, "error", err)
			continue
		}
		updates = append(updates, updateEntry{id: id, config: string(data)})
	}
	dataRows.Close()

	// Apply updates
	for _, u := range updates {
		if _, err := tx.ExecContext(ctx, "UPDATE publish_interfaces SET detect_filter_config = ? WHERE id = ?",
			u.config, u.id); err != nil {
			return fmt.Errorf("update detect_filter_config for id=%d: %w", u.id, err)
		}
	}

	// Drop the old column
	if _, err := tx.ExecContext(ctx, "ALTER TABLE publish_interfaces DROP COLUMN filter_invalid_source_ids"); err != nil {
		slog.Warn("Failed to drop old column filter_invalid_source_ids (may require SQLite 3.35.0+)", "error", err)
		// Non-fatal: older SQLite versions don't support DROP COLUMN.
		// The column will remain but is unused.
	}

	slog.Info("Detect filter config migration completed", "migrated_records", len(updates))
	return nil
}

// down20260426140000 is a no-op because converting back would risk data loss.
func down20260426140000(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Rollback of detect filter config migration is a no-op")
	return nil
}
