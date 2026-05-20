package pack

// MessagePack is the runtime-loadable VEGM message pack model.
type MessagePack struct {
	SchemaVersion string               `json:"schema_version"`
	PackName      string               `json:"pack_name"`
	PackVersion   string               `json:"pack_version"`
	Description   string               `json:"description,omitempty"`
	Extends       string               `json:"extends,omitempty"`
	Tags          []string             `json:"tags,omitempty"`
	Wire          WireConfig           `json:"wire"`
	ControlPlane  ControlPlaneConfig   `json:"control_plane"`
	Timers        TimersConfig         `json:"timers"`
	StateDefaults StateDefaultsConfig  `json:"state_defaults"`
	LoggingHints  LoggingHintsConfig   `json:"logging_hints,omitempty"`
	Operations    map[string]Operation `json:"operations"`
}

type WireConfig struct {
	ProtocolFamily string            `json:"protocol_family"`
	Transport      TransportConfig   `json:"transport"`
	Envelope       EnvelopeConfig    `json:"envelope"`
	Paths          PathsConfig       `json:"paths"`
	Namespaces     map[string]string `json:"namespaces"`
}

type TransportConfig struct {
	HTTPVersions       []string `json:"http_versions"`
	TLSModesSupported  []string `json:"tls_modes_supported"`
	ContentTypes       []string `json:"content_types"`
	SOAPActionRequired bool     `json:"soap_action_required,omitempty"`
}

type EnvelopeConfig struct {
	Kind            string `json:"kind"`
	SOAPNamespace   string `json:"soap_namespace,omitempty"`
	BodyXPathHint   string `json:"body_xpath_hint,omitempty"`
	HeaderXPathHint string `json:"header_xpath_hint,omitempty"`
}

type PathsConfig struct {
	DefaultListenerPath    string   `json:"default_listener_path"`
	AlternateListenerPaths []string `json:"alternate_listener_paths,omitempty"`
}

type ControlPlaneConfig struct {
	HotReloadableSections []string `json:"hot_reloadable_sections"`
	MutableFields         []string `json:"mutable_fields"`
	RequiresAuditLog      bool     `json:"requires_audit_log,omitempty"`
}

type TimersConfig struct {
	RequestTimeoutMS          int `json:"request_timeout_ms"`
	HeartbeatIntervalMS       int `json:"heartbeat_interval_ms"`
	HeartbeatTimeoutMS        int `json:"heartbeat_timeout_ms"`
	SessionIdleTimeoutMS      int `json:"session_idle_timeout_ms,omitempty"`
	ArtificialResponseDelayMS int `json:"artificial_response_delay_ms,omitempty"`
}

type StateDefaultsConfig struct {
	RegistrationMode string           `json:"registration_mode"`
	SessionState     string           `json:"session_state"`
	HeartbeatState   string           `json:"heartbeat_state"`
	AllowedHostIDs   []string         `json:"allowed_host_ids,omitempty"`
	RegisteredHosts  []RegisteredHost `json:"registered_hosts,omitempty"`
}

type RegisteredHost struct {
	HostID      string `json:"host_id"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role,omitempty"`
	Enabled     bool   `json:"enabled"`
	Notes       string `json:"notes,omitempty"`
}

type LoggingHintsConfig struct {
	CaptureRawXML      bool     `json:"capture_raw_xml,omitempty"`
	CaptureRenderedXML bool     `json:"capture_rendered_xml,omitempty"`
	ParsedFields       []string `json:"parsed_fields,omitempty"`
	RedactFields       []string `json:"redact_fields,omitempty"`
}

type Operation struct {
	Enabled       bool              `json:"enabled"`
	Description   string            `json:"description,omitempty"`
	Direction     string            `json:"direction"`
	Match         []MatchRule       `json:"match"`
	Preconditions []string          `json:"preconditions,omitempty"`
	Extract       []ExtractRule     `json:"extract"`
	Responses     []ResponseVariant `json:"responses"`
	FaultPolicy   FaultPolicy       `json:"fault_policy,omitempty"`
	LogFields     []string          `json:"log_fields,omitempty"`
}

type MatchRule struct {
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Source string `json:"source,omitempty"`
}

type ExtractRule struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	Selector string `json:"selector"`
	Default  any    `json:"default,omitempty"`
}

type ResponseVariant struct {
	VariantName string            `json:"variant_name"`
	Template    string            `json:"template"`
	When        []MatchRule       `json:"when,omitempty"`
	SetState    map[string]any    `json:"set_state,omitempty"`
	HTTPStatus  int               `json:"http_status,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	DelayMS     int               `json:"delay_ms,omitempty"`
}

type FaultPolicy struct {
	DropRequestProbability       float64 `json:"drop_request_probability,omitempty"`
	CorruptResponseProbability   float64 `json:"corrupt_response_probability,omitempty"`
	ForceMalformedOnce           bool    `json:"force_malformed_once,omitempty"`
	CloseConnectionAfterResponse bool    `json:"close_connection_after_response,omitempty"`
	OverrideHTTPStatus           int     `json:"override_http_status,omitempty"`
}

// MessageOverlay is a control-plane patch set applied to a MessagePack.
type MessageOverlay struct {
	SchemaVersion  string          `json:"schema_version"`
	OverlayName    string          `json:"overlay_name"`
	OverlayVersion string          `json:"overlay_version,omitempty"`
	TargetPack     string          `json:"target_pack"`
	Description    string          `json:"description,omitempty"`
	Changes        []OverlayChange `json:"changes"`
	Activate       OverlayActivate `json:"activate,omitempty"`
}

type OverlayChange struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

type OverlayActivate struct {
	ApplyImmediately bool   `json:"apply_immediately,omitempty"`
	Scope            string `json:"scope,omitempty"`
	TTLMS            int    `json:"ttl_ms,omitempty"`
}
