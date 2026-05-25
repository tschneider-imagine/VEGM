package runtime

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleWire(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isXMLContentType(r.Header.Get("Content-Type")) {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parsed, err := ParseG2SMessage(body)
	if err != nil {
		s.logger.Log("warn", "wire", "failed to parse inbound G2S message", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opName := parsed.RootLocalName
	if s.cfg.Logging.CaptureRawXML {
		_, _ = s.logger.WritePayload("inbound_request", opName, body)
	}
	hostID := firstNonEmpty(parsed.Fields["hostId"], parsed.Fields["hostID"], parsed.Fields["host_id"], "HOST-001")
	sessionID := firstNonEmpty(parsed.Fields["sessionId"], parsed.Fields["session_id"], fmt.Sprintf("%s-%d", s.cfg.InstanceID, time.Now().UnixNano()))
	egmID := firstNonEmpty(parsed.Fields["egmId"], parsed.Fields["egmID"], parsed.Fields["egm_id"], s.cfg.EGMID)

	s.mu.Lock()
	if !s.hostAllowedLocked(hostID) {
		s.state.LastError = fmt.Sprintf("host %q is not registered or enabled", hostID)
		s.mu.Unlock()
		http.Error(w, s.state.LastError, http.StatusForbidden)
		return
	}
	op, ok := s.pack.Operations[opName]
	if !ok || len(op.Responses) == 0 {
		s.state.LastError = fmt.Sprintf("operation %q is not defined", opName)
		s.mu.Unlock()
		http.Error(w, s.state.LastError, http.StatusBadRequest)
		return
	}
	variant := op.Responses[0]

	s.state.ConnectionState = "controller_connected"
	s.state.LastMessageType = opName
	s.state.LastCommandType = opName
	s.state.LastCommandAt = time.Now().UTC()
	s.state.LastCommandSource = hostID
	s.state.LastSessionID = sessionID
	s.state.LastHostID = hostID
	s.state.LastError = ""
	s.applyBuiltInOperationStateLocked(opName)
	if len(variant.SetState) > 0 {
		s.applyStateUpdatesLocked(variant.SetState, fmt.Sprintf("wire %s", opName))
	}
	requestFields := map[string]string{"hostId": hostID, "sessionId": sessionID, "egmId": egmID}
	responseXML := RenderTemplate(variant.Template, s.pack.Wire.Namespaces, requestFields, s.templateStateLocked())
	status := variant.HTTPStatus
	if status == 0 {
		status = http.StatusOK
	}
	s.state.LastAckStatus = fmt.Sprintf("http_%d", status)
	s.mu.Unlock()

	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("inbound_response", opName, []byte(responseXML))
	}
	s.logger.Log("info", "wire", "inbound G2S message handled", map[string]any{"message_type": opName, "host_id": hostID, "session_id": sessionID, "status": status})
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(responseXML))
}

func isXMLContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return ct == "text/xml" || ct == "application/xml" || ct == "application/soap+xml"
}

func (s *Server) hostAllowedLocked(hostID string) bool {
	if s.pack.StateDefaults.RegistrationMode == "open" || s.state.RegistrationState == "open" {
		return true
	}
	if len(s.state.RegisteredHosts) > 0 {
		for _, host := range s.state.RegisteredHosts {
			if host.HostID == hostID {
				return host.Enabled
			}
		}
		return false
	}
	for _, allowed := range s.state.AllowedHostIDs {
		if allowed == hostID {
			return true
		}
	}
	return false
}

func (s *Server) applyBuiltInOperationStateLocked(opName string) {
	switch opName {
	case "commsOnLine":
		s.applyStateUpdatesLocked(map[string]any{"session_state": "online", "heartbeat_state": "healthy"}, "wire commsOnLine")
	case "keepAlive":
		s.applyStateUpdatesLocked(map[string]any{"heartbeat_state": "healthy"}, "wire keepAlive")
	}
}
