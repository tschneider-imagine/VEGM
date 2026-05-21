package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tschneider-imagine/VEGM/fleet"
)

func launchFleet(ctx context.Context, generated []fleet.GeneratedConfig) ([]*exec.Cmd, error) {
	var cmds []*exec.Cmd
	for _, gen := range generated {
		cmd, err := startVEGMProcess(ctx, gen.Path)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

func startVEGMProcess(ctx context.Context, configPath string) (*exec.Cmd, error) {
	goExe, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("find go executable: %w", err)
	}
	cmd := exec.CommandContext(ctx, goExe, "run", "./cmd/vegm", "-config", configPath)
	repoRoot, _ := os.Getwd()
	cmd.Dir = repoRoot
	logPath := filepath.Join(filepath.Dir(configPath), filepath.Base(configPath)+".process.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open process log %q: %w", logPath, err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("start VEGM for config %q: %w", configPath, err)
	}
	return cmd, nil
}

func waitForFleetHealthy(generated []fleet.GeneratedConfig, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for _, gen := range generated {
		healthURL := fmt.Sprintf("http://%s:%d/healthz", gen.Instance.ListenHost, gen.Instance.ControlPort)
		for {
			if time.Now().After(deadline) {
				return fmt.Errorf("timed out waiting for %s to become healthy at %s", gen.Instance.InstanceID, healthURL)
			}
			ok, err := isHealthy(healthURL)
			if err == nil && ok {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func isHealthy(url string) (bool, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return true, nil
}
