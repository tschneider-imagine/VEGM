package runtime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	InstanceID    string              `json:"instance_id"`
	HostID        string              `json:"host_id,omitempty"`
	EGMID         string              `json:"egm_id"`
	EGMEndpoint   EGMEndpointConfig   `json:"egm_endpoint,omitempty"`
	HostEndpoint  HostEndpointConfig  `json:"host_endpoint,omitempty"`
	G2SXML        G2SXMLConfig        `json:"g2s_xml,omitempty"`
	SessionEngine SessionEngineConfig `json:"session_engine,omitempty"`
	Listen        ListenConfig        `json:"listen"`
	Security      SecurityConfig      `json:"security"`
	Logging       LoggingConfig       `json:"logging"`
	Storage       StorageConfig       `json:"storage,omitempty"`
	Control       ControlConfig       `json:"control"`
	Outbound      OutboundConfig      `json:"outbound,omitempty"`
	PackFile      string              `json:"pack_file"`
	Overlay       []string            `json:"overlay_files,omitempty"`
	Notes         map[string]string   `json:"notes,omitempty"`
}

type EGMEndpointConfig struct {
	Scheme string `json:"scheme,omitempty"`
	BindIP string `json:"bind_ip,omitempty"`
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Path   string `json:"path,omitempty"`
}

type HostEndpointConfig struct {
	URL string `json:"url,omitempty"`
}

type G2SXMLConfig struct {
	Mode        string `json:"mode,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	EGMLocation string `json:"egm_location,omitempty"`
}

type SessionEngineConfig struct {
	Enabled              bool `json:"enabled,omitempty"`
	AutoStart            bool `json:"auto_start,omitempty"`
	CommsOnlineTimeoutMS int  `json:"comms_online_timeout_ms,omitempty"`
	KeepAliveIntervalMS  int  `json:"keep_alive_interval_ms,omitempty"`
	ReconnectIntervalMS  int  `json:"reconnect_interval_ms,omitempty"`
	AcceptWrappedG2SResponseAck bool `json:"accept_wrapped_g2s_response_ack,omitempty"`
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
	if cfg.HostID == "" {
		cfg.HostID = firstNonEmpty(cfg.Notes["host_id"], "HOST-001")
	}
	if cfg.EGMID == "" {
		return fmt.Errorf("egm_id is required")
	}
	applyEndpointDefaults(cfg)
	applyG2SXMLDefaults(cfg)
	applySessionEngineDefaults(cfg)
	if err := validateEGMEndpoint(cfg.EGMEndpoint); err != nil {
		return err
	}
	if err := validateHostEndpoint(cfg.HostEndpoint); err != nil {
		return err
	}
	if err := validateG2SXML(cfg.G2SXML); err != nil {
		return err
	}
	rememberXMLModeInfo(cfg.InstanceID, cfg.G2SXML)
	if cfg.SessionEngine.Enabled && cfg.HostEndpoint.URL == "" {
		return fmt.Errorf("host_endpoint.url is required when session_engine.enabled is true")
	}
	if cfg.Listen.Host == "" {
		cfg.Listen.Host = cfg.EGMEndpoint.BindIP
	}
	if cfg.Listen.Port == 0 {
		cfg.Listen.Port = cfg.EGMEndpoint.Port
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
	return nil
}