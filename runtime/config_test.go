package runtime

import "testing"

func TestValidateConfig_Plaintext(t *testing.T) {
	cfg := &Config{
		InstanceID: "vegm-001",
		EGMID:      "EGM-001",
		Listen:     ListenConfig{Host: "127.0.0.1", Port: 18443},
		Security:   SecurityConfig{TrustMode: "plaintext_lab"},
		Logging:    LoggingConfig{Dir: "./logs/test", CaptureRawXML: true, CaptureRenderedXML: true},
		Control:    ControlConfig{Bind: "127.0.0.1:19001"},
		PackFile:   "./example.pack.json",
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected config to validate, got %v", err)
	}
}

func TestValidateConfig_StrictMTLSRequiresCerts(t *testing.T) {
	cfg := &Config{
		InstanceID: "vegm-001",
		EGMID:      "EGM-001",
		Listen:     ListenConfig{Host: "127.0.0.1", Port: 18443},
		Security:   SecurityConfig{TrustMode: "strict_mtls"},
		Logging:    LoggingConfig{Dir: "./logs/test", CaptureRawXML: true, CaptureRenderedXML: true},
		Control:    ControlConfig{Bind: "127.0.0.1:19001"},
		PackFile:   "./example.pack.json",
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected strict_mtls config to fail without certs")
	}
}
