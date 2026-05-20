package runtime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendOutbound_CommsOnLineSuccess(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<Envelope><Body><commsOnLineAck><sessionId>S-1</sessionId></commsOnLineAck></Body></Envelope>`))
	}))
	defer host.Close()

	srv := newTestServer(t)
	res, err := srv.SendOutbound(context.Background(), outboundRequest{
		MessageType: "commsOnLine",
		HostID:      "HOST-001",
		SessionID:   "S-1",
		TargetURL:   host.URL,
	})
	if err != nil {
		t.Fatalf("SendOutbound returned error: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected outbound success, got %#v", res)
	}
	if res.ResponseRoot != "commsOnLineAck" {
		t.Fatalf("expected commsOnLineAck, got %q", res.ResponseRoot)
	}
}

func TestOutboundScheduler_ReachesOnlineAndHealthy(t *testing.T) {
	var seen int
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		body := "<Envelope><Body><keepAliveAck><sessionId>S-1</sessionId></keepAliveAck></Body></Envelope>"
		if seen == 1 {
			body = "<Envelope><Body><commsOnLineAck><sessionId>S-1</sessionId></commsOnLineAck></Body></Envelope>"
		}
		_, _ = w.Write([]byte(body))
	}))
	defer host.Close()

	srv := newTestServer(t)
	srv.cfg.Outbound.DefaultTargetURL = host.URL
	srv.cfg.Outbound.Scheduler.Enabled = true
	srv.cfg.Outbound.Scheduler.AutoStart = false
	srv.cfg.Outbound.Scheduler.OpenSessionOnStart = true
	srv.cfg.Outbound.Scheduler.HeartbeatIntervalMS = 20
	srv.cfg.Outbound.Scheduler.OpenRetryIntervalMS = 10
	srv.cfg.Outbound.Scheduler.FailureThreshold = 2
	if err := srv.StartOutboundScheduler(); err != nil {
		t.Fatalf("StartOutboundScheduler failed: %v", err)
	}
	defer srv.StopOutboundScheduler()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		srv.mu.RLock()
		online := srv.state.SessionState == "online"
		healthy := srv.state.HeartbeatState == "healthy"
		opens := srv.state.OutboundScheduler.OpenAttempts
		heartbeats := srv.state.OutboundScheduler.HeartbeatAttempts
		srv.mu.RUnlock()
		if online && healthy && opens >= 1 && heartbeats >= 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	t.Fatalf("scheduler did not reach expected state: session=%q heartbeat=%q opens=%d heartbeats=%d", srv.state.SessionState, srv.state.HeartbeatState, srv.state.OutboundScheduler.OpenAttempts, srv.state.OutboundScheduler.HeartbeatAttempts)
}

func TestOutboundScheduler_DegradesAfterFailures(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Outbound.DefaultTargetURL = unusedLocalURL(t)
	srv.cfg.Outbound.Scheduler.Enabled = true
	srv.cfg.Outbound.Scheduler.AutoStart = false
	srv.cfg.Outbound.Scheduler.OpenSessionOnStart = true
	srv.cfg.Outbound.Scheduler.HeartbeatIntervalMS = 20
	srv.cfg.Outbound.Scheduler.OpenRetryIntervalMS = 10
	srv.cfg.Outbound.Scheduler.FailureThreshold = 1
	if err := srv.StartOutboundScheduler(); err != nil {
		t.Fatalf("StartOutboundScheduler failed: %v", err)
	}
	defer srv.StopOutboundScheduler()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		srv.mu.RLock()
		degraded := srv.state.HeartbeatState == "degraded"
		opens := srv.state.OutboundScheduler.OpenAttempts
		lastErr := srv.state.OutboundScheduler.LastError
		srv.mu.RUnlock()
		if degraded && opens >= 1 && lastErr != "" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	srv.mu.RLock()
	defer srv.mu.RUnlock()
	t.Fatalf("scheduler did not degrade as expected: heartbeat=%q opens=%d last_error=%q", srv.state.HeartbeatState, srv.state.OutboundScheduler.OpenAttempts, srv.state.OutboundScheduler.LastError)
}

func unusedLocalURL(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	if !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatalf("unexpected addr %q", addr)
	}
	return fmt.Sprintf("http://%s/host", addr)
}
