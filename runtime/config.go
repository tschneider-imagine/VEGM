package runtime

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	InstanceID string            `json:"instance_id"`
	EGMID      string            `json:"egm_id"`
	Listen     ListenConfig      `json:"listen"`
	Security   SecurityConfig    `json:"security"`
	Logging    LoggingConfig     `json:"logging"`
	Control    ControlConfig     `json:"control"`
	Outbound   OutboundConfig    `json:"outbound,omitempty"`
	PackFile   string            `json:"pack_file"`
	Overlay    []string          `json:"overlay_files,omitempty"`
	Notes      map[string]string `json:"notes,omitempty"`
}

type ListenConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type SecurityConfig struct {
	TrustMode string `json:"trust_mode"`
	CertFile  string `json:"cert_file,omitempty"`
	KeyFile   string `json:"key_file,omitempty"`
	CAFile    string `json:"ca_file,omitempty"`
}

type LoggingConfig struct {
	Dir                string `json:"dir"`
	CaptureRawXML      bool   `json:"capture_raw_xml"`
	CaptureRenderedXML bool   `json:"capture_rendered_xml"`
}

type ControlConfig struct {
	Bind string `json:"bind"`
}

type OutboundConfig struct {
	DefaultTargetURL string                  `json:"default_target_url,omitempty"`
	ServerName       string                  `json:"server_name,omitempty"`
	Path             string                  `json:"path,omitempty"`
	TimeoutMS        int                     `json:"timeout_ms,omitempty"`
	UseRuntimeCerts  bool                    `json:"use_runtime_certs,omitempty"`
	Scheduler        OutboundSchedulerConfig `json:"scheduler,omitempty"`
}

type OutboundSchedulerConfig struct {
	Enabled              bool `json:"enabled,omitempty"`
	AutoStart            bool `json:"auto_start,omitempty"`
	OpenSessionOnStart   bool `json:"open_session_on_start,omitempty"`
	HeartbeatIntervalMS  int  `json:"heartbeat_interval_ms,omitempty"`
	OpenRetryIntervalMS  int  `json:"open_retry_interval_ms,omitempty"`
	FailureThreshold     int  `json:"failure_threshold,omitempty"`
	RunHeartbeatWhenIdle bool `json:"run_heartbeat_when_idle,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode config json: %w", err)
	}
	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.InstanceID == "" {
		return fmt.Errorf("instance_id is required")
	}
	if cfg.EGMID == "" {
		return fmt.Errorf("egm_id is required")
	}
	if cfg.Listen.Host == "" {
		cfg.Listen.Host = "127.0.0.1"
	}
	if cfg.Logging.Dir == "" {
		return fmt.Errorf("logging.dir is required")
	}
	if cfg.Control.Bind == "" {
		cfg.Control.Bind = "127.0.0.1:0"
	}
	if cfg.PackFile == "" {
		return fmt.Errorf("pack_file is required")
	}
	if cfg.Outbound.TimeoutMS < 0 {
		return fmt.Errorf("outbound.timeout_ms must be >= 0")
	}
	if cfg.Outbound.Scheduler.HeartbeatIntervalMS < 0 {
		return fmt.Errorf("outbound.scheduler.heartbeat_interval_ms must be >= 0")
	}
	if cfg.Outbound.Scheduler.OpenRetryIntervalMS < 0 {
		return fmt.Errorf("outbound.scheduler.open_retry_interval_ms must be >= 0")
	}
	if cfg.Outbound.Scheduler.FailureThreshold < 0 {
		return fmt.Errorf("outbound.scheduler.failure_threshold must be >= 0")
	}
	switch cfg.Security.TrustMode {
	case "plaintext_lab":
	case "tls_server_only", "strict_mtls", "mtls_no_revocation", "accept_all_lab":
		if cfg.Security.CertFile == "" || cfg.Security.KeyFile == "" {
			return fmt.Errorf("cert_file and key_file are required for trust mode %q", cfg.Security.TrustMode)
		}
		if (cfg.Security.TrustMode == "strict_mtls" || cfg.Security.TrustMode == "mtls_no_revocation") && cfg.Security.CAFile == "" {
			return fmt.Errorf("ca_file is required for trust mode %q", cfg.Security.TrustMode)
		}
	default:
		return fmt.Errorf("unsupported trust_mode %q", cfg.Security.TrustMode)
	}
	return nil
}
