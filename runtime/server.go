package runtime

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	packpkg "github.com/tschneider-imagine/VEGM/pack"
	"github.com/tschneider-imagine/VEGM/storage"
)

type RuntimeState struct {
	InstanceID           string                   `json:"instance_id"`
	EGMID                string                   `json:"egm_id"`
	TrustMode            string                   `json:"trust_mode"`
	ConnectionState      string                   `json:"connection_state"`
	RegistrationState    string                   `json:"registration_state"`
	SessionState         string                   `json:"session_state"`
	HeartbeatState       string                   `json:"heartbeat_state"`
	AudioState           string                   `json:"audio_state"`
	HoldState            string                   `json:"hold_state"`
	LockState            string                   `json:"lock_state"`
	MachineState         string                   `json:"machine_state"`
	AllowedHostIDs       []string                 `json:"allowed_host_ids,omitempty"`
	RegisteredHosts      []packpkg.RegisteredHost `json:"registered_hosts,omitempty"`
	LastMessageType      string                   `json:"last_message_type,omitempty"`
	LastCommandType      string                   `json:"last_command_type,omitempty"`
	LastCommandAt        time.Time                `json:"last_command_at,omitempty"`
	LastCommandSource    string                   `json:"last_command_source,omitempty"`
	LastSessionID        string                   `json:"last_session_id,omitempty"`
	LastHostID           string                   `json:"last_host_id,omitempty"`
	LastAckStatus        string                   `json:"last_ack_status,omitempty"`
	LastTransitionAt     time.Time                `json:"last_transition_at,omitempty"`
	LastTransitionReason string                   `json:"last_transition_reason,omitempty"`
	LastError            string                   `json:"last_error,omitempty"`
	StartedAt            time.Time                `json:"started_at"`
	StorageBackend       string                   `json:"storage_backend,omitempty"`
	StorageSQLitePath    string                   `json:"storage_sqlite_path,omitempty"`
	OutboundScheduler    OutboundSchedulerState   `json:"outbound_scheduler"`
}

type Server struct {
	cfg             *Config
	pack            *packpkg.MessagePack
	logger          *Logger
	state           RuntimeState
	mu              sync.RWMutex
	wireSrv         *http.Server
	controlSrv      *http.Server
	wireLn          net.Listener
	controlLn       net.Listener
	schedulerCancel context.CancelFunc
}

func NewServer(cfg *Config) (*Server, error) {
	pk, err := packpkg.LoadPack(cfg.PackFile)
	if err != nil {
		return nil, err
	}
	for _, overlayPath := range cfg.Overlay {
		overlay, err := packpkg.LoadOverlay(overlayPath)
		if err != nil {
			return nil, err
		}
		pk, err = packpkg.ApplyOverlay(pk, overlay)
		if err != nil {
			return nil, err
		}
	}
	backend := cfg.Storage.Backend
	if backend == "" {
		backend = "noop"
	}
	sqlitePath := cfg.Storage.SQLitePath
	if backend == "sqlite" && sqlitePath == "" {
		sqlitePath = storage.DefaultSQLitePath(cfg.Logging.Dir)
	}
	idx, err := storage.NewIndex(backend, sqlitePath)
	if err != nil {
		return nil, err
	}
	logger, err := NewLoggerWithIndex(cfg.Logging.Dir, cfg.InstanceID, idx)
	if err != nil {
		return nil, err
	}
	allowed := append([]string(nil), pk.StateDefaults.AllowedHostIDs...)
	sort.Strings(allowed)
	registrationState := "open"
	if pk.StateDefaults.RegistrationMode != "open" {
		registrationState = "restricted"
	}
	return &Server{
		cfg:    cfg,
		pack:   pk,
		logger: logger,
		state: RuntimeState{
			InstanceID:        cfg.InstanceID,
			EGMID:             cfg.EGMID,
			TrustMode:         cfg.Security.TrustMode,
			ConnectionState:   "listening",
			RegistrationState: registrationState,
			SessionState:      pk.StateDefaults.SessionState,
			HeartbeatState:    pk.StateDefaults.HeartbeatState,
			AudioState:        "normal",
			HoldState:         "inactive",
			LockState:         "inactive",
			MachineState:      "available",
			AllowedHostIDs:    allowed,
			RegisteredHosts:   append([]packpkg.RegisteredHost(nil), pk.StateDefaults.RegisteredHosts...),
			StartedAt:         time.Now().UTC(),
			StorageBackend:    backend,
			StorageSQLitePath: sqlitePath,
		},
	}, nil
}

func (s *Server) WireAddr() string {
	if s.wireLn != nil {
		return s.wireLn.Addr().String()
	}
	return net.JoinHostPort(s.cfg.Listen.Host, fmt.Sprint(s.cfg.Listen.Port))
}

func (s *Server) ControlAddr() string {
	if s.controlLn != nil {
		return s.controlLn.Addr().String()
	}
	return s.cfg.Control.Bind
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.startControl(); err != nil {
		return err
	}
	if err := s.startWire(); err != nil {
		return err
	}
	if s.cfg.Outbound.Scheduler.Enabled && s.cfg.Outbound.Scheduler.AutoStart {
		_ = s.StartOutboundScheduler()
	}
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()
	s.logger.Log("info", "server", "VEGM started", map[string]any{"wire_addr": s.WireAddr(), "control_addr": s.ControlAddr(), "trust_mode": s.cfg.Security.TrustMode, "storage_backend": s.state.StorageBackend})
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.StopOutboundScheduler()
	if s.controlSrv != nil {
		_ = s.controlSrv.Shutdown(ctx)
	}
	if s.wireSrv != nil {
		_ = s.wireSrv.Shutdown(ctx)
	}
	if s.logger != nil {
		_ = s.logger.Close()
	}
	return nil
}

func (s *Server) startControl() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/ui", s.handleScenarioUIRoot)
	mux.Handle("/ui/", scenarioUIHandler())
	mux.HandleFunc("/control/state", s.handleControlState)
	mux.HandleFunc("/control/state/history", s.handleControlStateHistory)
	mux.HandleFunc("/control/audio", s.handleControlAudio)
	mux.HandleFunc("/control/machine-status", s.handleControlMachineStatus)
	mux.HandleFunc("/control/logs", s.handleControlLogs)
	mux.HandleFunc("/control/export", s.handleControlExport)
	mux.HandleFunc("/control/overlay", s.handleControlOverlay)
	mux.HandleFunc("/control/hosts/add", s.handleControlHostsAdd)
	mux.HandleFunc("/control/hosts/remove", s.handleControlHostsRemove)
	mux.HandleFunc("/control/security/mode", s.handleControlSecurityMode)
	mux.HandleFunc("/control/security/reload", s.handleControlSecurityReload)
	mux.HandleFunc("/control/pack/summary", s.handleControlPackSummary)
	mux.HandleFunc("/control/pack/operations", s.handleControlPackOperations)
	mux.HandleFunc("/control/inject-logical-command", s.handleControlInjectLogicalCommand)
	mux.HandleFunc("/control/outbound/session/open", s.handleControlOutboundSessionOpen)
	mux.HandleFunc("/control/outbound/heartbeat", s.handleControlOutboundHeartbeat)
	mux.HandleFunc("/control/outbound/send", s.handleControlOutboundSend)
	mux.HandleFunc("/control/outbound/scheduler/start", s.handleControlOutboundSchedulerStart)
	mux.HandleFunc("/control/outbound/scheduler/stop", s.handleControlOutboundSchedulerStop)
	ln, err := net.Listen("tcp", s.cfg.Control.Bind)
	if err != nil {
		return err
	}
	s.controlLn = ln
	s.controlSrv = &http.Server{Handler: withControlCORS(mux)}
	go func() { _ = s.controlSrv.Serve(ln) }()
	return nil
}

func (s *Server) startWire() error {
	mux := http.NewServeMux()
	path := s.cfg.EGMEndpoint.Path
	if path == "" {
		path = s.pack.Wire.Paths.DefaultListenerPath
	}
	mux.HandleFunc(path, s.handleWire)
	for _, p := range s.pack.Wire.Paths.AlternateListenerPaths {
		if p != path {
			mux.HandleFunc(p, s.handleWire)
		}
	}
	addr := net.JoinHostPort(s.cfg.Listen.Host, fmt.Sprint(s.cfg.Listen.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.wireLn = ln
	s.wireSrv = &http.Server{Handler: mux}
	if s.cfg.Security.TrustMode == "plaintext_lab" {
		go func() { _ = s.wireSrv.Serve(ln) }()
		return nil
	}
	tlsCfg, err := s.makeWireTLSConfig()
	if err != nil {
		return err
	}
	go func() { _ = s.wireSrv.Serve(tls.NewListener(ln, tlsCfg)) }()
	return nil
}

func (s *Server) makeWireTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(s.cfg.Security.CertFile, s.cfg.Security.KeyFile)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}
	switch s.cfg.Security.TrustMode {
	case "tls_server_only", "accept_all_lab":
		return cfg, nil
	case "strict_mtls", "mtls_no_revocation":
		caPEM, err := os.ReadFile(s.cfg.Security.CAFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caPEM)
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		return cfg, nil
	default:
		return nil, fmt.Errorf("unsupported trust mode %q", s.cfg.Security.TrustMode)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instance_id": s.cfg.InstanceID})
}

func (s *Server) handleControlState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.state)
}

func (s *Server) handleControlStateHistory(w http.ResponseWriter, r *http.Request) {
	events, err := s.logger.QueryEvents(EventFilter{Category: "state", Limit: 200})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events)
}

func (s *Server) handleControlAudio(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"instance_id":       s.state.InstanceID,
		"egm_id":            s.state.EGMID,
		"audio_state":       s.state.AudioState,
		"hold_state":        s.state.HoldState,
		"lock_state":        s.state.LockState,
		"last_command_type": s.state.LastCommandType,
		"last_command_at":   s.state.LastCommandAt,
	})
}

func (s *Server) handleControlMachineStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"instance_id":            s.state.InstanceID,
		"egm_id":                 s.state.EGMID,
		"machine_state":          s.state.MachineState,
		"connection_state":       s.state.ConnectionState,
		"registration_state":     s.state.RegistrationState,
		"session_state":          s.state.SessionState,
		"heartbeat_state":        s.state.HeartbeatState,
		"last_message_type":      s.state.LastMessageType,
		"last_transition_at":     s.state.LastTransitionAt,
		"last_transition_reason": s.state.LastTransitionReason,
	})
}

func (s *Server) handleControlLogs(w http.ResponseWriter, r *http.Request) {
	filter := EventFilter{Category: r.URL.Query().Get("category"), Level: r.URL.Query().Get("level"), Contains: r.URL.Query().Get("contains"), MessageType: r.URL.Query().Get("message_type")}
	events, err := s.logger.QueryEvents(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(events)
}

func (s *Server) handleControlExport(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	state := s.state
	cfg := s.cfg
	pkSummary := s.pack.Summary()
	s.mu.RUnlock()
	path, err := s.logger.ExportBundle(ExportOptions{StateSnapshot: state, ConfigSnapshot: cfg, PackSummary: pkSummary})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "path": path})
}

func (s *Server) handleControlOverlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var overlay packpkg.MessageOverlay
	if err := json.NewDecoder(r.Body).Decode(&overlay); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pk, err := packpkg.ApplyOverlay(s.pack, &overlay)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.pack = pk
	s.logger.Log("info", "control", "overlay applied", map[string]any{"overlay_name": overlay.OverlayName})
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pack_summary": s.pack.Summary()})
}

func (s *Server) handleControlHostsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct{ HostID string `json:"host_id"` }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.HostID == "" {
		http.Error(w, "host_id is required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.state.AllowedHostIDs {
		if v == in.HostID {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "allowed_host_ids": s.state.AllowedHostIDs})
			return
		}
	}
	s.state.AllowedHostIDs = append(s.state.AllowedHostIDs, in.HostID)
	s.state.RegisteredHosts = append(s.state.RegisteredHosts, packpkg.RegisteredHost{HostID: in.HostID, Role: "guest", Enabled: true})
	sort.Strings(s.state.AllowedHostIDs)
	s.state.RegistrationState = "restricted"
	s.logger.Log("info", "control", "host added", map[string]any{"host_id": in.HostID})
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "allowed_host_ids": s.state.AllowedHostIDs, "registered_hosts": s.state.RegisteredHosts})
}

func (s *Server) handleControlHostsRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct{ HostID string `json:"host_id"` }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.HostID == "" {
		http.Error(w, "host_id is required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.state.AllowedHostIDs[:0]
	for _, v := range s.state.AllowedHostIDs {
		if v != in.HostID {
			out = append(out, v)
		}
	}
	s.state.AllowedHostIDs = out
	hosts := s.state.RegisteredHosts[:0]
	for _, h := range s.state.RegisteredHosts {
		if h.HostID != in.HostID {
			hosts = append(hosts, h)
		}
	}
	s.state.RegisteredHosts = hosts
	if len(s.state.AllowedHostIDs) == 0 && s.pack.StateDefaults.RegistrationMode == "open" {
		s.state.RegistrationState = "open"
	}
	s.logger.Log("info", "control", "host removed", map[string]any{"host_id": in.HostID})
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "allowed_host_ids": s.state.AllowedHostIDs, "registered_hosts": s.state.RegisteredHosts})
}

func (s *Server) handleControlSecurityMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct{ TrustMode string `json:"trust_mode"` }
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.TrustMode == "" {
		http.Error(w, "trust_mode is required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Security.TrustMode = in.TrustMode
	s.state.TrustMode = in.TrustMode
	s.logger.Log("info", "control", "trust mode changed", map[string]any{"trust_mode": in.TrustMode, "note": "wire restart not yet automatic in repo seed"})
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "trust_mode": in.TrustMode, "note": "wire restart not yet automatic in repo seed"})
}

func (s *Server) handleControlSecurityReload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.logger.Log("info", "control", "security reload requested", nil)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "note": "security reload hook recorded"})
}

func (s *Server) handleControlPackSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.pack.Summary())
}

func (s *Server) handleControlPackOperations(w http.ResponseWriter, r *http.Request) {
	ops := make([]string, 0, len(s.pack.Operations))
	for name := range s.pack.Operations {
		ops = append(ops, name)
	}
	sort.Strings(ops)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"operations": ops})
}

func (s *Server) handleWire(w http.ResponseWriter, r *http.Request) {
	binding := validateG2SBindingRequest(r)
	if !binding.OK {
		s.logger.Log("warn", "binding", "g2s binding rejected", map[string]any{"method": r.Method, "path": r.URL.Path, "content_type": r.Header.Get("Content-Type"), "status": binding.StatusCode, "reason": binding.Message})
		http.Error(w, binding.Message, binding.StatusCode)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parsed, err := ParseG2SMessage(body)
	if err != nil {
		if s.cfg.Logging.CaptureRawXML {
			_, _ = s.logger.WritePayload("inbound_bad_soap", "invalid", body)
		}
		s.logger.Log("warn", "soap", "soap parse failed", map[string]any{"error": err.Error()})
		http.Error(w, "invalid soap xml", http.StatusBadRequest)
		return
	}
	if s.cfg.Logging.CaptureRawXML {
		_, _ = s.logger.WritePayload("inbound_request", parsed.RootLocalName, body)
	}
	hostID := firstNonEmpty(parsed.Fields["hostId"], parsed.Fields["host_id"], parsed.Fields["hostID"])
	sessionID := firstNonEmpty(parsed.Fields["sessionId"], parsed.Fields["session_id"])
	if !s.hostAllowed(hostID) {
		http.Error(w, "host is not registered", http.StatusForbidden)
		s.mu.Lock()
		s.state.ConnectionState = "controller_connected"
		s.state.LastError = "host is not registered"
		s.state.LastCommandType = parsed.RootLocalName
		s.state.LastCommandAt = time.Now().UTC()
		s.state.LastCommandSource = hostID
		s.mu.Unlock()
		s.logger.Log("warn", "wire", "unregistered host rejected", map[string]any{"host_id": hostID, "message_type": parsed.RootLocalName})
		return
	}
	opName, op := s.findOperation(parsed.RootLocalName)
	if op == nil || len(op.Responses) == 0 {
		http.Error(w, "operation not supported", http.StatusNotFound)
		s.mu.Lock()
		s.state.ConnectionState = "controller_connected"
		s.state.LastError = "operation not supported"
		s.state.LastCommandType = parsed.RootLocalName
		s.state.LastCommandAt = time.Now().UTC()
		s.state.LastCommandSource = hostID
		s.mu.Unlock()
		s.logger.Log("warn", "wire", "unsupported operation", map[string]any{"message_type": parsed.RootLocalName, "namespace": parsed.RootNamespace})
		return
	}
	s.mu.Lock()
	s.state.ConnectionState = "controller_connected"
	s.state.LastMessageType = parsed.RootLocalName
	s.state.LastCommandType = parsed.RootLocalName
	s.state.LastCommandAt = time.Now().UTC()
	s.state.LastCommandSource = hostID
	s.state.LastSessionID = sessionID
	s.state.LastHostID = hostID
	s.state.LastError = ""
	if parsed.RootLocalName == "commsOnLine" {
		s.applyStateUpdatesLocked(map[string]any{"session_state": "online", "heartbeat_state": "healthy"}, "inbound commsOnLine")
	}
	if parsed.RootLocalName == "keepAlive" {
		s.applyStateUpdatesLocked(map[string]any{"heartbeat_state": "healthy"}, "inbound keepAlive")
	}
	if parsed.RootLocalName == "commsClosing" {
		s.applyStateUpdatesLocked(map[string]any{"session_state": "closed"}, "inbound commsClosing")
	}
	stateValues := s.templateStateLocked()
	variant := op.Responses[0]
	if len(variant.SetState) > 0 {
		s.applyStateUpdatesLocked(variant.SetState, fmt.Sprintf("operation %s", parsed.RootLocalName))
		stateValues = s.templateStateLocked()
	}
	s.mu.Unlock()
	requestFields := map[string]string{"hostId": hostID, "sessionId": sessionID, "egmId": s.cfg.EGMID}
	respBody := RenderTemplate(variant.Template, s.pack.Wire.Namespaces, requestFields, stateValues)
	if delay := maxInt(variant.DelayMS, s.pack.Timers.ArtificialResponseDelayMS); delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("outbound_response", opName, []byte(respBody))
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	status := variant.HTTPStatus
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(respBody))
	s.mu.Lock()
	s.state.LastAckStatus = fmt.Sprintf("http_%d", status)
	s.mu.Unlock()
	s.logger.Log("info", "wire", "operation handled", map[string]any{"message_type": parsed.RootLocalName, "namespace": parsed.RootNamespace, "host_id": hostID, "session_id": sessionID, "status": status})
}

func (s *Server) hostAllowed(hostID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mode := s.pack.StateDefaults.RegistrationMode
	if mode == "open" {
		return true
	}
	if hostID == "" {
		return false
	}
	if len(s.state.RegisteredHosts) > 0 {
		for _, h := range s.state.RegisteredHosts {
			if h.HostID == hostID {
				return h.Enabled
			}
		}
		return false
	}
	for _, v := range s.state.AllowedHostIDs {
		if v == hostID {
			return true
		}
	}
	return false
}

func (s *Server) findOperation(root string) (string, *packpkg.Operation) {
	if op, ok := s.pack.Operations[root]; ok {
		return root, &op
	}
	for name, op := range s.pack.Operations {
		for _, m := range op.Match {
			if m.Kind == "message_type" && strings.EqualFold(m.Value, root) {
				copy := op
				return name, &copy
			}
		}
	}
	return "", nil
}

func (s *Server) templateStateLocked() map[string]any {
	return map[string]any{
		"instance_id":      s.state.InstanceID,
		"egm_id":           s.state.EGMID,
		"trust_mode":       s.state.TrustMode,
		"session_state":    s.state.SessionState,
		"heartbeat_state":  s.state.HeartbeatState,
		"audio_state":      s.state.AudioState,
		"hold_state":       s.state.HoldState,
		"lock_state":       s.state.LockState,
		"machine_state":    s.state.MachineState,
		"connection_state": s.state.ConnectionState,
	}
}

func (s *Server) applyStateUpdatesLocked(updates map[string]any, reason string) {
	if len(updates) == 0 {
		return
	}
	changed := map[string]any{}
	for key, value := range updates {
		newValue := fmt.Sprint(value)
		switch key {
		case "connection_state":
			if s.state.ConnectionState != newValue {
				changed[key] = map[string]string{"old": s.state.ConnectionState, "new": newValue}
				s.state.ConnectionState = newValue
			}
		case "registration_state":
			if s.state.RegistrationState != newValue {
				changed[key] = map[string]string{"old": s.state.RegistrationState, "new": newValue}
				s.state.RegistrationState = newValue
			}
		case "session_state":
			if s.state.SessionState != newValue {
				changed[key] = map[string]string{"old": s.state.SessionState, "new": newValue}
				s.state.SessionState = newValue
			}
		case "heartbeat_state":
			if s.state.HeartbeatState != newValue {
				changed[key] = map[string]string{"old": s.state.HeartbeatState, "new": newValue}
				s.state.HeartbeatState = newValue
			}
		case "audio_state":
			if s.state.AudioState != newValue {
				changed[key] = map[string]string{"old": s.state.AudioState, "new": newValue}
				s.state.AudioState = newValue
			}
		case "hold_state":
			if s.state.HoldState != newValue {
				changed[key] = map[string]string{"old": s.state.HoldState, "new": newValue}
				s.state.HoldState = newValue
			}
		case "lock_state":
			if s.state.LockState != newValue {
				changed[key] = map[string]string{"old": s.state.LockState, "new": newValue}
				s.state.LockState = newValue
			}
		case "machine_state":
			if s.state.MachineState != newValue {
				changed[key] = map[string]string{"old": s.state.MachineState, "new": newValue}
				s.state.MachineState = newValue
			}
		case "last_ack_status":
			s.state.LastAckStatus = newValue
		}
	}
	if len(changed) > 0 {
		s.state.LastTransitionAt = time.Now().UTC()
		s.state.LastTransitionReason = reason
		s.logger.Log("info", "state", "state changed", map[string]any{
			"reason":       reason,
			"changes":      changed,
			"message_type": s.state.LastCommandType,
			"host_id":      s.state.LastHostID,
			"session_id":   s.state.LastSessionID,
		})
	}
}
