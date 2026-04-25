package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up20260425230000, down20260425230000)
}

// up20260425230000 migrates old-format filter rule configs from a plain JSON
// array to the new object format with filter_mode.
//
// Old format: [{"match_mode":"regex","target":"name","pattern":"..."}]
// New format: {"filter_mode":"blacklist","rules":[{"match_mode":"regex","target":"name","pattern":"..."}]}
func up20260425230000(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Running filter rule config migration...")
	migrated := 0

	rows, err := tx.QueryContext(ctx, "SELECT id, config FROM aggregation_rules WHERE type = 'filter'")
	if err != nil {
		return fmt.Errorf("query filter rules: %w", err)
	}
	defer rows.Close()

	type ruleUpdate struct {
		id     uint
		config string
	}
	var updates []ruleUpdate

	for rows.Next() {
		var id uint
		var config string
		if err := rows.Scan(&id, &config); err != nil {
			return fmt.Errorf("scan filter rule: %w", err)
		}

		trimmed := strings.TrimSpace(config)
		if trimmed == "" {
			continue
		}

		// Already new format (JSON object)?
		if strings.HasPrefix(trimmed, "{") {
			continue
		}

		// Old format (JSON array)? Wrap it.
		if strings.HasPrefix(trimmed, "[") {
			// Validate it's valid JSON
			var arr json.RawMessage
			if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
				slog.Warn("Filter rule config is not valid JSON, skipping", "id", id, "error", err)
				continue
			}

			newConfig := map[string]interface{}{
				"filter_mode": "blacklist",
				"rules":       arr,
			}
			data, err := json.Marshal(newConfig)
			if err != nil {
				slog.Warn("Failed to marshal new filter config", "id", id, "error", err)
				continue
			}
			updates = append(updates, ruleUpdate{id: id, config: string(data)})
		}
	}
	rows.Close()

	for _, u := range updates {
		if _, err := tx.ExecContext(ctx, "UPDATE aggregation_rules SET config = ? WHERE id = ?",
			u.config, u.id); err != nil {
			return fmt.Errorf("update filter rule id=%d: %w", u.id, err)
		}
		migrated++
	}

	slog.Info("Filter rule config migration completed", "migrated_records", migrated)
	return nil
}

// down20260425230000 is a no-op because converting back would risk data loss.
func down20260425230000(ctx context.Context, tx *sql.Tx) error {
	slog.Info("Rollback of filter rule config migration is a no-op")
	return nil
}
