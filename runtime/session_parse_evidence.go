package runtime

import "sync"

type sessionParseEvidenceSet struct {
	LastParsedRootKind  string
	LastParsedClass     string
	LastParsedOperation string
	LastRawRoot         string
	LastExpectedAck     string
	LastActualAck       string
}

var sessionParseEvidence sync.Map

func (s *Server) recordParsedResponseEvidence(expectedAck string, parsed ParsedG2SEnvelope) {
	if s == nil || s.cfg == nil || s.cfg.InstanceID == "" {
		return
	}
	set := sessionParseEvidenceSet{
		LastParsedRootKind:  parsed.RootKind,
		LastParsedClass:     parsed.ClassName,
		LastParsedOperation: parsed.OperationName,
		LastRawRoot:         parsed.RawRoot,
		LastExpectedAck:     expectedAck,
		LastActualAck:       firstNonEmpty(parsed.OperationName, parsed.RawRoot),
	}
	sessionParseEvidence.Store(s.cfg.InstanceID, set)
}

func parseEvidenceForInstance(instanceID string) sessionParseEvidenceSet {
	current, _ := sessionParseEvidence.Load(instanceID)
	set, _ := current.(sessionParseEvidenceSet)
	return set
}
