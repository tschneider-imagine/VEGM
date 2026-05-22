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
	listenHost := firstNonEmpty(inst.BindHost, inst.EGMEndpoint.BindIP, profile.EGMEndpoint.BindIP, m.Defaults.EGMEndpoint.BindIP, m.Defaults.ListenHost, "127.0.0.1")
	wirePort := inst.WirePort
	if wirePort == 0 {
		wirePort = firstNonZero(inst.EGMEndpoint.Port, profile.EGMEndpoint.Port, m.Defaults.EGMEndpoint.Port)
	}
	if wirePort == 0 {
		wirePort = m.Defaults.WirePortBase + idx
	}
	controlPort := inst.ControlPort
	if controlPort == 0 {
		controlPort = m.Defaults.ControlPortBase + idx
	}
	hostID := firstNonEmpty(inst.HostID, profile.HostID, m.Defaults.HostID, "HOST-001")
	trustMode := m.Defaults.TrustMode
	if trustMode == "" {
		trustMode = "plaintext_lab"
	}
	packFile := firstNonEmpty(profile.PackFile, m.Defaults.PackFile)
	if packFile == "" {
		return EffectiveInstance{}, fmt.Errorf("no pack file resolved for instance %q", inst.InstanceID)
	}
	egmEndpoint := mergeEndpoint(m.Defaults.EGMEndpoint, profile.EGMEndpoint)
	egmEndpoint = mergeEndpoint(egmEndpoint, inst.EGMEndpoint)
	if egmEndpoint.Scheme == "" {
		if trustMode == "plaintext_lab" || trustMode == "" {
			egmEndpoint.Scheme = "http"
		} else {
			egmEndpoint.Scheme = "https"
		}
	}
	if egmEndpoint.BindIP == "" {
		egmEndpoint.BindIP = listenHost
	}
	if egmEndpoint.Host == "" {
		egmEndpoint.Host = firstNonEmpty(inst.AdvertisedHost, inst.AdvertisedIP, egmEndpoint.BindIP)
	}
	if egmEndpoint.Port == 0 {
		egmEndpoint.Port = wirePort
	}
	if egmEndpoint.Path == "" {
		egmEndpoint.Path = "/g2s"
	}
	hostEndpoint := mergeHostEndpoint(m.Defaults.HostEndpoint, profile.HostEndpoint)
	hostEndpoint = mergeHostEndpoint(hostEndpoint, inst.HostEndpoint)
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
	advertisedHost := firstNonEmpty(inst.AdvertisedHost, profile.AdvertisedHost, m.Defaults.AdvertisedHost, egmEndpoint.Host)
	advertisedIP := firstNonEmpty(inst.AdvertisedIP, profile.AdvertisedIP, m.Defaults.AdvertisedIP)
	dnsServers := firstNonEmptySlice(inst.DNSServers, profile.DNSServers, m.Defaults.DNSServers)
	subnetMask := firstNonEmpty(inst.SubnetMask, profile.SubnetMask, m.Defaults.SubnetMask)
	gateway := firstNonEmpty(inst.Gateway, profile.Gateway, m.Defaults.Gateway)
	serverName := firstNonEmpty(inst.ServerName, profile.ServerName, m.Defaults.ServerName)
	certFile := firstNonEmpty(inst.CertFile, profile.CertFile, m.Defaults.CertFile)
	keyFile := firstNonEmpty(inst.KeyFile, profile.KeyFile, m.Defaults.KeyFile)
	caFile := firstNonEmpty(inst.CAFile, profile.CAFile, m.Defaults.CAFile)
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
		HostID:          hostID,
		Group:           inst.Group,
		Profile:         group.Profile,
		Manufacturer:    profile.Manufacturer,
		ListenHost:      listenHost,
		WirePort:        wirePort,
		ControlPort:     controlPort,
		EGMEndpoint:     egmEndpoint,
		HostEndpoint:    hostEndpoint,
		AdvertisedHost:  advertisedHost,
		AdvertisedIP:    advertisedIP,
		DNSServers:      dnsServers,
		SubnetMask:      subnetMask,
		Gateway:         gateway,
		ServerName:      serverName,
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
		CertFile:        certFile,
		KeyFile:         keyFile,
		CAFile:          caFile,
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

func firstNonZero(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

func firstNonEmptySlice(values ...[]string) []string {
	for _, v := range values {
		if len(v) > 0 {
			out := make([]string, len(v))
			copy(out, v)
			return out
		}
	}
	return nil
}

func mergeEndpoint(base, overlay Endpoint) Endpoint {
	out := base
	if overlay.Scheme != "" { out.Scheme = overlay.Scheme }
	if overlay.BindIP != "" { out.BindIP = overlay.BindIP }
	if overlay.Host != "" { out.Host = overlay.Host }
	if overlay.Port != 0 { out.Port = overlay.Port }
	if overlay.Path != "" { out.Path = overlay.Path }
	return out
}

func mergeHostEndpoint(base, overlay HostEndpoint) HostEndpoint {
	out := base
	if overlay.URL != "" { out.URL = overlay.URL }
	return out
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
