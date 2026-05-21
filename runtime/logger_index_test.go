package runtime

import (
	"path/filepath"
	"testing"

	"github.com/tschneider-imagine/VEGM/storage"
)

type captureIndex struct {
	records []storage.EventRecord
}

func (c *captureIndex) Initialize() error { return nil }
func (c *captureIndex) WriteEvent(r storage.EventRecord) error {
	c.records = append(c.records, r)
	return nil
}
func (c *captureIndex) Query(storage.Query) ([]storage.EventRecord, error) { return c.records, nil }
func (c *captureIndex) Close() error { return nil }

func TestLogger_MirrorsEventsToIndex(t *testing.T) {
	idx := &captureIndex{}
	logger, err := NewLoggerWithIndex(filepath.Join(t.TempDir(), "logs"), "vegm-index-test", idx)
	if err != nil {
		t.Fatalf("NewLoggerWithIndex failed: %v", err)
	}
	defer logger.Close()

	logger.Log("info", "wire", "handled keepAlive", map[string]any{
		"message_type": "keepAlive",
		"host_id":      "HOST-001",
		"session_id":   "S-1",
	})

	if len(idx.records) != 1 {
		t.Fatalf("expected 1 indexed record, got %d", len(idx.records))
	}
	rec := idx.records[0]
	if rec.InstanceID != "vegm-index-test" {
		t.Fatalf("expected instance id vegm-index-test, got %q", rec.InstanceID)
	}
	if rec.MessageType != "keepAlive" {
		t.Fatalf("expected message type keepAlive, got %q", rec.MessageType)
	}
	if rec.HostID != "HOST-001" {
		t.Fatalf("expected host id HOST-001, got %q", rec.HostID)
	}
	if rec.SessionID != "S-1" {
		t.Fatalf("expected session id S-1, got %q", rec.SessionID)
	}
}
