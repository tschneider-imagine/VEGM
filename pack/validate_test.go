package pack

import "testing"

func TestValidatePack_BasicStarterShape(t *testing.T) {
	p := &MessagePack{
		SchemaVersion: MessagePackSchemaVersion,
		PackName:      "test-pack",
		PackVersion:   "0.1.0",
		Wire: WireConfig{
			ProtocolFamily: "g2s",
			Transport: TransportConfig{TLSModesSupported: []string{"plaintext_lab"}},
			Envelope: EnvelopeConfig{Kind: "soap-http"},
			Paths: PathsConfig{DefaultListenerPath: "/g2s"},
			Namespaces: map[string]string{"soapenv": "http://schemas.xmlsoap.org/soap/envelope/"},
		},
		ControlPlane: ControlPlaneConfig{HotReloadableSections: []string{"timers"}, MutableFields: []string{"/timers/artificial_response_delay_ms"}},
		Timers: TimersConfig{RequestTimeoutMS: 3000, HeartbeatIntervalMS: 5000, HeartbeatTimeoutMS: 12000},
		StateDefaults: StateDefaultsConfig{RegistrationMode: "strict_match", SessionState: "idle", HeartbeatState: "idle"},
		Operations: map[string]Operation{
			"commsOnLine": {
				Enabled:   true,
				Direction: "inbound",
				Match:     []MatchRule{{Kind: "message_type", Value: "commsOnLine"}},
				Responses: []ResponseVariant{{VariantName: "ack", Template: "<x/>"}},
			},
		},
	}
	if err := ValidatePack(p); err != nil {
		t.Fatalf("expected pack to validate, got %v", err)
	}
}

func TestValidateOverlay_Basic(t *testing.T) {
	o := &MessageOverlay{
		SchemaVersion: MessageOverlaySchemaVersion,
		OverlayName:   "test-overlay",
		TargetPack:    "test-pack",
		Changes:       []OverlayChange{{Op: "set", Path: "/timers/artificial_response_delay_ms", Value: 100}},
	}
	if err := ValidateOverlay(o); err != nil {
		t.Fatalf("expected overlay to validate, got %v", err)
	}
}
