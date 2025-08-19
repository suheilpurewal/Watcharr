package groupview

import "database/sql"

func Migrate(db *sql.DB) error {
    stmts := []string{
        `PRAGMA foreign_keys=ON;`,
        `CREATE TABLE IF NOT EXISTS members (
            id TEXT PRIMARY KEY,
            slug TEXT NOT NULL UNIQUE,
            display_name TEXT NOT NULL,
            is_active INTEGER NOT NULL DEFAULT 1,
            created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )`,
        `CREATE TABLE IF NOT EXISTS viewing_sessions (
            id TEXT PRIMARY KEY,
            media_id TEXT NOT NULL,
            media_type TEXT NOT NULL CHECK (media_type IN ('movie','episode')),
            started_at TEXT NOT NULL,
            finished_at TEXT,
            source TEXT,
            notes TEXT
        )`,
        `CREATE TABLE IF NOT EXISTS attendances (
            id TEXT PRIMARY KEY,
            viewing_session_id TEXT NOT NULL REFERENCES viewing_sessions(id) ON DELETE CASCADE,
            member_id TEXT NOT NULL REFERENCES members(id) ON DELETE RESTRICT,
            rating REAL,
            rated_at TEXT,
            UNIQUE (viewing_session_id, member_id)
        )`,
    }
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil { return err }
    }
    return nil
}
