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
	return json.Marshal(struct {
		runtimeStateAlias
		LastCommsOnlineAt time.Time `json:"last_comms_online_at,omitempty"`
		LastKeepAliveAt   time.Time `json:"last_keep_alive_at,omitempty"`
		LastAckAt         time.Time `json:"last_ack_at,omitempty"`
	}{
		runtimeStateAlias: runtimeStateAlias(r),
		LastCommsOnlineAt: set.LastCommsOnlineAt,
		LastKeepAliveAt:   set.LastKeepAliveAt,
		LastAckAt:         set.LastAckAt,
	})
}
