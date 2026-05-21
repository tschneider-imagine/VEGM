package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tschneider-imagine/VEGM/storage"
)

type Logger struct {
	mu         sync.Mutex
	eventFile  *os.File
	eventPath  string
	payloadDir string
	logDir     string
	instanceID string
	index      storage.Index
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
	OutputDir        string
	Since            time.Time
	Until            time.Time
	IncludePayloads  bool
	StateSnapshot    any
	ConfigSnapshot   any
	PackSummary      any
}

type ExportBundle struct {
	GeneratedAt     time.Time `json:"generated_at"`
	InstanceID      string    `json:"instance_id"`
	Since           time.Time `json:"since,omitempty"`
	Until           time.Time `json:"until,omitempty"`
	IncludePayloads bool      `json:"include_payloads"`
	Events          []Event   `json:"events"`
	PayloadFiles    []string  `json:"payload_files,omitempty"`
	StateSnapshot   any       `json:"state_snapshot,omitempty"`
	ConfigSnapshot  any       `json:"config_snapshot,omitempty"`
	PackSummary     any       `json:"pack_summary,omitempty"`
}

func NewLogger(dir, instanceID string) (*Logger, error) {
	return NewLoggerWithIndex(dir, instanceID, &storage.NoopIndex{})
}

func NewLoggerWithIndex(dir, instanceID string, idx storage.Index) (*Logger, error) {
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
	if idx == nil {
		idx = &storage.NoopIndex{}
	}
	if err := idx.Initialize(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("initialize storage index: %w", err)
	}
	return &Logger{eventFile: f, eventPath: eventPath, payloadDir: payloadDir, logDir: dir, instanceID: instanceID, index: idx}, nil
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	if l.index != nil {
		_ = l.index.Close()
	}
	if l.eventFile == nil {
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
	if l.index != nil {
		_ = l.index.WriteEvent(eventToRecord(evt, ""))
	}
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
	return l.queryEventsLocked(filter)
}

func (l *Logger) ExportBundle(opts ExportOptions) (string, error) {
	if l == nil {
		return "", fmt.Errorf("logger is nil")
	}
	outDir := opts.OutputDir
	if outDir == "" {
		outDir = filepath.Join(l.logDir, "exports")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	bundlePath := filepath.Join(outDir, fmt.Sprintf("%s_%s_bundle.json", time.Now().UTC().Format("20060102T150405Z"), sanitizeFilePart(l.instanceID)))
	l.mu.Lock()
	defer l.mu.Unlock()
	events, err := l.queryEventsLocked(EventFilter{Since: opts.Since, Until: opts.Until})
	if err != nil {
		return "", err
	}
	bundle := ExportBundle{
		GeneratedAt:     time.Now().UTC(),
		InstanceID:      l.instanceID,
		Since:           opts.Since,
		Until:           opts.Until,
		IncludePayloads: opts.IncludePayloads,
		Events:          events,
		StateSnapshot:   opts.StateSnapshot,
		ConfigSnapshot:  opts.ConfigSnapshot,
		PackSummary:     opts.PackSummary,
	}
	if opts.IncludePayloads {
		entries, err := os.ReadDir(l.payloadDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				bundle.PayloadFiles = append(bundle.PayloadFiles, entry.Name())
			}
			sort.Strings(bundle.PayloadFiles)
		}
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(bundlePath, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return bundlePath, nil
}

func (l *Logger) queryEventsLocked(filter EventFilter) ([]Event, error) {
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
		blob := strings.ToLower(evt.Message + " " + flattenFields(evt.Fields))
		if !strings.Contains(blob, needle) {
			return false
		}
	}
	if filter.MessageType != "" && fmt.Sprint(evt.Fields["message_type"]) != filter.MessageType {
		return false
	}
	if filter.HostID != "" && fmt.Sprint(evt.Fields["hostId"]) != filter.HostID && fmt.Sprint(evt.Fields["host_id"]) != filter.HostID {
		return false
	}
	if filter.SessionID != "" && fmt.Sprint(evt.Fields["sessionId"]) != filter.SessionID && fmt.Sprint(evt.Fields["session_id"]) != filter.SessionID {
		return false
	}
	return true
}

func flattenFields(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(fmt.Sprint(fields[k]))
		b.WriteByte(' ')
	}
	return b.String()
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

func eventToRecord(evt Event, payloadPath string) storage.EventRecord {
	return storage.EventRecord{
		Time:        evt.Time,
		InstanceID:  evt.InstanceID,
		Level:       evt.Level,
		Category:    evt.Category,
		Message:     evt.Message,
		MessageType: firstField(evt.Fields, "message_type", "messageType"),
		HostID:      firstField(evt.Fields, "host_id", "hostId"),
		SessionID:   firstField(evt.Fields, "session_id", "sessionId"),
		PayloadPath: payloadPath,
	}
}

func firstField(fields map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := fields[k]; ok {
			s := fmt.Sprint(v)
			if s != "" {
				return s
			}
		}
	}
	return ""
}
