package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteIndex_WriteAndQuery(t *testing.T) {
	idx := NewSQLiteIndex(filepath.Join(t.TempDir(), "vegm-index.db"))
	if err := idx.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	defer idx.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := idx.WriteEvent(EventRecord{
		Time:        now,
		InstanceID:  "vegm-001",
		Level:       "info",
		Category:    "wire",
		Message:     "handled keepAlive",
		MessageType: "keepAlive",
		HostID:      "HOST-001",
		SessionID:   "S-1",
	}); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	results, err := idx.Query(Query{MessageType: "keepAlive", HostID: "HOST-001", Limit: 10})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SessionID != "S-1" {
		t.Fatalf("expected session S-1, got %q", results[0].SessionID)
	}
}
