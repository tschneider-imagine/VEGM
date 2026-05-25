package runtime

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func (s *Server) handleWire(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
	s.mu.Lock()
	op, ok := s.pack.Operations[opName]
	if !ok || len(op.Responses) == 0 {
		s.state.LastError = fmt.Sprintf("operation %q is not defined", opName)
		s.mu.Unlock()
		http.Error(w, s.state.LastError, http.StatusBadRequest)
		return
	}
	variant := op.Responses[0]
	hostID := firstNonEmpty(parsed.Fields["hostId"], parsed.Fields["hostID"], parsed.HostID, "HOST-001")
	sessionID := firstNonEmpty(parsed.Fields["sessionId"], parsed.SessionID, fmt.Sprintf("%s-%d", s.cfg.InstanceID, time.Now().UnixNano()))
	egmID := firstNonEmpty(parsed.Fields["egmId"], parsed.EGMID, s.cfg.EGMID)

	s.state.ConnectionState = "controller_connected"
	s.state.LastMessageType = opName
	s.state.LastCommandType = opName
	s.state.LastCommandAt = time.Now().UTC()
	s.state.LastCommandSource = hostID
	s.state.LastSessionID = sessionID
	s.state.LastHostID = hostID
	s.state.LastError = ""
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
