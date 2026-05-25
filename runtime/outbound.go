package runtime

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type outboundRequest struct {
	MessageType     string `json:"message_type"`
	HostID          string `json:"host_id,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	TargetURL       string `json:"target_url,omitempty"`
	AllowGenericAck bool   `json:"allow_generic_ack,omitempty"`
}

type outboundResult struct {
	OK           bool   `json:"ok"`
	HTTPStatus   int    `json:"http_status"`
	ResponseRoot string `json:"response_root,omitempty"`
	Error        string `json:"error,omitempty"`
}

func (s *Server) SendOutbound(ctx context.Context, req outboundRequest) (outboundResult, error) {
	s.mu.RLock()
	pk := s.pack
	state := s.templateStateLocked()
	trustMode := s.cfg.Security.TrustMode
	cfg := s.cfg.Outbound
	hostEndpointURL := s.cfg.HostEndpoint.URL
	s.mu.RUnlock()
	if pk == nil {
		return outboundResult{}, fmt.Errorf("pack is nil")
	}
	if req.MessageType == "" {
		return outboundResult{}, fmt.Errorf("message_type is required")
	}
	targetURL := req.TargetURL
	if targetURL == "" {
		targetURL = cfg.DefaultTargetURL
	}
	if targetURL == "" {
		targetURL = hostEndpointURL
	}
	if targetURL == "" {
		return outboundResult{}, fmt.Errorf("target url is required")
	}
	sessionID := firstNonEmpty(req.SessionID, fmt.Sprintf("%s-%d", s.cfg.InstanceID, time.Now().UnixNano()))
	body := s.renderConfiguredOutboundBody(req.MessageType, sessionID)
	if body == "" {
		op, ok := pk.Operations[req.MessageType]
		if !ok || len(op.Responses) == 0 {
			return outboundResult{}, fmt.Errorf("operation %q is not defined for outbound use", req.MessageType)
		}
		requestFields := map[string]string{
			"hostId":    firstNonEmpty(req.HostID, s.cfg.HostID, s.cfg.InstanceID),
			"sessionId": sessionID,
			"egmId":     s.cfg.EGMID,
		}
		body = RenderTemplate(op.Responses[0].Template, pk.Wire.Namespaces, requestFields, state)
	}
	if s.cfg.Logging.CaptureRenderedXML {
		_, _ = s.logger.WritePayload("outbound_request", req.MessageType, []byte(body))
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBufferString(body))
	if err != nil {
		return outboundResult{}, err
	}
	httpReq.Header.Set("Content-Type", firstContentType(pk))
	client := &http.Client{Timeout: time.Duration(maxInt(cfg.TimeoutMS, 3000)) * time.Millisecond}
	if transport, err := s.outboundTransport(trustMode, cfg); err != nil {
		return outboundResult{}, err
	} else if transport != nil {
		client.Transport = transport
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		s.logger.Log("warn", "outbound", "outbound request failed", map[string]any{"message_type": req.MessageType, "target_url": targetURL, "error": err.Error()})
		return outboundResult{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	parsedRoot := rootLocalNameFromResponse(respBody)
	if s.cfg.Logging.CaptureRawXML {
		_, _ = s.logger.WritePayload("outbound_response", req.MessageType, respBody)
	}
	result := outboundResult{HTTPStatus: resp.StatusCode, ResponseRoot: parsedRoot}
	ackExpected := expectedAckRoot(req.MessageType)
	strictAckOK := ackExpected == "" || strings.EqualFold(parsedRoot, ackExpected)
	genericAckOK := req.AllowGenericAck && strings.EqualFold(parsedRoot, "g2sResponse")
	result.OK = resp.StatusCode >= 200 && resp.StatusCode < 300 && (strictAckOK || genericAckOK)
	if !result.OK && result.Error == "" {
		result.Error = fmt.Sprintf("unexpected http_status=%d response_root=%s", resp.StatusCode, parsedRoot)
	}
	s.logger.Log("info", "outbound", "outbound request complete", map[string]any{"message_type": req.MessageType, "target_url": targetURL, "http_status": resp.StatusCode, "response_root": parsedRoot, "ok": result.OK, "allow_generic_ack": req.AllowGenericAck, "xml_mode": s.cfg.G2SXML.Mode})
	return result, nil
}

func (s *Server) renderConfiguredOutboundBody(messageType, sessionID string) string {
	switch messageType {
	case "commsOnLine":
		return s.renderCommsOnline(sessionID)
	case "getDescriptor":
		return s.renderGetDescriptor(sessionID)
	case "setKeepAlive":
		return s.renderSetKeepAlive(sessionID)
	case "keepAlive":
		return s.renderKeepAlive(sessionID)
	default:
		return ""
	}
}

func rootLocalNameFromResponse(data []byte) string {
	if parsed, err := ParseG2SEnvelope(data); err == nil {
		return firstNonEmpty(parsed.OperationName, parsed.RawRoot)
	}
	return ParseMessage(data).RootLocalName
}

func (s *Server) outboundTransport(trustMode string, cfg OutboundConfig) (*http.Transport, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	switch trustMode {
	case "plaintext_lab":
		return nil, nil
	case "accept_all_lab":
		tlsCfg.InsecureSkipVerify = true
	case "tls_server_only", "strict_mtls", "mtls_no_revocation":
		if cfg.UseRuntimeCerts && s.cfg.Security.CertFile != "" && s.cfg.Security.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(s.cfg.Security.CertFile, s.cfg.Security.KeyFile)
			if err != nil {
				return nil, err
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}
		if s.cfg.Security.CAFile != "" {
			caPEM, err := os.ReadFile(s.cfg.Security.CAFile)
			if err != nil {
				return nil, err
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caPEM)
			tlsCfg.RootCAs = pool
		}
		if cfg.ServerName != "" {
			tlsCfg.ServerName = cfg.ServerName
		}
	default:
		return nil, fmt.Errorf("unsupported outbound trust mode %q", trustMode)
	}
	return &http.Transport{TLSClientConfig: tlsCfg}, nil
}

func (s *Server) handleControlOutboundSessionOpen(w http.ResponseWriter, r *http.Request) {
	s.handleOutboundSendLike(w, r, "commsOnLine")
}

func (s *Server) handleControlOutboundHeartbeat(w http.ResponseWriter, r *http.Request) {
	s.handleOutboundSendLike(w, r, "keepAlive")
}

func (s *Server) handleControlOutboundSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req outboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.SendOutbound(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if res.OK {
		s.mu.Lock()
		if req.MessageType == "commsOnLine" {
			s.state.SessionState = "online"
		}
		if req.MessageType == "keepAlive" {
			s.state.HeartbeatState = "healthy"
		}
		s.mu.Unlock()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleOutboundSendLike(w http.ResponseWriter, r *http.Request, msgType string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req outboundRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.MessageType = msgType
	if msgType == "keepAlive" {
		req.AllowGenericAck = true
	}
	res, err := s.SendOutbound(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if res.OK {
		s.mu.Lock()
		if msgType == "commsOnLine" {
			s.state.SessionState = "online"
		}
		if msgType == "keepAlive" {
			s.state.HeartbeatState = "healthy"
		}
		s.mu.Unlock()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func expectedAckRoot(messageType string) string {
	switch messageType {
	case "commsOnLine":
		return "commsOnLineAck"
	case "keepAlive":
		return "keepAliveAck"
	case "setKeepAlive":
		return "setKeepAliveAck"
	case "commsClosing":
		return "commsClosingAck"
	default:
		return ""
	}
}

func firstContentType(pk interface{}) string {
	if p, ok := pk.(*struct{}); ok && p == nil {
		return "text/xml; charset=utf-8"
	}
	return "text/xml; charset=utf-8"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func maxInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
