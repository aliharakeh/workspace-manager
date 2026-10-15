package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// ChangeEntry is one row of the change log.
type ChangeEntry struct {
	ID     int64
	Entity string
	IDs    []int64
}

// LogChange appends a write to the change log, so another process that shares
// the database (the GUI) can refresh what it shows.
func (d *DB) LogChange(ctx context.Context, entity string, ids []int64) error {
	if ids == nil {
		ids = []int64{}
	}
	b, _ := json.Marshal(ids)
	_, err := d.SQL.ExecContext(ctx, `INSERT INTO changes (entity, ids) VALUES (?, ?)`, entity, string(b))
	return err
}

// LastChangeID is the id of the newest change, 0 when the log is empty.
func (d *DB) LastChangeID(ctx context.Context) (int64, error) {
	var id sql.NullInt64
	err := d.SQL.QueryRowContext(ctx, `SELECT MAX(id) FROM changes`).Scan(&id)
	return id.Int64, err
}

// ChangesSince returns the changes logged after id, oldest first.
func (d *DB) ChangesSince(ctx context.Context, id int64) ([]ChangeEntry, error) {
	rows, err := d.SQL.QueryContext(ctx, `SELECT id, entity, ids FROM changes WHERE id > ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChangeEntry
	for rows.Next() {
		var c ChangeEntry
		var raw string
		if err := rows.Scan(&c.ID, &c.Entity, &raw); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &c.IDs)
		out = append(out, c)
	}
	return out, rows.Err()
}

// PruneChanges drops log rows older than a day; the log only needs to cover
// the gap between two GUI polls.
func (d *DB) PruneChanges(ctx context.Context) error {
	_, err := d.SQL.ExecContext(ctx, `DELETE FROM changes WHERE created_at < datetime('now', '-1 day')`)
	return err
}

// SessionClaim says who is running an app: the GUI's runner, or an agent that
// asked the MCP server to prepare a run.
type SessionClaim struct {
	AppID       int64
	SessionID   string
	Owner       string
	Kind        string
	PID         int
	ConfigSetID int64
	Dir         string
	StartedAt   time.Time
}

// ErrSessionHeld is returned by ClaimSession when a live session already
// holds the app.
var ErrSessionHeld = errors.New("app already has a session")

func scanSession(row interface{ Scan(...any) error }) (SessionClaim, error) {
	var s SessionClaim
	var started string
	if err := row.Scan(&s.AppID, &s.SessionID, &s.Owner, &s.Kind, &s.PID, &s.ConfigSetID, &s.Dir, &started); err != nil {
		return SessionClaim{}, err
	}
	s.StartedAt, _ = time.Parse("2006-01-02 15:04:05", started)
	return s, nil
}

const sessionCols = `app_id, session_id, owner, kind, pid, config_set_id, dir, started_at`

// GetAppSession returns the app's session row, (zero, false) when none.
func (d *DB) GetAppSession(ctx context.Context, appID int64) (SessionClaim, bool, error) {
	s, err := scanSession(d.SQL.QueryRowContext(ctx, `SELECT `+sessionCols+` FROM app_sessions WHERE app_id = ?`, appID))
	if errors.Is(err, sql.ErrNoRows) {
		return SessionClaim{}, false, nil
	}
	return s, err == nil, err
}

// ListAppSessions returns every session row.
func (d *DB) ListAppSessions(ctx context.Context) ([]SessionClaim, error) {
	rows, err := d.SQL.QueryContext(ctx, `SELECT `+sessionCols+` FROM app_sessions ORDER BY app_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionClaim
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// TryInsertAppSession inserts the row unless the app already has one, and
// reports whether it did.
func (d *DB) TryInsertAppSession(ctx context.Context, s SessionClaim) (bool, error) {
	res, err := d.SQL.ExecContext(ctx, `INSERT INTO app_sessions (app_id, session_id, owner, kind, pid, config_set_id, dir)
		VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(app_id) DO NOTHING`,
		s.AppID, s.SessionID, s.Owner, s.Kind, s.PID, s.ConfigSetID, s.Dir)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// DeleteAppSession removes the app's row; with a session id, only that
// session's, so a late release cannot drop a newer claim.
func (d *DB) DeleteAppSession(ctx context.Context, appID int64, sessionID string) error {
	if sessionID == "" {
		_, err := d.SQL.ExecContext(ctx, `DELETE FROM app_sessions WHERE app_id = ?`, appID)
		return err
	}
	_, err := d.SQL.ExecContext(ctx, `DELETE FROM app_sessions WHERE app_id = ? AND session_id = ?`, appID, sessionID)
	return err
}
