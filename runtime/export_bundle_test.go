package runtime

import (
	"encoding/json"
	"os"
	"testing"
)

func TestExportBundleIncludesEvidenceSummary(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "vegm-evidence")
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	logger.Log("info", "wire", "operation handled", map[string]any{"message_type": "audioMuteOn", "host_id": "HOST-001", "session_id": "S-1", "status": 200})
	logger.Log("info", "state", "state changed", map[string]any{"message_type": "audioMuteOn", "host_id": "HOST-001", "session_id": "S-1"})

	path, err := logger.ExportBundle(ExportOptions{RunID: "RUN-001", IncludePayloads: true, PassFailNotes: "expected mute applied"})
	if err != nil {
		t.Fatalf("ExportBundle failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	if bundle.RunID != "RUN-001" {
		t.Fatalf("expected run id RUN-001, got %q", bundle.RunID)
	}
	if bundle.ReasonBucket != "none" {
		t.Fatalf("expected reason bucket none, got %q", bundle.ReasonBucket)
	}
	if len(bundle.MessageSummary) != 1 {
		t.Fatalf("expected one message summary entry, got %d", len(bundle.MessageSummary))
	}
	if bundle.MessageSummary[0].MessageType != "audioMuteOn" {
		t.Fatalf("expected audioMuteOn message summary, got %#v", bundle.MessageSummary[0])
	}
	if len(bundle.StateTransitions) != 1 {
		t.Fatalf("expected one state transition, got %d", len(bundle.StateTransitions))
	}
	if bundle.FirstMessageWorksheet == nil {
		t.Fatalf("expected first message worksheet")
	}
}

func TestExportBundleReasonBucketCapturesFailure(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "vegm-evidence")
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	logger.Log("warn", "soap", "soap parse failed", map[string]any{"error": "bad envelope"})
	path, err := logger.ExportBundle(ExportOptions{RunID: "RUN-FAIL"})
	if err != nil {
		t.Fatalf("ExportBundle failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	var bundle ExportBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	if bundle.ReasonBucket != "soap" {
		t.Fatalf("expected soap reason bucket, got %q", bundle.ReasonBucket)
	}
}
