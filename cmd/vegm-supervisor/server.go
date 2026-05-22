package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/tschneider-imagine/VEGM/fleet"
	"github.com/tschneider-imagine/VEGM/webui"
)

type supervisorServer struct {
	manifestPath string
	generatedDir string
	generated    []fleet.GeneratedConfig
	mu           sync.Mutex
	cmds         map[string]*exec.Cmd
	ctx          context.Context
	cancel       context.CancelFunc
}

type instanceView struct {
	InstanceID     string   `json:"instance_id"`
	EGMID          string   `json:"egm_id"`
	Group          string   `json:"group"`
	Profile        string   `json:"profile"`
	Manufacturer   string   `json:"manufacturer,omitempty"`
	WireURL        string   `json:"wire_url"`
	ControlURL     string   `json:"control_url"`
	UIURL          string   `json:"ui_url"`
	ConfigPath     string   `json:"config_path"`
	Running        bool     `json:"running"`
	Healthy        bool     `json:"healthy"`
	LogDir         string   `json:"log_dir"`
	ListenHost     string   `json:"listen_host"`
	WirePort       int      `json:"wire_port"`
	ControlPort    int      `json:"control_port"`
	AdvertisedHost string   `json:"advertised_host,omitempty"`
	AdvertisedIP   string   `json:"advertised_ip,omitempty"`
	DNSServers     []string `json:"dns_servers,omitempty"`
	SubnetMask     string   `json:"subnet_mask,omitempty"`
	Gateway        string   `json:"gateway,omitempty"`
	ServerName     string   `json:"server_name,omitempty"`
	TrustMode      string   `json:"trust_mode"`
	CertFile       string   `json:"cert_file,omitempty"`
	KeyFile        string   `json:"key_file,omitempty"`
	CAFile         string   `json:"ca_file,omitempty"`
}

func newSupervisorServer(manifestPath, generatedDir string, generated []fleet.GeneratedConfig) *supervisorServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &supervisorServer{manifestPath: manifestPath, generatedDir: generatedDir, generated: generated, cmds: map[string]*exec.Cmd{}, ctx: ctx, cancel: cancel}
}

func (s *supervisorServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/ui", s.handleUIRoot)
	mux.Handle("/ui/", s.uiHandler())
	mux.HandleFunc("/api/instances", s.handleInstances)
	mux.HandleFunc("/api/instances/start-all", s.handleStartAll)
	mux.HandleFunc("/api/instances/stop-all", s.handleStopAll)
	mux.HandleFunc("/api/instances/", s.handleInstanceAction)
	return withSupervisorCORS(mux)
}

func (s *supervisorServer) uiHandler() http.Handler {
	sub, err := fs.Sub(webui.StaticFS, "static")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/ui/", http.FileServer(http.FS(sub)))
}

func withSupervisorCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *supervisorServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instances": len(s.generated)})
}

func (s *supervisorServer) handleUIRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui/supervisor.html", http.StatusTemporaryRedirect)
}

func (s *supervisorServer) handleInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	views := s.instanceViews()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"instances": views})
}

func (s *supervisorServer) handleStartAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var started []string
	for _, gen := range s.generated {
		ok, err := s.startOne(gen.Instance.InstanceID)
		if err == nil && ok {
			started = append(started, gen.Instance.InstanceID)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "started": started})
}

func (s *supervisorServer) handleStopAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var stopped []string
	for _, gen := range s.generated {
		if s.stopOne(gen.Instance.InstanceID) {
			stopped = append(stopped, gen.Instance.InstanceID)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "stopped": stopped})
}

func (s *supervisorServer) handleInstanceAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/instances/"), "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, action := parts[0], parts[1]
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var ok bool
	var err error
	if action == "start" {
		ok, err = s.startOne(id)
	} else if action == "stop" {
		ok = s.stopOne(id)
	} else if action == "restart" {
		s.stopOne(id)
		ok, err = s.startOne(id)
	} else {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": ok, "instance_id": id, "action": action})
}

func (s *supervisorServer) instanceViews() []instanceView {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []instanceView
	for _, gen := range s.generated {
		inst := gen.Instance
		controlURL := fmt.Sprintf("http://%s:%d", inst.ListenHost, inst.ControlPort)
		wireURL := fmt.Sprintf("http://%s:%d", inst.ListenHost, inst.WirePort)
		cmd, running := s.cmds[inst.InstanceID]
		healthy := false
		if running && cmd != nil && cmd.Process != nil {
			healthy, _ = isHealthy(controlURL + "/healthz")
		}
		out = append(out, instanceView{
			InstanceID:     inst.InstanceID,
			EGMID:          inst.EGMID,
			Group:          inst.Group,
			Profile:        inst.Profile,
			Manufacturer:   inst.Manufacturer,
			WireURL:        wireURL,
			ControlURL:     controlURL,
			UIURL:          controlURL + "/ui/scenario-runner.html",
			ConfigPath:     gen.Path,
			Running:        running,
			Healthy:        healthy,
			LogDir:         inst.LogDir,
			ListenHost:     inst.ListenHost,
			WirePort:       inst.WirePort,
			ControlPort:    inst.ControlPort,
			AdvertisedHost: inst.AdvertisedHost,
			AdvertisedIP:   inst.AdvertisedIP,
			DNSServers:     inst.DNSServers,
			SubnetMask:     inst.SubnetMask,
			Gateway:        inst.Gateway,
			ServerName:     inst.ServerName,
			TrustMode:      inst.TrustMode,
			CertFile:       inst.CertFile,
			KeyFile:        inst.KeyFile,
			CAFile:         inst.CAFile,
		})
	}
	return out
}

func (s *supervisorServer) startOne(instanceID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cmd := s.cmds[instanceID]; cmd != nil && cmd.Process != nil {
		return false, nil
	}
	gen, ok := s.generatedByID(instanceID)
	if !ok {
		return false, fmt.Errorf("instance %q not found", instanceID)
	}
	cmd, err := startVEGMProcess(s.ctx, gen.Path)
	if err != nil {
		return false, err
	}
	s.cmds[instanceID] = cmd
	return true, nil
}

func (s *supervisorServer) stopOne(instanceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cmd := s.cmds[instanceID]
	if cmd == nil || cmd.Process == nil {
		return false
	}
	_ = cmd.Process.Kill()
	delete(s.cmds, instanceID)
	return true
}

func (s *supervisorServer) generatedByID(instanceID string) (fleet.GeneratedConfig, bool) {
	for _, gen := range s.generated {
		if gen.Instance.InstanceID == instanceID {
			return gen, true
		}
	}
	return fleet.GeneratedConfig{}, false
}

func (s *supervisorServer) shutdown() {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, cmd := range s.cmds {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		delete(s.cmds, id)
	}
}

func serveSupervisor(bind string, server *supervisorServer) error {
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		return err
	}
	httpServer := &http.Server{Handler: server.routes()}
	go func() { _ = httpServer.Serve(ln) }()
	return nil
}
