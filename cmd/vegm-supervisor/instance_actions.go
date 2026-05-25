package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (s *supervisorServer) handleInstanceAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/instances/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "expected /api/instances/{instance_id}/{action}", http.StatusBadRequest)
		return
	}
	id := parts[0]
	action := parts[1]

	var changed bool
	var err error
	switch action {
	case "start":
		changed, err = s.startOne(id)
	case "stop":
		changed = s.stopOne(id)
	case "restart":
		s.stopOne(id)
		changed, err = s.startOne(id)
	default:
		http.Error(w, fmt.Sprintf("unsupported action %q", action), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "instance_id": id, "action": action, "changed": changed, "instances": s.instanceViews()})
}
