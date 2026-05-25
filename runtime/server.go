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
	mux.HandleFunc("/control/evidence/latest", s.handleControlEvidenceLatest)
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