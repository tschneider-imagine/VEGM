package storage

import "time"

// EventRecord is the normalized metadata shape intended for future index/search backends.
type EventRecord struct {
	Time        time.Time
	InstanceID  string
	Level       string
	Category    string
	Message     string
	MessageType string
	HostID      string
	SessionID   string
	PayloadPath string
}

// Query describes a future storage-backed search request.
type Query struct {
	Category    string
	Level       string
	MessageType string
	HostID      string
	SessionID   string
	Contains    string
	Since       time.Time
	Until       time.Time
	Limit       int
}

// Index is the seam intended for future SQLite-backed search/index integration.
type Index interface {
	Initialize() error
	WriteEvent(EventRecord) error
	Query(Query) ([]EventRecord, error)
	Close() error
}
