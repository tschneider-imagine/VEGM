package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tschneider-imagine/VEGM/fleet"
)

func main() {
	manifestPath := flag.String("manifest", "./example.fleet.json", "path to VEGM fleet manifest JSON")
	output := flag.String("output", "summary", "output mode: summary or json")
	generateDir := flag.String("generate-dir", "./generated", "directory for generated per-instance configs")
	generate := flag.Bool("generate", true, "generate per-instance configs from the manifest")
	serve := flag.Bool("serve", false, "serve the supervisor web UI and lifecycle API")
	bind := flag.String("bind", "127.0.0.1:18081", "supervisor HTTP bind address when -serve is used")
	flag.Parse()

	m, err := fleet.LoadManifest(*manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	effective, err := fleet.ResolveInstances(m)
	if err != nil {
		log.Fatal(err)
	}

	var generated []fleet.GeneratedConfig
	if *generate || *serve {
		generated, err = fleet.GenerateConfigs(m, *generateDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	switch *output {
	case "summary":
		fmt.Printf("Fleet: %s\n", m.FleetName)
		if m.Description != "" {
			fmt.Printf("Description: %s\n", m.Description)
		}
		fmt.Printf("Instances: %d\n", len(effective))
		if len(generated) > 0 {
			fmt.Printf("Generated configs: %s\n", *generateDir)
		}
		if *serve {
			fmt.Printf("Supervisor UI: http://%s/ui/supervisor.html\n", *bind)
		}
		fmt.Println()
		for _, inst := range effective {
			fmt.Printf("- %s (%s)\n", inst.InstanceID, inst.EGMID)
			fmt.Printf("  group=%s profile=%s manufacturer=%s\n", inst.Group, inst.Profile, inst.Manufacturer)
			fmt.Printf("  wire=%s:%d control=%s:%d\n", inst.ListenHost, inst.WirePort, inst.ListenHost, inst.ControlPort)
			fmt.Printf("  trust=%s pack=%s\n", inst.TrustMode, inst.PackFile)
			fmt.Printf("  log_dir=%s\n", inst.LogDir)
			if inst.StorageBackend == "sqlite" {
				fmt.Printf("  sqlite=%s\n", inst.SQLitePath)
			}
			fmt.Printf("  ui=http://%s:%d/ui/scenario-runner.html\n", inst.ListenHost, inst.ControlPort)
			fmt.Println()
		}
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{"fleet_name": m.FleetName, "instances": effective}); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported output mode %q", *output)
	}

	if *serve {
		server := newSupervisorServer(*manifestPath, *generateDir, generated)
		if err := serveSupervisor(*bind, server); err != nil {
			log.Fatal(err)
		}
		select {}
	}
}
