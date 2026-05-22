package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tschneider-imagine/VEGM/fleet"
)

type childIndicators struct {
	SessionState    string `json:"session_state,omitempty"`
	HeartbeatState  string `json:"heartbeat_state,omitempty"`
	ConnectionState string `json:"connection_state,omitempty"`
	AudioState      string `json:"audio_state,omitempty"`
	HoldState       string `json:"hold_state,omitempty"`
	LockState       string `json:"lock_state,omitempty"`
	MachineState    string `json:"machine_state,omitempty"`
	LastCommandType string `json:"last_command_type,omitempty"`
}

type instanceSettings struct {
	InstanceID      string             `json:"instance_id"`
	HostID          string             `json:"host_id"`
	EGMEndpoint     fleet.Endpoint     `json:"egm_endpoint"`
	HostEndpoint    fleet.HostEndpoint `json:"host_endpoint,omitempty"`
	ListenHost      string             `json:"listen_host"`
	WirePort        int                `json:"wire_port"`
	ControlPort     int                `json:"control_port"`
	AdvertisedHost  string             `json:"advertised_host,omitempty"`
	AdvertisedIP    string             `json:"advertised_ip,omitempty"`
	DNSServers      []string           `json:"dns_servers,omitempty"`
	SubnetMask      string             `json:"subnet_mask,omitempty"`
	Gateway         string             `json:"gateway,omitempty"`
	ServerName      string             `json:"server_name,omitempty"`
	TrustMode       string             `json:"trust_mode"`
	CertFile        string             `json:"cert_file,omitempty"`
	KeyFile         string             `json:"key_file,omitempty"`
	CAFile          string             `json:"ca_file,omitempty"`
}

func fetchChildIndicators(controlBase string) (childIndicators, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(controlBase + "/control/state")
	if err != nil {
		return childIndicators{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return childIndicators{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var out childIndicators
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return childIndicators{}, err
	}
	return out, nil
}

func (s *supervisorServer) handleInstanceSettings(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/instance-settings/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet {
		settings, err := s.getInstanceSettings(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings)
		return
	}
	if r.Method == http.MethodPost {
		var in instanceSettings
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated, err := s.updateInstanceSettings(id, in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "settings": updated})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *supervisorServer) getInstanceSettings(id string) (instanceSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, gen := range s.generated {
		if gen.Instance.InstanceID == id {
			inst := gen.Instance
			return instanceSettings{
				InstanceID:     inst.InstanceID,
				HostID:         inst.HostID,
				EGMEndpoint:    inst.EGMEndpoint,
				HostEndpoint:   inst.HostEndpoint,
				ListenHost:     inst.ListenHost,
				WirePort:       inst.WirePort,
				ControlPort:    inst.ControlPort,
				AdvertisedHost: inst.AdvertisedHost,
				AdvertisedIP:   inst.AdvertisedIP,
				DNSServers:     append([]string(nil), inst.DNSServers...),
				SubnetMask:     inst.SubnetMask,
				Gateway:        inst.Gateway,
				ServerName:     inst.ServerName,
				TrustMode:      inst.TrustMode,
				CertFile:       inst.CertFile,
				KeyFile:        inst.KeyFile,
				CAFile:         inst.CAFile,
			}, nil
		}
	}
	return instanceSettings{}, fmt.Errorf("instance %q not found", id)
}

func (s *supervisorServer) updateInstanceSettings(id string, in instanceSettings) (instanceSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.generated {
		if s.generated[i].Instance.InstanceID != id {
			continue
		}
		gen := &s.generated[i]
		inst := &gen.Instance
		cfg := &gen.Config

		inst.HostID = in.HostID
		inst.EGMEndpoint = in.EGMEndpoint
		inst.HostEndpoint = in.HostEndpoint
		inst.ListenHost = in.ListenHost
		inst.WirePort = in.WirePort
		inst.ControlPort = in.ControlPort
		inst.AdvertisedHost = in.AdvertisedHost
		inst.AdvertisedIP = in.AdvertisedIP
		inst.DNSServers = append([]string(nil), in.DNSServers...)
		inst.SubnetMask = in.SubnetMask
		inst.Gateway = in.Gateway
		inst.ServerName = in.ServerName
		inst.TrustMode = in.TrustMode
		inst.CertFile = in.CertFile
		inst.KeyFile = in.KeyFile
		inst.CAFile = in.CAFile

		cfg.HostID = in.HostID
		cfg.EGMEndpoint.Scheme = in.EGMEndpoint.Scheme
		cfg.EGMEndpoint.BindIP = in.EGMEndpoint.BindIP
		cfg.EGMEndpoint.Host = in.EGMEndpoint.Host
		cfg.EGMEndpoint.Port = in.EGMEndpoint.Port
		cfg.EGMEndpoint.Path = in.EGMEndpoint.Path
		cfg.HostEndpoint.URL = in.HostEndpoint.URL
		cfg.Listen.Host = in.ListenHost
		cfg.Listen.Port = in.WirePort
		cfg.Control.Bind = fmt.Sprintf("%s:%d", in.ListenHost, in.ControlPort)
		cfg.Security.TrustMode = in.TrustMode
		cfg.Security.CertFile = in.CertFile
		cfg.Security.KeyFile = in.KeyFile
		cfg.Security.CAFile = in.CAFile
		if cfg.Notes == nil {
			cfg.Notes = map[string]string{}
		}
		cfg.Notes["host_id"] = in.HostID
		cfg.Notes["egm_endpoint_scheme"] = in.EGMEndpoint.Scheme
		cfg.Notes["egm_endpoint_bind_ip"] = in.EGMEndpoint.BindIP
		cfg.Notes["egm_endpoint_host"] = in.EGMEndpoint.Host
		cfg.Notes["egm_endpoint_path"] = in.EGMEndpoint.Path
		cfg.Notes["host_endpoint_url"] = in.HostEndpoint.URL
		cfg.Notes["advertised_host"] = in.AdvertisedHost
		cfg.Notes["advertised_ip"] = in.AdvertisedIP
		cfg.Notes["dns_servers"] = strings.Join(in.DNSServers, ",")
		cfg.Notes["subnet_mask"] = in.SubnetMask
		cfg.Notes["gateway"] = in.Gateway
		cfg.Notes["server_name"] = in.ServerName
		cfg.Notes["cert_file"] = in.CertFile
		cfg.Notes["key_file"] = in.KeyFile
		cfg.Notes["ca_file"] = in.CAFile

		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return instanceSettings{}, err
		}
		if err := os.WriteFile(gen.Path, append(b, '\n'), 0o644); err != nil {
			return instanceSettings{}, err
		}
		return instanceSettings{
			InstanceID:     inst.InstanceID,
			HostID:         inst.HostID,
			EGMEndpoint:    inst.EGMEndpoint,
			HostEndpoint:   inst.HostEndpoint,
			ListenHost:     inst.ListenHost,
			WirePort:       inst.WirePort,
			ControlPort:    inst.ControlPort,
			AdvertisedHost: inst.AdvertisedHost,
			AdvertisedIP:   inst.AdvertisedIP,
			DNSServers:     append([]string(nil), inst.DNSServers...),
			SubnetMask:     inst.SubnetMask,
			Gateway:        inst.Gateway,
			ServerName:     inst.ServerName,
			TrustMode:      inst.TrustMode,
			CertFile:       inst.CertFile,
			KeyFile:        inst.KeyFile,
			CAFile:         inst.CAFile,
		}, nil
	}
	return instanceSettings{}, fmt.Errorf("instance %q not found", id)
}
