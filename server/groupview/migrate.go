package groupview

import (
    "database/sql"
    "log/slog"
)

func Migrate(db *sql.DB) error {
    stmts := []string{
        `PRAGMA foreign_keys=ON;`,
        
        // Create groups table for family groups
        `CREATE TABLE IF NOT EXISTS groups (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )`,
        
        // Create group_members table to link users to groups
        `CREATE TABLE IF NOT EXISTS group_members (
            id TEXT PRIMARY KEY,
            group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
            user_id INTEGER NOT NULL,
            role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin','member')),
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now')),
            UNIQUE (group_id, user_id)
        )`,
        
        // Legacy members table - keeping for backward compatibility
        `CREATE TABLE IF NOT EXISTS members (
            id TEXT PRIMARY KEY,
            slug TEXT NOT NULL UNIQUE,
            display_name TEXT NOT NULL,
            is_active INTEGER NOT NULL DEFAULT 1,
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )`,
        
        `CREATE TABLE IF NOT EXISTS viewing_sessions (
            id TEXT PRIMARY KEY,
            media_id TEXT NOT NULL,
            media_type TEXT NOT NULL CHECK (media_type IN ('movie','episode')),
            started_at TEXT NOT NULL,
            finished_at TEXT,
            source TEXT,
            notes TEXT,
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )`,
        
        // Updated attendances table with user_id support
        `CREATE TABLE IF NOT EXISTS attendances (
            id TEXT PRIMARY KEY,
            viewing_session_id TEXT NOT NULL REFERENCES viewing_sessions(id) ON DELETE CASCADE,
            member_id TEXT REFERENCES members(id) ON DELETE RESTRICT,
            user_id INTEGER,
            rating REAL,
            rated_at TEXT,
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )`,
        
        // Add indexes for performance
        `CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id)`,
        `CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON group_members(user_id)`,
        `CREATE INDEX IF NOT EXISTS idx_attendances_user_id ON attendances(user_id)`,
        `CREATE INDEX IF NOT EXISTS idx_attendances_viewing_session_id ON attendances(viewing_session_id)`,
    }
    
    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil { 
            slog.Error("Migration failed", "statement", s, "error", err)
            return err 
        }
    }
    
    slog.Info("Group view migration completed successfully")
    return nil
}
