package runtime

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleWire_CommsOnLineHappyPath(t *testing.T) {
	srv := newTestServer(t)
	reqBody := `<Envelope><Body><commsOnLine><hostId>HOST-001</hostId><sessionId>S-1</sessionId></commsOnLine></Body></Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/g2s", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	srv.handleWire(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "commsOnLineAck") {
		t.Fatalf("expected commsOnLineAck response, got %s", rr.Body.String())
	}
	if srv.state.SessionState != "online" {
		t.Fatalf("expected session_state online, got %q", srv.state.SessionState)
	}
	if srv.state.LastHostID != "HOST-001" {
		t.Fatalf("expected last host id HOST-001, got %q", srv.state.LastHostID)
	}
}

func TestHandleWire_UnregisteredHostRejected(t *testing.T) {
	srv := newTestServer(t)
	reqBody := `<Envelope><Body><keepAlive><hostId>HOST-999</hostId><sessionId>S-9</sessionId></keepAlive></Body></Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/g2s", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	srv.handleWire(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestControlHostsAddAndRemove(t *testing.T) {
	srv := newTestServer(t)

	addReq := httptest.NewRequest(http.MethodPost, "/control/hosts/add", bytes.NewBufferString(`{"host_id":"HOST-ABC"}`))
	addRR := httptest.NewRecorder()
	srv.handleControlHostsAdd(addRR, addReq)
	if addRR.Code != http.StatusOK {
		t.Fatalf("expected add 200, got %d body=%s", addRR.Code, addRR.Body.String())
	}
	if !containsString(srv.state.AllowedHostIDs, "HOST-ABC") {
		t.Fatalf("expected HOST-ABC to be present after add")
	}

	removeReq := httptest.NewRequest(http.MethodPost, "/control/hosts/remove", bytes.NewBufferString(`{"host_id":"HOST-ABC"}`))
	removeRR := httptest.NewRecorder()
	srv.handleControlHostsRemove(removeRR, removeReq)
	if removeRR.Code != http.StatusOK {
		t.Fatalf("expected remove 200, got %d body=%s", removeRR.Code, removeRR.Body.String())
	}
	if containsString(srv.state.AllowedHostIDs, "HOST-ABC") {
		t.Fatalf("expected HOST-ABC to be removed")
	}
}

func TestControlOverlay_ChangesRegistrationMode(t *testing.T) {
	srv := newTestServer(t)
	payload := map[string]any{
		"schema_version": "vegm.message-overlay/v1",
		"overlay_name":   "open-reg",
		"target_pack":    "test-pack",
		"changes": []map[string]any{
			{"op": "set", "path": "/state_defaults/registration_mode", "value": "open"},
		},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/control/overlay", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.handleControlOverlay(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if srv.pack.StateDefaults.RegistrationMode != "open" {
		t.Fatalf("expected pack registration_mode open, got %q", srv.pack.StateDefaults.RegistrationMode)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	packPath := filepath.Join(dir, "pack.json")
	packJSON := `{
  "schema_version": "vegm.message-pack/v1",
  "pack_name": "test-pack",
  "pack_version": "0.1.0",
  "wire": {
    "protocol_family": "g2s",
    "transport": {
      "http_versions": ["1.1"],
      "tls_modes_supported": ["plaintext_lab"],
      "content_types": ["text/xml; charset=utf-8"],
      "soap_action_required": false
    },
    "envelope": {"kind": "soap-http"},
    "paths": {"default_listener_path": "/g2s"},
    "namespaces": {"soapenv": "http://schemas.xmlsoap.org/soap/envelope/", "g2s": "urn:test:g2s"}
  },
  "control_plane": {
    "hot_reloadable_sections": ["timers", "state_defaults", "operations"],
    "mutable_fields": ["/state_defaults/registration_mode"]
  },
  "timers": {
    "request_timeout_ms": 3000,
    "heartbeat_interval_ms": 5000,
    "heartbeat_timeout_ms": 12000,
    "artificial_response_delay_ms": 0
  },
  "state_defaults": {
    "registration_mode": "strict_match",
    "session_state": "idle",
    "heartbeat_state": "idle",
    "allowed_host_ids": ["HOST-001"]
  },
  "operations": {
    "commsOnLine": {
      "enabled": true,
      "direction": "inbound",
      "match": [{"kind": "message_type", "value": "commsOnLine"}],
      "extract": [],
      "responses": [{
        "variant_name": "ack",
        "template": "<Envelope><Body><commsOnLineAck><hostId>{{request.hostId}}</hostId><egmId>{{request.egmId}}</egmId><sessionId>{{request.sessionId}}</sessionId></commsOnLineAck></Body></Envelope>",
        "http_status": 200
      }]
    },
    "keepAlive": {
      "enabled": true,
      "direction": "inbound",
      "match": [{"kind": "message_type", "value": "keepAlive"}],
      "extract": [],
      "responses": [{
        "variant_name": "ack",
        "template": "<Envelope><Body><keepAliveAck><sessionId>{{request.sessionId}}</sessionId></keepAliveAck></Body></Envelope>",
        "http_status": 200
      }]
    }
  }
}`
	if err := os.WriteFile(packPath, []byte(packJSON), 0o644); err != nil {
		t.Fatalf("write pack: %v", err)
	}
	cfg := &Config{
		InstanceID: "vegm-test",
		EGMID:      "EGM-TEST",
		Listen:     ListenConfig{Host: "127.0.0.1", Port: 0},
		Security:   SecurityConfig{TrustMode: "plaintext_lab"},
		Logging:    LoggingConfig{Dir: filepath.Join(dir, "logs"), CaptureRawXML: true, CaptureRenderedXML: true},
		Control:    ControlConfig{Bind: "127.0.0.1:0"},
		PackFile:   packPath,
	}
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	return srv
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
