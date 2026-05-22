package fleet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	runtimecfg "github.com/tschneider-imagine/VEGM/runtime"
)

type GeneratedConfig struct {
	Instance EffectiveInstance `json:"instance"`
	Path     string            `json:"path"`
	Config   runtimecfg.Config `json:"config"`
}

func GenerateConfigs(m *Manifest, outDir string) ([]GeneratedConfig, error) {
	effective, err := ResolveInstances(m)
	if err != nil {
		return nil, err
	}
	if outDir == "" {
		outDir = "./generated"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir generated dir: %w", err)
	}
	var out []GeneratedConfig
	for _, eff := range effective {
		notes := map[string]string{
			"group":             eff.Group,
			"profile":           eff.Profile,
			"manufacturer":      eff.Manufacturer,
			"host_id":           eff.HostID,
			"egm_endpoint_scheme": eff.EGMEndpoint.Scheme,
			"egm_endpoint_bind_ip": eff.EGMEndpoint.BindIP,
			"egm_endpoint_host": eff.EGMEndpoint.Host,
			"egm_endpoint_path": eff.EGMEndpoint.Path,
			"host_endpoint_url": eff.HostEndpoint.URL,
			"advertised_host":   eff.AdvertisedHost,
			"advertised_ip":     eff.AdvertisedIP,
			"subnet_mask":       eff.SubnetMask,
			"gateway":           eff.Gateway,
			"server_name":       eff.ServerName,
			"dns_servers":       strings.Join(eff.DNSServers, ","),
			"cert_file":         eff.CertFile,
			"key_file":          eff.KeyFile,
			"ca_file":           eff.CAFile,
			"storage_backend":   eff.StorageBackend,
			"sqlite_path":       eff.SQLitePath,
		}
		cfg := runtimecfg.Config{
			InstanceID: eff.InstanceID,
			HostID:     eff.HostID,
			EGMID:      eff.EGMID,
			EGMEndpoint: runtimecfg.EGMEndpointConfig{
				Scheme: eff.EGMEndpoint.Scheme,
				BindIP: eff.EGMEndpoint.BindIP,
				Host:   eff.EGMEndpoint.Host,
				Port:   eff.EGMEndpoint.Port,
				Path:   eff.EGMEndpoint.Path,
			},
			HostEndpoint: runtimecfg.HostEndpointConfig{URL: eff.HostEndpoint.URL},
			Listen: runtimecfg.ListenConfig{
				Host: eff.ListenHost,
				Port: eff.WirePort,
			},
			Security: runtimecfg.SecurityConfig{
				TrustMode: eff.TrustMode,
				CertFile:  eff.CertFile,
				KeyFile:   eff.KeyFile,
				CAFile:    eff.CAFile,
			},
			Logging: runtimecfg.LoggingConfig{
				Dir:                eff.LogDir,
				CaptureRawXML:      true,
				CaptureRenderedXML: true,
			},
			Storage: runtimecfg.StorageConfig{
				Backend:    eff.StorageBackend,
				SQLitePath: eff.SQLitePath,
			},
			Control: runtimecfg.ControlConfig{
				Bind: fmt.Sprintf("%s:%d", eff.ListenHost, eff.ControlPort),
			},
			PackFile: eff.PackFile,
			Overlay:  eff.OverlayFiles,
			Notes:    notes,
		}
		cfgPath := filepath.Join(outDir, eff.InstanceID+".json")
		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal generated config for %s: %w", eff.InstanceID, err)
		}
		if err := os.WriteFile(cfgPath, append(b, '\n'), 0o644); err != nil {
			return nil, fmt.Errorf("write generated config for %s: %w", eff.InstanceID, err)
		}
		out = append(out, GeneratedConfig{Instance: eff, Path: cfgPath, Config: cfg})
	}
	return out, nil
}
