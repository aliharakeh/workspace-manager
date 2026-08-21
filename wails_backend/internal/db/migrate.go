package db

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed migrations/*.sql migrations/journal.json
var migrationFS embed.FS

type journal struct {
	Entries []journalEntry `json:"entries"`
}

type journalEntry struct {
	Idx  int    `json:"idx"`
	Tag  string `json:"tag"`
	When int64  `json:"when"`
}

func applyDrizzleMigrations(sqlDB *sql.DB) error {
	if _, err := sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS "__drizzle_migrations" (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash text NOT NULL,
			created_at numeric
		)
	`); err != nil {
		return err
	}

	raw, err := migrationFS.ReadFile("migrations/journal.json")
	if err != nil {
		return err
	}
	var j journal
	if err := json.Unmarshal(raw, &j); err != nil {
		return err
	}

	applied := map[string]struct{}{}
	rows, err := sqlDB.Query(`SELECT hash FROM __drizzle_migrations`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return err
		}
		applied[hash] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, entry := range j.Entries {
		body, err := migrationFS.ReadFile("migrations/" + entry.Tag + ".sql")
		if err != nil {
			return fmt.Errorf("migration %s: %w", entry.Tag, err)
		}
		sum := sha256.Sum256(body)
		hash := hex.EncodeToString(sum[:])
		if _, ok := applied[hash]; ok {
			continue
		}
		for _, stmt := range strings.Split(string(body), "--> statement-breakpoint") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := sqlDB.Exec(stmt); err != nil {
				return fmt.Errorf("migration %s: %w", entry.Tag, err)
			}
		}
		if _, err := sqlDB.Exec(
			`INSERT INTO __drizzle_migrations (hash, created_at) VALUES (?, ?)`,
			hash, entry.When,
		); err != nil {
			return err
		}
	}
	return nil
}
