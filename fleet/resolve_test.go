package fleet

import "testing"

func TestResolveInstances_ExampleShape(t *testing.T) {
	m := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		FleetName:     "test-floor",
		Defaults: Defaults{
			ListenHost:      "127.0.0.1",
			WirePortBase:    18443,
			ControlPortBase: 19001,
			TrustMode:       "plaintext_lab",
			PackFile:        "./example.pack.json",
			LogRoot:         "./logs/test-floor",
			StorageBackend:  "sqlite",
			SQLiteRoot:      "./logs/test-floor",
			Heartbeat:       map[string]any{"interval_ms": 5000},
			NormalizedState: map[string]any{"audio_state": "normal"},
		},
		Profiles: map[string]Profile{
			"baseline": {
				PackFile: "./example.pack.json",
				LogicalCommands: map[string]string{
					"audio_mute_on": "audioMuteOn",
				},
			},
		},
		Groups: map[string]Group{
			"bank_a": {Profile: "baseline"},
		},
		Instances: []Instance{
			{InstanceID: "vegm-001", EGMID: "EGM-001", Group: "bank_a"},
			{InstanceID: "vegm-002", EGMID: "EGM-002", Group: "bank_a", WirePort: 18499, ControlPort: 19111},
		},
	}

	out, err := ResolveInstances(m)
	if err != nil {
		t.Fatalf("ResolveInstances failed: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 effective instances, got %d", len(out))
	}
	if out[0].WirePort != 18443 || out[0].ControlPort != 19001 {
		t.Fatalf("expected defaults to assign base ports, got wire=%d control=%d", out[0].WirePort, out[0].ControlPort)
	}
	if out[1].WirePort != 18499 || out[1].ControlPort != 19111 {
		t.Fatalf("expected explicit ports to win, got wire=%d control=%d", out[1].WirePort, out[1].ControlPort)
	}
	if out[0].StorageBackend != "sqlite" {
		t.Fatalf("expected sqlite storage backend, got %q", out[0].StorageBackend)
	}
	if out[0].LogicalCommands["audio_mute_on"] != "audioMuteOn" {
		t.Fatalf("expected logical command mapping to survive resolution")
	}
	if out[0].SQLitePath == "" {
		t.Fatalf("expected resolved sqlite path")
	}
}

func TestValidateManifest_DuplicateInstanceIDFails(t *testing.T) {
	m := &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		FleetName:     "test-floor",
		Profiles: map[string]Profile{
			"baseline": {PackFile: "./example.pack.json", LogicalCommands: map[string]string{"audio_mute_on": "audioMuteOn"}},
		},
		Groups: map[string]Group{
			"bank_a": {Profile: "baseline"},
		},
		Instances: []Instance{
			{InstanceID: "vegm-001", EGMID: "EGM-001", Group: "bank_a"},
			{InstanceID: "vegm-001", EGMID: "EGM-002", Group: "bank_a"},
		},
	}
	if err := ValidateManifest(m); err == nil {
		t.Fatalf("expected duplicate instance_id validation failure")
	}
}
