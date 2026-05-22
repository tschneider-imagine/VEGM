package fleet

type Manifest struct {
	SchemaVersion string             `json:"schema_version"`
	FleetName     string             `json:"fleet_name"`
	Description   string             `json:"description,omitempty"`
	Defaults      Defaults           `json:"defaults"`
	Profiles      map[string]Profile `json:"profiles"`
	Groups        map[string]Group   `json:"groups"`
	Instances     []Instance         `json:"instances"`
}

type Defaults struct {
	ListenHost       string         `json:"listen_host,omitempty"`
	WirePortBase     int            `json:"wire_port_base,omitempty"`
	ControlPortBase  int            `json:"control_port_base,omitempty"`
	TrustMode        string         `json:"trust_mode,omitempty"`
	PackFile         string         `json:"pack_file,omitempty"`
	OverlayFiles     []string       `json:"overlay_files,omitempty"`
	LogRoot          string         `json:"log_root,omitempty"`
	StorageBackend   string         `json:"storage_backend,omitempty"`
	SQLiteRoot       string         `json:"sqlite_root,omitempty"`
	AdvertisedHost   string         `json:"advertised_host,omitempty"`
	AdvertisedIP     string         `json:"advertised_ip,omitempty"`
	DNSServers       []string       `json:"dns_servers,omitempty"`
	SubnetMask       string         `json:"subnet_mask,omitempty"`
	Gateway          string         `json:"gateway,omitempty"`
	ServerName       string         `json:"server_name,omitempty"`
	CertFile         string         `json:"cert_file,omitempty"`
	KeyFile          string         `json:"key_file,omitempty"`
	CAFile           string         `json:"ca_file,omitempty"`
	Heartbeat        map[string]any `json:"heartbeat,omitempty"`
	NormalizedState  map[string]any `json:"normalized_state,omitempty"`
	Faults           map[string]any `json:"faults,omitempty"`
}

type Profile struct {
	Label           string            `json:"label,omitempty"`
	Manufacturer    string            `json:"manufacturer,omitempty"`
	PackFile        string            `json:"pack_file"`
	OverlayFiles    []string          `json:"overlay_files,omitempty"`
	LogicalCommands map[string]string `json:"logical_commands"`
	AdvertisedHost  string            `json:"advertised_host,omitempty"`
	AdvertisedIP    string            `json:"advertised_ip,omitempty"`
	DNSServers      []string          `json:"dns_servers,omitempty"`
	SubnetMask      string            `json:"subnet_mask,omitempty"`
	Gateway         string            `json:"gateway,omitempty"`
	ServerName      string            `json:"server_name,omitempty"`
	CertFile        string            `json:"cert_file,omitempty"`
	KeyFile         string            `json:"key_file,omitempty"`
	CAFile          string            `json:"ca_file,omitempty"`
	Heartbeat       map[string]any    `json:"heartbeat,omitempty"`
	NormalizedState map[string]any    `json:"normalized_state,omitempty"`
	Faults          map[string]any    `json:"faults,omitempty"`
}

type Group struct {
	Label     string         `json:"label,omitempty"`
	Profile   string         `json:"profile"`
	Zone      string         `json:"zone,omitempty"`
	Overrides map[string]any `json:"overrides,omitempty"`
}

type Instance struct {
	InstanceID     string         `json:"instance_id"`
	EGMID          string         `json:"egm_id"`
	Group          string         `json:"group"`
	WirePort       int            `json:"wire_port,omitempty"`
	ControlPort    int            `json:"control_port,omitempty"`
	BindHost       string         `json:"bind_host,omitempty"`
	AdvertisedHost string         `json:"advertised_host,omitempty"`
	AdvertisedIP   string         `json:"advertised_ip,omitempty"`
	DNSServers     []string       `json:"dns_servers,omitempty"`
	SubnetMask     string         `json:"subnet_mask,omitempty"`
	Gateway        string         `json:"gateway,omitempty"`
	ServerName     string         `json:"server_name,omitempty"`
	LogDir         string         `json:"log_dir,omitempty"`
	SQLitePath     string         `json:"sqlite_path,omitempty"`
	CertFile       string         `json:"cert_file,omitempty"`
	KeyFile        string         `json:"key_file,omitempty"`
	CAFile         string         `json:"ca_file,omitempty"`
	Overrides      map[string]any `json:"overrides,omitempty"`
}

type EffectiveInstance struct {
	InstanceID      string            `json:"instance_id"`
	EGMID           string            `json:"egm_id"`
	Group           string            `json:"group"`
	Profile         string            `json:"profile"`
	Manufacturer    string            `json:"manufacturer,omitempty"`
	ListenHost      string            `json:"listen_host"`
	WirePort        int               `json:"wire_port"`
	ControlPort     int               `json:"control_port"`
	AdvertisedHost  string            `json:"advertised_host,omitempty"`
	AdvertisedIP    string            `json:"advertised_ip,omitempty"`
	DNSServers      []string          `json:"dns_servers,omitempty"`
	SubnetMask      string            `json:"subnet_mask,omitempty"`
	Gateway         string            `json:"gateway,omitempty"`
	ServerName      string            `json:"server_name,omitempty"`
	TrustMode       string            `json:"trust_mode"`
	PackFile        string            `json:"pack_file"`
	OverlayFiles    []string          `json:"overlay_files,omitempty"`
	LogDir          string            `json:"log_dir"`
	StorageBackend  string            `json:"storage_backend"`
	SQLitePath      string            `json:"sqlite_path,omitempty"`
	LogicalCommands map[string]string `json:"logical_commands,omitempty"`
	Heartbeat       map[string]any    `json:"heartbeat,omitempty"`
	NormalizedState map[string]any    `json:"normalized_state,omitempty"`
	Faults          map[string]any    `json:"faults,omitempty"`
	CertFile        string            `json:"cert_file,omitempty"`
	KeyFile         string            `json:"key_file,omitempty"`
	CAFile          string            `json:"ca_file,omitempty"`
}
