package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/tschneider-imagine/VEGM/fleet"
)

type addInstanceRequest struct {
	Remove         bool               `json:"remove,omitempty"`
	InstanceID     string             `json:"instance_id"`
	EGMID          string             `json:"egm_id"`
	HostID         string             `json:"host_id,omitempty"`
	Group          string             `json:"group"`
	WirePort       int                `json:"wire_port,omitempty"`
	ControlPort    int                `json:"control_port,omitempty"`
	BindHost       string             `json:"bind_host,omitempty"`
	EGMEndpoint    fleet.Endpoint     `json:"egm_endpoint,omitempty"`
	HostEndpoint   fleet.HostEndpoint `json:"host_endpoint,omitempty"`
	G2SXML         fleet.G2SXML       `json:"g2s_xml,omitempty"`
	AdvertisedHost string             `json:"advertised_host,omitempty"`
	AdvertisedIP   string             `json:"advertised_ip,omitempty"`
}

type removeInstanceRequest struct {
	InstanceID string `json:"instance_id"`
}

func (s *supervisorServer) handleFleetInstancesAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in addInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if in.Remove {
		id := strings.TrimSpace(in.InstanceID)
		if id == "" {
			http.Error(w, "instance_id is required", http.StatusBadRequest)
			return
		}
		if err := s.removeFleetInstance(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "removed": id, "instances": s.instanceViews()})
		return
	}
	inst := fleet.Instance{
		InstanceID:     strings.TrimSpace(in.InstanceID),
		EGMID:          strings.TrimSpace(in.EGMID),
		HostID:         strings.TrimSpace(in.HostID),
		Group:          strings.TrimSpace(in.Group),
		WirePort:       in.WirePort,
		ControlPort:    in.ControlPort,
		BindHost:       strings.TrimSpace(in.BindHost),
		EGMEndpoint:    in.EGMEndpoint,
		HostEndpoint:   in.HostEndpoint,
		G2SXML:         defaultFleetG2SXML(in.G2SXML, in.EGMEndpoint),
		AdvertisedHost: strings.TrimSpace(in.AdvertisedHost),
		AdvertisedIP:   strings.TrimSpace(in.AdvertisedIP),
	}
	if inst.InstanceID == "" || inst.EGMID == "" || inst.Group == "" {
		http.Error(w, "instance_id, egm_id, and group are required", http.StatusBadRequest)
		return
	}
	if err := s.addFleetInstance(inst); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instance_id": inst.InstanceID, "instances": s.instanceViews()})
}

func (s *supervisorServer) handleFleetInstancesRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in removeInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(in.InstanceID)
	if id == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}
	if err := s.removeFleetInstance(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "removed": id, "instances": s.instanceViews()})
}

func (s *supervisorServer) addFleetInstance(inst fleet.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := fleet.LoadManifest(s.manifestPath)
	if err != nil {
		return err
	}
	for _, existing := range m.Instances {
		if existing.InstanceID == inst.InstanceID {
			return fmt.Errorf("instance_id %q already exists", inst.InstanceID)
		}
		if existing.EGMID == inst.EGMID {
			return fmt.Errorf("egm_id %q already exists", inst.EGMID)
		}
	}
	if _, ok := m.Groups[inst.Group]; !ok {
		return fmt.Errorf("group %q not found", inst.Group)
	}
	m.Instances = append(m.Instances, inst)
	generated, err := validateAndGenerateFleet(m, s.generatedDir)
	if err != nil {
		return err
	}
	if err := writeManifest(s.manifestPath, m); err != nil {
		return err
	}
	s.generated = generated
	return nil
}

func (s *supervisorServer) removeFleetInstance(instanceID string) error {
	s.stopOne(instanceID)
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := fleet.LoadManifest(s.manifestPath)
	if err != nil {
		return err
	}
	found := false
	out := make([]fleet.Instance, 0, len(m.Instances))
	for _, inst := range m.Instances {
		if inst.InstanceID == instanceID {
			found = true
			continue
		}
		out = append(out, inst)
	}
	if !found {
		return fmt.Errorf("instance %q not found", instanceID)
	}
	m.Instances = out
	generated, err := validateAndGenerateFleet(m, s.generatedDir)
	if err != nil {
		return err
	}
	if err := writeManifest(s.manifestPath, m); err != nil {
		return err
	}
	s.generated = generated
	delete(s.cmds, instanceID)
	delete(s.restart, instanceID)
	return nil
}

func validateAndGenerateFleet(m *fleet.Manifest, generatedDir string) ([]fleet.GeneratedConfig, error) {
	if _, err := fleet.ResolveInstances(m); err != nil {
		return nil, err
	}
	return fleet.GenerateConfigs(m, generatedDir)
}

func writeManifest(path string, m *fleet.Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func defaultFleetG2SXML(in fleet.G2SXML, ep fleet.Endpoint) fleet.G2SXML {
	if in.Mode == "" {
		in.Mode = "lab_legacy_xml"
	}
	if in.Namespace == "" {
		in.Namespace = "http://www.gamingstandards.com/g2s/schemas/v1.0.3"
	}
	if in.EGMLocation == "" && ep.Host != "" && ep.Port > 0 {
		in.EGMLocation = fmt.Sprintf("%s:%d", ep.Host, ep.Port)
	}
	return in
}
