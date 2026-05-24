package fleet

import "testing"

func TestGenerateConfigsIncludesG2SXMLMetadata(t *testing.T) {
	m := &Manifest{
		SchemaVersion: "vegm.fleet-manifest/v1",
		FleetName:     "g2s-xml-test",
		Defaults: Defaults{
			HostID:          "HOST-001",
			ListenHost:      "127.0.0.1",
			WirePortBase:    18443,
			ControlPortBase: 19001,
			EGMEndpoint: Endpoint{
				Scheme: "http",
				BindIP: "127.0.0.1",
				Host:   "127.0.0.1",
				Path:   "/g2s",
			},
			G2SXML: G2SXML{
				Mode:      "lab_legacy_xml",
				Namespace: "http://www.gamingstandards.com/g2s/schemas/v1.0.3",
			},
			TrustMode: "plaintext_lab",
			PackFile:  "../example.pack.json",
			LogRoot:   t.TempDir(),
		},
		Profiles: map[string]Profile{
			"baseline": {
				PackFile: "../example.pack.json",
				LogicalCommands: map[string]string{
					"audio_mute_on":  "audioMuteOn",
					"audio_mute_off": "audioMuteOff",
				},
			},
		},
		Groups: map[string]Group{
			"bank_a": {Profile: "baseline"},
		},
		Instances: []Instance{
			{
				InstanceID:  "vegm-001",
				EGMID:       "EGM-001",
				Group:       "bank_a",
				WirePort:    18443,
				ControlPort: 19001,
			},
			{
				InstanceID:  "vegm-002",
				EGMID:       "EGM-002",
				Group:       "bank_a",
				WirePort:    18444,
				ControlPort: 19002,
				G2SXML: G2SXML{
					Mode:        "xsd_g2s_message",
					EGMLocation: "127.0.0.1:18444",
				},
			},
		},
	}

	generated, err := GenerateConfigs(m, t.TempDir())
	if err != nil {
		t.Fatalf("generate configs: %v", err)
	}
	if len(generated) != 2 {
		t.Fatalf("generated = %d, want 2", len(generated))
	}

	legacy := generated[0].Config.G2SXML
	if legacy.Mode != "lab_legacy_xml" {
		t.Fatalf("legacy mode = %q", legacy.Mode)
	}
	if legacy.Namespace != "http://www.gamingstandards.com/g2s/schemas/v1.0.3" {
		t.Fatalf("legacy namespace = %q", legacy.Namespace)
	}
	if legacy.EGMLocation != "127.0.0.1:18443" {
		t.Fatalf("legacy egm_location = %q", legacy.EGMLocation)
	}
	if generated[0].Config.Notes["g2s_xml_egm_location"] != "127.0.0.1:18443" {
		t.Fatalf("legacy notes missing egm_location: %#v", generated[0].Config.Notes)
	}

	xsd := generated[1].Config.G2SXML
	if xsd.Mode != "xsd_g2s_message" {
		t.Fatalf("xsd mode = %q", xsd.Mode)
	}
	if xsd.Namespace != "http://www.gamingstandards.com/g2s/schemas/v1.0.3" {
		t.Fatalf("xsd namespace = %q", xsd.Namespace)
	}
	if xsd.EGMLocation != "127.0.0.1:18444" {
		t.Fatalf("xsd egm_location = %q", xsd.EGMLocation)
	}
}
