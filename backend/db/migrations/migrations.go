package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// Files contains the ordered, append-only PostgreSQL schema migrations.
//
//go:embed *.sql
var Files embed.FS

func Apply(ctx context.Context, db *sql.DB) error {
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		return fmt.Errorf("read database migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		body, readErr := Files.ReadFile(name)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", name, readErr)
		}
		prefix := strings.SplitN(name, "_", 2)[0]
		version, parseErr := strconv.ParseInt(prefix, 10, 64)
		if parseErr != nil {
			return fmt.Errorf("migration %s has no numeric version: %w", name, parseErr)
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(body))
		var recordedChecksum string
		lookupErr := db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version = $1", version).Scan(&recordedChecksum)
		if lookupErr == nil {
			if recordedChecksum != checksum {
				return fmt.Errorf("migration %s checksum changed", name)
			}
			continue
		}
		if lookupErr != sql.ErrNoRows {
			// The first migration creates schema_migrations itself.
			if version != 1 {
				return fmt.Errorf("read migration state for %s: %w", name, lookupErr)
			}
		}
		tx, beginErr := db.BeginTx(ctx, nil)
		if beginErr != nil {
			return fmt.Errorf("begin migration %s: %w", name, beginErr)
		}
		if _, execErr := tx.ExecContext(ctx, string(body)); execErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, execErr)
		}
		if _, execErr := tx.ExecContext(ctx, `
			INSERT INTO schema_migrations(version, name, checksum, applied_at, result)
			VALUES ($1, $2, $3, NOW(), 'applied')
			ON CONFLICT (version) DO NOTHING`, version, name, checksum); execErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, execErr)
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return fmt.Errorf("commit migration %s: %w", name, commitErr)
		}
	}
	return nil
}
