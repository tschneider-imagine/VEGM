package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"sync"

	"github.com/tschneider-imagine/VEGM/fleet"
	runtimecfg "github.com/tschneider-imagine/VEGM/runtime"
	"github.com/tschneider-imagine/VEGM/webui"
)

type supervisorServer struct {
	manifestPath string
	generatedDir string
	generated    []fleet.GeneratedConfig
	mu           sync.Mutex
	cmds         map[string]*exec.Cmd
	restart      map[string]restartMeta
	ctx          context.Context
	cancel       context.CancelFunc
}

type instanceView struct {
	InstanceID        string                       `json:"instance_id"`
	EGMID             string                       `json:"egm_id"`
	HostID            string                       `json:"host_id"`
	Group             string                       `json:"group"`
	Profile           string                       `json:"profile"`
	Manufacturer      string                       `json:"manufacturer,omitempty"`
	EGMEndpoint       fleet.Endpoint               `json:"egm_endpoint"`
	HostEndpoint      fleet.HostEndpoint           `json:"host_endpoint,omitempty"`
	G2SXML            runtimecfg.G2SXMLConfig      `json:"g2s_xml,omitempty"`
	WireURL           string                       `json:"wire_url"`
	ControlURL        string                       `json:"control_url"`
	UIURL             string                       `json:"ui_url"`
	ConfigPath        string                       `json:"config_path"`
	Running           bool                         `json:"running"`
	Healthy           bool                         `json:"healthy"`
	RestartDesired    bool                         `json:"restart_desired"`
	RestartCount      int                          `json:"restart_count"`
	LastExit          string                       `json:"last_exit,omitempty"`
	LogDir            string                       `json:"log_dir"`
	ListenHost        string                       `json:"listen_host"`
	WirePort          int                          `json:"wire_port"`
	ControlPort       int                          `json:"control_port"`
	AdvertisedHost    string                       `json:"advertised_host,omitempty"`
	AdvertisedIP      string                       `json:"advertised_ip,omitempty"`
	DNSServers        []string                     `json:"dns_servers,omitempty"`
	SubnetMask        string                       `json:"subnet_mask,omitempty"`
	Gateway           string                       `json:"gateway,omitempty"`
	ServerName        string                       `json:"server_name,omitempty"`
	TrustMode         string                       `json:"trust_mode"`
	CertFile          string                       `json:"cert_file,omitempty"`
	KeyFile           string                       `json:"key_file,omitempty"`
	CAFile            string                       `json:"ca_file,omitempty"`
	CertificateStatus runtimecfg.CertificateStatus `json:"certificate_status"`
	SessionState      string                       `json:"session_state,omitempty"`
	HeartbeatState    string                       `json:"heartbeat_state,omitempty"`
	ConnectionState   string                       `json:"connection_state,omitempty"`
	AudioState        string                       `json:"audio_state,omitempty"`
	HoldState         string                       `json:"hold_state,omitempty"`
	LockState         string                       `json:"lock_state,omitempty"`
	MachineState      string                       `json:"machine_state,omitempty"`
	LastCommandType   string                       `json:"last_command_type,omitempty"`
}

func newSupervisorServer(manifestPath, generatedDir string, generated []fleet.GeneratedConfig) *supervisorServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &supervisorServer{manifestPath: manifestPath, generatedDir: generatedDir, generated: generated, cmds: map[string]*exec.Cmd{}, restart: map[string]restartMeta{}, ctx: ctx, cancel: cancel}
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
	mux.HandleFunc("/api/instance-settings/", s.handleInstanceSettings)
	mux.HandleFunc("/api/fleet/instances/add", s.handleFleetInstancesAdd)
	mux.HandleFunc("/api/export/fleet", s.handleFleetExport)
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
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			started = append(started, gen.Instance.InstanceID)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"started": started, "instances": s.instanceViews()})
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
	_ = json.NewEncoder(w).Encode(map[string]any{"stopped": stopped, "instances": s.instanceViews()})
}

func (s *supervisorServer) instanceViews() []instanceView {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]instanceView, 0, len(s.generated))
	for _, gen := range s.generated {
		inst := gen.Instance
		wireURL := fmt.Sprintf("%s://%s:%d%s", inst.EGMEndpoint.Scheme, inst.EGMEndpoint.Host, inst.WirePort, inst.EGMEndpoint.Path)
		controlURL := fmt.Sprintf("http://%s:%d", inst.ListenHost, inst.ControlPort)
		cmd := s.cmds[inst.InstanceID]
		running := processRunning(cmd)
		meta := s.restartMeta(inst.InstanceID)
		healthy := false
		indicators := childIndicators{}
		if running {
			healthy, _ = isHealthy(controlURL + "/healthz")
			if healthy {
				indicators, _ = fetchChildIndicators(controlURL)
			}
		}
		out = append(out, instanceView{
			InstanceID:        inst.InstanceID,
			EGMID:             inst.EGMID,
			HostID:            inst.HostID,
			Group:             inst.Group,
			Profile:           inst.Profile,
			Manufacturer:      inst.Manufacturer,
			EGMEndpoint:       inst.EGMEndpoint,
			HostEndpoint:      inst.HostEndpoint,
			G2SXML:            gen.Config.G2SXML,
			WireURL:           wireURL,
			ControlURL:        controlURL,
			UIURL:             controlURL + "/ui/scenario-runner.html",
			ConfigPath:        gen.Path,
			Running:           running,
			Healthy:           healthy,
			RestartDesired:    meta.Desired,
			RestartCount:      meta.RestartCount,
			LastExit:          meta.LastExit,
			LogDir:            inst.LogDir,
			ListenHost:        inst.ListenHost,
			WirePort:          inst.WirePort,
			ControlPort:       inst.ControlPort,
			AdvertisedHost:    inst.AdvertisedHost,
			AdvertisedIP:      inst.AdvertisedIP,
			DNSServers:        inst.DNSServers,
			SubnetMask:        inst.SubnetMask,
			Gateway:           inst.Gateway,
			ServerName:        inst.ServerName,
			TrustMode:         inst.TrustMode,
			CertFile:          inst.CertFile,
			KeyFile:           inst.KeyFile,
			CAFile:            inst.CAFile,
			CertificateStatus: runtimecfg.BuildCertificateStatus(&gen.Config),
			SessionState:      indicators.SessionState,
			HeartbeatState:    indicators.HeartbeatState,
			ConnectionState:   indicators.ConnectionState,
			AudioState:        indicators.AudioState,
			HoldState:         indicators.HoldState,
			LockState:         indicators.LockState,
			MachineState:      indicators.MachineState,
			LastCommandType:   indicators.LastCommandType,
		})
	}
	return out
}

func (s *supervisorServer) startOne(instanceID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cmd := s.cmds[instanceID]; processRunning(cmd) {
		return false, nil
	}
	delete(s.cmds, instanceID)
	gen, ok := s.generatedByID(instanceID)
	if !ok {
		return false, fmt.Errorf("instance %q not found", instanceID)
	}
	cmd, err := startVEGMProcessForScale(s.ctx, gen.Path, gen.Instance.LogDir)
	if err != nil {
		return false, err
	}
	s.cmds[instanceID] = cmd
	s.markDesired(instanceID, true)
	go s.monitorChild(instanceID)
	return true, nil
}

func (s *supervisorServer) stopOne(instanceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.markDesired(instanceID, false)
	cmd := s.cmds[instanceID]
	if cmd == nil || cmd.Process == nil {
		delete(s.cmds, instanceID)
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
		if s.restart != nil {
			meta := s.restart[id]
			meta.Desired = false
			s.restart[id] = meta
		}
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
