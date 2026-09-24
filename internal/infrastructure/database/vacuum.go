package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Vacuum rebuilds the database file, reclaiming free pages. It is the
// "compact" operation offered by the settings page.
func Vacuum(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `VACUUM`); err != nil {
		return fmt.Errorf("database: vacuum: %w", err)
	}
	return nil
}
