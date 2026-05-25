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
			return
		}
		if err := s.runStartupExchange(ctx, sessionID); err != nil {
			s.mu.Lock()
			s.state.SessionState = "startup_failed"
			s.state.ConnectionState = "host_connected"
			s.state.HeartbeatState = "failed"
			s.state.LastError = err.Error()
			s.mu.Unlock()
			return
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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("commsOnLine status %d", resp.StatusCode)
	}
	parsed, err := ParseG2SEnvelope(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("parse commsOnLineAck: %w", err)
	}

	actual := firstNonEmpty(parsed.OperationName, parsed.RawRoot)

	// minimal wrapped support
	if actual != "commsOnLineAck" && s.cfg.SessionEngine.AcceptWrappedG2SResponseAck {
		if actual == "g2sResponse" && firstNestedAckName(buf.Bytes()) == "commsOnLineAck" {
			actual = "commsOnLineAck"
		}
	}

	if actual != "commsOnLineAck" {
		return "", fmt.Errorf("expected commsOnLineAck, got %s", actual)
	}

	return sessionID, nil
}

func (s *Server) renderCommsOnline(sessionID string) string {
	return "" 
}
