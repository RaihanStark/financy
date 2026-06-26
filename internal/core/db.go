package core

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"
)

// ErrFileTooNew is returned when a document's schema version is higher than this
// build understands — it was written by a newer version of Financy. We refuse to
// open it rather than risk misreading (or silently corrupting) the data.
var ErrFileTooNew = errors.New("this file was created by a newer version of Financy")

// migrations are applied in order; the DB's PRAGMA user_version tracks how many
// have run. Append new migrations — never edit an existing one.
var migrations = []string{
	// v1 — initial schema.
	`
	CREATE TABLE accounts (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		type        TEXT NOT NULL,
		institution TEXT NOT NULL DEFAULT '',
		notes       TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE transactions (
		id    TEXT PRIMARY KEY,
		date  INTEGER NOT NULL,
		payee TEXT NOT NULL DEFAULT '',
		memo  TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE postings (
		txn_id     TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
		seq        INTEGER NOT NULL,
		account_id TEXT NOT NULL,
		amount     INTEGER NOT NULL,
		PRIMARY KEY (txn_id, seq)
	);
	CREATE INDEX idx_postings_account ON postings(account_id);
	CREATE TABLE settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`,
	// v2 — recurring transaction templates.
	`
	CREATE TABLE recurring (
		id       TEXT PRIMARY KEY,
		kind     TEXT NOT NULL,
		acct_a   TEXT NOT NULL,
		acct_b   TEXT NOT NULL,
		amount   INTEGER NOT NULL,
		payee    TEXT NOT NULL DEFAULT '',
		memo     TEXT NOT NULL DEFAULT '',
		freq     TEXT NOT NULL,
		next_due INTEGER NOT NULL,
		enabled  INTEGER NOT NULL DEFAULT 1
	);
	`,
	// v3 — zero-based budget assignments (one row per month × category).
	`
	CREATE TABLE budget (
		month       TEXT NOT NULL,
		category_id TEXT NOT NULL,
		amount      INTEGER NOT NULL,
		PRIMARY KEY (month, category_id)
	);
	`,
	// v4 — per-account "off budget" (tracking) flag. Default 0 keeps every
	// existing account on-budget.
	`
	ALTER TABLE accounts ADD COLUMN off_budget INTEGER NOT NULL DEFAULT 0;
	`,
}

// schemaVersion is the number of migrations that define the current schema.
func schemaVersion() int { return len(migrations) }

// openDB opens (or creates) a SQLite database file and applies pending
// migrations. A ":memory:" path yields an ephemeral in-memory database.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// One connection keeps writes serialized and avoids locking surprises for a
	// single-user document; enforce foreign keys for cascade deletes.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// migrate applies any migrations newer than the DB's current user_version.
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version;`).Scan(&version); err != nil {
		return err
	}
	if version > len(migrations) {
		return ErrFileTooNew
	}
	for v := version; v < len(migrations); v++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[v]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d: %w", v+1, err)
		}
		// PRAGMA user_version doesn't accept placeholders.
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d;`, v+1)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("set version %d: %w", v+1, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// dbVersion reports a database file's schema version without migrating it.
func dbVersion(path string) (int, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = db.Close() }()
	var v int
	if err := db.QueryRow(`PRAGMA user_version;`).Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}
