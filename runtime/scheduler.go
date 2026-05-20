package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OutboundSchedulerState struct {
	Running             bool      `json:"running"`
	TargetURL           string    `json:"target_url,omitempty"`
	LastAttemptAt       time.Time `json:"last_attempt_at,omitempty"`
	LastTickAt          time.Time `json:"last_tick_at,omitempty"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
	LastError           string    `json:"last_error,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	OpenAttempts        int64     `json:"open_attempts"`
	HeartbeatAttempts   int64     `json:"heartbeat_attempts"`
	LastOperation       string    `json:"last_operation,omitempty"`
}

func (s *Server) schedulerConfig() OutboundSchedulerConfig {
	cfg := s.cfg.Outbound.Scheduler
	if cfg.HeartbeatIntervalMS <= 0 {
		cfg.HeartbeatIntervalMS = 5000
	}
	if cfg.OpenRetryIntervalMS <= 0 {
		cfg.OpenRetryIntervalMS = 2000
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 2
	}
	if !cfg.RunHeartbeatWhenIdle {
		cfg.OpenSessionOnStart = true
	}
	return cfg
}

func (s *Server) StartOutboundScheduler() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.cfg.Outbound.Scheduler.Enabled {
		return fmt.Errorf("outbound scheduler is not enabled in config")
	}
	if s.cfg.Outbound.DefaultTargetURL == "" {
		return fmt.Errorf("outbound.default_target_url is required for scheduler")
	}
	if s.schedulerCancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.schedulerCancel = cancel
	s.state.OutboundScheduler.Running = true
	s.state.OutboundScheduler.TargetURL = s.cfg.Outbound.DefaultTargetURL
	s.state.OutboundScheduler.LastError = ""
	s.logger.Log("info", "scheduler", "outbound scheduler started", map[string]any{"target_url": s.cfg.Outbound.DefaultTargetURL})
	go s.runOutboundScheduler(ctx)
	return nil
}

func (s *Server) StopOutboundScheduler() {
	s.mu.Lock()
	cancel := s.schedulerCancel
	if cancel != nil {
		s.schedulerCancel = nil
		s.state.OutboundScheduler.Running = false
		s.logger.Log("info", "scheduler", "outbound scheduler stopped", nil)
	}
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Server) runOutboundScheduler(ctx context.Context) {
	cfg := s.schedulerConfig()
	interval := time.Duration(cfg.HeartbeatIntervalMS) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	s.runOutboundSchedulerTick(ctx, cfg)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOutboundSchedulerTick(ctx, cfg)
		}
	}
}

func (s *Server) runOutboundSchedulerTick(ctx context.Context, cfg OutboundSchedulerConfig) {
	s.mu.RLock()
	schedRunning := s.state.OutboundScheduler.Running
	sessionState := s.state.SessionState
	lastAttempt := s.state.OutboundScheduler.LastAttemptAt
	s.mu.RUnlock()
	if !schedRunning {
		return
	}
	now := time.Now().UTC()
	operation := "keepAlive"
	if sessionState != "online" {
		if !cfg.OpenSessionOnStart && !cfg.RunHeartbeatWhenIdle {
			return
		}
		operation = "commsOnLine"
		if !lastAttempt.IsZero() && now.Sub(lastAttempt) < time.Duration(cfg.OpenRetryIntervalMS)*time.Millisecond {
			return
		}
	}
	if sessionState != "online" && cfg.RunHeartbeatWhenIdle {
		operation = "keepAlive"
	}
	res, err := s.SendOutbound(ctx, outboundRequest{MessageType: operation})
	s.updateSchedulerAfterResult(operation, res, err, cfg)
}

func (s *Server) updateSchedulerAfterResult(operation string, res outboundResult, err error, cfg OutboundSchedulerConfig) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	st := &s.state.OutboundScheduler
	st.LastTickAt = now
	st.LastAttemptAt = now
	st.LastOperation = operation
	if operation == "commsOnLine" {
		st.OpenAttempts++
	} else if operation == "keepAlive" {
		st.HeartbeatAttempts++
	}
	ok := err == nil && res.OK
	if ok {
		st.LastSuccessAt = now
		st.ConsecutiveFailures = 0
		st.LastError = ""
		if operation == "keepAlive" {
			s.state.HeartbeatState = "healthy"
		}
		return
	}
	st.ConsecutiveFailures++
	if err != nil {
		st.LastError = err.Error()
	} else if res.Error != "" {
		st.LastError = res.Error
	} else {
		st.LastError = fmt.Sprintf("http_status=%d response_root=%s", res.HTTPStatus, res.ResponseRoot)
	}
	fields := map[string]any{"operation": operation, "error": st.LastError, "consecutive_failures": st.ConsecutiveFailures, "failure_threshold": cfg.FailureThreshold, "target_url": st.TargetURL}
	if st.ConsecutiveFailures >= cfg.FailureThreshold {
		if operation == "commsOnLine" {
			s.state.SessionState = "idle"
		}
		s.state.HeartbeatState = "degraded"
		s.logger.Log("warn", "scheduler", "outbound scheduler failure threshold reached", fields)
	} else {
		s.logger.Log("warn", "scheduler", "outbound scheduler tick failed", fields)
	}
}

func (s *Server) handleControlOutboundSchedulerStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.StartOutboundScheduler(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "scheduler": s.state.OutboundScheduler})
}

func (s *Server) handleControlOutboundSchedulerStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.StopOutboundScheduler()
	w.Header().Set("Content-Type", "application/json")
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "scheduler": s.state.OutboundScheduler})
}
