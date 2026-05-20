package pack

import "sort"

type Summary struct {
	PackName            string   `json:"pack_name"`
	PackVersion         string   `json:"pack_version"`
	ProtocolFamily      string   `json:"protocol_family"`
	DefaultListenerPath string   `json:"default_listener_path"`
	Namespaces          []string `json:"namespaces"`
	Operations          []string `json:"operations"`
	TLSModes            []string `json:"tls_modes"`
	AllowedHostIDs      []string `json:"allowed_host_ids"`
	RegisteredHostIDs   []string `json:"registered_host_ids"`
}

func (p *MessagePack) Summary() Summary {
	var namespaces []string
	for k := range p.Wire.Namespaces {
		namespaces = append(namespaces, k)
	}
	sort.Strings(namespaces)
	var ops []string
	for k := range p.Operations {
		ops = append(ops, k)
	}
	sort.Strings(ops)
	var registered []string
	for _, h := range p.StateDefaults.RegisteredHosts {
		registered = append(registered, h.HostID)
	}
	sort.Strings(registered)
	allowed := append([]string(nil), p.StateDefaults.AllowedHostIDs...)
	sort.Strings(allowed)
	tlsModes := append([]string(nil), p.Wire.Transport.TLSModesSupported...)
	sort.Strings(tlsModes)
	return Summary{
		PackName:            p.PackName,
		PackVersion:         p.PackVersion,
		ProtocolFamily:      p.Wire.ProtocolFamily,
		DefaultListenerPath: p.Wire.Paths.DefaultListenerPath,
		Namespaces:          namespaces,
		Operations:          ops,
		TLSModes:            tlsModes,
		AllowedHostIDs:      allowed,
		RegisteredHostIDs:   registered,
	}
}
