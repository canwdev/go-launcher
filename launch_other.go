//go:build !windows

package main

import (
	"os/exec"
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

func startTracked(path string, proc *runningProc) error {
	cmd := exec.Command(path)
	if err := cmd.Start(); err != nil {
		return err
	}
	proc.start = time.Now()
	proc.wait = cmd.Wait
	proc.stop = func() error { return cmd.Process.Kill() }
	return nil
}
