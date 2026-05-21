package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteIndex struct {
	path string
	db   *sql.DB
}

func NewSQLiteIndex(path string) *SQLiteIndex {
	return &SQLiteIndex{path: path}
}

func (s *SQLiteIndex) Initialize() error {
	if s.path == "" {
		return fmt.Errorf("sqlite path is required")
	}
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return err
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS event_index (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  time_utc TEXT NOT NULL,
  instance_id TEXT NOT NULL,
  level TEXT,
  category TEXT,
  message TEXT,
  message_type TEXT,
  host_id TEXT,
  session_id TEXT,
  payload_path TEXT
);
CREATE INDEX IF NOT EXISTS idx_event_index_time ON event_index(time_utc);
CREATE INDEX IF NOT EXISTS idx_event_index_message_type ON event_index(message_type);
CREATE INDEX IF NOT EXISTS idx_event_index_host_id ON event_index(host_id);
CREATE INDEX IF NOT EXISTS idx_event_index_session_id ON event_index(session_id);
CREATE INDEX IF NOT EXISTS idx_event_index_category ON event_index(category);
`); err != nil {
		_ = db.Close()
		return err
	}
	s.db = db
	return nil
}

func (s *SQLiteIndex) WriteEvent(rec EventRecord) error {
	if s.db == nil {
		return fmt.Errorf("sqlite index is not initialized")
	}
	_, err := s.db.Exec(`
INSERT INTO event_index (
  time_utc, instance_id, level, category, message, message_type, host_id, session_id, payload_path
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, rec.Time.UTC().Format(time.RFC3339Nano), rec.InstanceID, rec.Level, rec.Category, rec.Message, rec.MessageType, rec.HostID, rec.SessionID, rec.PayloadPath)
	return err
}

func (s *SQLiteIndex) Query(q Query) ([]EventRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("sqlite index is not initialized")
	}
	where := []string{"1=1"}
	args := []any{}
	if q.Category != "" {
		where = append(where, "category = ?")
		args = append(args, q.Category)
	}
	if q.Level != "" {
		where = append(where, "level = ?")
		args = append(args, q.Level)
	}
	if q.MessageType != "" {
		where = append(where, "message_type = ?")
		args = append(args, q.MessageType)
	}
	if q.HostID != "" {
		where = append(where, "host_id = ?")
		args = append(args, q.HostID)
	}
	if q.SessionID != "" {
		where = append(where, "session_id = ?")
		args = append(args, q.SessionID)
	}
	if q.Contains != "" {
		where = append(where, "LOWER(message) LIKE ?")
		args = append(args, "%"+strings.ToLower(q.Contains)+"%")
	}
	if !q.Since.IsZero() {
		where = append(where, "time_utc >= ?")
		args = append(args, q.Since.UTC().Format(time.RFC3339Nano))
	}
	if !q.Until.IsZero() {
		where = append(where, "time_utc <= ?")
		args = append(args, q.Until.UTC().Format(time.RFC3339Nano))
	}
	stmt := `SELECT time_utc, instance_id, level, category, message, message_type, host_id, session_id, payload_path FROM event_index WHERE ` + strings.Join(where, " AND ") + ` ORDER BY time_utc DESC, id DESC`
	if q.Limit > 0 {
		stmt += fmt.Sprintf(" LIMIT %d", q.Limit)
	}
	rows, err := s.db.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventRecord
	for rows.Next() {
		var ts string
		var rec EventRecord
		if err := rows.Scan(&ts, &rec.InstanceID, &rec.Level, &rec.Category, &rec.Message, &rec.MessageType, &rec.HostID, &rec.SessionID, &rec.PayloadPath); err != nil {
			return nil, err
		}
		rec.Time, err = time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteIndex) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func DefaultSQLitePath(logDir string) string {
	return filepath.Join(logDir, "vegm-index.db")
}
