package pack

import (
	"fmt"
	"sort"
	"strings"
)

const (
	MessagePackSchemaVersion    = "vegm.message-pack/v1"
	MessageOverlaySchemaVersion = "vegm.message-overlay/v1"
)

var (
	allowedProtocolFamilies = setOf("g2s")
	allowedTransportKinds   = setOf("soap-http")
	allowedDirections       = setOf("inbound", "outbound", "bidirectional")
	allowedMatchKinds       = setOf("message_type", "path", "namespace", "field_equals", "always")
	allowedRegistrationMode = setOf("open", "strict_match", "deny_all", "simulate_vendor_quirk")
	allowedSessionState     = setOf("idle", "online", "closed", "degraded")
	allowedHeartbeatState   = setOf("idle", "healthy", "degraded", "timeout")
	allowedTLSModes         = setOf("plaintext_lab", "tls_server_only", "strict_mtls", "mtls_no_revocation", "accept_all_lab")
	allowedOverlayOps       = setOf("set", "merge", "remove", "append")
	allowedOverlayScopes    = setOf("instance", "fleet", "session", "")
)

func ValidatePack(p *MessagePack) error {
	if p == nil {
		return fmt.Errorf("pack is nil")
	}
	var errs []string
	if p.SchemaVersion != MessagePackSchemaVersion {
		errs = append(errs, fmt.Sprintf("schema_version must be %q", MessagePackSchemaVersion))
	}
	if strings.TrimSpace(p.PackName) == "" {
		errs = append(errs, "pack_name is required")
	}
	if strings.TrimSpace(p.PackVersion) == "" {
		errs = append(errs, "pack_version is required")
	}
	if !allowedProtocolFamilies[p.Wire.ProtocolFamily] {
		errs = append(errs, fmt.Sprintf("wire.protocol_family %q is invalid", p.Wire.ProtocolFamily))
	}
	if !allowedTransportKinds[p.Wire.Envelope.Kind] {
		errs = append(errs, fmt.Sprintf("wire.envelope.kind %q is invalid", p.Wire.Envelope.Kind))
	}
	if strings.TrimSpace(p.Wire.Paths.DefaultListenerPath) == "" {
		errs = append(errs, "wire.paths.default_listener_path is required")
	}
	for _, mode := range p.Wire.Transport.TLSModesSupported {
		if !allowedTLSModes[mode] {
			errs = append(errs, fmt.Sprintf("wire.transport.tls_modes_supported contains invalid mode %q", mode))
		}
	}
	for _, dup := range duplicates(p.Wire.Transport.TLSModesSupported) {
		errs = append(errs, fmt.Sprintf("wire.transport.tls_modes_supported contains duplicate %q", dup))
	}
	if !allowedRegistrationMode[p.StateDefaults.RegistrationMode] {
		errs = append(errs, fmt.Sprintf("state_defaults.registration_mode %q is invalid", p.StateDefaults.RegistrationMode))
	}
	if !allowedSessionState[p.StateDefaults.SessionState] {
		errs = append(errs, fmt.Sprintf("state_defaults.session_state %q is invalid", p.StateDefaults.SessionState))
	}
	if !allowedHeartbeatState[p.StateDefaults.HeartbeatState] {
		errs = append(errs, fmt.Sprintf("state_defaults.heartbeat_state %q is invalid", p.StateDefaults.HeartbeatState))
	}
	if len(p.Operations) == 0 {
		errs = append(errs, "operations must not be empty")
	}
	for name, op := range p.Operations {
		if strings.TrimSpace(name) == "" {
			errs = append(errs, "operations contains empty key")
			continue
		}
		if !op.Enabled {
			continue
		}
		if !allowedDirections[op.Direction] {
			errs = append(errs, fmt.Sprintf("operations.%s.direction %q is invalid", name, op.Direction))
		}
		if len(op.Match) == 0 {
			errs = append(errs, fmt.Sprintf("operations.%s.match must not be empty", name))
		}
		if len(op.Responses) == 0 {
			errs = append(errs, fmt.Sprintf("operations.%s.responses must not be empty", name))
		}
		for i, m := range op.Match {
			if !allowedMatchKinds[m.Kind] {
				errs = append(errs, fmt.Sprintf("operations.%s.match[%d].kind %q is invalid", name, i, m.Kind))
			}
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("pack validation failed:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}

func ValidateOverlay(o *MessageOverlay) error {
	if o == nil {
		return fmt.Errorf("overlay is nil")
	}
	var errs []string
	if o.SchemaVersion != MessageOverlaySchemaVersion {
		errs = append(errs, fmt.Sprintf("schema_version must be %q", MessageOverlaySchemaVersion))
	}
	if strings.TrimSpace(o.OverlayName) == "" {
		errs = append(errs, "overlay_name is required")
	}
	if strings.TrimSpace(o.TargetPack) == "" {
		errs = append(errs, "target_pack is required")
	}
	if len(o.Changes) == 0 {
		errs = append(errs, "changes must not be empty")
	}
	for i, change := range o.Changes {
		if !allowedOverlayOps[change.Op] {
			errs = append(errs, fmt.Sprintf("changes[%d].op %q is invalid", i, change.Op))
		}
		if !strings.HasPrefix(change.Path, "/") {
			errs = append(errs, fmt.Sprintf("changes[%d].path must start with /", i))
		}
		if change.Op != "remove" && change.Value == nil {
			errs = append(errs, fmt.Sprintf("changes[%d].value is required for op %q", i, change.Op))
		}
	}
	if !allowedOverlayScopes[o.Activate.Scope] {
		errs = append(errs, fmt.Sprintf("activate.scope %q is invalid", o.Activate.Scope))
	}
	if o.Activate.TTLMS < 0 {
		errs = append(errs, "activate.ttl_ms must be >= 0")
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("overlay validation failed:\n- %s", strings.Join(errs, "\n- "))
	}
	return nil
}

func setOf(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}

func duplicates(values []string) []string {
	seen := map[string]int{}
	for _, v := range values {
		seen[v]++
	}
	var out []string
	for v, n := range seen {
		if n > 1 {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
