package runtime

import (
	"encoding/json"
	"net/http"
	"time"

	packpkg "github.com/tschneider-imagine/VEGM/pack"
)

type audioControlRequest struct {
	Muted bool `json:"muted"`
}

type securityModeRequest struct {
	TrustMode string `json:"trust_mode"`
}

type hostControlRequest struct {
	HostID string `json:"host_id"`
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instance_id": s.cfg.InstanceID})
}

func (s *Server) handleControlState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) handleControlStateHistory(w http.ResponseWriter, r *http.Request) {
	events, err := s.logger.QueryEvents(EventFilter{Category: "state", Limit: 100})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"events": events})
}

func (s *Server) handleControlAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req audioControlRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.mu.Lock()
	if req.Muted {
		s.applyStateUpdatesLocked(map[string]any{"audio_state": "muted"}, "control audio muted")
	} else {
		s.applyStateUpdatesLocked(map[string]any{"audio_state": "normal"}, "control audio normal")
	}
	state := s.state
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) handleControlMachineStatus(w http.ResponseWriter, r *http.Request) {
	s.handleControlState(w, r)
}

func (s *Server) handleControlLogs(w http.ResponseWriter, r *http.Request) {
	events, err := s.logger.QueryEvents(EventFilter{Limit: 200})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"events": events})
}

func (s *Server) handleControlExport(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	state := s.state
	cfg := s.cfg
	s.mu.RUnlock()
	path, err := s.logger.ExportBundle(ExportOptions{IncludePayloads: true, StateSnapshot: state, ConfigSnapshot: cfg})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	updated, err := packpkg.ApplyOverlay(s.pack, &overlay)
	if err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.pack = updated
	if updated.StateDefaults.RegistrationMode == "open" {
		s.state.RegistrationState = "open"
	} else {
		s.state.RegistrationState = "restricted"
	}
	state := s.state
	s.mu.Unlock()
	s.logger.Log("info", "control", "overlay applied", map[string]any{"overlay_name": overlay.OverlayName, "target_pack": overlay.TargetPack})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pack_name": updated.PackName, "registration_state": state.RegistrationState})
}

func (s *Server) handleControlHostsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HostID == "" {
		http.Error(w, "host_id is required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.state.AllowedHostIDs = append(s.state.AllowedHostIDs, req.HostID)
	state := s.state
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) handleControlHostsRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.HostID == "" {
		http.Error(w, "host_id is required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	out := s.state.AllowedHostIDs[:0]
	for _, id := range s.state.AllowedHostIDs {
		if id != req.HostID {
			out = append(out, id)
		}
	}
	s.state.AllowedHostIDs = out
	state := s.state
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) handleControlSecurityMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req securityModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TrustMode == "" {
		http.Error(w, "trust_mode is required", http.StatusBadRequest)
		return
	}
	s.cfg.Security.TrustMode = req.TrustMode
	s.mu.Lock()
	s.state.TrustMode = req.TrustMode
	state := s.state
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(state)
}

func (s *Server) handleControlSecurityReload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(BuildCertificateStatus(s.cfg))
}

func (s *Server) handleControlPackSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"pack_name": s.pack.PackName, "pack_version": s.pack.PackVersion, "operations": len(s.pack.Operations)})
}

func (s *Server) handleControlPackOperations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.pack.Operations)
}

func (s *Server) applyStateUpdatesLocked(updates map[string]any, reason string) {
	if len(updates) == 0 {
		return
	}
	if v, ok := updates["audio_state"].(string); ok {
		s.state.AudioState = v
	}
	if v, ok := updates["hold_state"].(string); ok {
		s.state.HoldState = v
	}
	if v, ok := updates["lock_state"].(string); ok {
		s.state.LockState = v
	}
	if v, ok := updates["machine_state"].(string); ok {
		s.state.MachineState = v
	}
	if v, ok := updates["session_state"].(string); ok {
		s.state.SessionState = v
	}
	if v, ok := updates["heartbeat_state"].(string); ok {
		s.state.HeartbeatState = v
	}
	if v, ok := updates["connection_state"].(string); ok {
		s.state.ConnectionState = v
	}
	s.state.LastTransitionAt = time.Now().UTC()
	s.state.LastTransitionReason = reason
	s.logger.Log("info", "state", "state updated", map[string]any{"reason": reason, "updates": updates})
}

func (s *Server) templateStateLocked() map[string]any {
	return map[string]any{
		"instanceId":        s.state.InstanceID,
		"egmId":             s.state.EGMID,
		"trustMode":         s.state.TrustMode,
		"connectionState":   s.state.ConnectionState,
		"sessionState":      s.state.SessionState,
		"heartbeatState":    s.state.HeartbeatState,
		"audioState":        s.state.AudioState,
		"holdState":         s.state.HoldState,
		"lockState":         s.state.LockState,
		"machineState":      s.state.MachineState,
		"lastMessageType":   s.state.LastMessageType,
		"lastCommandType":   s.state.LastCommandType,
		"lastCommandSource": s.state.LastCommandSource,
		"lastSessionId":     s.state.LastSessionID,
		"lastHostId":        s.state.LastHostID,
		"lastAckStatus":     s.state.LastAckStatus,
	}
}
