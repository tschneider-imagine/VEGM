package runtime

// resolveAck centralizes ACK normalization for strict + wrapped modes
func resolveAck(expected string, actual string, raw []byte, cfg SessionEngineConfig) string {
	if actual == expected {
		return actual
	}
	if cfg.AcceptWrappedG2SResponseAck {
		if actual == "g2sResponse" {
			if firstNestedAckName(raw) == expected {
				return expected
			}
		}
	}
	return actual
}
