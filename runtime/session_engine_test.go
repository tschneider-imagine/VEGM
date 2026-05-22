package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunCommsOnlineOnce_MockHostAck(t *testing.T) {
	srv := newTestServer(t)
	seenRequest := false
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); !strings.Contains(got, "text/xml") {
			t.Fatalf("expected text/xml content type, got %q", got)
		}
		seenRequest = true
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:g2s="urn:test:g2s"><soapenv:Body><g2s:commsOnLineAck><g2s:hostId>HOST-001</g2s:hostId><g2s:egmId>EGM-TEST</g2s:egmId><g2s:sessionId>S-ACK</g2s:sessionId></g2s:commsOnLineAck></soapenv:Body></soapenv:Envelope>`))
	}))
	defer host.Close()

	srv.cfg.HostEndpoint.URL = host.URL + "/g2s"
	srv.cfg.SessionEngine.Enabled = true
	srv.cfg.SessionEngine.CommsOnlineTimeoutMS = 3000

	if err := srv.runCommsOnlineOnce(context.Background()); err != nil {
		t.Fatalf("runCommsOnlineOnce failed: %v", err)
	}
	if !seenRequest {
		t.Fatalf("mock host did not receive commsOnLine")
	}
	if srv.state.SessionState != "online" {
		t.Fatalf("expected session online, got %q", srv.state.SessionState)
	}
	if srv.state.ConnectionState != "host_connected" {
		t.Fatalf("expected host_connected, got %q", srv.state.ConnectionState)
	}
	if srv.state.LastMessageType != "commsOnLineAck" {
		t.Fatalf("expected last message commsOnLineAck, got %q", srv.state.LastMessageType)
	}
	if srv.state.LastHostID != "HOST-001" {
		t.Fatalf("expected host id HOST-001, got %q", srv.state.LastHostID)
	}
}

func TestRunCommsOnlineOnce_RejectsWrongAck(t *testing.T) {
	srv := newTestServer(t)
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"><soapenv:Body><wrongAck/></soapenv:Body></soapenv:Envelope>`))
	}))
	defer host.Close()

	srv.cfg.HostEndpoint.URL = host.URL + "/g2s"
	srv.cfg.SessionEngine.Enabled = true
	srv.cfg.SessionEngine.CommsOnlineTimeoutMS = 3000

	if err := srv.runCommsOnlineOnce(context.Background()); err == nil {
		t.Fatalf("expected wrong ACK error")
	}
}
