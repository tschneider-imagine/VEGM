package fleet

import "testing"

func TestResolveInstances_ResolvedWireEndpointCollisionFails(t *testing.T) {
	m := scaleValidationManifest()
	m.Defaults.EGMEndpoint.Port = 18443
	m.Instances = []Instance{
		{InstanceID: "vegm-001", EGMID: "EGM-001", Group: "bank_a"},
		{InstanceID: "vegm-002", EGMID: "EGM-002", Group: "bank_a"},
	}
	if _, err := ResolveInstances(m); err == nil {
		t.Fatalf("expected resolved wire endpoint collision failure")
	}
}

func TestResolveInstances_ResolvedControlEndpointCollisionFails(t *testing.T) {
	m := scaleValidationManifest()
	m.Instances = []Instance{
		{InstanceID: "vegm-001", EGMID: "EGM-001", Group: "bank_a"},
		{InstanceID: "vegm-002", EGMID: "EGM-002", Group: "bank_a", ControlPort: 19001},
	}
	if _, err := ResolveInstances(m); err == nil {
		t.Fatalf("expected resolved control endpoint collision failure")
	}
}

func TestResolveInstances_ResolvedEGMEndpointCollisionFails(t *testing.T) {
	m := scaleValidationManifest()
	m.Instances = []Instance{
		{InstanceID: "vegm-001", EGMID: "EGM-001", Group: "bank_a", WirePort: 18443, EGMEndpoint: Endpoint{Host: "same.floor.local", Port: 18443}},
		{InstanceID: "vegm-002", EGMID: "EGM-002", Group: "bank_a", WirePort: 18444, EGMEndpoint: Endpoint{Host: "same.floor.local", Port: 18443}},
	}
	if _, err := ResolveInstances(m); err == nil {
		t.Fatalf("expected resolved EGM endpoint collision failure")
	}
}

func scaleValidationManifest() *Manifest {
	return &Manifest{
		SchemaVersion: ManifestSchemaVersion,
		FleetName:     "scale-test-floor",
		Defaults: Defaults{
			HostID:          "HOST-001",
			ListenHost:      "127.0.0.1",
			WirePortBase:    18443,
			ControlPortBase: 19001,
			EGMEndpoint:     Endpoint{Scheme: "http", BindIP: "127.0.0.1", Host: "127.0.0.1", Path: "/g2s"},
			TrustMode:       "plaintext_lab",
			PackFile:        "./example.pack.json",
		},
		Profiles: map[string]Profile{
			"baseline": {PackFile: "./example.pack.json", LogicalCommands: map[string]string{"audio_mute_on": "audioMuteOn"}},
		},
		Groups: map[string]Group{"bank_a": {Profile: "baseline"}},
	}
}
