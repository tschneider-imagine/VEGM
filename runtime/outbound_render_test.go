package runtime

import (
	"strings"
	"testing"
)

func TestRenderConfiguredOutboundBody_LegacyKeepAliveUsesRequestShape(t *testing.T) {
	srv := newTestServer(t)
	body := srv.renderConfiguredOutboundBody("keepAlive", "S-KEEP")
	if body == "" {
		t.Fatalf("expected keepAlive body")
	}
	if !strings.Contains(body, ":keepAlive>") {
		t.Fatalf("expected keepAlive request element, got %s", body)
	}
	if strings.Contains(body, "keepAliveAck") {
		t.Fatalf("forced keepAlive rendered ACK response template: %s", body)
	}
	if !strings.Contains(body, "S-KEEP") {
		t.Fatalf("expected session id in body, got %s", body)
	}
}

func TestRenderConfiguredOutboundBody_LegacyCommsOnLineUsesRequestShape(t *testing.T) {
	srv := newTestServer(t)
	body := srv.renderConfiguredOutboundBody("commsOnLine", "S-COMMS")
	if body == "" {
		t.Fatalf("expected commsOnLine body")
	}
	if !strings.Contains(body, ":commsOnLine>") {
		t.Fatalf("expected commsOnLine request element, got %s", body)
	}
	if strings.Contains(body, "commsOnLineAck") {
		t.Fatalf("forced commsOnLine rendered ACK response template: %s", body)
	}
	if !strings.Contains(body, "S-COMMS") {
		t.Fatalf("expected session id in body, got %s", body)
	}
}

func TestRenderConfiguredOutboundBody_UnknownFallsBackToPack(t *testing.T) {
	srv := newTestServer(t)
	body := srv.renderConfiguredOutboundBody("audioMuteOn", "S-MUTE")
	if body != "" {
		t.Fatalf("expected non-session command to fall back to pack rendering, got %s", body)
	}
}
