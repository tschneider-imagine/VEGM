package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tschneider-imagine/VEGM/fleet"
)

func main() {
	manifestPath := flag.String("manifest", "./example.fleet.json", "path to VEGM fleet manifest JSON")
	output := flag.String("output", "summary", "output mode: summary or json")
	generateDir := flag.String("generate-dir", "./generated", "directory for generated per-instance configs")
	generate := flag.Bool("generate", true, "generate per-instance configs from the manifest")
	launch := flag.Bool("launch", false, "launch resolved VEGM instances after generation")
	waitSeconds := flag.Int("wait-seconds", 60, "max seconds to wait for launched instances to become healthy")
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
	if *generate || *launch {
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

	if !*launch {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cmds, err := launchFleet(ctx, generated)
	if err != nil {
		log.Fatal(err)
	}
	if err := waitForFleetHealthy(generated, time.Duration(*waitSeconds)*time.Second); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Fleet healthy. UI URLs:")
	for _, gen := range generated {
		fmt.Printf("- %s: http://%s:%d/ui/scenario-runner.html\n", gen.Instance.InstanceID, gen.Instance.ListenHost, gen.Instance.ControlPort)
	}
	<-ctx.Done()
	for _, cmd := range cmds {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
}
