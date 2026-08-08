// Package store provides the bot's persistent storage, backed by SQLite.
//
// It uses the pure-Go modernc.org/sqlite driver (no cgo) so the binary stays
// statically linked and runs on the distroless image.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// DB is a handle to the bot's SQLite database.
type DB struct {
	sql *sql.DB
}

// Open opens (creating if needed) the database at path and applies the schema.
func Open(ctx context.Context, path string) (*DB, error) {
	// busy_timeout avoids "database is locked" errors under brief contention;
	// foreign_keys keeps referential integrity on for future tables.
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	// A single connection avoids SQLite write contention for this tiny workload.
	sqlDB.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}
	if _, err := sqlDB.ExecContext(ctx, schema); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &DB{sql: sqlDB}, nil
}

// Close releases the database handle.
func (db *DB) Close() error {
	return db.sql.Close()
}
