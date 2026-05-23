package main

import (
	"fmt"
	"os/exec"
	"time"
)

const maxSupervisorRestarts = 3

type restartMeta struct {
	Desired      bool   `json:"desired"`
	RestartCount int    `json:"restart_count"`
	LastExit     string `json:"last_exit,omitempty"`
}

func (s *supervisorServer) markDesired(instanceID string, desired bool) {
	if s.restart == nil {
		s.restart = map[string]restartMeta{}
	}
	meta := s.restart[instanceID]
	meta.Desired = desired
	if !desired {
		meta.RestartCount = 0
	}
	s.restart[instanceID] = meta
}

func (s *supervisorServer) restartMeta(instanceID string) restartMeta {
	if s.restart == nil {
		return restartMeta{}
	}
	return s.restart[instanceID]
}

func (s *supervisorServer) monitorChild(instanceID string) {
	for {
		time.Sleep(2 * time.Second)
		s.mu.Lock()
		meta := s.restartMeta(instanceID)
		cmd := s.cmds[instanceID]
		if !meta.Desired {
			s.mu.Unlock()
			return
		}
		if cmd == nil || cmd.Process == nil {
			s.mu.Unlock()
			return
		}
		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			s.mu.Unlock()
			continue
		}
		meta.LastExit = cmd.ProcessState.String()
		if meta.RestartCount >= maxSupervisorRestarts {
			meta.Desired = false
			s.restart[instanceID] = meta
			delete(s.cmds, instanceID)
			s.mu.Unlock()
			return
		}
		meta.RestartCount++
		s.restart[instanceID] = meta
		gen, ok := s.generatedByID(instanceID)
		if !ok {
			delete(s.cmds, instanceID)
			s.mu.Unlock()
			return
		}
		newCmd, err := startVEGMProcessForScale(s.ctx, gen.Path, gen.Instance.LogDir)
		if err != nil {
			meta.LastExit = fmt.Sprintf("restart failed: %v", err)
			s.restart[instanceID] = meta
			delete(s.cmds, instanceID)
			s.mu.Unlock()
			return
		}
		s.cmds[instanceID] = newCmd
		s.mu.Unlock()
	}
}

func processRunning(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.Process != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited())
}
