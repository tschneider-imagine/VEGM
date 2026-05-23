package runtime

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSessionEngineAutoStartSendsRecurringKeepAlive(t *testing.T) {
	var mu sync.Mutex
	counts := map[string]int{}

	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		parsed, err := ParseG2SMessage(body)
		if err != nil {
			t.Fatalf("parse request: %v", err)
		}
		mu.Lock()
		counts[parsed.RootLocalName]++
		mu.Unlock()

		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		switch parsed.RootLocalName {
		case "commsOnLine":
			_, _ = io.WriteString(w, soapResponse("commsOnLineAck", parsed.Fields["hostId"], parsed.Fields["egmId"], parsed.Fields["sessionId"]))
		case "keepAlive":
			_, _ = io.WriteString(w, soapResponse("keepAliveAck", parsed.Fields["hostId"], parsed.Fields["egmId"], parsed.Fields["sessionId"]))
		default:
			http.Error(w, "unsupported", http.StatusBadRequest)
		}
	}))
	defer host.Close()

	cfg := &Config{
		InstanceID: "vegm-loop-test",
		HostID:     "HOST-001",
		EGMID:      "EGM-LOOP-001",
		EGMEndpoint: EGMEndpointConfig{
			Scheme: "http",
			BindIP: "127.0.0.1",
			Host:   "127.0.0.1",
			Port:   18443,
			Path:   "/g2s",
		},
		HostEndpoint: HostEndpointConfig{URL: host.URL + "/g2s"},
		SessionEngine: SessionEngineConfig{
			Enabled:              true,
			AutoStart:            true,
			CommsOnlineTimeoutMS: 500,
			KeepAliveIntervalMS:  40,
			ReconnectIntervalMS:  100,
		},
		Listen:   ListenConfig{Host: "127.0.0.1", Port: 18443},
		Security: SecurityConfig{TrustMode: "plaintext_lab"},
		Logging:  LoggingConfig{Dir: t.TempDir(), CaptureRawXML: true, CaptureRenderedXML: true},
		Control:  ControlConfig{Bind: "127.0.0.1:0"},
		PackFile: filepath.Join("..", "example.pack.json"),
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartSessionEngine(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		comms := counts["commsOnLine"]
		keep := counts["keepAlive"]
		mu.Unlock()
		if comms >= 1 && keep >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	comms := counts["commsOnLine"]
	keep := counts["keepAlive"]
	mu.Unlock()
	if comms < 1 {
		t.Fatalf("expected commsOnLine, got %d", comms)
	}
	if keep < 2 {
		t.Fatalf("expected recurring keepAlive, got %d", keep)
	}

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if srv.state.SessionState != "online" {
		t.Fatalf("session state = %q, want online", srv.state.SessionState)
	}
	if srv.state.HeartbeatState != "healthy" {
		t.Fatalf("heartbeat state = %q, want healthy", srv.state.HeartbeatState)
	}
	if srv.state.LastMessageType != "keepAliveAck" {
		t.Fatalf("last message = %q, want keepAliveAck", srv.state.LastMessageType)
	}
}

func soapResponse(messageType, hostID, egmID, sessionID string) string {
	return `<soapenv:Envelope xmlns:soapenv="` + SOAP11Namespace + `" xmlns:g2s="urn:g2s:lab"><soapenv:Body><g2s:` + messageType + `><g2s:hostId>` + hostID + `</g2s:hostId><g2s:egmId>` + egmID + `</g2s:egmId><g2s:sessionId>` + sessionID + `</g2s:sessionId></g2s:` + messageType + `></soapenv:Body></soapenv:Envelope>`
}
