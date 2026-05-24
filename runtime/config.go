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

type StorageConfig struct {
	Backend    string `json:"backend,omitempty"`
	SQLitePath string `json:"sqlite_path,omitempty"`
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
	switch cfg.Storage.Backend {
	case "", "noop":
	case "sqlite":
		if cfg.Storage.SQLitePath == "" {
			return fmt.Errorf("storage.sqlite_path is required when storage.backend is %q", cfg.Storage.Backend)
		}
	default:
		return fmt.Errorf("unsupported storage.backend %q", cfg.Storage.Backend)
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

func applyEndpointDefaults(cfg *Config) {
	if cfg.EGMEndpoint.Scheme == "" {
		if cfg.Security.TrustMode == "plaintext_lab" || cfg.Security.TrustMode == "" {
			cfg.EGMEndpoint.Scheme = "http"
		} else {
			cfg.EGMEndpoint.Scheme = "https"
		}
	}
	if cfg.EGMEndpoint.BindIP == "" {
		cfg.EGMEndpoint.BindIP = firstNonEmpty(cfg.Listen.Host, "127.0.0.1")
	}
	if cfg.EGMEndpoint.Host == "" {
		cfg.EGMEndpoint.Host = cfg.EGMEndpoint.BindIP
	}
	if cfg.EGMEndpoint.Port == 0 {
		cfg.EGMEndpoint.Port = cfg.Listen.Port
	}
	if cfg.EGMEndpoint.Path == "" {
		cfg.EGMEndpoint.Path = "/g2s"
	}
	if cfg.Outbound.DefaultTargetURL == "" && cfg.HostEndpoint.URL != "" {
		cfg.Outbound.DefaultTargetURL = cfg.HostEndpoint.URL
	}
}

func applyG2SXMLDefaults(cfg *Config) {
	if cfg.G2SXML.Mode == "" {
		cfg.G2SXML.Mode = "lab_legacy_xml"
	}
	if cfg.G2SXML.Namespace == "" {
		cfg.G2SXML.Namespace = "http://www.gamingstandards.com/g2s/schemas/v1.0.3"
	}
	if cfg.G2SXML.EGMLocation == "" {
		cfg.G2SXML.EGMLocation = fmt.Sprintf("%s:%d", cfg.EGMEndpoint.Host, cfg.EGMEndpoint.Port)
	}
}

func validateG2SXML(x G2SXMLConfig) error {
	switch x.Mode {
	case "lab_legacy_xml", "xsd_g2s_message":
	default:
		return fmt.Errorf("unsupported g2s_xml.mode %q", x.Mode)
	}
	if x.Namespace == "" {
		return fmt.Errorf("g2s_xml.namespace is required")
	}
	if x.EGMLocation == "" {
		return fmt.Errorf("g2s_xml.egm_location is required")
	}
	return nil
}

func applySessionEngineDefaults(cfg *Config) {
	if cfg.SessionEngine.CommsOnlineTimeoutMS == 0 {
		cfg.SessionEngine.CommsOnlineTimeoutMS = 3000
	}
	if cfg.SessionEngine.KeepAliveIntervalMS == 0 {
		cfg.SessionEngine.KeepAliveIntervalMS = 5000
	}
	if cfg.SessionEngine.ReconnectIntervalMS == 0 {
		cfg.SessionEngine.ReconnectIntervalMS = 5000
	}
}

func validateEGMEndpoint(ep EGMEndpointConfig) error {
	if ep.Scheme != "http" && ep.Scheme != "https" {
		return fmt.Errorf("egm_endpoint.scheme must be http or https")
	}
	if ep.BindIP == "" {
		return fmt.Errorf("egm_endpoint.bind_ip is required")
	}
	if ep.Host == "" {
		return fmt.Errorf("egm_endpoint.host is required")
	}
	if ep.Port <= 0 || ep.Port > 65535 {
		return fmt.Errorf("egm_endpoint.port must be between 1 and 65535")
	}
	if ep.Path == "" || ep.Path[0] != '/' {
		return fmt.Errorf("egm_endpoint.path must start with /")
	}
	return nil
}

func validateHostEndpoint(ep HostEndpointConfig) error {
	if ep.URL == "" {
		return nil
	}
	u, err := url.Parse(ep.URL)
	if err != nil {
		return fmt.Errorf("host_endpoint.url is invalid: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("host_endpoint.url scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("host_endpoint.url must include host")
	}
	if u.Path == "" || u.Path[0] != '/' {
		return fmt.Errorf("host_endpoint.url must include path")
	}
	return nil
}
