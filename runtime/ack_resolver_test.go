package runtime

import "testing"

func TestResolveAckStrictMode(t *testing.T) {
	cfg := SessionEngineConfig{}

	if got := resolveAck("commsOnLineAck", "commsOnLineAck", nil, cfg); got != "commsOnLineAck" {
		t.Fatalf("strict exact ACK = %q, want commsOnLineAck", got)
	}

	if got := resolveAck("commsOnLineAck", "g2sResponse", []byte(`<g2sResponse/>`), cfg); got != "g2sResponse" {
		t.Fatalf("strict bare g2sResponse = %q, want g2sResponse", got)
	}
}

func TestResolveAckWrappedCompatibility(t *testing.T) {
	cfg := SessionEngineConfig{AcceptWrappedG2SResponseAck: true}

	if got := resolveAck("keepAliveAck", "g2sResponse", []byte(`<g2s:g2sResponse xmlns:g2s="urn:test"><g2s:keepAliveAck/></g2s:g2sResponse>`), cfg); got != "keepAliveAck" {
		t.Fatalf("wrapped nested ACK = %q, want keepAliveAck", got)
	}

	if got := resolveAck("commsOnLineAck", "g2sResponse", []byte(`<g2s:g2sResponse xmlns:g2s="urn:test"/>`), cfg); got != "commsOnLineAck" {
		t.Fatalf("lab bare g2sResponse fallback = %q, want commsOnLineAck", got)
	}
}
