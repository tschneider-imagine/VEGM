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
		if err := s.runCommsOnlineOnce(ctx); err != nil {
			s.mu.Lock()
			s.state.SessionState = "connect_failed"
			s.state.ConnectionState = "host_unreachable"
			s.state.LastError = err.Error()
			s.mu.Unlock()
			s.logger.Log("warn", "session", "commsOnLine failed", map[string]any{"error": err.Error(), "host_endpoint": s.cfg.HostEndpoint.URL})
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(s.cfg.SessionEngine.ReconnectIntervalMS) * time.Millisecond):
				continue
			}
		}
		return
	}
}

func (s *Server) runCommsOnlineOnce(ctx context.Context) error {
	if s.cfg.HostEndpoint.URL == "" {
		return fmt.Errorf("host endpoint url is empty")
	}
	sessionID := fmt.Sprintf("%s-%d", s.cfg.InstanceID, time.Now().UTC().UnixNano())
	body := s.renderCommsOnline(sessionID)
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("outbound_request", "commsOnLine", []byte(body))
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
		_, _ = s.logger.WritePayload("inbound_response", "commsOnLineAck", buf.Bytes())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("commsOnLine status %d", resp.StatusCode)
	}
	parsed, err := ParseG2SMessage(buf.Bytes())
	if err != nil {
		return fmt.Errorf("parse commsOnLineAck: %w", err)
	}
	if parsed.RootLocalName != "commsOnLineAck" {
		return fmt.Errorf("expected commsOnLineAck, got %s", parsed.RootLocalName)
	}
	s.mu.Lock()
	s.state.SessionState = "online"
	s.state.ConnectionState = "host_connected"
	s.state.HeartbeatState = "idle"
	s.state.LastMessageType = "commsOnLineAck"
	s.state.LastCommandType = "commsOnLine"
	s.state.LastCommandAt = time.Now().UTC()
	s.state.LastSessionID = sessionID
	s.state.LastHostID = s.cfg.HostID
	s.state.LastAckStatus = fmt.Sprintf("http_%d", resp.StatusCode)
	s.state.LastError = ""
	s.mu.Unlock()
	s.logger.Log("info", "session", "commsOnLine acknowledged", map[string]any{"host_id": s.cfg.HostID, "egm_id": s.cfg.EGMID, "session_id": sessionID, "status": resp.StatusCode})
	return nil
}

func (s *Server) renderCommsOnline(sessionID string) string {
	soapNS := firstNonEmpty(s.pack.Wire.Namespaces["soapenv"], SOAP11Namespace)
	g2sNS := firstNonEmpty(s.pack.Wire.Namespaces["g2s"], "urn:g2s:lab")
	return fmt.Sprintf(`<soapenv:Envelope xmlns:soapenv="%s" xmlns:g2s="%s"><soapenv:Body><g2s:commsOnLine><g2s:hostId>%s</g2s:hostId><g2s:egmId>%s</g2s:egmId><g2s:sessionId>%s</g2s:sessionId></g2s:commsOnLine></soapenv:Body></soapenv:Envelope>`, soapNS, g2sNS, s.cfg.HostID, s.cfg.EGMID, sessionID)
}
