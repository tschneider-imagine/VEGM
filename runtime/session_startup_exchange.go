package runtime

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) runStartupExchange(ctx context.Context, sessionID string) error {
	if err := s.runGetDescriptorOnce(ctx, sessionID); err != nil {
		return err
	}
	if err := s.runSetKeepAliveOnce(ctx, sessionID); err != nil {
		return err
	}
	return nil
}

func (s *Server) runGetDescriptorOnce(ctx context.Context, sessionID string) error {
	body := s.renderGetDescriptor(sessionID)
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("outbound_request", "getDescriptor", []byte(body))
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
		_, _ = s.logger.WritePayload("inbound_response", "descriptorList", buf.Bytes())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("getDescriptor status %d", resp.StatusCode)
	}
	parsed, err := ParseG2SEnvelope(buf.Bytes())
	if err != nil {
		return fmt.Errorf("parse descriptorList: %w", err)
	}
	s.recordParsedResponseEvidence("descriptorList", parsed)
	actual := firstNonEmpty(parsed.OperationName, parsed.RawRoot)
	if actual != "descriptorList" {
		return fmt.Errorf("expected descriptorList, got %s", actual)
	}
	s.mu.Lock()
	s.state.LastMessageType = "descriptorList"
	s.state.LastCommandType = "getDescriptor"
	s.state.LastCommandAt = time.Now().UTC()
	s.state.LastSessionID = sessionID
	s.state.LastHostID = s.cfg.HostID
	s.state.LastAckStatus = fmt.Sprintf("http_%d", resp.StatusCode)
	s.state.LastError = ""
	s.mu.Unlock()
	s.logger.Log("info", "session", "descriptorList received", map[string]any{"host_id": s.cfg.HostID, "egm_id": s.cfg.EGMID, "session_id": sessionID, "status": resp.StatusCode, "message_type": "descriptorList", "parsed_root_kind": parsed.RootKind, "parsed_class": parsed.ClassName, "parsed_operation": parsed.OperationName, "raw_root": parsed.RawRoot, "expected_ack": "descriptorList", "actual_ack": actual})
	return nil
}

func (s *Server) runSetKeepAliveOnce(ctx context.Context, sessionID string) error {
	body := s.renderSetKeepAlive(sessionID)
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("outbound_request", "setKeepAlive", []byte(body))
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
		_, _ = s.logger.WritePayload("inbound_response", "setKeepAliveAck", buf.Bytes())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("setKeepAlive status %d", resp.StatusCode)
	}
	parsed, err := ParseG2SEnvelope(buf.Bytes())
	if err != nil {
		return fmt.Errorf("parse setKeepAliveAck: %w", err)
	}
	s.recordParsedResponseEvidence("setKeepAliveAck", parsed)
	actual := firstNonEmpty(parsed.OperationName, parsed.RawRoot)
	if actual != "setKeepAliveAck" {
		return fmt.Errorf("expected setKeepAliveAck, got %s", actual)
	}
	s.mu.Lock()
	s.state.LastMessageType = "setKeepAliveAck"
	s.state.LastCommandType = "setKeepAlive"
	s.state.LastCommandAt = time.Now().UTC()
	s.state.LastSessionID = sessionID
	s.state.LastHostID = s.cfg.HostID
	s.state.LastAckStatus = fmt.Sprintf("http_%d", resp.StatusCode)
	s.state.LastError = ""
	s.mu.Unlock()
	s.logger.Log("info", "session", "setKeepAlive acknowledged", map[string]any{"host_id": s.cfg.HostID, "egm_id": s.cfg.EGMID, "session_id": sessionID, "status": resp.StatusCode, "message_type": "setKeepAliveAck", "parsed_root_kind": parsed.RootKind, "parsed_class": parsed.ClassName, "parsed_operation": parsed.OperationName, "raw_root": parsed.RawRoot, "expected_ack": "setKeepAliveAck", "actual_ack": actual})
	return nil
}

func (s *Server) renderGetDescriptor(sessionID string) string {
	if s.shouldRenderXSDG2SMessage() {
		return s.renderXSDCommunicationsMessage("getDescriptor", map[string]string{
			"includeOwners":  boolAttr(true),
			"includeConfigs": boolAttr(true),
			"includeGuests":  boolAttr(true),
			"includeOthers":  boolAttr(true),
		})
	}
	soapNS := firstNonEmpty(s.pack.Wire.Namespaces["soapenv"], SOAP11Namespace)
	g2sNS := firstNonEmpty(s.pack.Wire.Namespaces["g2s"], "urn:g2s:lab")
	return fmt.Sprintf(`<soapenv:Envelope xmlns:soapenv="%s" xmlns:g2s="%s"><soapenv:Body><g2s:getDescriptor><g2s:hostId>%s</g2s:hostId><g2s:egmId>%s</g2s:egmId><g2s:sessionId>%s</g2s:sessionId></g2s:getDescriptor></soapenv:Body></soapenv:Envelope>`, soapNS, g2sNS, s.cfg.HostID, s.cfg.EGMID, sessionID)
}

func (s *Server) renderSetKeepAlive(sessionID string) string {
	if s.shouldRenderXSDG2SMessage() {
		return s.renderXSDCommunicationsMessage("setKeepAlive", map[string]string{
			"interval": intAttr(s.cfg.SessionEngine.KeepAliveIntervalMS),
		})
	}
	soapNS := firstNonEmpty(s.pack.Wire.Namespaces["soapenv"], SOAP11Namespace)
	g2sNS := firstNonEmpty(s.pack.Wire.Namespaces["g2s"], "urn:g2s:lab")
	return fmt.Sprintf(`<soapenv:Envelope xmlns:soapenv="%s" xmlns:g2s="%s"><soapenv:Body><g2s:setKeepAlive><g2s:hostId>%s</g2s:hostId><g2s:egmId>%s</g2s:egmId><g2s:sessionId>%s</g2s:sessionId><g2s:keepAliveIntervalMS>%d</g2s:keepAliveIntervalMS></g2s:setKeepAlive></soapenv:Body></soapenv:Envelope>`, soapNS, g2sNS, s.cfg.HostID, s.cfg.EGMID, sessionID, s.cfg.SessionEngine.KeepAliveIntervalMS)
}
