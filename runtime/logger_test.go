package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogger_QueryAndExportBundle(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "vegm-test")
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	logger.Log("info", "wire", "handled keepAlive", map[string]any{"message_type": "keepAlive", "sessionId": "S-1"})
	logger.Log("warn", "control", "host removed", map[string]any{"host_id": "HOST-002"})
	if _, err := logger.WritePayload("inbound_request", "keepAlive", []byte("<keepAlive/>")); err != nil {
		t.Fatalf("WritePayload failed: %v", err)
	}

	events, err := logger.QueryEvents(EventFilter{MessageType: "keepAlive"})
	if err != nil {
		t.Fatalf("QueryEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 keepAlive event, got %d", len(events))
	}

	bundlePath, err := logger.ExportBundle(ExportOptions{IncludePayloads: true, Since: time.Now().Add(-time.Hour)})
	if err != nil {
		t.Fatalf("ExportBundle failed: %v", err)
	}
	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("expected bundle file to exist: %v", err)
	}
	if filepath.Ext(bundlePath) != ".json" {
		t.Fatalf("expected json bundle output, got %s", bundlePath)
	}
}
