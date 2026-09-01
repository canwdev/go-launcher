//go:build !windows

package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func openWithDefaultHandler(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func revealFile(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}

func startTracked(path string, args []string, workDir string, proc *runningProc) error {
	cmd := exec.Command(path, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	proc.start = time.Now()
	proc.wait = cmd.Wait
	proc.stop = func() error { return cmd.Process.Kill() }
	return nil
}
