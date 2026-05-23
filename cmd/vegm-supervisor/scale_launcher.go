package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func startVEGMProcessForScale(ctx context.Context, configPath string, logDir string) (*exec.Cmd, error) {
	if logDir == "" {
		logDir = filepath.Join("logs", "supervisor-child")
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir child log dir: %w", err)
	}
	stdoutPath := filepath.Join(logDir, "process.stdout.log")
	stderrPath := filepath.Join(logDir, "process.stderr.log")
	stdout, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open child stdout log: %w", err)
	}
	stderr, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("open child stderr log: %w", err)
	}

	cmd := buildVEGMCommand(ctx, configPath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, err
	}
	go func() {
		_ = cmd.Wait()
		_ = stdout.Close()
		_ = stderr.Close()
	}()
	return cmd, nil
}

func buildVEGMCommand(ctx context.Context, configPath string) *exec.Cmd {
	if explicit := os.Getenv("VEGM_CHILD_BINARY"); explicit != "" {
		return exec.CommandContext(ctx, explicit, "-config", configPath)
	}
	if bin := siblingVEGMBinary(); bin != "" {
		return exec.CommandContext(ctx, bin, "-config", configPath)
	}
	return exec.CommandContext(ctx, "go", "run", "./cmd/vegm", "-config", configPath)
}

func siblingVEGMBinary() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	name := "vegm"
	if runtime.GOOS == "windows" {
		name = "vegm.exe"
	}
	candidate := filepath.Join(filepath.Dir(exe), name)
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		return candidate
	}
	candidate = filepath.Join(filepath.Dir(exe), "bin", name)
	if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
		return candidate
	}
	return ""
}
