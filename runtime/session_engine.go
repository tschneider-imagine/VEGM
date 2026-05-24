package runtime

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) StartSessionEngine(ctx context.Context) {
	if !s.cfg.SessionEngine.Enabled || !s.cfg.SessionEngine.AutoStart {
		return
	}
	go s.sessionEngineLoop(ctx)
}

func (s *Server) sessionEngineLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		sessionID, err := s.runCommsOnlineOnce(ctx)
		if err != nil {
			s.mu.Lock()
			s.state.SessionState = "connect_failed"
			s.state.ConnectionState = "host_unreachable"
			s.state.HeartbeatState = "failed"
			s.state.LastError = err.Error()
			s.mu.Unlock()
			s.logger.Log("warn", "session", "commsOnLine failed", map[string]any{"error": err.Error(), "host_endpoint": s.cfg.HostEndpoint.URL, "message_type": "commsOnLine"})
			if !sleepOrDone(ctx, time.Duration(s.cfg.SessionEngine.ReconnectIntervalMS)*time.Millisecond) {
				return
			}
			continue
		}
		if err := s.runStartupExchange(ctx, sessionID); err != nil {
			s.mu.Lock()
			s.state.SessionState = "startup_failed"
			s.state.ConnectionState = "host_connected"
			s.state.HeartbeatState = "failed"
			s.state.LastError = err.Error()
			s.mu.Unlock()
			s.logger.Log("warn", "session", "startup exchange failed", map[string]any{"error": err.Error(), "host_endpoint": s.cfg.HostEndpoint.URL, "session_id": sessionID})
			if !sleepOrDone(ctx, time.Duration(s.cfg.SessionEngine.ReconnectIntervalMS)*time.Millisecond) {
				return
			}
			continue
		}
		if ok := s.runKeepAliveLoop(ctx, sessionID); !ok {
			return
		}
		if !sleepOrDone(ctx, time.Duration(s.cfg.SessionEngine.ReconnectIntervalMS)*time.Millisecond) {
			return
		}
	}
}

func (s *Server) runKeepAliveLoop(ctx context.Context, sessionID string) bool {
	interval := time.Duration(s.cfg.SessionEngine.KeepAliveIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if err := s.runKeepAliveOnce(ctx, sessionID); err != nil {
		s.recordKeepAliveFailure(sessionID, err)
		return true
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if err := s.runKeepAliveOnce(ctx, sessionID); err != nil {
				s.recordKeepAliveFailure(sessionID, err)
				return true
			}
		}
	}
}

func (s *Server) recordKeepAliveFailure(sessionID string, err error) {
	s.mu.Lock()
	s.state.HeartbeatState = "failed"
	s.state.ConnectionState = "host_unreachable"
	s.state.LastError = err.Error()
	s.mu.Unlock()
	s.logger.Log("warn", "session", "keepAlive failed", map[string]any{"error": err.Error(), "host_endpoint": s.cfg.HostEndpoint.URL, "session_id": sessionID, "message_type": "keepAlive"})
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		d = 5 * time.Second
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func (s *Server) runCommsOnlineOnce(ctx context.Context) (string, error) {
	if s.cfg.HostEndpoint.URL == "" {
		return "", fmt.Errorf("host endpoint url is empty")
	}
	sessionID := fmt.Sprintf("%s-%d", s.cfg.InstanceID, time.Now().UTC().UnixNano())
	body := s.renderCommsOnline(sessionID)
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("outbound_request", "commsOnLine", []byte(body))
	}
	client := &http.Client{Timeout: time.Duration(s.cfg.SessionEngine.CommsOnlineTimeoutMS) * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.HostEndpoint.URL, bytes.NewBufferString(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	if s.cfg.Logging.CaptureRawXML {
		_, _ = s.logger.WritePayload("inbound_response", "commsOnLineAck", buf.Bytes())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("commsOnLine status %d", resp.StatusCode)
	}
	parsed, err := ParseG2SEnvelope(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("parse commsOnLineAck: %w", err)
	}
	s.recordParsedResponseEvidence("commsOnLineAck", parsed)
	actual := firstNonEmpty(parsed.OperationName, parsed.RawRoot)
	if actual != "commsOnLineAck" {
		return "", fmt.Errorf("expected commsOnLineAck, got %s", actual)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	s.state.SessionState = "online"
	s.state.ConnectionState = "host_connected"
	s.state.HeartbeatState = "idle"
	s.state.LastMessageType = "commsOnLineAck"
	s.state.LastCommandType = "commsOnLine"
	s.state.LastCommandAt = now
	s.state.LastSessionID = sessionID
	s.state.LastHostID = s.cfg.HostID
	s.state.LastAckStatus = fmt.Sprintf("http_%d", resp.StatusCode)
	s.state.LastError = ""
	s.mu.Unlock()
	s.recordSessionTimestamp("commsOnLine", now)
	s.logger.Log("info", "session", "commsOnLine acknowledged", map[string]any{"host_id": s.cfg.HostID, "egm_id": s.cfg.EGMID, "session_id": sessionID, "status": resp.StatusCode, "message_type": "commsOnLineAck", "parsed_root_kind": parsed.RootKind, "parsed_class": parsed.ClassName, "parsed_operation": parsed.OperationName, "raw_root": parsed.RawRoot, "expected_ack": "commsOnLineAck", "actual_ack": actual})
	return sessionID, nil
}

func (s *Server) renderCommsOnline(sessionID string) string {
	if s.shouldRenderXSDG2SMessage() {
		return s.renderXSDCommunicationsMessage("commsOnLine", map[string]string{
			"equipmentType":    "G2S_egm",
			"egmLocation":      s.cfg.G2SXML.EGMLocation,
			"deviceReset":      boolAttr(false),
			"deviceChanged":    boolAttr(false),
			"subscriptionLost": boolAttr(false),
			"metersReset":      boolAttr(false),
		})
	}
	soapNS := firstNonEmpty(s.pack.Wire.Namespaces["soapenv"], SOAP11Namespace)
	g2sNS := firstNonEmpty(s.pack.Wire.Namespaces["g2s"], "urn:g2s:lab")
	return fmt.Sprintf(`<soapenv:Envelope xmlns:soapenv="%s" xmlns:g2s="%s"><soapenv:Body><g2s:commsOnLine><g2s:hostId>%s</g2s:hostId><g2s:egmId>%s</g2s:egmId><g2s:sessionId>%s</g2s:sessionId></g2s:commsOnLine></soapenv:Body></soapenv:Envelope>`, soapNS, g2sNS, s.cfg.HostID, s.cfg.EGMID, sessionID)
}
