package runtime

import (
	"encoding/json"
	"testing"
)

func TestRuntimeStateJSONIncludesParsedResponseEvidence(t *testing.T) {
	s := &Server{cfg: &Config{InstanceID: "vegm-evidence-json-test"}}
	s.recordParsedResponseEvidence("keepAliveAck", ParsedG2SEnvelope{
		RootKind:      "g2sMessage",
		ClassName:     "communications",
		OperationName: "keepAliveAck",
		RawRoot:       "g2sMessage",
	})

	state := RuntimeState{InstanceID: "vegm-evidence-json-test"}
	body, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	checks := map[string]string{
		"last_parsed_root_kind":  "g2sMessage",
		"last_parsed_class":      "communications",
		"last_parsed_operation":  "keepAliveAck",
		"last_raw_root":          "g2sMessage",
		"last_expected_ack":      "keepAliveAck",
		"last_actual_ack":        "keepAliveAck",
	}
	for key, want := range checks {
		if got := out[key]; got != want {
			t.Fatalf("%s = %#v, want %q; json=%s", key, got, want, string(body))
		}
	}
}
