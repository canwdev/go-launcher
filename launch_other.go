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

// startProcess launches path with args in workDir and returns the running
// command; the caller decides how to wait/cleanup.
func startProcess(path string, args []string, workDir string) (*exec.Cmd, error) {
	cmd := exec.Command(path, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// startUntracked launches path without returning a process handle, so nothing
// can Stop or time it. The child is reaped in the background so it does not
// become a zombie.
func startUntracked(path string, args []string, workDir string) error {
	cmd, err := startProcess(path, args, workDir)
	if err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func startTracked(path string, args []string, workDir string, proc *runningProc) error {
	cmd, err := startProcess(path, args, workDir)
	if err != nil {
		return err
	}
	proc.start = time.Now()
	proc.wait = cmd.Wait
	proc.stop = func() error { return cmd.Process.Kill() }
	return nil
}
