package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type fleetExport struct {
	GeneratedAt  time.Time      `json:"generated_at"`
	RunID        string         `json:"run_id"`
	ManifestPath string         `json:"manifest_path"`
	GeneratedDir string         `json:"generated_dir"`
	Instances    []instanceView `json:"instances"`
	Configs      []configRef    `json:"configs"`
}

type configRef struct {
	InstanceID string `json:"instance_id"`
	EGMID      string `json:"egm_id"`
	HostID     string `json:"host_id"`
	Path       string `json:"path"`
}

func (s *supervisorServer) handleFleetExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		runID = time.Now().UTC().Format("20060102T150405Z")
	}
	outDir := filepath.Join(s.generatedDir, "exports")
	if s.generatedDir == "" {
		outDir = filepath.Join("generated", "exports")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	exp := fleetExport{GeneratedAt: time.Now().UTC(), RunID: runID, ManifestPath: s.manifestPath, GeneratedDir: s.generatedDir, Instances: s.instanceViews()}
	s.mu.Lock()
	for _, gen := range s.generated {
		exp.Configs = append(exp.Configs, configRef{InstanceID: gen.Instance.InstanceID, EGMID: gen.Instance.EGMID, HostID: gen.Instance.HostID, Path: gen.Path})
	}
	s.mu.Unlock()
	path := filepath.Join(outDir, runID+"_fleet_bundle.json")
	data, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "run_id": runID, "path": path, "instances": len(exp.Instances)})
}
