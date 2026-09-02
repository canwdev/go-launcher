//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// singleInstanceLockName is the file name (inside the OS temp dir) used as an
// advisory lock to enforce single-instance behaviour on non-Windows platforms.
const singleInstanceLockName = ".go-launcher-singleton.lock"

// instanceLock is the open lock file held for the lifetime of the process.
var instanceLock *os.File

// acquireSingleton claims the single-instance advisory lock via flock(). It
// returns true when this process is the only running copy, and false when
// another copy already holds the lock. The lock is released automatically when
// the process exits (the kernel drops flock locks on fd close / process exit),
// so a crashed instance never leaves a stale lock behind.
func acquireSingleton() (bool, error) {
	path := filepath.Join(os.TempDir(), singleInstanceLockName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return false, nil
		}
		return false, fmt.Errorf("flock %s: %w", path, err)
	}
	instanceLock = f
	return true, nil
}

// releaseSingleton drops the advisory lock and closes the lock file.
func releaseSingleton() {
	if instanceLock != nil {
		_ = syscall.Flock(int(instanceLock.Fd()), syscall.LOCK_UN)
		_ = instanceLock.Close()
		instanceLock = nil
	}
}

// activateExistingInstance is a no-op on non-Windows platforms.
func activateExistingInstance() {}
