package runtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type latestEvidenceResponse struct {
	InstanceID      string                 `json:"instance_id"`
	XMLMode         string                 `json:"xml_mode,omitempty"`
	XMLNamespace    string                 `json:"xml_namespace,omitempty"`
	XMLEGMLocation  string                 `json:"xml_egm_location,omitempty"`
	ExpectedAck     string                 `json:"expected_ack,omitempty"`
	ActualAck       string                 `json:"actual_ack,omitempty"`
	ParsedRootKind  string                 `json:"parsed_root_kind,omitempty"`
	ParsedClass     string                 `json:"parsed_class,omitempty"`
	ParsedOperation string                 `json:"parsed_operation,omitempty"`
	RawRoot         string                 `json:"raw_root,omitempty"`
	Pairing         evidencePairing        `json:"pairing"`
	Transcript      string                 `json:"transcript,omitempty"`
	Request         latestEvidencePayload  `json:"request,omitempty"`
	Response        latestEvidencePayload  `json:"response,omitempty"`
	State           map[string]interface{} `json:"state,omitempty"`
}

type evidencePairing struct {
	Strategy            string `json:"strategy"`
	RequestMessageType  string `json:"request_message_type,omitempty"`
	ResponseMessageType string `json:"response_message_type,omitempty"`
	MatchedMessageType  bool   `json:"matched_message_type"`
	Warning             string `json:"warning,omitempty"`
}

type latestEvidencePayload struct {
	Path         string    `json:"path,omitempty"`
	Name         string    `json:"name,omitempty"`
	Direction    string    `json:"direction,omitempty"`
	MessageType  string    `json:"message_type,omitempty"`
	ModifiedAt   time.Time `json:"modified_at,omitempty"`
	SizeBytes    int64     `json:"size_bytes,omitempty"`
	Content      string    `json:"content,omitempty"`
	ParsedRoot   string    `json:"parsed_root,omitempty"`
	ParsedClass  string    `json:"parsed_class,omitempty"`
	ParsedOp     string    `json:"parsed_operation,omitempty"`
	ParseError   string    `json:"parse_error,omitempty"`
}

func (s *Server) handleControlEvidenceLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	evidence := parseEvidenceForInstance(s.cfg.InstanceID)
	xmlInfo := xmlModeInfoForInstance(s.cfg.InstanceID)
	stateSnapshot := map[string]interface{}{}
	s.mu.RLock()
	stateSnapshot["session_state"] = s.state.SessionState
	stateSnapshot["heartbeat_state"] = s.state.HeartbeatState
	stateSnapshot["connection_state"] = s.state.ConnectionState
	stateSnapshot["last_error"] = s.state.LastError
	stateSnapshot["last_message_type"] = s.state.LastMessageType
	stateSnapshot["last_command_type"] = s.state.LastCommandType
	stateSnapshot["last_ack_status"] = s.state.LastAckStatus
	s.mu.RUnlock()
	payloadDir := ""
	if s.logger != nil {
		payloadDir = s.logger.payloadDir
	}
	request := latestPayload(payloadDir, "outbound_request")
	response, pairing := pairedLatestResponse(payloadDir, request)
	out := latestEvidenceResponse{
		InstanceID:      s.cfg.InstanceID,
		XMLMode:         firstNonEmpty(evidence.G2SXMLMode, xmlInfo.Mode, s.cfg.G2SXML.Mode),
		XMLNamespace:    firstNonEmpty(evidence.G2SXMLNamespace, xmlInfo.Namespace, s.cfg.G2SXML.Namespace),
		XMLEGMLocation:  firstNonEmpty(evidence.G2SXMLEGMLocation, xmlInfo.EGMLocation, s.cfg.G2SXML.EGMLocation),
		ExpectedAck:     evidence.LastExpectedAck,
		ActualAck:       evidence.LastActualAck,
		ParsedRootKind:  evidence.LastParsedRootKind,
		ParsedClass:     evidence.LastParsedClass,
		ParsedOperation: evidence.LastParsedOperation,
		RawRoot:         evidence.LastRawRoot,
		Pairing:         pairing,
		Request:         request,
		Response:        response,
		State:           stateSnapshot,
	}
	out.Transcript = evidenceTranscript(out)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func pairedLatestResponse(payloadDir string, request latestEvidencePayload) (latestEvidencePayload, evidencePairing) {
	pairing := evidencePairing{RequestMessageType: request.MessageType}
	if request.MessageType != "" {
		response := latestPayloadForMessageType(payloadDir, "outbound_response", request.MessageType)
		if response.Path != "" {
			pairing.Strategy = "matched_outbound_response"
			pairing.ResponseMessageType = response.MessageType
			pairing.MatchedMessageType = true
			return response, pairing
		}
		response = latestPayloadForMessageType(payloadDir, "inbound_response", request.MessageType)
		if response.Path != "" {
			pairing.Strategy = "matched_inbound_response"
			pairing.ResponseMessageType = response.MessageType
			pairing.MatchedMessageType = true
			return response, pairing
		}
	}
	response := latestPayload(payloadDir, "outbound_response")
	if response.Path == "" {
		response = latestPayload(payloadDir, "inbound_response")
	}
	pairing.Strategy = "fallback_latest_response"
	pairing.ResponseMessageType = response.MessageType
	pairing.MatchedMessageType = request.MessageType != "" && request.MessageType == response.MessageType
	if request.Path != "" && response.Path != "" && !pairing.MatchedMessageType {
		pairing.Warning = "latest response does not match latest request message_type"
	}
	return response, pairing
}

func evidenceTranscript(out latestEvidenceResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "instance=%s xml_mode=%s\n", out.InstanceID, out.XMLMode)
	fmt.Fprintf(&b, "request=%s parsed=%s file=%s\n", out.Request.MessageType, firstNonEmpty(out.Request.ParsedOp, out.Request.ParsedRoot), out.Request.Name)
	fmt.Fprintf(&b, "response=%s parsed=%s file=%s\n", out.Response.MessageType, firstNonEmpty(out.Response.ParsedOp, out.Response.ParsedRoot), out.Response.Name)
	fmt.Fprintf(&b, "pairing=%s matched=%t", out.Pairing.Strategy, out.Pairing.MatchedMessageType)
	if out.Pairing.Warning != "" {
		fmt.Fprintf(&b, " warning=%s", out.Pairing.Warning)
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "expected_ack=%s actual_ack=%s\n", out.ExpectedAck, out.ActualAck)
	if out.State != nil {
		fmt.Fprintf(&b, "state session=%v connection=%v heartbeat=%v error=%v\n", out.State["session_state"], out.State["connection_state"], out.State["heartbeat_state"], out.State["last_error"])
	}
	return b.String()
}

func latestPayload(payloadDir, direction string) latestEvidencePayload {
	return latestPayloadForMessageType(payloadDir, direction, "")
}

func latestPayloadForMessageType(payloadDir, direction, messageType string) latestEvidencePayload {
	if payloadDir == "" {
		return latestEvidencePayload{}
	}
	entries, err := os.ReadDir(payloadDir)
	if err != nil {
		return latestEvidencePayload{ParseError: err.Error()}
	}
	var files []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.Contains(name, "_"+direction+"_") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		if messageType != "" && messageTypeFromPayloadName(name, direction) != messageType {
			continue
		}
		files = append(files, entry)
	}
	sort.Slice(files, func(i, j int) bool {
		ii, _ := files[i].Info()
		jj, _ := files[j].Info()
		return ii.ModTime().After(jj.ModTime())
	})
	if len(files) == 0 {
		return latestEvidencePayload{}
	}
	entry := files[0]
	info, _ := entry.Info()
	path := filepath.Join(payloadDir, entry.Name())
	data, err := os.ReadFile(path)
	out := latestEvidencePayload{
		Path:        path,
		Name:        entry.Name(),
		Direction:   direction,
		MessageType: messageTypeFromPayloadName(entry.Name(), direction),
		ModifiedAt:  info.ModTime(),
		SizeBytes:   info.Size(),
	}
	if err != nil {
		out.ParseError = err.Error()
		return out
	}
	out.Content = string(data)
	if parsed, err := ParseG2SEnvelope(data); err == nil {
		out.ParsedRoot = firstNonEmpty(parsed.OperationName, parsed.RawRoot)
		out.ParsedClass = parsed.ClassName
		out.ParsedOp = parsed.OperationName
	} else {
		out.ParsedRoot = ParseMessage(data).RootLocalName
		out.ParseError = err.Error()
	}
	return out
}

func messageTypeFromPayloadName(name, direction string) string {
	marker := "_" + direction + "_"
	idx := strings.Index(name, marker)
	if idx < 0 {
		return ""
	}
	msg := strings.TrimSuffix(name[idx+len(marker):], ".xml")
	if msg == "" {
		return ""
	}
	return fmt.Sprintf("%s", msg)
}
