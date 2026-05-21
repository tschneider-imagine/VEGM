package fleet

import (
	"fmt"
	"path/filepath"
)

func ResolveInstances(m *Manifest) ([]EffectiveInstance, error) {
	if err := ValidateManifest(m); err != nil {
		return nil, err
	}
	var out []EffectiveInstance
	for i, inst := range m.Instances {
		eff, err := resolveInstance(m, i, inst)
		if err != nil {
			return nil, err
		}
		out = append(out, eff)
	}
	return out, nil
}

func resolveInstance(m *Manifest, idx int, inst Instance) (EffectiveInstance, error) {
	group := m.Groups[inst.Group]
	profile := m.Profiles[group.Profile]
	listenHost := firstNonEmpty(inst.BindHost, m.Defaults.ListenHost, "127.0.0.1")
	wirePort := inst.WirePort
	if wirePort == 0 {
		wirePort = m.Defaults.WirePortBase + idx
	}
	controlPort := inst.ControlPort
	if controlPort == 0 {
		controlPort = m.Defaults.ControlPortBase + idx
	}
	trustMode := m.Defaults.TrustMode
	if trustMode == "" {
		trustMode = "plaintext_lab"
	}
	packFile := firstNonEmpty(profile.PackFile, m.Defaults.PackFile)
	if packFile == "" {
		return EffectiveInstance{}, fmt.Errorf("no pack file resolved for instance %q", inst.InstanceID)
	}
	overlays := append([]string(nil), m.Defaults.OverlayFiles...)
	overlays = append(overlays, profile.OverlayFiles...)
	logDir := firstNonEmpty(inst.LogDir)
	if logDir == "" {
		root := firstNonEmpty(m.Defaults.LogRoot, "./logs")
		logDir = filepath.Join(root, inst.InstanceID)
	}
	storageBackend := firstNonEmpty(m.Defaults.StorageBackend, "noop")
	sqlitePath := firstNonEmpty(inst.SQLitePath)
	if sqlitePath == "" && storageBackend == "sqlite" {
		root := firstNonEmpty(m.Defaults.SQLiteRoot, logDir)
		sqlitePath = filepath.Join(root, inst.InstanceID+"-index.db")
	}
	heartbeat := mergeMaps(m.Defaults.Heartbeat, profile.Heartbeat)
	normalizedState := mergeMaps(m.Defaults.NormalizedState, profile.NormalizedState)
	faults := mergeMaps(m.Defaults.Faults, profile.Faults)
	if group.Overrides != nil {
		if v, ok := group.Overrides["heartbeat"].(map[string]any); ok {
			heartbeat = mergeMaps(heartbeat, v)
		}
		if v, ok := group.Overrides["normalized_state"].(map[string]any); ok {
			normalizedState = mergeMaps(normalizedState, v)
		}
		if v, ok := group.Overrides["faults"].(map[string]any); ok {
			faults = mergeMaps(faults, v)
		}
	}
	if inst.Overrides != nil {
		if v, ok := inst.Overrides["heartbeat"].(map[string]any); ok {
			heartbeat = mergeMaps(heartbeat, v)
		}
		if v, ok := inst.Overrides["normalized_state"].(map[string]any); ok {
			normalizedState = mergeMaps(normalizedState, v)
		}
		if v, ok := inst.Overrides["faults"].(map[string]any); ok {
			faults = mergeMaps(faults, v)
		}
	}
	return EffectiveInstance{
		InstanceID:      inst.InstanceID,
		EGMID:           inst.EGMID,
		Group:           inst.Group,
		Profile:         group.Profile,
		Manufacturer:    profile.Manufacturer,
		ListenHost:      listenHost,
		WirePort:        wirePort,
		ControlPort:     controlPort,
		TrustMode:       trustMode,
		PackFile:        packFile,
		OverlayFiles:    overlays,
		LogDir:          logDir,
		StorageBackend:  storageBackend,
		SQLitePath:      sqlitePath,
		LogicalCommands: copyStringMap(profile.LogicalCommands),
		Heartbeat:       heartbeat,
		NormalizedState: normalizedState,
		Faults:          faults,
		CertFile:        inst.CertFile,
		KeyFile:         inst.KeyFile,
		CAFile:          inst.CAFile,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func mergeMaps(base, overlay map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
