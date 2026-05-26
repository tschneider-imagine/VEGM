package runtime

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) runKeepAliveOnce(ctx context.Context, sessionID string) error {
	body := s.renderKeepAlive(sessionID)
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("outbound_request", "keepAlive", []byte(body))
	}
	client := &http.Client{Timeout: time.Duration(s.cfg.SessionEngine.CommsOnlineTimeoutMS) * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.HostEndpoint.URL, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	if s.cfg.Logging.CaptureRawXML {
		_, _ = s.logger.WritePayload("inbound_response", "keepAliveAck", buf.Bytes())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("keepAlive status %d", resp.StatusCode)
	}
	parsed, err := ParseG2SEnvelope(buf.Bytes())
	if err != nil {
		return fmt.Errorf("parse keepAliveAck: %w", err)
	}
	s.recordParsedResponseEvidence("keepAliveAck", parsed)
	actual := firstNonEmpty(parsed.OperationName, parsed.RawRoot)
	if actual != "keepAliveAck" && s.cfg.SessionEngine.AcceptWrappedG2SResponseAck {
		if actual == "g2sResponse" && firstNestedAckName(buf.Bytes()) == "keepAliveAck" {
			actual = "keepAliveAck"
		}
	}
	if actual != "keepAliveAck" {
		return fmt.Errorf("expected keepAliveAck, got %s", actual)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	s.state.HeartbeatState = "healthy"
	s.state.ConnectionState = "host_connected"
	s.state.LastMessageType = "keepAliveAck"
	s.state.LastCommandType = "keepAlive"
	s.state.LastCommandAt = now
	s.state.LastSessionID = sessionID
	s.state.LastHostID = s.cfg.HostID
	s.state.LastAckStatus = fmt.Sprintf("http_%d", resp.StatusCode)
	s.state.LastError = ""
	s.mu.Unlock()
	s.recordSessionTimestamp("keepAlive", now)
	s.logger.Log("info", "session", "keepAlive acknowledged", map[string]any{"host_id": s.cfg.HostID, "egm_id": s.cfg.EGMID, "session_id": sessionID, "status": resp.StatusCode, "message_type": "keepAliveAck", "parsed_root_kind": parsed.RootKind, "parsed_class": parsed.ClassName, "parsed_operation": parsed.OperationName, "raw_root": parsed.RawRoot, "expected_ack": "keepAliveAck", "actual_ack": actual})
	return nil
}

func (s *Server) renderKeepAlive(sessionID string) string {
	if s.shouldRenderXSDG2SMessage() {
		return s.renderXSDCommunicationsMessage("keepAlive", map[string]string{})
	}
	soapNS := firstNonEmpty(s.pack.Wire.Namespaces["soapenv"], SOAP11Namespace)
	g2sNS := firstNonEmpty(s.pack.Wire.Namespaces["g2s"], "urn:g2s:lab")
	return fmt.Sprintf(`<soapenv:Envelope xmlns:soapenv="%s" xmlns:g2s="%s"><soapenv:Body><g2s:keepAlive><g2s:hostId>%s</g2s:hostId><g2s:egmId>%s</g2s:egmId><g2s:sessionId>%s</g2s:sessionId></g2s:keepAlive></soapenv:Body></soapenv:Envelope>`, soapNS, g2sNS, s.cfg.HostID, s.cfg.EGMID, sessionID)
}
