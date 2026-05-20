package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu         sync.Mutex
	eventFile  *os.File
	eventPath  string
	payloadDir string
	logDir     string
	instanceID string
}

type Event struct {
	Time       time.Time      `json:"time"`
	Level      string         `json:"level"`
	Category   string         `json:"category"`
	Message    string         `json:"message"`
	InstanceID string         `json:"instance_id"`
	Fields     map[string]any `json:"fields,omitempty"`
}

type EventFilter struct {
	Category    string
	Level       string
	Contains    string
	MessageType string
	HostID      string
	SessionID   string
	Since       time.Time
	Until       time.Time
	Limit       int
}

type ExportOptions struct {
	OutputDir       string
	Since           time.Time
	Until           time.Time
	IncludePayloads bool
	StateSnapshot   any
	ConfigSnapshot  any
	PackSummary     any
}

func NewLogger(dir, instanceID string) (*Logger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir log dir: %w", err)
	}
	payloadDir := filepath.Join(dir, "payloads")
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir payload dir: %w", err)
	}
	eventPath := filepath.Join(dir, "events.jsonl")
	f, err := os.OpenFile(eventPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open event log: %w", err)
	}
	return &Logger{eventFile: f, eventPath: eventPath, payloadDir: payloadDir, logDir: dir, instanceID: instanceID}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.eventFile == nil {
		return nil
	}
	return l.eventFile.Close()
}

func (l *Logger) Log(level, category, message string, fields map[string]any) {
	if l == nil {
		return
	}
	evt := Event{Time: time.Now().UTC(), Level: level, Category: category, Message: message, InstanceID: l.instanceID, Fields: fields}
	data, _ := json.Marshal(evt)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.eventFile.Write(append(data, '\n'))
}

func (l *Logger) WritePayload(direction, op string, data []byte) (string, error) {
	if l == nil {
		return "", nil
	}
	name := fmt.Sprintf("%s_%s_%s.xml", time.Now().UTC().Format("20060102T150405.000000000Z"), direction, sanitizeFilePart(op))
	path := filepath.Join(l.payloadDir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (l *Logger) QueryEvents(filter EventFilter) ([]Event, error) {
	if l == nil {
		return nil, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.Open(l.eventPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Event
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for s.Scan() {
		var evt Event
		if err := json.Unmarshal(s.Bytes(), &evt); err != nil {
			continue
		}
		if !matchesEventFilter(evt, filter) {
			continue
		}
		out = append(out, evt)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (l *Logger) ExportBundle(opts ExportOptions) (string, error) {
	return "", fmt.Errorf("bundle export not implemented in this repo seed")
}

func matchesEventFilter(evt Event, filter EventFilter) bool {
	if filter.Category != "" && evt.Category != filter.Category {
		return false
	}
	if filter.Level != "" && evt.Level != filter.Level {
		return false
	}
	if !filter.Since.IsZero() && evt.Time.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && evt.Time.After(filter.Until) {
		return false
	}
	if filter.Contains != "" {
		needle := strings.ToLower(filter.Contains)
		blob := strings.ToLower(evt.Message)
		if !strings.Contains(blob, needle) {
			return false
		}
	}
	if filter.MessageType != "" && fmt.Sprint(evt.Fields["message_type"]) != filter.MessageType {
		return false
	}
	return true
}

func sanitizeFilePart(s string) string {
	if s == "" {
		return "unknown"
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
