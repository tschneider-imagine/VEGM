package pack

import (
	"encoding/json"
	"fmt"
)

// ApplyOverlay currently supports top-level merge semantics for simple control-plane changes.
// It keeps the repo compileable and can be expanded to full JSON-pointer mutation later.
func ApplyOverlay(base *MessagePack, overlay *MessageOverlay) (*MessagePack, error) {
	if base == nil {
		return nil, fmt.Errorf("base pack is nil")
	}
	if overlay == nil {
		return nil, fmt.Errorf("overlay is nil")
	}
	if err := ValidatePack(base); err != nil {
		return nil, fmt.Errorf("base pack invalid: %w", err)
	}
	if err := ValidateOverlay(overlay); err != nil {
		return nil, err
	}
	if overlay.TargetPack != base.PackName {
		return nil, fmt.Errorf("overlay target_pack %q does not match base pack %q", overlay.TargetPack, base.PackName)
	}
	data, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("marshal base pack: %w", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, fmt.Errorf("decode base pack: %w", err)
	}
	for _, change := range overlay.Changes {
		if change.Path == "/timers/artificial_response_delay_ms" && change.Op == "set" {
			if generic["timers"] == nil {
				generic["timers"] = map[string]any{}
			}
			if timers, ok := generic["timers"].(map[string]any); ok {
				timers["artificial_response_delay_ms"] = change.Value
			}
		}
		if change.Path == "/state_defaults/registration_mode" && change.Op == "set" {
			if generic["state_defaults"] == nil {
				generic["state_defaults"] = map[string]any{}
			}
			if sd, ok := generic["state_defaults"].(map[string]any); ok {
				sd["registration_mode"] = change.Value
			}
		}
	}
	out, err := json.Marshal(generic)
	if err != nil {
		return nil, fmt.Errorf("marshal merged pack: %w", err)
	}
	return ParsePack(out)
}
