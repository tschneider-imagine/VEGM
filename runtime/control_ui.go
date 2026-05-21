package runtime

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/tschneider-imagine/VEGM/webui"
)

type logicalCommandRequest struct {
	LogicalCommand string `json:"logical_command"`
	HostID         string `json:"host_id,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
}

func withControlCORS(next http.Handler) http.Handler {
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

func (s *Server) handleScenarioUIRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui/scenario-runner.html", http.StatusTemporaryRedirect)
}

func (s *Server) handleControlInjectLogicalCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req logicalCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opName, err := logicalCommandToOperation(req.LogicalCommand)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hostID := firstNonEmpty(req.HostID, "HOST-001")
	sessionID := firstNonEmpty(req.SessionID, "SCENARIO")
	responseXML, status, err := s.processSimulatedInbound(opName, hostID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":           true,
		"operation":    opName,
		"status":       status,
		"response_xml": responseXML,
		"state":        s.state,
	})
}

func logicalCommandToOperation(logical string) (string, error) {
	switch strings.TrimSpace(logical) {
	case "audio_mute_on", "audioMuteOn":
		return "audioMuteOn", nil
	case "audio_mute_off", "audioMuteOff":
		return "audioMuteOff", nil
	case "hold_on", "holdOn":
		return "holdOn", nil
	case "hold_off", "holdOff":
		return "holdOff", nil
	case "lock_on", "lockOn":
		return "lockOn", nil
	case "lock_off", "lockOff":
		return "lockOff", nil
	case "comms_online", "commsOnLine":
		return "commsOnLine", nil
	case "keep_alive", "keepAlive":
		return "keepAlive", nil
	default:
		return "", fmt.Errorf("unsupported logical command %q", logical)
	}
}

func (s *Server) processSimulatedInbound(opName, hostID, sessionID string) (string, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, ok := s.pack.Operations[opName]
	if !ok || len(op.Responses) == 0 {
		return "", 0, fmt.Errorf("operation %q is not defined", opName)
	}
	variant := op.Responses[0]
	s.state.ConnectionState = "controller_simulated"
	s.state.LastMessageType = opName
	s.state.LastCommandType = opName
	s.state.LastCommandAt = time.Now().UTC()
	s.state.LastCommandSource = hostID
	s.state.LastSessionID = sessionID
	s.state.LastHostID = hostID
	s.state.LastError = ""
	if opName == "commsOnLine" {
		s.applyStateUpdatesLocked(map[string]any{"session_state": "online", "heartbeat_state": "healthy"}, "scenario commsOnLine")
	}
	if opName == "keepAlive" {
		s.applyStateUpdatesLocked(map[string]any{"heartbeat_state": "healthy"}, "scenario keepAlive")
	}
	if len(variant.SetState) > 0 {
		s.applyStateUpdatesLocked(variant.SetState, fmt.Sprintf("scenario %s", opName))
	}
	requestFields := map[string]string{"hostId": hostID, "sessionId": sessionID, "egmId": s.cfg.EGMID}
	responseXML := RenderTemplate(variant.Template, s.pack.Wire.Namespaces, requestFields, s.templateStateLocked())
	status := variant.HTTPStatus
	if status == 0 {
		status = http.StatusOK
	}
	s.state.LastAckStatus = fmt.Sprintf("http_%d", status)
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("scenario_response", opName, []byte(responseXML))
	}
	s.logger.Log("info", "scenario", "logical command injected", map[string]any{"logical_command": opName, "host_id": hostID, "session_id": sessionID, "status": status})
	return responseXML, status, nil
}

func scenarioUIHandler() http.Handler {
	sub, err := fs.Sub(webui.StaticFS, "static")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/ui/", http.FileServer(http.FS(sub)))
}
