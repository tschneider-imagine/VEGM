package runtime

import (
	"encoding/json"
	"testing"
)

func TestRuntimeStateJSONIncludesParsedResponseEvidence(t *testing.T) {
	s := &Server{cfg: &Config{
		InstanceID: "vegm-evidence-json-test",
		G2SXML: G2SXMLConfig{
			Mode:        G2SXMLModeXSDMessage,
			Namespace:   G2SDefaultNamespace,
			EGMLocation: "192.168.10.161:18443",
		},
	}}
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
		"g2s_xml_mode":          G2SXMLModeXSDMessage,
		"g2s_xml_namespace":     G2SDefaultNamespace,
		"g2s_xml_egm_location":  "192.168.10.161:18443",
		"last_parsed_root_kind": "g2sMessage",
		"last_parsed_class":     "communications",
		"last_parsed_operation": "keepAliveAck",
		"last_raw_root":         "g2sMessage",
		"last_expected_ack":     "keepAliveAck",
		"last_actual_ack":       "keepAliveAck",
	}
	for key, want := range checks {
		if got := out[key]; got != want {
			t.Fatalf("%s = %#v, want %q; json=%s", key, got, want, string(body))
		}
	}
}

func TestRuntimeStateJSONIncludesConfiguredXMLMetadataBeforeParsedResponse(t *testing.T) {
	cfg := &Config{
		InstanceID: "vegm-xml-before-response-test",
		HostID:     "HOST-001",
		EGMID:      "EGM-XML-001",
		EGMEndpoint: EGMEndpointConfig{
			Scheme: "http",
			BindIP: "192.168.10.161",
			Host:   "192.168.10.161",
			Port:   18443,
			Path:   "/g2s",
		},
		G2SXML: G2SXMLConfig{
			Mode:        G2SXMLModeXSDMessage,
			Namespace:   G2SDefaultNamespace,
			EGMLocation: "192.168.10.161:18443",
		},
		Listen:   ListenConfig{Host: "192.168.10.161", Port: 18443},
		Security: SecurityConfig{TrustMode: "plaintext_lab"},
		Logging:  LoggingConfig{Dir: t.TempDir()},
		Control:  ControlConfig{Bind: "127.0.0.1:0"},
		PackFile: "../example.pack.json",
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	state := RuntimeState{InstanceID: cfg.InstanceID}
	body, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	checks := map[string]string{
		"g2s_xml_mode":         G2SXMLModeXSDMessage,
		"g2s_xml_namespace":    G2SDefaultNamespace,
		"g2s_xml_egm_location": "192.168.10.161:18443",
	}
	for key, want := range checks {
		if got := out[key]; got != want {
			t.Fatalf("%s = %#v, want %q; json=%s", key, got, want, string(body))
		}
	}
}
