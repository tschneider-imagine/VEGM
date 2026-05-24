package runtime

import (
	"encoding/json"
	"sync"
	"time"
)

type sessionTimestampSet struct {
	LastCommsOnlineAt time.Time `json:"last_comms_online_at,omitempty"`
	LastKeepAliveAt   time.Time `json:"last_keep_alive_at,omitempty"`
	LastAckAt         time.Time `json:"last_ack_at,omitempty"`
}

var sessionTimestamps sync.Map

func (s *Server) recordSessionTimestamp(kind string, at time.Time) {
	if at.IsZero() || s == nil || s.cfg == nil || s.cfg.InstanceID == "" {
		return
	}
	current, _ := sessionTimestamps.Load(s.cfg.InstanceID)
	set, _ := current.(sessionTimestampSet)
	switch kind {
	case "commsOnLine":
		set.LastCommsOnlineAt = at
		set.LastAckAt = at
	case "keepAlive":
		set.LastKeepAliveAt = at
		set.LastAckAt = at
	case "ack":
		set.LastAckAt = at
	}
	sessionTimestamps.Store(s.cfg.InstanceID, set)
}

func timestampsForInstance(instanceID string) sessionTimestampSet {
	current, _ := sessionTimestamps.Load(instanceID)
	set, _ := current.(sessionTimestampSet)
	return set
}

func (r RuntimeState) MarshalJSON() ([]byte, error) {
	type runtimeStateAlias RuntimeState
	set := timestampsForInstance(r.InstanceID)
	if set.LastKeepAliveAt.IsZero() && r.LastCommandType == "keepAlive" {
		set.LastKeepAliveAt = r.LastCommandAt
	}
	if set.LastAckAt.IsZero() && r.LastAckStatus != "" {
		set.LastAckAt = r.LastCommandAt
	}
	evidence := parseEvidenceForInstance(r.InstanceID)
	return json.Marshal(struct {
		runtimeStateAlias
		LastCommsOnlineAt   time.Time `json:"last_comms_online_at,omitempty"`
		LastKeepAliveAt     time.Time `json:"last_keep_alive_at,omitempty"`
		LastAckAt           time.Time `json:"last_ack_at,omitempty"`
		G2SXMLMode          string    `json:"g2s_xml_mode,omitempty"`
		G2SXMLNamespace     string    `json:"g2s_xml_namespace,omitempty"`
		G2SXMLEGMLocation   string    `json:"g2s_xml_egm_location,omitempty"`
		LastParsedRootKind  string    `json:"last_parsed_root_kind,omitempty"`
		LastParsedClass     string    `json:"last_parsed_class,omitempty"`
		LastParsedOperation string    `json:"last_parsed_operation,omitempty"`
		LastRawRoot         string    `json:"last_raw_root,omitempty"`
		LastExpectedAck     string    `json:"last_expected_ack,omitempty"`
		LastActualAck       string    `json:"last_actual_ack,omitempty"`
	}{
		runtimeStateAlias:   runtimeStateAlias(r),
		LastCommsOnlineAt:   set.LastCommsOnlineAt,
		LastKeepAliveAt:     set.LastKeepAliveAt,
		LastAckAt:           set.LastAckAt,
		G2SXMLMode:          evidence.G2SXMLMode,
		G2SXMLNamespace:     evidence.G2SXMLNamespace,
		G2SXMLEGMLocation:   evidence.G2SXMLEGMLocation,
		LastParsedRootKind:  evidence.LastParsedRootKind,
		LastParsedClass:     evidence.LastParsedClass,
		LastParsedOperation: evidence.LastParsedOperation,
		LastRawRoot:         evidence.LastRawRoot,
		LastExpectedAck:     evidence.LastExpectedAck,
		LastActualAck:       evidence.LastActualAck,
	})
}
